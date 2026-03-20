package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"welink/backend/export"
	"welink/backend/service"

	"github.com/gin-gonic/gin"
)

type chatLabEnvelope struct {
	FileName string         `json:"file_name"`
	MIMEType string         `json:"mime_type"`
	Data     export.ChatLab `json:"data"`
	Summary  map[string]any `json:"summary,omitempty"`
}

func registerExportRoutes(api *gin.RouterGroup, contactSvc *service.ContactService, ready func() bool) {
	withAnalysisData := func(handler gin.HandlerFunc) gin.HandlerFunc {
		return func(c *gin.Context) {
			if ready != nil && !ready() {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "analysis data not ready"})
				return
			}
			handler(c)
		}
	}
	api.GET("/export/chatlab/contact", withAnalysisData(func(c *gin.Context) {
		username := strings.TrimSpace(c.Query("username"))
		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username required"})
			return
		}
		limit := parseExportLimit(c.DefaultQuery("limit", "200"))
		payload := buildContactChatLab(contactSvc, username, limit)
		summary := map[string]any{
			"scope":             "contact",
			"username":          username,
			"limit":             limit,
			"conversation_name": payload.Meta.Name,
			"message_count":     len(payload.Messages),
			"member_count":      len(payload.Members),
		}
		respondChatLab(c, payload, sanitizeExportFilename(username)+".chatlab.json", summary)
	}))

	api.GET("/export/chatlab/group", withAnalysisData(func(c *gin.Context) {
		username := strings.TrimSpace(c.Query("username"))
		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username required"})
			return
		}
		date := strings.TrimSpace(c.Query("date"))
		if date == "" {
			date = inferGroupExportDate(contactSvc, username)
		}
		payload := buildGroupChatLab(contactSvc, username, date)
		summary := map[string]any{
			"scope":             "group",
			"username":          username,
			"date":              date,
			"conversation_name": payload.Meta.Name,
			"message_count":     len(payload.Messages),
			"member_count":      len(payload.Members),
		}
		respondChatLab(c, payload, sanitizeExportFilename(username+"-"+date)+".chatlab.json", summary)
	}))

	api.GET("/export/chatlab/search", withAnalysisData(func(c *gin.Context) {
		query := strings.TrimSpace(c.Query("q"))
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "q required"})
			return
		}
		limit := parseExportLimit(c.DefaultQuery("limit", "200"))
		includeMine := c.DefaultQuery("include_mine", "true") != "false"
		payload := buildSearchChatLab(contactSvc, query, includeMine, limit)
		summary := map[string]any{
			"scope":             "search",
			"query":             query,
			"include_mine":      includeMine,
			"limit":             limit,
			"conversation_name": payload.Meta.Name,
			"message_count":     len(payload.Messages),
			"member_count":      len(payload.Members),
		}
		respondChatLab(c, payload, sanitizeExportFilename("search-"+query)+".chatlab.json", summary)
	}))

	api.POST("/export/chatlab", withAnalysisData(func(c *gin.Context) {
		var body struct {
			Scope       string `json:"scope"`
			Username    string `json:"username"`
			Query       string `json:"query"`
			Q           string `json:"q"`
			Date        string `json:"date"`
			IncludeMine string `json:"include_mine"`
			Limit       int    `json:"limit"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}

		scope := strings.TrimSpace(body.Scope)
		username := strings.TrimSpace(body.Username)
		query := strings.TrimSpace(body.Query)
		if query == "" {
			query = strings.TrimSpace(body.Q)
		}
		date := strings.TrimSpace(body.Date)
		limit := body.Limit
		if limit <= 0 {
			limit = 200
		}
		includeMine := strings.TrimSpace(body.IncludeMine) != "false"

		switch scope {
		case "contact":
			if username == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "username required"})
				return
			}
			payload := buildContactChatLab(contactSvc, username, limit)
			summary := map[string]any{
				"scope":             "contact",
				"username":          username,
				"limit":             limit,
				"conversation_name": payload.Meta.Name,
				"message_count":     len(payload.Messages),
				"member_count":      len(payload.Members),
			}
			respondChatLab(c, payload, sanitizeExportFilename(username)+".chatlab.json", summary)
		case "group":
			if username == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "username required"})
				return
			}
			if date == "" {
				date = inferGroupExportDate(contactSvc, username)
			}
			payload := buildGroupChatLab(contactSvc, username, date)
			summary := map[string]any{
				"scope":             "group",
				"username":          username,
				"date":              date,
				"conversation_name": payload.Meta.Name,
				"message_count":     len(payload.Messages),
				"member_count":      len(payload.Members),
			}
			respondChatLab(c, payload, sanitizeExportFilename(username+"-"+date)+".chatlab.json", summary)
		case "search":
			if query == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "query required"})
				return
			}
			payload := buildSearchChatLab(contactSvc, query, includeMine, limit)
			summary := map[string]any{
				"scope":             "search",
				"query":             query,
				"include_mine":      includeMine,
				"limit":             limit,
				"conversation_name": payload.Meta.Name,
				"message_count":     len(payload.Messages),
				"member_count":      len(payload.Members),
			}
			respondChatLab(c, payload, sanitizeExportFilename("search-"+query)+".chatlab.json", summary)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported scope"})
		}
	}))
}

func respondChatLab(c *gin.Context, payload export.ChatLab, fileName string, summary map[string]any) {
	if c.Query("download") == "true" {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	}
	c.JSON(http.StatusOK, chatLabEnvelope{
		FileName: fileName,
		MIMEType: "application/json",
		Data:     payload,
		Summary:  summary,
	})
}

func buildContactChatLab(contactSvc *service.ContactService, username string, limit int) export.ChatLab {
	displayName := username
	for _, contact := range contactSvc.GetCachedStats() {
		if contact.Username == username {
			if contact.Remark != "" {
				displayName = contact.Remark
			} else if contact.Nickname != "" {
				displayName = contact.Nickname
			}
			break
		}
	}

	raw := contactSvc.GetMessageHistory(username, 0, limit)
	messages := make([]export.MessageRecord, 0, len(raw))
	for _, msg := range raw {
		sender := username
		accountName := displayName
		if msg.IsMine {
			sender = "__self__"
			accountName = "我"
		}
		messages = append(messages, export.MessageRecord{
			Sender:      sender,
			AccountName: accountName,
			Timestamp:   msg.Timestamp,
			Type:        export.InferWeLinkMessageType(msg.Type, msg.Content),
			Content:     msg.Content,
		})
	}

	return export.BuildChatLab(export.ConversationMeta{
		Name:        displayName,
		Platform:    "wechat",
		Type:        "private",
		Generator:   "WeLink",
		Description: "WeLink contact export",
	}, []export.MemberRecord{
		{PlatformID: "__self__", AccountName: "我"},
		{PlatformID: username, AccountName: displayName},
	}, messages)
}

func buildGroupChatLab(contactSvc *service.ContactService, username, date string) export.ChatLab {
	groupName := username
	for _, group := range contactSvc.GetGroups() {
		if group.Username == username {
			if group.Name != "" {
				groupName = group.Name
			}
			break
		}
	}

	raw := contactSvc.GetGroupDayMessages(username, date)
	messages := make([]export.MessageRecord, 0, len(raw))
	memberMap := map[string]export.MemberRecord{}
	for _, msg := range raw {
		senderID := msg.Speaker
		memberMap[senderID] = export.MemberRecord{
			PlatformID:    senderID,
			AccountName:   msg.Speaker,
			GroupNickname: msg.Speaker,
		}
		messages = append(messages, export.MessageRecord{
			Sender:        senderID,
			AccountName:   msg.Speaker,
			GroupNickname: msg.Speaker,
			Timestamp:     parseDayTime(date, msg.Time),
			Type:          export.InferWeLinkMessageType(msg.Type, msg.Content),
			Content:       msg.Content,
		})
	}

	members := make([]export.MemberRecord, 0, len(memberMap))
	for _, member := range memberMap {
		members = append(members, member)
	}

	return export.BuildChatLab(export.ConversationMeta{
		Name:        groupName,
		Platform:    "wechat",
		Type:        "group",
		GroupID:     username,
		Generator:   "WeLink",
		Description: "WeLink group export for " + date,
	}, members, messages)
}

func buildSearchChatLab(contactSvc *service.ContactService, query string, includeMine bool, limit int) export.ChatLab {
	raw := contactSvc.SearchAllMessages(query, includeMine, limit)
	messages := make([]export.MessageRecord, 0, len(raw))
	memberMap := map[string]export.MemberRecord{
		"__self__": {PlatformID: "__self__", AccountName: "我"},
	}
	for _, hit := range raw {
		senderID := hit.Username
		accountName := hit.Name
		groupNickname := ""
		if hit.IsMine {
			senderID = "__self__"
			accountName = "我"
			groupNickname = hit.Name
		} else {
			memberMap[hit.Username] = export.MemberRecord{
				PlatformID:    hit.Username,
				AccountName:   hit.Name,
				GroupNickname: hit.Name,
			}
			groupNickname = hit.Name
		}
		messages = append(messages, export.MessageRecord{
			Sender:        senderID,
			AccountName:   accountName,
			GroupNickname: groupNickname,
			Timestamp:     hit.Ts,
			Type:          export.InferWeLinkMessageType(hit.Type, hit.Content),
			Content:       hit.Content,
		})
	}

	members := make([]export.MemberRecord, 0, len(memberMap))
	for _, member := range memberMap {
		members = append(members, member)
	}

	return export.BuildChatLab(export.ConversationMeta{
		Name:        "搜索结果",
		Platform:    "wechat",
		Type:        "group",
		GroupID:     "search:" + query,
		Generator:   "WeLink",
		Description: "WeLink search export for query: " + query,
	}, members, messages)
}

func inferGroupExportDate(contactSvc *service.ContactService, username string) string {
	for _, group := range contactSvc.GetGroups() {
		if group.Username != username {
			continue
		}
		if len(group.LastMessage) >= 10 {
			return group.LastMessage[:10]
		}
		break
	}
	return time.Now().Format("2006-01-02")
}

func parseExportLimit(raw string) int {
	limit := 200
	if _, err := fmt.Sscanf(raw, "%d", &limit); err != nil || limit <= 0 {
		return 200
	}
	return limit
}

func parseDayTime(date, hhmm string) int64 {
	t, err := time.ParseInLocation("2006-01-02 15:04", date+" "+hhmm, time.Local)
	if err != nil {
		return 0
	}
	return t.Unix()
}

func sanitizeExportFilename(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "welink-export"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", ":", "-", "@", "_", "\"", "")
	return replacer.Replace(value)
}
