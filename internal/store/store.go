package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSourceNotFound = errors.New("source not found")

type IngestInput struct {
	Source      string
	EventType   string
	ExternalID  *string
	Payload     json.RawMessage
	Metadata    json.RawMessage
	RequestInfo json.RawMessage
	BatchID     *string
}

type IngestResult struct {
	ID        int64  `json:"id"`
	Source    string `json:"source"`
	EventType string `json:"event_type"`
}

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Ingest(ctx context.Context, in IngestInput) (IngestResult, error) {
	if in.Source == "" {
		in.Source = "default"
	}
	if in.EventType == "" {
		in.EventType = "record"
	}
	if len(in.Payload) == 0 {
		return IngestResult{}, fmt.Errorf("payload is required")
	}
	if !json.Valid(in.Payload) {
		return IngestResult{}, fmt.Errorf("payload must be valid JSON")
	}
	var payloadCheck any
	if err := json.Unmarshal(in.Payload, &payloadCheck); err != nil {
		return IngestResult{}, fmt.Errorf("payload must be valid JSON")
	}
	if _, ok := payloadCheck.(map[string]any); !ok {
		return IngestResult{}, fmt.Errorf("payload must be a JSON object")
	}
	if len(in.Metadata) == 0 {
		in.Metadata = json.RawMessage(`{}`)
	} else if !json.Valid(in.Metadata) {
		return IngestResult{}, fmt.Errorf("metadata must be valid JSON")
	}
	if len(in.RequestInfo) == 0 {
		in.RequestInfo = json.RawMessage(`{}`)
	} else if !json.Valid(in.RequestInfo) {
		return IngestResult{}, fmt.Errorf("request_info must be valid JSON")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return IngestResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var sourceID string
	err = tx.QueryRow(ctx, `SELECT id::text FROM sources WHERE name = $1`, in.Source).Scan(&sourceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return IngestResult{}, ErrSourceNotFound
	}
	if err != nil {
		return IngestResult{}, fmt.Errorf("lookup source: %w", err)
	}

	var batchID *string
	if in.BatchID != nil && *in.BatchID != "" {
		var exists string
		err = tx.QueryRow(ctx,
			`SELECT id::text FROM batches WHERE id = $1::uuid AND source_id = $2::uuid`,
			*in.BatchID, sourceID,
		).Scan(&exists)
		if errors.Is(err, pgx.ErrNoRows) {
			return IngestResult{}, fmt.Errorf("batch not found for source")
		}
		if err != nil {
			return IngestResult{}, fmt.Errorf("lookup batch: %w", err)
		}
		batchID = &exists
	}

	var eventID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO events (source_id, batch_id, event_type, external_id, payload, metadata, request_info)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5::jsonb, $6::jsonb, $7::jsonb)
		RETURNING id
	`, sourceID, batchID, in.EventType, in.ExternalID, in.Payload, in.Metadata, in.RequestInfo).Scan(&eventID)
	if err != nil {
		return IngestResult{}, fmt.Errorf("insert event: %w", err)
	}

	if batchID != nil {
		_, err = tx.Exec(ctx, `UPDATE batches SET record_count = record_count + 1 WHERE id = $1::uuid`, *batchID)
		if err != nil {
			return IngestResult{}, fmt.Errorf("update batch count: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return IngestResult{}, fmt.Errorf("commit tx: %w", err)
	}

	return IngestResult{
		ID:        eventID,
		Source:    in.Source,
		EventType: in.EventType,
	}, nil
}

type CreateBatchInput struct {
	Source string
}

type Batch struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Status string `json:"status"`
}

func (s *Store) CreateBatch(ctx context.Context, in CreateBatchInput) (Batch, error) {
	if in.Source == "" {
		in.Source = "default"
	}

	var batch Batch
	err := s.pool.QueryRow(ctx, `
		INSERT INTO batches (source_id)
		SELECT id FROM sources WHERE name = $1
		RETURNING id::text, $1, status
	`, in.Source).Scan(&batch.ID, &batch.Source, &batch.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return Batch{}, ErrSourceNotFound
	}
	if err != nil {
		return Batch{}, fmt.Errorf("create batch: %w", err)
	}

	return batch, nil
}

func (s *Store) CloseBatch(ctx context.Context, batchID string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE batches
		SET status = 'closed', finished_at = now()
		WHERE id = $1::uuid AND status = 'open'
	`, batchID)
	if err != nil {
		return fmt.Errorf("close batch: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("batch not found or already closed")
	}
	return nil
}

func (s *Store) CreateSource(ctx context.Context, name, description string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sources (name, description) VALUES ($1, $2)
		ON CONFLICT (name) DO NOTHING
	`, name, description)
	if err != nil {
		return fmt.Errorf("create source: %w", err)
	}
	return nil
}

func (s *Store) Health(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
