package store

import (
	"encoding/json"
	"fmt"

	"github.com/main/ingest/internal/crypto"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool      *pgxpool.Pool
	encryptor *crypto.Encryptor
}

func New(pool *pgxpool.Pool, encryptor *crypto.Encryptor) *Store {
	return &Store{pool: pool, encryptor: encryptor}
}

func (s *Store) sealPayload(plaintext json.RawMessage) (json.RawMessage, error) {
	if s.encryptor == nil {
		return plaintext, nil
	}
	sealed, err := s.encryptor.Encrypt(plaintext)
	if err != nil {
		return nil, fmt.Errorf("encrypt payload: %w", err)
	}
	return sealed, nil
}
