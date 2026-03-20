package service

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
	"welink/backend/model"
	"welink/backend/pkg/db"
)

const (
	relationRecentDays         = 7
	relationPreviousDays       = 30
	relationColdWindowDays     = 90
	relationReplyMaxSeconds    = 24 * 3600
	relationLongTextThreshold  = 36
	relationOverviewLimit      = 8
	relationEvidenceLimit      = 5
	relationMinMessages        = 20
	relationMinTextMessages    = 8
	relationMinSessions        = 3
	controversyMinMessages     = 28
	controversyMinReplySamples = 3
	confidenceFreshDays1       = 30
	confidenceFreshDays2       = 90
	confidenceFreshDays3       = 180
)

var taskKeywords = []string{
	"付款", "转账", "支付", "报销", "文件", "地址", "定位", "帮忙", "发我", "给我", "发下", "发一下",
	"弄一下", "弄个", "麻烦", "资料", "表格", "合同", "发票", "收款", "二维码", "链接", "快递",
	"寄", "拿一下", "带一下", "处理", "安排", "确认", "提交", "下载", "截图", "转发", "接龙",
}

type RelationOverview struct {
	Warming    []RelationOverviewItem `json:"warming"`
	Cooling    []RelationOverviewItem `json:"cooling"`
	Initiative []RelationOverviewItem `json:"initiative"`
	FastReply  []RelationOverviewItem `json:"fast_reply"`
}

type RelationOverviewGrouped struct {
	All    RelationOverview `json:"all"`
	Male   RelationOverview `json:"male"`
	Female RelationOverview `json:"female"`
}

type RelationOverviewItem struct {
	Username         string   `json:"username"`
	Name             string   `json:"name"`
	Gender           string   `json:"gender"`
	Score            float64  `json:"score"`
	Confidence       float64  `json:"confidence,omitempty"`
	StaleHint        string   `json:"stale_hint,omitempty"`
	ConfidenceReason string   `json:"confidence_reason,omitempty"`
	Why              string   `json:"why,omitempty"`
	EvidencePreview  []string `json:"evidence_preview,omitempty"`
	RankScore        float64  `json:"-"`
}

type ControversyOverview struct {
	Simp         []ControversyItem `json:"simp"`
	Ambiguity    []ControversyItem `json:"ambiguity"`
	Faded        []ControversyItem `json:"faded"`
	ToolPerson   []ControversyItem `json:"tool_person"`
	ColdViolence []ControversyItem `json:"cold_violence"`
}

type ControversyOverviewGrouped struct {
	All    ControversyOverview `json:"all"`
	Male   ControversyOverview `json:"male"`
	Female ControversyOverview `json:"female"`
}

type ControversyItem struct {
	Username         string             `json:"username"`
	Name             string             `json:"name"`
	Gender           string             `json:"gender"`
	Label            string             `json:"label"`
	Score            float64            `json:"score"`
	Confidence       float64            `json:"confidence"`
	StaleHint        string             `json:"stale_hint,omitempty"`
	ConfidenceReason string             `json:"confidence_reason,omitempty"`
	Why              string             `json:"why"`
	EvidencePreview  []RelationEvidence `json:"evidence_preview"`
	RankScore        float64            `json:"-"`
}

type ControversyDetail struct {
	Username            string               `json:"username"`
	Name                string               `json:"name"`
	ControversialLabels []ControversialLabel `json:"controversial_labels"`
}

type RelationDetail struct {
	Username            string                  `json:"username"`
	Name                string                  `json:"name"`
	Confidence          float64                 `json:"confidence,omitempty"`
	StaleHint           string                  `json:"stale_hint,omitempty"`
	ConfidenceReason    string                  `json:"confidence_reason,omitempty"`
	StageTimeline       []RelationStageItem     `json:"stage_timeline"`
	ObjectiveSummary    string                  `json:"objective_summary"`
	PlayfulSummary      string                  `json:"playful_summary"`
	Metrics             []RelationMetricItem    `json:"metrics"`
	ControversialLabels []ControversialLabel    `json:"controversial_labels"`
	EvidenceGroups      []RelationEvidenceGroup `json:"evidence_groups"`
}

type RelationStageItem struct {
	ID        string  `json:"id,omitempty"`
	Stage     string  `json:"stage"`
	StartDate string  `json:"start_date"`
	EndDate   string  `json:"end_date,omitempty"`
	Summary   string  `json:"summary,omitempty"`
	Score     float64 `json:"score,omitempty"`
}

type RelationMetricItem struct {
	Key      string  `json:"key"`
	Label    string  `json:"label"`
	Value    string  `json:"value"`
	SubValue string  `json:"sub_value,omitempty"`
	Trend    string  `json:"trend,omitempty"`
	Hint     string  `json:"hint,omitempty"`
	RawValue float64 `json:"raw_value,omitempty"`
}

type RelationEvidenceGroup struct {
	ID       string             `json:"id,omitempty"`
	Title    string             `json:"title"`
	Subtitle string             `json:"subtitle,omitempty"`
	Items    []RelationEvidence `json:"items"`
}

type RelationEvidence struct {
	Date    string `json:"date"`
	Time    string `json:"time"`
	Content string `json:"content"`
	IsMine  bool   `json:"is_mine"`
	Reason  string `json:"reason"`
}

type ControversialLabel struct {
	Label            string              `json:"label"`
	Score            float64             `json:"score"`
	Confidence       float64             `json:"confidence"`
	StaleHint        string              `json:"stale_hint,omitempty"`
	ConfidenceReason string              `json:"confidence_reason,omitempty"`
	Why              string              `json:"why"`
	Metrics          []ControversyMetric `json:"metrics,omitempty"`
	EvidenceGroups   []RelationEvidence  `json:"evidence_groups,omitempty"`
}

type ControversyMetric struct {
	Key          string  `json:"key"`
	Label        string  `json:"label"`
	Value        float64 `json:"value"`
	DisplayValue string  `json:"display_value,omitempty"`
}

type relationMessage struct {
	Ts      int64
	Content string
	IsMine  bool
	Type    int
}

type relationProfile struct {
	Username             string
	Name                 string
	Gender               model.Gender
	ContactKind          string
	IsDeleted            bool
	IsBiz                bool
	LikelyMarketing      bool
	IsLikelyAlt          bool
	Messages             []relationMessage
	TotalMessages        int
	MyMessages           int
	TheirMessages        int
	TextMessages         int
	LateNightMessages    int
	LongTextMessages     int
	TaskMessages         int
	TotalSessions        int
	MyInitiatedSessions  int
	Recent7Messages      int
	Previous30Messages   int
	Recent7Sessions      int
	Previous30Sessions   int
	ReplySamples         []float64
	MedianReplySeconds   float64
	P80ReplySeconds      float64
	LateNightRatio       float64
	LongTextRatio        float64
	TaskRatio            float64
	MyInitiationRatio    float64
	MyMessageShare       float64
	TrendScore           float64
	CoolingScore         float64
	FastReplyScore       float64
	SharedGroupsCount    int
	PeakMonth            string
	PeakMonthCount       int
	CurrentStage         string
	DaysSinceLastContact int
	FirstTs              int64
	LastTs               int64
	CurrentMonthCount    int
	PreviousMonthCount   int
	Labels               []ControversialLabel
	ObjectiveSummary     string
	PlayfulSummary       string
	StageTimeline        []RelationStageItem
	MetricItems          []RelationMetricItem
	EvidenceGroups       []RelationEvidenceGroup
}

func (s *ContactService) GetRelationOverview() *RelationOverviewGrouped {
	profiles := s.buildAllRelationProfiles()
	grouped := &RelationOverviewGrouped{
		All:    buildRelationOverviewFromProfiles(profiles),
		Male:   buildRelationOverviewFromProfiles(filterProfilesByGender(profiles, model.GenderMale)),
		Female: buildRelationOverviewFromProfiles(filterProfilesByGender(profiles, model.GenderFemale)),
	}
	return grouped
}

func buildRelationOverviewFromProfiles(profiles []*relationProfile) RelationOverview {
	overview := RelationOverview{}
	warming := make([]RelationOverviewItem, 0, len(profiles))
	cooling := make([]RelationOverviewItem, 0, len(profiles))
	initiative := make([]RelationOverviewItem, 0, len(profiles))
	fastReply := make([]RelationOverviewItem, 0, len(profiles))
	for _, profile := range profiles {
		if shouldIncludeWarmingBoard(profile) {
			confidence, confidenceReason := confidenceWithReason(profile, 15)
			warming = append(warming, RelationOverviewItem{
				Username:         profile.Username,
				Name:             profile.Name,
				Gender:           string(profile.Gender),
				Score:            clamp100(profile.TrendScore),
				Confidence:       confidence,
				StaleHint:        buildStaleHint(profile.DaysSinceLastContact, false),
				ConfidenceReason: confidenceReason,
				Why:              fmt.Sprintf("近7天 %.0f 条/天，前30天 %.0f 条/天，最近%s", float64(profile.Recent7Messages)/relationRecentDays, float64(profile.Previous30Messages)/relationPreviousDays, profile.lastSeenText()),
				EvidencePreview:  []string{fmt.Sprintf("近7天 %d 条，前30天 %d 条", profile.Recent7Messages, profile.Previous30Messages), fmt.Sprintf("最近 %d 个会话里你主动 %.0f%%", maxInt(1, profile.Recent7Sessions), profile.MyInitiationRatio)},
				RankScore:        relationRankScore("warming", clamp100(profile.TrendScore), profile),
			})
		}
		if shouldIncludeCoolingBoard(profile) {
			confidence, confidenceReason := confidenceWithReason(profile, 15)
			cooling = append(cooling, RelationOverviewItem{
				Username:         profile.Username,
				Name:             profile.Name,
				Gender:           string(profile.Gender),
				Score:            clamp100(profile.CoolingScore),
				Confidence:       confidence,
				StaleHint:        buildStaleHint(profile.DaysSinceLastContact, true),
				ConfidenceReason: confidenceReason,
				Why:              fmt.Sprintf("峰值月份 %s 有 %d 条，最近30天仅 %d 条", profile.PeakMonth, profile.PeakMonthCount, profile.CurrentMonthCount),
				EvidencePreview:  []string{fmt.Sprintf("距今 %d 天没新消息", profile.DaysSinceLastContact), fmt.Sprintf("近7天 %d 条，前30天 %d 条", profile.Recent7Messages, profile.Previous30Messages)},
				RankScore:        relationRankScore("cooling", clamp100(profile.CoolingScore), profile),
			})
		}
		if shouldIncludeInitiativeBoard(profile) {
			confidence, confidenceReason := confidenceWithReason(profile, 15)
			initiative = append(initiative, RelationOverviewItem{
				Username:         profile.Username,
				Name:             profile.Name,
				Gender:           string(profile.Gender),
				Score:            clamp100(profile.MyInitiationRatio),
				Confidence:       confidence,
				StaleHint:        buildStaleHint(profile.DaysSinceLastContact, false),
				ConfidenceReason: confidenceReason,
				Why:              fmt.Sprintf("共 %d 段对话，你先开口 %d 次", profile.TotalSessions, profile.MyInitiatedSessions),
				EvidencePreview:  []string{fmt.Sprintf("你发出 %d 条，对方 %d 条", profile.MyMessages, profile.TheirMessages), fmt.Sprintf("回复中位数 %s", formatDurationCN(profile.MedianReplySeconds))},
				RankScore:        relationRankScore("initiative", clamp100(profile.MyInitiationRatio), profile),
			})
		}
		if shouldIncludeFastReplyBoard(profile) {
			confidence, confidenceReason := confidenceWithReason(profile, 15)
			fastReply = append(fastReply, RelationOverviewItem{
				Username:         profile.Username,
				Name:             profile.Name,
				Gender:           string(profile.Gender),
				Score:            clamp100(profile.FastReplyScore),
				Confidence:       confidence,
				StaleHint:        buildStaleHint(profile.DaysSinceLastContact, false),
				ConfidenceReason: confidenceReason,
				Why:              fmt.Sprintf("TA 回复中位数 %s，P80 %s", formatDurationCN(profile.MedianReplySeconds), formatDurationCN(profile.P80ReplySeconds)),
				EvidencePreview:  []string{fmt.Sprintf("有效回复样本 %d 条", len(profile.ReplySamples)), fmt.Sprintf("最近 %d 天仍有来往", maxInt(0, relationColdWindowDays-profile.DaysSinceLastContact))},
				RankScore:        relationRankScore("fast_reply", clamp100(profile.FastReplyScore), profile),
			})
		}
	}
	sortRelationItems(warming)
	sortRelationItems(cooling)
	sortRelationItems(initiative)
	sortRelationItems(fastReply)
	overview.Warming = trimRelationItems(warming, relationOverviewLimit)
	overview.Cooling = trimRelationItems(cooling, relationOverviewLimit)
	overview.Initiative = trimRelationItems(initiative, relationOverviewLimit)
	overview.FastReply = trimRelationItems(fastReply, relationOverviewLimit)
	return overview
}

func (s *ContactService) GetRelationDetail(username string) *RelationDetail {
	profile := s.buildRelationProfile(username)
	if profile == nil {
		return &RelationDetail{Username: username, Name: username}
	}
	return &RelationDetail{
		Username:            profile.Username,
		Name:                profile.Name,
		Confidence:          profile.confidence(),
		StaleHint:           buildStaleHint(profile.DaysSinceLastContact, profile.CoolingScore >= 70),
		ConfidenceReason:    profile.confidenceReason(15),
		StageTimeline:       profile.StageTimeline,
		ObjectiveSummary:    profile.ObjectiveSummary,
		PlayfulSummary:      profile.PlayfulSummary,
		Metrics:             profile.MetricItems,
		ControversialLabels: profile.Labels,
		EvidenceGroups:      profile.EvidenceGroups,
	}
}

func (s *ContactService) GetControversyOverview() *ControversyOverviewGrouped {
	profiles := s.buildAllRelationProfiles()
	grouped := &ControversyOverviewGrouped{
		All:    buildControversyOverviewFromProfiles(profiles),
		Male:   buildControversyOverviewFromProfiles(filterProfilesByGender(profiles, model.GenderMale)),
		Female: buildControversyOverviewFromProfiles(filterProfilesByGender(profiles, model.GenderFemale)),
	}
	return grouped
}

func buildControversyOverviewFromProfiles(profiles []*relationProfile) ControversyOverview {
	overview := ControversyOverview{}
	for _, profile := range profiles {
		if !shouldIncludeInControversyRanking(profile) {
			continue
		}
		for _, label := range profile.Labels {
			historical := label.Label == "faded"
			item := ControversyItem{
				Username:         profile.Username,
				Name:             profile.Name,
				Gender:           string(profile.Gender),
				Label:            label.Label,
				Score:            label.Score,
				Confidence:       label.Confidence,
				StaleHint:        buildStaleHint(profile.DaysSinceLastContact, historical),
				ConfidenceReason: label.ConfidenceReason,
				Why:              label.Why,
				EvidencePreview:  trimEvidence(label.EvidenceGroups, 2),
				RankScore:        controversyRankScore(label.Label, label.Score, profile),
			}
			switch label.Label {
			case "simp":
				overview.Simp = append(overview.Simp, item)
			case "ambiguity":
				overview.Ambiguity = append(overview.Ambiguity, item)
			case "faded":
				overview.Faded = append(overview.Faded, item)
			case "tool_person":
				overview.ToolPerson = append(overview.ToolPerson, item)
			case "cold_violence":
				overview.ColdViolence = append(overview.ColdViolence, item)
			}
		}
	}
	sortControversyItems(overview.Simp)
	sortControversyItems(overview.Ambiguity)
	sortControversyItems(overview.Faded)
	sortControversyItems(overview.ToolPerson)
	sortControversyItems(overview.ColdViolence)
	overview.Simp = trimControversyItems(overview.Simp, relationOverviewLimit)
	overview.Ambiguity = trimControversyItems(overview.Ambiguity, relationOverviewLimit)
	overview.Faded = trimControversyItems(overview.Faded, relationOverviewLimit)
	overview.ToolPerson = trimControversyItems(overview.ToolPerson, relationOverviewLimit)
	overview.ColdViolence = trimControversyItems(overview.ColdViolence, relationOverviewLimit)
	return overview
}

func (s *ContactService) GetControversyDetail(username string) *ControversyDetail {
	profile := s.buildRelationProfile(username)
	if profile == nil {
		return &ControversyDetail{Username: username, Name: username}
	}
	return &ControversyDetail{
		Username:            profile.Username,
		Name:                profile.Name,
		ControversialLabels: profile.Labels,
	}
}

func (s *ContactService) buildAllRelationProfiles() []*relationProfile {
	stats := s.GetCachedStats()
	if len(stats) == 0 {
		return nil
	}
	profiles := make([]*relationProfile, 0, len(stats))
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, maxInt(1, s.cfg.WorkerCount))
	for _, stat := range stats {
		username := stat.Username
		if isUnsupportedAnalysisUsername(username) || stat.TotalMessages == 0 {
			continue
		}
		wg.Add(1)
		go func(uname string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			profile := s.buildRelationProfile(uname)
			if profile == nil || profile.TotalMessages == 0 {
				return
			}
			mu.Lock()
			profiles = append(profiles, profile)
			mu.Unlock()
		}(username)
	}
	wg.Wait()
	return profiles
}

func filterProfilesByGender(profiles []*relationProfile, gender model.Gender) []*relationProfile {
	if len(profiles) == 0 {
		return nil
	}
	filtered := make([]*relationProfile, 0, len(profiles))
	for _, profile := range profiles {
		if profile == nil {
			continue
		}
		if profile.Gender == gender {
			filtered = append(filtered, profile)
		}
	}
	return filtered
}

func (s *ContactService) buildRelationProfile(username string) *relationProfile {
	if isUnsupportedAnalysisUsername(username) {
		return nil
	}
	meta := s.lookupContactMeta(username)
	messages := s.loadRelationMessages(username)
	if len(messages) == 0 {
		return &relationProfile{
			Username:          username,
			Name:              meta.Name,
			Gender:            meta.Gender,
			ContactKind:       meta.ContactKind,
			IsDeleted:         meta.IsDeleted,
			IsBiz:             meta.IsBiz,
			LikelyMarketing:   meta.LikelyMarketing,
			IsLikelyAlt:       meta.IsLikelyAlt,
			SharedGroupsCount: meta.SharedGroupsCount,
		}
	}
	profile := &relationProfile{
		Username:          username,
		Name:              meta.Name,
		Gender:            meta.Gender,
		ContactKind:       meta.ContactKind,
		IsDeleted:         meta.IsDeleted,
		IsBiz:             meta.IsBiz,
		LikelyMarketing:   meta.LikelyMarketing,
		IsLikelyAlt:       meta.IsLikelyAlt,
		Messages:          messages,
		TotalMessages:     len(messages),
		SharedGroupsCount: meta.SharedGroupsCount,
		FirstTs:           messages[0].Ts,
		LastTs:            messages[len(messages)-1].Ts,
	}
	now := time.Now().In(s.tz)
	recentCutoff := now.AddDate(0, 0, -relationRecentDays).Unix()
	previousCutoff := now.AddDate(0, 0, -(relationRecentDays + relationPreviousDays)).Unix()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, s.tz).Unix()
	previousMonthStart := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, s.tz).Unix()

	monthCounts := make(map[string]int)
	var lastSessionTs int64
	var sessionStartMessages []relationMessage
	for idx, msg := range messages {
		dt := time.Unix(msg.Ts, 0).In(s.tz)
		monthCounts[dt.Format("2006-01")]++
		if msg.IsMine {
			profile.MyMessages++
		} else {
			profile.TheirMessages++
		}
		if isMeaningfulText(msg) {
			profile.TextMessages++
			if utf8.RuneCountInString(strings.TrimSpace(msg.Content)) >= relationLongTextThreshold {
				profile.LongTextMessages++
			}
			if isTaskOrientedText(msg.Content) {
				profile.TaskMessages++
			}
		}
		if dt.Hour() >= 0 && dt.Hour() < 5 {
			profile.LateNightMessages++
		}
		if msg.Ts >= recentCutoff {
			profile.Recent7Messages++
		}
		if msg.Ts >= previousCutoff && msg.Ts < recentCutoff {
			profile.Previous30Messages++
		}
		if msg.Ts >= currentMonthStart {
			profile.CurrentMonthCount++
		}
		if msg.Ts >= previousMonthStart && msg.Ts < currentMonthStart {
			profile.PreviousMonthCount++
		}
		if lastSessionTs == 0 || msg.Ts-lastSessionTs > s.cfg.SessionGapSeconds {
			profile.TotalSessions++
			sessionStartMessages = append(sessionStartMessages, msg)
			if msg.IsMine {
				profile.MyInitiatedSessions++
				if msg.Ts >= recentCutoff {
					profile.Recent7Sessions++
				}
				if msg.Ts >= previousCutoff && msg.Ts < recentCutoff {
					profile.Previous30Sessions++
				}
			} else {
				if msg.Ts >= recentCutoff {
					profile.Recent7Sessions++
				}
				if msg.Ts >= previousCutoff && msg.Ts < recentCutoff {
					profile.Previous30Sessions++
				}
			}
		}
		lastSessionTs = msg.Ts

		if msg.IsMine {
			for next := idx + 1; next < len(messages); next++ {
				candidate := messages[next]
				if candidate.Ts-msg.Ts > relationReplyMaxSeconds {
					break
				}
				if candidate.IsMine || !isMeaningfulMessage(candidate) {
					continue
				}
				profile.ReplySamples = append(profile.ReplySamples, float64(candidate.Ts-msg.Ts))
				break
			}
		}
	}

	profile.MedianReplySeconds = percentile(profile.ReplySamples, 0.5)
	profile.P80ReplySeconds = percentile(profile.ReplySamples, 0.8)
	profile.MyInitiationRatio = ratioPercent(float64(profile.MyInitiatedSessions), float64(profile.TotalSessions))
	profile.MyMessageShare = ratioPercent(float64(profile.MyMessages), float64(profile.TotalMessages))
	profile.LateNightRatio = ratioPercent(float64(profile.LateNightMessages), float64(profile.TotalMessages))
	profile.LongTextRatio = ratioPercent(float64(profile.LongTextMessages), float64(maxInt(1, profile.TextMessages)))
	profile.TaskRatio = ratioPercent(float64(profile.TaskMessages), float64(maxInt(1, profile.TextMessages)))
	profile.DaysSinceLastContact = maxInt(0, int(now.Sub(time.Unix(profile.LastTs, 0).In(s.tz)).Hours()/24))
	profile.PeakMonth, profile.PeakMonthCount = peakMonth(monthCounts)
	profile.TrendScore = computeWarmingScore(profile)
	profile.CoolingScore = computeCoolingScore(profile)
	profile.FastReplyScore = computeFastReplyScore(profile)
	profile.CurrentStage = deriveCurrentStage(profile)
	profile.StageTimeline = buildStageTimeline(profile)
	profile.ObjectiveSummary = buildObjectiveSummary(profile)
	profile.PlayfulSummary = buildPlayfulSummary(profile)
	profile.MetricItems = buildRelationMetrics(profile)
	profile.EvidenceGroups = buildObjectiveEvidenceGroups(profile, sessionStartMessages)
	profile.Labels = buildControversialLabels(profile, sessionStartMessages)
	sort.Slice(profile.Labels, func(i, j int) bool { return profile.Labels[i].Score > profile.Labels[j].Score })
	return profile
}

type relationContactMeta struct {
	Name              string
	Gender            model.Gender
	SharedGroupsCount int
	ContactKind       string
	IsDeleted         bool
	IsBiz             bool
	LikelyMarketing   bool
	IsLikelyAlt       bool
}

func (s *ContactService) lookupContactMeta(username string) relationContactMeta {
	for _, item := range s.GetCachedStats() {
		if item.Username == username {
			name := strings.TrimSpace(item.Remark)
			if name == "" {
				name = strings.TrimSpace(item.Nickname)
			}
			if name == "" {
				name = username
			}
			return relationContactMeta{
				Name:              name,
				Gender:            item.Gender,
				SharedGroupsCount: item.SharedGroupsCount,
				ContactKind:       item.ContactKind,
				IsDeleted:         item.IsDeleted,
				IsBiz:             item.IsBiz,
				LikelyMarketing:   item.LikelyMarketing,
				IsLikelyAlt:       item.IsLikelyAlt,
			}
		}
	}
	var contact model.Contact
	var extraBuffer []byte
	_ = s.dbMgr.ContactDB.QueryRow(
		"SELECT COALESCE(remark,''), COALESCE(nick_name,''), COALESCE(alias,''), COALESCE(description,''), COALESCE(delete_flag,0), COALESCE(extra_buffer, x'') FROM contact WHERE username = ? LIMIT 1",
		username,
	).Scan(&contact.Remark, &contact.Nickname, &contact.Alias, &contact.Description, &contact.DeleteFlag, &extraBuffer)
	contact.Username = username
	contact.Gender = parseExplicitGenderFromExtraBuffer(extraBuffer)
	contact.IsDeleted = contact.DeleteFlag != 0
	contact.ContactKind, contact.IsBiz, contact.LikelyMarketing, contact.IsLikelyAlt = classifyContactKind(contact)
	name := strings.TrimSpace(contact.Remark)
	if name == "" {
		name = strings.TrimSpace(contact.Nickname)
	}
	if name == "" {
		name = username
	}
	return relationContactMeta{
		Name:            name,
		Gender:          contact.Gender,
		ContactKind:     contact.ContactKind,
		IsDeleted:       contact.IsDeleted,
		IsBiz:           contact.IsBiz,
		LikelyMarketing: contact.LikelyMarketing,
		IsLikelyAlt:     contact.IsLikelyAlt,
	}
}

func (s *ContactService) loadRelationMessages(username string) []relationMessage {
	tableName := db.GetTableName(username)
	whereClause := s.timeWhere()
	query := fmt.Sprintf("SELECT create_time, local_type, message_content, COALESCE(WCDB_CT_message_content,0), COALESCE(real_sender_id,0) FROM [%s]%s ORDER BY create_time ASC", tableName, whereClause)
	messages := make([]relationMessage, 0, 512)
	for _, mdb := range s.dbMgr.MessageDBs {
		var contactRowID int64 = -1
		_ = mdb.QueryRow("SELECT rowid FROM Name2Id WHERE user_name = ?", username).Scan(&contactRowID)
		rows, err := mdb.Query(query)
		if err != nil {
			continue
		}
		for rows.Next() {
			var ts int64
			var lt int
			var rawContent []byte
			var ct, senderID int64
			if err := rows.Scan(&ts, &lt, &rawContent, &ct, &senderID); err != nil {
				continue
			}
			content := normalizeMessageContent(decodeGroupContent(rawContent, ct), lt)
			if content == "" {
				continue
			}
			if s.isSys(content) {
				continue
			}
			messages = append(messages, relationMessage{
				Ts:      ts,
				Content: content,
				IsMine:  contactRowID < 0 || senderID != contactRowID,
				Type:    lt,
			})
		}
		rows.Close()
	}
	sort.Slice(messages, func(i, j int) bool { return messages[i].Ts < messages[j].Ts })
	return messages
}

func normalizeMessageContent(content string, lt int) string {
	content = strings.TrimSpace(content)
	switch lt {
	case 1:
		return content
	case 3:
		return "[图片]"
	case 34:
		return "[语音]"
	case 43:
		return "[视频]"
	case 47:
		return "[表情]"
	case 49:
		if strings.Contains(content, "wcpay") || strings.Contains(content, "redenvelope") {
			return "[红包/转账]"
		}
		if content == "" {
			return "[文件/链接]"
		}
		return "[链接/文件]"
	default:
		if content != "" {
			return content
		}
		return fmt.Sprintf("[消息类型 %d]", lt)
	}
}

func isMeaningfulText(msg relationMessage) bool {
	return msg.Type == 1 && strings.TrimSpace(msg.Content) != ""
}

func isMeaningfulMessage(msg relationMessage) bool {
	return strings.TrimSpace(msg.Content) != ""
}

func isTaskOrientedText(content string) bool {
	text := strings.TrimSpace(strings.ToLower(content))
	if text == "" {
		return false
	}
	for _, keyword := range taskKeywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	copyValues := append([]float64(nil), values...)
	sort.Float64s(copyValues)
	if len(copyValues) == 1 {
		return copyValues[0]
	}
	pos := p * float64(len(copyValues)-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))
	if lower == upper {
		return copyValues[lower]
	}
	weight := pos - float64(lower)
	return copyValues[lower]*(1-weight) + copyValues[upper]*weight
}

func ratioPercent(part, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return clamp100(part / total * 100)
}

func computeWarmingScore(profile *relationProfile) float64 {
	recentRate := float64(profile.Recent7Messages) / relationRecentDays
	previousRate := float64(profile.Previous30Messages) / relationPreviousDays
	delta := recentRate - previousRate
	trend := clamp100(50 + delta*18)
	freshness := clamp100(100 - float64(profile.DaysSinceLastContact)*6)
	initiativeBoost := clamp100(profile.MyInitiationRatio)
	return clamp100(trend*0.55 + freshness*0.25 + initiativeBoost*0.20)
}

func computeCoolingScore(profile *relationProfile) float64 {
	historyBase := clamp100(float64(profile.PeakMonthCount) * 1.2)
	recentWeak := clamp100(100 - float64(profile.Recent7Messages*8))
	lastGap := clamp100(float64(profile.DaysSinceLastContact) * 4)
	monthDrop := clamp100(float64(maxInt(0, profile.PreviousMonthCount-profile.CurrentMonthCount)) * 6)
	return clamp100(historyBase*0.25 + recentWeak*0.25 + lastGap*0.30 + monthDrop*0.20)
}

func computeFastReplyScore(profile *relationProfile) float64 {
	if profile.MedianReplySeconds <= 0 {
		return 0
	}
	medianScore := clamp100(100 - profile.MedianReplySeconds/90)
	p80Score := clamp100(100 - profile.P80ReplySeconds/120)
	return clamp100(medianScore*0.7 + p80Score*0.3)
}

func deriveCurrentStage(profile *relationProfile) string {
	switch {
	case profile.CoolingScore >= 72:
		return "明显冷却"
	case profile.TrendScore >= 70 && profile.LateNightRatio >= 12:
		return "升温靠近"
	case profile.TaskRatio >= 55:
		return "事务型维系"
	case profile.MyInitiationRatio >= 65 && profile.P80ReplySeconds > 6*3600:
		return "单向拉扯"
	default:
		return "稳定来往"
	}
}

func buildStageTimeline(profile *relationProfile) []RelationStageItem {
	start := time.Unix(profile.FirstTs, 0).In(time.Local)
	last := time.Unix(profile.LastTs, 0).In(time.Local)
	items := []RelationStageItem{{
		ID:        "start",
		Stage:     "建立联系",
		StartDate: start.Format("2006-01-02"),
		EndDate:   monthEndString(start),
		Summary:   fmt.Sprintf("第一次记录出现在 %s，当前累计 %d 条消息。", start.Format("2006-01-02"), profile.TotalMessages),
		Score:     32,
	}}
	if profile.PeakMonth != "" {
		items = append(items, RelationStageItem{
			ID:        "peak",
			Stage:     "关系高点",
			StartDate: profile.PeakMonth + "-01",
			Summary:   fmt.Sprintf("峰值月份 %s 共 %d 条消息，是这段关系最热的时候。", profile.PeakMonth, profile.PeakMonthCount),
			Score:     clamp100(float64(profile.PeakMonthCount) * 1.2),
		})
	}
	items = append(items, RelationStageItem{
		ID:        "current",
		Stage:     profile.CurrentStage,
		StartDate: last.AddDate(0, 0, -30).Format("2006-01-02"),
		EndDate:   last.Format("2006-01-02"),
		Summary:   fmt.Sprintf("最近 7 天 %d 条，前 30 天 %d 条，TA 回复中位数 %s。", profile.Recent7Messages, profile.Previous30Messages, formatDurationCN(profile.MedianReplySeconds)),
		Score:     clamp100(math.Max(profile.TrendScore, profile.CoolingScore)),
	})
	return items
}

func buildObjectiveSummary(profile *relationProfile) string {
	return fmt.Sprintf("最近 7 天 %d 条消息，前 30 天 %d 条；%d 段对话里你先开口 %d 次（%.0f%%）。TA 回复中位数 %s，深夜占比 %.0f%%，事务型沟通占比 %.0f%%，当前更像“%s”。",
		profile.Recent7Messages,
		profile.Previous30Messages,
		profile.TotalSessions,
		profile.MyInitiatedSessions,
		profile.MyInitiationRatio,
		formatDurationCN(profile.MedianReplySeconds),
		profile.LateNightRatio,
		profile.TaskRatio,
		profile.CurrentStage,
	)
}

func buildPlayfulSummary(profile *relationProfile) string {
	switch profile.CurrentStage {
	case "升温靠近":
		return "这条线最近明显有点发烫：联系变密、夜聊抬头、长消息也在变多，像是从普通来往往更靠近的方向滑。"
	case "明显冷却":
		return "这段关系像是从高峰退潮，历史上热闹过，但最近热度掉得很明显，属于‘不是没故事，是故事暂时停更’。"
	case "事务型维系":
		return "目前更像一条高效率协作线：需求有来有回，但闲聊和情绪浓度不算高，功能性明显强于氛围感。"
	case "单向拉扯":
		return "你这边像在持续点火，对面回得不算快也不算密，整体呈现一种‘你负责开场，对方负责偶尔续命’的味道。"
	default:
		return "整体节奏偏稳定，没有极端升温也没彻底凉透，属于有来有回、但火候还没到失控的那种关系。"
	}
}

func buildRelationMetrics(profile *relationProfile) []RelationMetricItem {
	trend := "flat"
	if profile.TrendScore >= 65 {
		trend = "up"
	} else if profile.CoolingScore >= 65 {
		trend = "down"
	}
	return []RelationMetricItem{
		{Key: "stage", Label: "关系阶段", Value: profile.CurrentStage, Hint: "综合升温/降温/主动度后的当前判断"},
		{Key: "initiative", Label: "主动度", Value: fmt.Sprintf("%.0f%%", profile.MyInitiationRatio), SubValue: fmt.Sprintf("%d/%d 段由你开场", profile.MyInitiatedSessions, profile.TotalSessions), Trend: trend, RawValue: profile.MyInitiationRatio},
		{Key: "reply", Label: "回复速度", Value: formatDurationCN(profile.MedianReplySeconds), SubValue: fmt.Sprintf("P80 %s", formatDurationCN(profile.P80ReplySeconds)), Hint: "仅统计 24 小时内有效响应", RawValue: profile.MedianReplySeconds},
		{Key: "late_night", Label: "深夜浓度", Value: fmt.Sprintf("%.0f%%", profile.LateNightRatio), SubValue: fmt.Sprintf("%d 条发生在 0-5 点", profile.LateNightMessages), RawValue: profile.LateNightRatio},
		{Key: "long_text", Label: "长文本倾向", Value: fmt.Sprintf("%.0f%%", profile.LongTextRatio), SubValue: fmt.Sprintf("%d/%d 条文本超过 %d 字", profile.LongTextMessages, profile.TextMessages, relationLongTextThreshold), RawValue: profile.LongTextRatio},
		{Key: "task", Label: "事务型沟通", Value: fmt.Sprintf("%.0f%%", profile.TaskRatio), SubValue: fmt.Sprintf("命中 %d 条事务关键词消息", profile.TaskMessages), RawValue: profile.TaskRatio},
		{Key: "groups", Label: "共同群聊依赖度", Value: fmt.Sprintf("%d 个", profile.SharedGroupsCount), SubValue: sharedGroupHint(profile.SharedGroupsCount)},
	}
}

func buildObjectiveEvidenceGroups(profile *relationProfile, sessionStarts []relationMessage) []RelationEvidenceGroup {
	groups := make([]RelationEvidenceGroup, 0, 3)
	trendItems := make([]RelationEvidence, 0, 3)
	initiativeItems := make([]RelationEvidence, 0, 3)
	replyItems := make([]RelationEvidence, 0, 3)
	for _, msg := range reverseMessages(profile.Messages) {
		if len(trendItems) < 3 && msg.Ts >= time.Now().In(safeTZ()).AddDate(0, 0, -relationRecentDays).Unix() {
			trendItems = append(trendItems, evidenceFromMessage(msg, "最近7天高频互动样本"))
		}
		if len(replyItems) < 3 && !msg.IsMine {
			replyItems = append(replyItems, evidenceFromMessage(msg, "对方的实际回应消息"))
		}
		if len(trendItems) >= 3 && len(replyItems) >= 3 {
			break
		}
	}
	for _, msg := range reverseMessages(sessionStarts) {
		if msg.IsMine && len(initiativeItems) < 3 {
			initiativeItems = append(initiativeItems, evidenceFromMessage(msg, "该对话段由你主动发起"))
		}
	}
	groups = append(groups, RelationEvidenceGroup{ID: "trend", Title: "最近关系变化", Subtitle: "用于解释最近升温/降温走势", Items: trimEvidence(trendItems, relationEvidenceLimit)})
	groups = append(groups, RelationEvidenceGroup{ID: "initiative", Title: "主动度证据", Subtitle: "按 6 小时切段，段首谁先开口", Items: trimEvidence(initiativeItems, relationEvidenceLimit)})
	groups = append(groups, RelationEvidenceGroup{ID: "reply", Title: "回复节奏证据", Subtitle: "展示对方的回应样本", Items: trimEvidence(replyItems, relationEvidenceLimit)})
	return groups
}

func buildControversialLabels(profile *relationProfile, sessionStarts []relationMessage) []ControversialLabel {
	labels := []ControversialLabel{
		buildSimpLabel(profile, sessionStarts),
		buildAmbiguityLabel(profile),
		buildFadedLabel(profile),
		buildToolPersonLabel(profile),
		buildColdViolenceLabel(profile, sessionStarts),
	}
	return labels
}

func buildSimpLabel(profile *relationProfile, sessionStarts []relationMessage) ControversialLabel {
	score := clamp100(profile.MyInitiationRatio*0.45 + clamp100(profile.MedianReplySeconds/180)*0.30 + profile.MyMessageShare*0.25)
	confidence, confidenceReason := confidenceWithReason(profile, 18)
	evidence := make([]RelationEvidence, 0, relationEvidenceLimit)
	for _, msg := range reverseMessages(sessionStarts) {
		if msg.IsMine && len(evidence) < 3 {
			evidence = append(evidence, evidenceFromMessage(msg, "新对话段由你先发起"))
		}
	}
	for _, msg := range reverseMessages(profile.Messages) {
		if !msg.IsMine && len(evidence) < relationEvidenceLimit {
			evidence = append(evidence, evidenceFromMessage(msg, "对方回复样本，可对照回复延迟"))
		}
	}
	return ControversialLabel{
		Label:            "simp",
		Score:            score,
		Confidence:       confidence,
		StaleHint:        buildStaleHint(profile.DaysSinceLastContact, false),
		ConfidenceReason: confidenceReason,
		Why:              fmt.Sprintf("你先开口占 %.0f%%，你发消息占 %.0f%%，TA 回复中位数 %s，明显是你在更努力维持这条线。", profile.MyInitiationRatio, profile.MyMessageShare, formatDurationCN(profile.MedianReplySeconds)),
		Metrics: []ControversyMetric{
			{Key: "initiative_ratio", Label: "你先开口", Value: profile.MyInitiationRatio, DisplayValue: fmt.Sprintf("%.0f%%", profile.MyInitiationRatio)},
			{Key: "my_share", Label: "你消息占比", Value: profile.MyMessageShare, DisplayValue: fmt.Sprintf("%.0f%%", profile.MyMessageShare)},
			{Key: "reply_median", Label: "回复中位数", Value: profile.MedianReplySeconds, DisplayValue: formatDurationCN(profile.MedianReplySeconds)},
		},
		EvidenceGroups: trimEvidence(evidence, relationEvidenceLimit),
	}
}

func buildAmbiguityLabel(profile *relationProfile) ControversialLabel {
	score := clamp100(profile.TrendScore*0.45 + profile.LateNightRatio*0.25 + profile.LongTextRatio*0.20 + clamp100(float64(profile.Recent7Sessions*12))*0.10)
	confidence, confidenceReason := confidenceWithReason(profile, 12)
	evidence := pickEvidence(profile.Messages, func(msg relationMessage) bool {
		dt := time.Unix(msg.Ts, 0).In(safeTZ())
		return dt.Hour() < 5 || (isMeaningfulText(msg) && utf8.RuneCountInString(msg.Content) >= relationLongTextThreshold)
	}, relationEvidenceLimit, "深夜或长文本互动样本")
	return ControversialLabel{
		Label:            "ambiguity",
		Score:            score,
		Confidence:       confidence,
		StaleHint:        buildStaleHint(profile.DaysSinceLastContact, false),
		ConfidenceReason: confidenceReason,
		Why:              fmt.Sprintf("最近明显升温，深夜占比 %.0f%%，长文本占比 %.0f%%，已经有点从普通聊天往暧昧氛围滑的意思。", profile.LateNightRatio, profile.LongTextRatio),
		Metrics: []ControversyMetric{
			{Key: "warming", Label: "升温分", Value: profile.TrendScore, DisplayValue: fmt.Sprintf("%.0f", profile.TrendScore)},
			{Key: "late_night", Label: "深夜占比", Value: profile.LateNightRatio, DisplayValue: fmt.Sprintf("%.0f%%", profile.LateNightRatio)},
			{Key: "long_text", Label: "长文本占比", Value: profile.LongTextRatio, DisplayValue: fmt.Sprintf("%.0f%%", profile.LongTextRatio)},
		},
		EvidenceGroups: evidence,
	}
}

func buildFadedLabel(profile *relationProfile) ControversialLabel {
	score := clamp100(profile.CoolingScore*0.6 + clamp100(float64(profile.DaysSinceLastContact)*2.5)*0.2 + clamp100(float64(maxInt(0, profile.PeakMonthCount-profile.CurrentMonthCount))*2)*0.2)
	confidence, confidenceReason := confidenceWithReason(profile, 12)
	evidence := make([]RelationEvidence, 0, relationEvidenceLimit)
	peakMonth := profile.PeakMonth
	for _, msg := range reverseMessages(profile.Messages) {
		if peakMonth != "" && strings.HasPrefix(time.Unix(msg.Ts, 0).In(safeTZ()).Format("2006-01"), peakMonth) && len(evidence) < 2 {
			evidence = append(evidence, evidenceFromMessage(msg, "历史高频阶段消息"))
		}
	}
	for _, msg := range reverseMessages(profile.Messages) {
		if len(evidence) < relationEvidenceLimit {
			evidence = append(evidence, evidenceFromMessage(msg, "近期最后留下的消息样本"))
			break
		}
	}
	return ControversialLabel{
		Label:            "faded",
		Score:            score,
		Confidence:       confidence,
		StaleHint:        buildStaleHint(profile.DaysSinceLastContact, true),
		ConfidenceReason: confidenceReason,
		Why:              fmt.Sprintf("你们在 %s 达到过高点（%d 条），但最近 30 天只剩 %d 条，已经是典型的‘曾经热络，现在发凉’。", emptyIf(profile.PeakMonth, "历史高点月份"), profile.PeakMonthCount, profile.CurrentMonthCount),
		Metrics: []ControversyMetric{
			{Key: "cooling", Label: "降温分", Value: profile.CoolingScore, DisplayValue: fmt.Sprintf("%.0f", profile.CoolingScore)},
			{Key: "peak_month", Label: "峰值月份", Value: float64(profile.PeakMonthCount), DisplayValue: fmt.Sprintf("%s / %d 条", emptyIf(profile.PeakMonth, "-"), profile.PeakMonthCount)},
			{Key: "last_gap", Label: "最近断联", Value: float64(profile.DaysSinceLastContact), DisplayValue: fmt.Sprintf("%d 天", profile.DaysSinceLastContact)},
		},
		EvidenceGroups: trimEvidence(evidence, relationEvidenceLimit),
	}
}

func buildToolPersonLabel(profile *relationProfile) ControversialLabel {
	score := clamp100(profile.TaskRatio*0.6 + clamp100(100-profile.LongTextRatio)*0.2 + clamp100(100-profile.LateNightRatio)*0.2)
	confidence, confidenceReason := confidenceWithReason(profile, 10)
	evidence := pickEvidence(profile.Messages, func(msg relationMessage) bool {
		return isMeaningfulText(msg) && isTaskOrientedText(msg.Content)
	}, relationEvidenceLimit, "事务型关键词命中")
	return ControversialLabel{
		Label:            "tool_person",
		Score:            score,
		Confidence:       confidence,
		StaleHint:        buildStaleHint(profile.DaysSinceLastContact, false),
		ConfidenceReason: confidenceReason,
		Why:              fmt.Sprintf("事务型关键词消息占 %.0f%%，深夜和长文本都不高，这段互动更像需求流转而不是情绪交流。", profile.TaskRatio),
		Metrics: []ControversyMetric{
			{Key: "task_ratio", Label: "事务占比", Value: profile.TaskRatio, DisplayValue: fmt.Sprintf("%.0f%%", profile.TaskRatio)},
			{Key: "long_text", Label: "长文本占比", Value: profile.LongTextRatio, DisplayValue: fmt.Sprintf("%.0f%%", profile.LongTextRatio)},
			{Key: "late_night", Label: "深夜占比", Value: profile.LateNightRatio, DisplayValue: fmt.Sprintf("%.0f%%", profile.LateNightRatio)},
		},
		EvidenceGroups: evidence,
	}
}

func buildColdViolenceLabel(profile *relationProfile, sessionStarts []relationMessage) ControversialLabel {
	score := clamp100(profile.MyInitiationRatio*0.40 + clamp100(profile.P80ReplySeconds/240)*0.35 + clamp100(float64(profile.DaysSinceLastContact)*3)*0.25)
	confidence, confidenceReason := confidenceWithReason(profile, 10)
	evidence := make([]RelationEvidence, 0, relationEvidenceLimit)
	for _, msg := range reverseMessages(sessionStarts) {
		if msg.IsMine && len(evidence) < 3 {
			evidence = append(evidence, evidenceFromMessage(msg, "你持续担任开场角色"))
		}
	}
	for _, msg := range reverseMessages(profile.Messages) {
		if !msg.IsMine && len(evidence) < relationEvidenceLimit {
			evidence = append(evidence, evidenceFromMessage(msg, "对方低频回应样本"))
		}
	}
	return ControversialLabel{
		Label:            "cold_violence",
		Score:            score,
		Confidence:       confidence,
		StaleHint:        buildStaleHint(profile.DaysSinceLastContact, false),
		ConfidenceReason: confidenceReason,
		Why:              fmt.Sprintf("你先开口 %.0f%%，对方回复 P80 已经拖到 %s，最近还断了 %d 天，冷处理的体感非常强。", profile.MyInitiationRatio, formatDurationCN(profile.P80ReplySeconds), profile.DaysSinceLastContact),
		Metrics: []ControversyMetric{
			{Key: "initiative_ratio", Label: "你先开口", Value: profile.MyInitiationRatio, DisplayValue: fmt.Sprintf("%.0f%%", profile.MyInitiationRatio)},
			{Key: "reply_p80", Label: "回复 P80", Value: profile.P80ReplySeconds, DisplayValue: formatDurationCN(profile.P80ReplySeconds)},
			{Key: "last_gap", Label: "最近断联", Value: float64(profile.DaysSinceLastContact), DisplayValue: fmt.Sprintf("%d 天", profile.DaysSinceLastContact)},
		},
		EvidenceGroups: trimEvidence(evidence, relationEvidenceLimit),
	}
}

func controversyConfidence(profile *relationProfile, minMessages int) float64 {
	confidence, _ := confidenceWithReason(profile, minMessages)
	return confidence
}

func confidenceWithReason(profile *relationProfile, minMessages int) (float64, string) {
	messageFactor := clamp100(float64(profile.TotalMessages) / float64(maxInt(1, minMessages)) * 100)
	replyFactor := clamp100(float64(len(profile.ReplySamples)) / 8 * 100)
	sessionFactor := clamp100(float64(profile.TotalSessions) / 8 * 100)
	sampleConfidence := clamp100(messageFactor*0.45 + replyFactor*0.25 + sessionFactor*0.30)
	freshnessFactor, freshnessReason := confidenceFreshnessFactor(profile.DaysSinceLastContact)
	finalConfidence := clamp100(sampleConfidence * freshnessFactor)
	return finalConfidence, fmt.Sprintf("样本充分度 %.0f/100；%s", sampleConfidence, freshnessReason)
}

func pickEvidence(messages []relationMessage, predicate func(relationMessage) bool, limit int, reason string) []RelationEvidence {
	result := make([]RelationEvidence, 0, limit)
	for _, msg := range reverseMessages(messages) {
		if predicate(msg) {
			result = append(result, evidenceFromMessage(msg, reason))
			if len(result) >= limit {
				break
			}
		}
	}
	return result
}

func evidenceFromMessage(msg relationMessage, reason string) RelationEvidence {
	t := time.Unix(msg.Ts, 0).In(safeTZ())
	return RelationEvidence{
		Date:    t.Format("2006-01-02"),
		Time:    t.Format("15:04"),
		Content: msg.Content,
		IsMine:  msg.IsMine,
		Reason:  reason,
	}
}

func reverseMessages[T any](items []T) []T {
	out := make([]T, len(items))
	for i := range items {
		out[len(items)-1-i] = items[i]
	}
	return out
}

func trimRelationItems(items []RelationOverviewItem, n int) []RelationOverviewItem {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

func trimControversyItems(items []ControversyItem, n int) []ControversyItem {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

func sortRelationItems(items []RelationOverviewItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].RankScore == items[j].RankScore {
			if items[i].Score == items[j].Score {
				return items[i].Confidence > items[j].Confidence
			}
			return items[i].Score > items[j].Score
		}
		return items[i].RankScore > items[j].RankScore
	})
}

func sortControversyItems(items []ControversyItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].RankScore == items[j].RankScore {
			if items[i].Score == items[j].Score {
				return items[i].Confidence > items[j].Confidence
			}
			return items[i].Score > items[j].Score
		}
		return items[i].RankScore > items[j].RankScore
	})
}

func trimEvidence(items []RelationEvidence, n int) []RelationEvidence {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

func clamp100(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func shouldIncludeInRelationRanking(profile *relationProfile) bool {
	if profile == nil {
		return false
	}
	if profile.TotalMessages < relationMinMessages {
		return false
	}
	if profile.TextMessages < relationMinTextMessages {
		return false
	}
	if profile.TotalSessions < relationMinSessions {
		return false
	}
	if profile.MyMessages < 5 || profile.TheirMessages < 5 {
		return false
	}
	return true
}

func shouldIncludeInControversyRanking(profile *relationProfile) bool {
	if !shouldIncludeInRelationRanking(profile) {
		return false
	}
	if profile.TotalMessages < controversyMinMessages {
		return false
	}
	if profile.MyMessages < 8 || profile.TheirMessages < 8 {
		return false
	}
	if len(profile.ReplySamples) < controversyMinReplySamples {
		return false
	}
	return true
}

func shouldIncludeWarmingBoard(profile *relationProfile) bool {
	return shouldIncludeInRelationRanking(profile) &&
		profile.Recent7Messages >= 18 &&
		(profile.Recent7Messages+profile.Previous30Messages) >= 60 &&
		profile.DaysSinceLastContact <= confidenceFreshDays2
}

func shouldIncludeCoolingBoard(profile *relationProfile) bool {
	return shouldIncludeInRelationRanking(profile) &&
		profile.PeakMonthCount >= 45 &&
		(profile.CurrentMonthCount+profile.PreviousMonthCount) >= 45
}

func shouldIncludeInitiativeBoard(profile *relationProfile) bool {
	return shouldIncludeInRelationRanking(profile) &&
		profile.TotalMessages >= 45 &&
		profile.TotalSessions >= 6 &&
		(profile.Recent7Messages+profile.Previous30Messages) >= 12 &&
		profile.DaysSinceLastContact <= 120
}

func shouldIncludeFastReplyBoard(profile *relationProfile) bool {
	return shouldIncludeInRelationRanking(profile) &&
		profile.TotalMessages >= 45 &&
		len(profile.ReplySamples) >= 6 &&
		(profile.Recent7Messages+profile.Previous30Messages) >= 10 &&
		profile.DaysSinceLastContact <= 120
}

func confidenceFreshnessFactor(daysSinceLastContact int) (float64, string) {
	switch {
	case daysSinceLastContact <= confidenceFreshDays1:
		return 1.00, "近 30 天仍有联系，新鲜度不衰减"
	case daysSinceLastContact <= confidenceFreshDays2:
		return 0.85, "近 31-90 天联系变少，置信度轻度下调"
	case daysSinceLastContact <= confidenceFreshDays3:
		return 0.65, "近 91-180 天联系稀疏，置信度中度下调"
	default:
		return 0.45, "超过 180 天未联系，置信度重度下调"
	}
}

func freshnessRankFactor(board string, daysSinceLastContact int) float64 {
	isHistoricalBoard := board == "cooling" || board == "faded"
	switch {
	case daysSinceLastContact <= confidenceFreshDays1:
		return 1.00
	case daysSinceLastContact <= confidenceFreshDays2:
		if isHistoricalBoard {
			return 0.95
		}
		return 0.82
	case daysSinceLastContact <= confidenceFreshDays3:
		if isHistoricalBoard {
			return 0.90
		}
		return 0.58
	default:
		if isHistoricalBoard {
			return 0.85
		}
		return 0.35
	}
}

func recentActivityWeight(profile *relationProfile, board string) float64 {
	recentMessageSignal := clamp100(float64(profile.Recent7Messages)*5 + float64(profile.Previous30Messages)*1.6)
	recentSessionSignal := clamp100(float64(profile.Recent7Sessions)*14 + float64(profile.Previous30Sessions)*5)
	activitySignal := (recentMessageSignal*0.7 + recentSessionSignal*0.3) / 100
	if board == "cooling" || board == "faded" {
		return 0.75 + activitySignal*0.25
	}
	return 0.45 + activitySignal*0.55
}

func relationRankScore(board string, score float64, profile *relationProfile) float64 {
	return clamp100(score * freshnessRankFactor(board, profile.DaysSinceLastContact) * recentActivityWeight(profile, board))
}

func controversyRankScore(label string, score float64, profile *relationProfile) float64 {
	return clamp100(score * freshnessRankFactor(label, profile.DaysSinceLastContact) * recentActivityWeight(profile, label))
}

func buildStaleHint(daysSinceLastContact int, historical bool) string {
	switch {
	case daysSinceLastContact <= confidenceFreshDays1:
		return ""
	case daysSinceLastContact <= confidenceFreshDays2:
		return "近 90 天联系较少，当前判断置信度已下调"
	case daysSinceLastContact <= confidenceFreshDays3:
		if historical {
			return "近半年联系较少，该结论更偏历史回看"
		}
		return "近半年联系较少，当前判断置信度已下调"
	default:
		if historical {
			return "长期未联系，该结论主要基于历史数据"
		}
		return "长期未联系，当前判断置信度已显著下调"
	}
}

func peakMonth(monthCounts map[string]int) (string, int) {
	bestMonth := ""
	bestCount := 0
	for month, count := range monthCounts {
		if count > bestCount || (count == bestCount && month > bestMonth) {
			bestMonth = month
			bestCount = count
		}
	}
	return bestMonth, bestCount
}

func monthEndString(t time.Time) string {
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	end := start.AddDate(0, 1, -1)
	return end.Format("2006-01-02")
}

func sharedGroupHint(count int) string {
	switch {
	case count >= 6:
		return "共同群聊很多，社交重叠高"
	case count >= 2:
		return "有一定共同圈层"
	default:
		return "主要靠私聊维系"
	}
}

func formatDurationCN(seconds float64) string {
	if seconds <= 0 {
		return "暂无样本"
	}
	d := time.Duration(seconds) * time.Second
	if d < time.Minute {
		return fmt.Sprintf("%d 秒", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d 分钟", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes == 0 {
			return fmt.Sprintf("%d 小时", hours)
		}
		return fmt.Sprintf("%d 小时 %d 分", hours, minutes)
	}
	return fmt.Sprintf("%.1f 天", d.Hours()/24)
}

func emptyIf(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func safeTZ() *time.Location {
	return time.Local
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (p *relationProfile) confidence() float64 {
	return controversyConfidence(p, 15)
}

func (p *relationProfile) confidenceReason(minMessages int) string {
	_, reason := confidenceWithReason(p, minMessages)
	return reason
}

func (p *relationProfile) lastSeenText() string {
	if p.DaysSinceLastContact <= 0 {
		return "刚刚还在聊"
	}
	if p.DaysSinceLastContact == 1 {
		return "昨天还有联系"
	}
	return fmt.Sprintf("%d 天前联系过", p.DaysSinceLastContact)
}

var _ = model.ContactStats{}
