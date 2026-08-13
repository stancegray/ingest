package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/main/ingest/internal/discord"
)

var ErrWebhookNotFound = errors.New("webhook not found")
var ErrMessageNotFound = errors.New("message not found")

type WebhookRecord struct {
	ID        string
	Token     string
	Source    string
	Name      string
	Avatar    *string
	ChannelID string
	GuildID   *string
}

type CreateWebhookInput struct {
	Name   string
	Source string
	Avatar *string
}

type CreateWebhookResult struct {
	ID        string `json:"id"`
	Token     string `json:"token"`
	Name      string `json:"name"`
	ChannelID string `json:"channel_id"`
	URL       string `json:"url"`
	Source    string `json:"source"`
}

func (s *Store) CreateWebhook(ctx context.Context, in CreateWebhookInput) (CreateWebhookResult, error) {
	if in.Source == "" {
		in.Source = "webhook"
	}
	if in.Name == "" {
		in.Name = "Captain Hook"
	}

	var sourceID string
	err := s.pool.QueryRow(ctx, `SELECT id::text FROM sources WHERE name = $1`, in.Source).Scan(&sourceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return CreateWebhookResult{}, ErrSourceNotFound
	}
	if err != nil {
		return CreateWebhookResult{}, fmt.Errorf("lookup source: %w", err)
	}

	var seq int64
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM webhooks`).Scan(&seq); err != nil {
		return CreateWebhookResult{}, fmt.Errorf("count webhooks: %w", err)
	}

	webhookID := discord.WebhookSnowflake(seq)
	channelID := discord.ChannelSnowflake(seq)
	token, err := randomToken(32)
	if err != nil {
		return CreateWebhookResult{}, fmt.Errorf("generate token: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO webhooks (id, token, source_id, name, avatar, channel_id)
		VALUES ($1, $2, $3::uuid, $4, $5, $6)
	`, webhookID, token, sourceID, in.Name, in.Avatar, channelID)
	if err != nil {
		return CreateWebhookResult{}, fmt.Errorf("insert webhook: %w", err)
	}

	return CreateWebhookResult{
		ID:        webhookID,
		Token:     token,
		Name:      in.Name,
		ChannelID: channelID,
		URL:       "/api/webhooks/" + webhookID + "/" + token,
		Source:    in.Source,
	}, nil
}

func (s *Store) GetWebhook(ctx context.Context, id, token string) (WebhookRecord, error) {
	var wh WebhookRecord
	err := s.pool.QueryRow(ctx, `
		SELECT w.id, w.token, s.name, w.name, w.avatar, w.channel_id, w.guild_id
		FROM webhooks w
		JOIN sources s ON s.id = w.source_id
		WHERE w.id = $1 AND w.token = $2
	`, id, token).Scan(&wh.ID, &wh.Token, &wh.Source, &wh.Name, &wh.Avatar, &wh.ChannelID, &wh.GuildID)
	if errors.Is(err, pgx.ErrNoRows) {
		return WebhookRecord{}, ErrWebhookNotFound
	}
	if err != nil {
		return WebhookRecord{}, fmt.Errorf("get webhook: %w", err)
	}
	return wh, nil
}

type DiscordIngestInput struct {
	WebhookID string
	Token     string
	Payload   json.RawMessage
	Metadata  json.RawMessage
	ThreadID  *string
	Wait      bool
}

type DiscordIngestResult struct {
	EventID   int64
	MessageID string
	Webhook   WebhookRecord
	Payload   json.RawMessage
}

func (s *Store) IngestDiscord(ctx context.Context, in DiscordIngestInput) (DiscordIngestResult, error) {
	wh, err := s.GetWebhook(ctx, in.WebhookID, in.Token)
	if err != nil {
		return DiscordIngestResult{}, err
	}

	if len(in.Payload) == 0 || !json.Valid(in.Payload) {
		return DiscordIngestResult{}, fmt.Errorf("invalid payload")
	}
	if len(in.Metadata) == 0 {
		in.Metadata = json.RawMessage(`{}`)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return DiscordIngestResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var sourceID string
	err = tx.QueryRow(ctx, `SELECT id::text FROM sources WHERE name = $1`, wh.Source).Scan(&sourceID)
	if err != nil {
		return DiscordIngestResult{}, fmt.Errorf("lookup source: %w", err)
	}

	var eventID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO events (source_id, event_type, payload, metadata)
		VALUES ($1::uuid, 'discord.webhook', $2::jsonb, $3::jsonb)
		RETURNING id
	`, sourceID, in.Payload, in.Metadata).Scan(&eventID)
	if err != nil {
		return DiscordIngestResult{}, fmt.Errorf("insert event: %w", err)
	}

	messageID := discord.MessageSnowflake(eventID)
	_, err = tx.Exec(ctx, `UPDATE events SET external_id = $1 WHERE id = $2`, messageID, eventID)
	if err != nil {
		return DiscordIngestResult{}, fmt.Errorf("set message id: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return DiscordIngestResult{}, fmt.Errorf("commit tx: %w", err)
	}

	return DiscordIngestResult{
		EventID:   eventID,
		MessageID: messageID,
		Webhook:   wh,
		Payload:   in.Payload,
	}, nil
}

func (s *Store) UpdateDiscordMessage(ctx context.Context, webhookID, token, messageID string, payload json.RawMessage) (DiscordIngestResult, error) {
	wh, err := s.GetWebhook(ctx, webhookID, token)
	if err != nil {
		return DiscordIngestResult{}, err
	}

	tag, err := s.pool.Exec(ctx, `
		UPDATE events e
		SET payload = $1::jsonb,
		    metadata = metadata || jsonb_build_object('edited_at', to_jsonb(now()))
		FROM sources s, webhooks w
		WHERE e.external_id = $2
		  AND e.source_id = s.id
		  AND w.source_id = s.id
		  AND w.id = $3
		  AND w.token = $4
		  AND e.event_type = 'discord.webhook'
	`, payload, messageID, webhookID, token)
	if err != nil {
		return DiscordIngestResult{}, fmt.Errorf("update message: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return DiscordIngestResult{}, ErrMessageNotFound
	}

	var eventID int64
	err = s.pool.QueryRow(ctx, `
		SELECT e.id FROM events e
		JOIN sources s ON s.id = e.source_id
		JOIN webhooks w ON w.source_id = s.id
		WHERE e.external_id = $1 AND w.id = $2 AND w.token = $3
	`, messageID, webhookID, token).Scan(&eventID)
	if err != nil {
		return DiscordIngestResult{}, fmt.Errorf("lookup event: %w", err)
	}

	return DiscordIngestResult{
		EventID:   eventID,
		MessageID: messageID,
		Webhook:   wh,
		Payload:   payload,
	}, nil
}

func (s *Store) DeleteDiscordMessage(ctx context.Context, webhookID, token, messageID string) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM events e
		USING sources s, webhooks w
		WHERE e.external_id = $1
		  AND e.source_id = s.id
		  AND w.source_id = s.id
		  AND w.id = $2
		  AND w.token = $3
		  AND e.event_type = 'discord.webhook'
	`, messageID, webhookID, token)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
