package service

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	snsKindAll         = "all"
	snsKindPost        = "post"
	snsKindInteraction = "interaction"
	snsKindIndex       = "index"
	snsDefaultLimit    = 100
	snsMaxLimit        = 500
)

type SnsSearchParams struct {
	Q        string
	Username string
	Kind     string
	From     string
	To       string
	Limit    int
}

type SnsSearchItem struct {
	Kind                 string `json:"kind"`
	FeedID               string `json:"feed_id"`
	Username             string `json:"username"`
	DisplayName          string `json:"display_name"`
	CreatedAt            string `json:"created_at"`
	ContentText          string `json:"content_text"`
	RawContent           string `json:"raw_content"`
	CounterpartyUsername string `json:"counterparty_username,omitempty"`
	CounterpartyName     string `json:"counterparty_name,omitempty"`
	createdAtUnix        int64
}

type SnsSearchResponse struct {
	Available         bool            `json:"available"`
	HasSNSDB          bool            `json:"has_sns_db"`
	Message           string          `json:"message,omitempty"`
	UnavailableReason string          `json:"unavailable_reason,omitempty"`
	Items             []SnsSearchItem `json:"items"`
	Total             int             `json:"total"`
}

type snsTimelinePayload struct {
	TimelineObject struct {
		ID          string `xml:"id"`
		Username    string `xml:"username"`
		CreateTime  string `xml:"createTime"`
		ContentDesc string `xml:"contentDesc"`
		SourceNick  string `xml:"sourceNickName"`
		LocalExtra  struct {
			Nickname string `xml:"nickname"`
		} `xml:"LocalExtraInfo"`
	} `xml:"TimelineObject"`
}

func (s *ContactService) SearchSNS(params SnsSearchParams) (*SnsSearchResponse, error) {
	kind := strings.ToLower(strings.TrimSpace(params.Kind))
	if kind == "" {
		kind = snsKindAll
	}
	if kind != snsKindAll && kind != snsKindPost && kind != snsKindInteraction && kind != snsKindIndex {
		return nil, fmt.Errorf("kind must be all|post|interaction|index")
	}

	limit := params.Limit
	if limit <= 0 {
		limit = snsDefaultLimit
	}
	if limit > snsMaxLimit {
		limit = snsMaxLimit
	}

	fromTs, err := s.parseSNSBound(params.From, false)
	if err != nil {
		return nil, fmt.Errorf("invalid from: %w", err)
	}
	toTs, err := s.parseSNSBound(params.To, true)
	if err != nil {
		return nil, fmt.Errorf("invalid to: %w", err)
	}
	if fromTs > 0 && toTs > 0 && fromTs > toTs {
		return nil, fmt.Errorf("from must be <= to")
	}

	snsPath := filepath.Join(s.dbMgr.DataDir, "sns", "sns.db")
	if info, statErr := os.Stat(snsPath); statErr != nil || info.IsDir() {
		return &SnsSearchResponse{
			Available:         false,
			HasSNSDB:          false,
			Message:           "sns database not found",
			UnavailableReason: "未检测到 sns.db",
			Items:             []SnsSearchItem{},
			Total:             0,
		}, nil
	}

	snsDB, err := sql.Open("sqlite", snsPath)
	if err != nil {
		return nil, fmt.Errorf("open sns db failed: %w", err)
	}
	defer snsDB.Close()

	nameMap := s.loadContactNameMap()
	queryText := strings.ToLower(strings.TrimSpace(params.Q))
	username := strings.TrimSpace(params.Username)
	scanLimit := limit * 20
	if scanLimit < 200 {
		scanLimit = 200
	}
	if scanLimit > 5000 {
		scanLimit = 5000
	}

	items := make([]SnsSearchItem, 0, limit)
	if kind == snsKindAll || kind == snsKindPost {
		postItems, postErr := s.searchSNSPosts(snsDB, nameMap, queryText, username, fromTs, toTs, scanLimit)
		if postErr != nil {
			return nil, postErr
		}
		items = append(items, postItems...)
	}
	if kind == snsKindAll || kind == snsKindInteraction {
		interactionItems, interactionErr := s.searchSNSInteractions(snsDB, nameMap, queryText, username, fromTs, toTs, scanLimit)
		if interactionErr != nil {
			return nil, interactionErr
		}
		items = append(items, interactionItems...)
	}
	if kind == snsKindAll || kind == snsKindIndex {
		indexItems, indexErr := s.searchSNSIndex(snsDB, nameMap, queryText, username, fromTs, toTs, scanLimit)
		if indexErr != nil {
			return nil, indexErr
		}
		items = append(items, indexItems...)
	}

	sortSNSItems(items, kind == snsKindAll)
	if len(items) > limit {
		items = items[:limit]
	}

	return &SnsSearchResponse{
		Available: true,
		HasSNSDB:  true,
		Items:     items,
		Total:     len(items),
	}, nil
}

func sortSNSItems(items []SnsSearchItem, prioritizeKind bool) {
	sort.Slice(items, func(i, j int) bool {
		iRank := snsKindRank(items[i].Kind)
		jRank := snsKindRank(items[j].Kind)
		if prioritizeKind && iRank != jRank {
			return iRank < jRank
		}
		if items[i].createdAtUnix != items[j].createdAtUnix {
			return items[i].createdAtUnix > items[j].createdAtUnix
		}
		if !prioritizeKind && iRank != jRank {
			return iRank < jRank
		}
		return items[i].FeedID > items[j].FeedID
	})
}

func snsKindRank(kind string) int {
	switch kind {
	case snsKindPost:
		return 0
	case snsKindInteraction:
		return 1
	case snsKindIndex:
		return 2
	default:
		return 99
	}
}

func (s *ContactService) searchSNSPosts(db *sql.DB, nameMap map[string]string, queryText, username string, fromTs, toTs int64, scanLimit int) ([]SnsSearchItem, error) {
	rows, err := db.Query(
		"SELECT COALESCE(tid,''), COALESCE(user_name,''), COALESCE(content,'') FROM SnsTimeLine ORDER BY rowid DESC LIMIT ?",
		scanLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("query SnsTimeLine failed: %w", err)
	}
	defer rows.Close()

	items := make([]SnsSearchItem, 0, 64)
	for rows.Next() {
		var tid, userName, rawContent string
		if scanErr := rows.Scan(&tid, &userName, &rawContent); scanErr != nil {
			continue
		}
		feedID, actor, createdAt, contentText, xmlDisplay := parseTimelineContent(rawContent, tid, userName)
		if actor == "" {
			actor = userName
		}
		if username != "" && actor != username {
			continue
		}
		if !withinSNSRange(createdAt, fromTs, toTs) {
			continue
		}
		if queryText != "" && !snsContains(contentText, queryText) && !snsContains(rawContent, queryText) {
			continue
		}

		display := strings.TrimSpace(xmlDisplay)
		if display == "" {
			display = strings.TrimSpace(nameMap[actor])
		}
		if display == "" {
			display = actor
		}
		if contentText == "" {
			contentText = "[朋友圈内容]"
		}

		items = append(items, SnsSearchItem{
			Kind:          snsKindPost,
			FeedID:        feedID,
			Username:      actor,
			DisplayName:   display,
			CreatedAt:     s.formatSNSUnix(createdAt),
			ContentText:   contentText,
			RawContent:    rawContent,
			createdAtUnix: createdAt,
		})
	}
	return items, nil
}

func (s *ContactService) searchSNSInteractions(db *sql.DB, nameMap map[string]string, queryText, username string, fromTs, toTs int64, scanLimit int) ([]SnsSearchItem, error) {
	rows, err := db.Query(
		`SELECT
			COALESCE(create_time,0),
			COALESCE(feed_id,''),
			COALESCE(from_username,''),
			COALESCE(from_nickname,''),
			COALESCE(to_username,''),
			COALESCE(to_nickname,''),
			COALESCE(content,'')
		FROM SnsMessage_tmp3
		ORDER BY create_time DESC
		LIMIT ?`,
		scanLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("query SnsMessage_tmp3 failed: %w", err)
	}
	defer rows.Close()

	items := make([]SnsSearchItem, 0, 64)
	for rows.Next() {
		var createdAt int64
		var feedID, fromUsername, fromNick, toUsername, toNick, content string
		if scanErr := rows.Scan(&createdAt, &feedID, &fromUsername, &fromNick, &toUsername, &toNick, &content); scanErr != nil {
			continue
		}
		if username != "" && fromUsername != username && toUsername != username {
			continue
		}
		if !withinSNSRange(createdAt, fromTs, toTs) {
			continue
		}
		if queryText != "" &&
			!snsContains(content, queryText) &&
			!snsContains(fromUsername, queryText) &&
			!snsContains(fromNick, queryText) &&
			!snsContains(toUsername, queryText) &&
			!snsContains(toNick, queryText) {
			continue
		}

		displayName := strings.TrimSpace(fromNick)
		if displayName == "" {
			displayName = strings.TrimSpace(nameMap[fromUsername])
		}
		if displayName == "" {
			displayName = fromUsername
		}
		counterpartyName := strings.TrimSpace(toNick)
		if counterpartyName == "" {
			counterpartyName = strings.TrimSpace(nameMap[toUsername])
		}
		if counterpartyName == "" {
			counterpartyName = toUsername
		}

		items = append(items, SnsSearchItem{
			Kind:                 snsKindInteraction,
			FeedID:               strings.TrimSpace(feedID),
			Username:             strings.TrimSpace(fromUsername),
			DisplayName:          displayName,
			CreatedAt:            s.formatSNSUnix(createdAt),
			ContentText:          strings.TrimSpace(content),
			RawContent:           content,
			CounterpartyUsername: strings.TrimSpace(toUsername),
			CounterpartyName:     counterpartyName,
			createdAtUnix:        createdAt,
		})
	}
	return items, nil
}

func (s *ContactService) searchSNSIndex(db *sql.DB, nameMap map[string]string, queryText, username string, fromTs, toTs int64, scanLimit int) ([]SnsSearchItem, error) {
	rows, err := db.Query(
		`SELECT
			COALESCE(CAST(tid AS TEXT), ''),
			COALESCE(username, ''),
			COALESCE(summary, ''),
			COALESCE(MAX(create_time), 0),
			COALESCE(MAX(last_read_time), 0),
			COALESCE(MAX(is_read), 0)
		FROM SnsTopItem_1
		GROUP BY CAST(tid AS TEXT), username, summary
		ORDER BY MAX(create_time) DESC
		LIMIT ?`,
		scanLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("query SnsTopItem_1 failed: %w", err)
	}
	defer rows.Close()

	items := make([]SnsSearchItem, 0, 64)
	for rows.Next() {
		var tid, actor, summary string
		var createdAt, lastReadAt int64
		var isRead int
		if scanErr := rows.Scan(&tid, &actor, &summary, &createdAt, &lastReadAt, &isRead); scanErr != nil {
			continue
		}
		tid = strings.TrimSpace(tid)
		actor = strings.TrimSpace(actor)
		summary = strings.TrimSpace(summary)
		if actor == "" {
			continue
		}
		if username != "" && actor != username {
			continue
		}
		if !withinSNSRange(createdAt, fromTs, toTs) {
			continue
		}
		displayName := strings.TrimSpace(nameMap[actor])
		if displayName == "" {
			displayName = actor
		}
		searchBlob := strings.Join([]string{
			actor,
			displayName,
			summary,
			tid,
		}, " ")
		if queryText != "" && !snsContains(searchBlob, queryText) {
			continue
		}

		contentText := summary
		if contentText == "" {
			contentText = "[朋友圈索引记录，正文未同步到 sns.db]"
		}
		rawContent := fmt.Sprintf(`{"summary":%q,"last_read_time":%d,"is_read":%t}`, summary, lastReadAt, isRead != 0)
		items = append(items, SnsSearchItem{
			Kind:          snsKindIndex,
			FeedID:        tid,
			Username:      actor,
			DisplayName:   displayName,
			CreatedAt:     s.formatSNSUnix(createdAt),
			ContentText:   contentText,
			RawContent:    rawContent,
			createdAtUnix: createdAt,
		})
	}
	return items, nil
}

func parseTimelineContent(rawContent, fallbackFeedID, fallbackUsername string) (feedID string, username string, createdAt int64, contentText string, displayName string) {
	feedID = strings.TrimSpace(fallbackFeedID)
	username = strings.TrimSpace(fallbackUsername)
	contentText = ""
	displayName = ""

	rawContent = strings.TrimSpace(rawContent)
	if rawContent == "" {
		return feedID, username, 0, contentText, displayName
	}

	var payload snsTimelinePayload
	if err := xml.Unmarshal([]byte(rawContent), &payload); err != nil {
		return feedID, username, 0, contentText, displayName
	}

	if id := strings.TrimSpace(payload.TimelineObject.ID); id != "" {
		feedID = id
	}
	if actor := strings.TrimSpace(payload.TimelineObject.Username); actor != "" {
		username = actor
	}
	if ct := strings.TrimSpace(payload.TimelineObject.CreateTime); ct != "" {
		if ts, parseErr := strconv.ParseInt(ct, 10, 64); parseErr == nil {
			createdAt = ts
		}
	}
	contentText = strings.TrimSpace(payload.TimelineObject.ContentDesc)
	if nick := strings.TrimSpace(payload.TimelineObject.LocalExtra.Nickname); nick != "" {
		displayName = nick
	} else if sourceNick := strings.TrimSpace(payload.TimelineObject.SourceNick); sourceNick != "" {
		displayName = sourceNick
	}

	return feedID, username, createdAt, contentText, displayName
}

func withinSNSRange(createdAt, fromTs, toTs int64) bool {
	if createdAt <= 0 {
		return fromTs == 0 && toTs == 0
	}
	if fromTs > 0 && createdAt < fromTs {
		return false
	}
	if toTs > 0 && createdAt > toTs {
		return false
	}
	return true
}

func snsContains(content, query string) bool {
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(content), query)
}

func (s *ContactService) parseSNSBound(raw string, endOfDay bool) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	if ts, err := strconv.ParseInt(value, 10, 64); err == nil {
		return ts, nil
	}

	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, value, s.tz)
		if err != nil {
			continue
		}
		if layout == "2006-01-02" && endOfDay {
			return t.Add(23*time.Hour + 59*time.Minute + 59*time.Second).Unix(), nil
		}
		return t.Unix(), nil
	}
	return 0, fmt.Errorf("unsupported datetime format: %s", value)
}

func (s *ContactService) formatSNSUnix(ts int64) string {
	if ts <= 0 {
		return "-"
	}
	return time.Unix(ts, 0).In(s.tz).Format("2006-01-02 15:04:05")
}
