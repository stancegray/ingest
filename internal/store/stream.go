package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Event struct {
	ID          int64           `json:"id"`
	Source      string          `json:"source"`
	EventType   string          `json:"event_type"`
	ExternalID  *string         `json:"external_id,omitempty"`
	Payload     json.RawMessage `json:"payload"`
	Metadata    json.RawMessage `json:"metadata"`
	RequestInfo json.RawMessage `json:"request_info"`
	IngestedAt  time.Time       `json:"ingested_at"`
}

func (s *Store) GetEventByID(ctx context.Context, id int64) (Event, error) {
	var event Event
	err := s.pool.QueryRow(ctx, `
		SELECT e.id, s.name, e.event_type, e.external_id, e.payload, e.metadata, e.request_info, e.ingested_at
		FROM events e
		JOIN sources s ON s.id = e.source_id
		WHERE e.id = $1
	`, id).Scan(
		&event.ID,
		&event.Source,
		&event.EventType,
		&event.ExternalID,
		&event.Payload,
		&event.Metadata,
		&event.RequestInfo,
		&event.IngestedAt,
	)
	if err != nil {
		return Event{}, fmt.Errorf("get event: %w", err)
	}
	return event, nil
}

func (s *Store) ListenEvents(ctx context.Context, onEvent func(Event) error) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire conn: %w", err)
	}
	defer conn.Release()

	pgConn := conn.Conn()
	if _, err := pgConn.Exec(ctx, "LISTEN new_events"); err != nil {
		return fmt.Errorf("listen new_events: %w", err)
	}

	for {
		notification, err := pgConn.WaitForNotification(ctx)
		if err != nil {
			return fmt.Errorf("wait for notification: %w", err)
		}

		var payload struct {
			ID int64 `json:"id"`
		}
		if err := json.Unmarshal([]byte(notification.Payload), &payload); err != nil {
			continue
		}

		event, err := s.GetEventByID(ctx, payload.ID)
		if err != nil {
			continue
		}

		if err := onEvent(event); err != nil {
			return err
		}
	}
}
