package discord

import (
	"encoding/json"
	"time"
)

const EpochMS = 1420070400000

type ExecutePayload struct {
	Content         *string         `json:"content"`
	Username        *string         `json:"username"`
	AvatarURL       *string         `json:"avatar_url"`
	TTS             *bool           `json:"tts"`
	Embeds          json.RawMessage `json:"embeds"`
	AllowedMentions json.RawMessage `json:"allowed_mentions"`
	Components      json.RawMessage `json:"components"`
	Flags           *int            `json:"flags"`
	ThreadName      *string         `json:"thread_name"`
	Attachments     json.RawMessage `json:"attachments"`
	Poll            json.RawMessage `json:"poll"`
}

func (p ExecutePayload) HasDeliverableContent(hasFiles bool) bool {
	if hasFiles {
		return true
	}
	if p.Content != nil && *p.Content != "" {
		return true
	}
	if len(p.Embeds) > 0 && string(p.Embeds) != "null" && string(p.Embeds) != "[]" {
		return true
	}
	if len(p.Components) > 0 && string(p.Components) != "null" && string(p.Components) != "[]" {
		return true
	}
	if len(p.Poll) > 0 && string(p.Poll) != "null" && string(p.Poll) != "{}" {
		return true
	}
	return false
}

type Webhook struct {
	ID          string  `json:"id"`
	Type        int     `json:"type"`
	Name        string  `json:"name"`
	Avatar      *string `json:"avatar"`
	ChannelID   string  `json:"channel_id"`
	GuildID     *string `json:"guild_id,omitempty"`
	Token       string  `json:"token,omitempty"`
	Application *string `json:"application_id"`
}

type Author struct {
	ID            string  `json:"id"`
	Username      string  `json:"username"`
	Avatar        *string `json:"avatar"`
	Discriminator string  `json:"discriminator"`
	Bot           bool    `json:"bot"`
}

type Message struct {
	ID              string          `json:"id"`
	Type            int             `json:"type"`
	Content         string          `json:"content"`
	ChannelID       string          `json:"channel_id"`
	Author          Author          `json:"author"`
	Attachments     []any           `json:"attachments"`
	Embeds          json.RawMessage `json:"embeds"`
	Mentions        []any           `json:"mentions"`
	MentionRoles    []any           `json:"mention_roles"`
	Pinned          bool            `json:"pinned"`
	MentionEveryone bool            `json:"mention_everyone"`
	TTS             bool            `json:"tts"`
	Timestamp       string          `json:"timestamp"`
	EditedTimestamp *string         `json:"edited_timestamp"`
	Flags           int             `json:"flags"`
	Components      json.RawMessage `json:"components,omitempty"`
	WebhookID       string          `json:"webhook_id"`
}

type APIError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func MessageSnowflake(eventID int64) string {
	ts := time.Now().UnixMilli() - EpochMS
	if ts < 0 {
		ts = 0
	}
	id := (ts << 22) | (eventID & 0x3FFFFF)
	return formatSnowflake(id)
}

func WebhookSnowflake(n int64) string {
	ts := time.Now().UnixMilli() - EpochMS
	if ts < 0 {
		ts = 0
	}
	id := (ts << 22) | ((n + 1) << 10) | (n & 0x3FF)
	return formatSnowflake(id)
}

func ChannelSnowflake(n int64) string {
	ts := time.Now().UnixMilli() - EpochMS
	if ts < 0 {
		ts = 0
	}
	id := (ts << 22) | (2 << 17) | (n & 0x1FFFF)
	return formatSnowflake(id)
}

func formatSnowflake(id int64) string {
	if id < 0 {
		id = -id
	}
	return jsonNumber(id)
}

func jsonNumber(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func BuildMessage(eventID int64, webhook Webhook, payload ExecutePayload, username string, avatar *string) Message {
	content := ""
	if payload.Content != nil {
		content = *payload.Content
	}
	tts := false
	if payload.TTS != nil {
		tts = *payload.TTS
	}
	flags := 0
	if payload.Flags != nil {
		flags = *payload.Flags
	}
	embeds := payload.Embeds
	if len(embeds) == 0 {
		embeds = json.RawMessage("[]")
	}
	components := payload.Components
	if len(components) == 0 {
		components = nil
	}

	return Message{
		ID:        MessageSnowflake(eventID),
		Type:      0,
		Content:   content,
		ChannelID: webhook.ChannelID,
		Author: Author{
			ID:            webhook.ID,
			Username:      username,
			Avatar:        avatar,
			Discriminator: "0000",
			Bot:           true,
		},
		Attachments:     []any{},
		Embeds:          embeds,
		Mentions:        []any{},
		MentionRoles:    []any{},
		Pinned:          false,
		MentionEveryone: false,
		TTS:             tts,
		Timestamp:       time.Now().UTC().Format(time.RFC3339Nano),
		EditedTimestamp: nil,
		Flags:           flags,
		Components:      components,
		WebhookID:       webhook.ID,
	}
}
