package export

import "time"

const (
	ChatLabVersion = "0.0.1"

	ChatLabTypeText     = 0
	ChatLabTypeImage    = 1
	ChatLabTypeVoice    = 2
	ChatLabTypeVideo    = 3
	ChatLabTypeFile     = 4
	ChatLabTypeEmoji    = 5
	ChatLabTypeLink     = 7
	ChatLabTypeLocation = 8

	ChatLabTypeRedPacket = 20
	ChatLabTypeTransfer  = 21
	ChatLabTypePoke      = 22
	ChatLabTypeCall      = 23
	ChatLabTypeShare     = 24
	ChatLabTypeReply     = 25
	ChatLabTypeForward   = 26
	ChatLabTypeContact   = 27

	ChatLabTypeSystem = 80
	ChatLabTypeRecall = 81
	ChatLabTypeOther  = 99
)

type ChatLab struct {
	ChatLab  Header    `json:"chatlab"`
	Meta     Meta      `json:"meta"`
	Members  []Member  `json:"members"`
	Messages []Message `json:"messages"`
}

type Header struct {
	Version     string `json:"version"`
	ExportedAt  int64  `json:"exportedAt"`
	Generator   string `json:"generator,omitempty"`
	Description string `json:"description,omitempty"`
}

type Meta struct {
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	Type        string `json:"type"`
	GroupID     string `json:"groupId,omitempty"`
	GroupAvatar string `json:"groupAvatar,omitempty"`
}

type Member struct {
	PlatformID    string   `json:"platformId"`
	AccountName   string   `json:"accountName"`
	GroupNickname string   `json:"groupNickname,omitempty"`
	Aliases       []string `json:"aliases,omitempty"`
	Avatar        string   `json:"avatar,omitempty"`
}

type Message struct {
	Sender        string `json:"sender"`
	AccountName   string `json:"accountName"`
	GroupNickname string `json:"groupNickname,omitempty"`
	Timestamp     int64  `json:"timestamp"`
	Type          int    `json:"type"`
	Content       string `json:"content"`
}

type ConversationMeta struct {
	Name        string
	Platform    string
	Type        string
	GroupID     string
	GroupAvatar string
	Description string
	Generator   string
}

type MemberRecord struct {
	PlatformID    string
	AccountName   string
	GroupNickname string
	Aliases       []string
	Avatar        string
}

type MessageRecord struct {
	Sender        string
	AccountName   string
	GroupNickname string
	Timestamp     int64
	Type          int
	Content       string
}

func BuildChatLab(meta ConversationMeta, members []MemberRecord, messages []MessageRecord) ChatLab {
	if meta.Platform == "" {
		meta.Platform = "wechat"
	}
	if meta.Type == "" {
		meta.Type = "private"
	}
	if meta.Generator == "" {
		meta.Generator = "WeLink"
	}

	out := ChatLab{
		ChatLab: Header{
			Version:     ChatLabVersion,
			ExportedAt:  time.Now().Unix(),
			Generator:   meta.Generator,
			Description: meta.Description,
		},
		Meta: Meta{
			Name:        meta.Name,
			Platform:    meta.Platform,
			Type:        meta.Type,
			GroupID:     meta.GroupID,
			GroupAvatar: meta.GroupAvatar,
		},
		Members:  make([]Member, 0, len(members)),
		Messages: make([]Message, 0, len(messages)),
	}

	for _, member := range members {
		out.Members = append(out.Members, Member{
			PlatformID:    member.PlatformID,
			AccountName:   member.AccountName,
			GroupNickname: member.GroupNickname,
			Aliases:       member.Aliases,
			Avatar:        member.Avatar,
		})
	}

	for _, message := range messages {
		out.Messages = append(out.Messages, Message{
			Sender:        message.Sender,
			AccountName:   message.AccountName,
			GroupNickname: message.GroupNickname,
			Timestamp:     message.Timestamp,
			Type:          message.Type,
			Content:       message.Content,
		})
	}

	return out
}

func InferWeLinkMessageType(localType int, content string) int {
	switch localType {
	case 1:
		return ChatLabTypeText
	case 3:
		return ChatLabTypeImage
	case 34:
		return ChatLabTypeVoice
	case 43:
		return ChatLabTypeVideo
	case 47:
		return ChatLabTypeEmoji
	case 48:
		return ChatLabTypeLocation
	case 49:
		switch content {
		case "[红包/转账]":
			return ChatLabTypeRedPacket
		case "[链接/文件]":
			return ChatLabTypeLink
		case "[文件/链接]":
			return ChatLabTypeFile
		default:
			return ChatLabTypeShare
		}
	default:
		if localType >= 10000 {
			return ChatLabTypeSystem
		}
		return ChatLabTypeOther
	}
}
