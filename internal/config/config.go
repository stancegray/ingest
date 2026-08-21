package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port          int
	DatabaseURL   string
	PublicKeyPEM  []byte
}

func Load() (Config, error) {
	_ = os.Setenv("TZ", "UTC")

	port := 8080
	if raw := os.Getenv("PORT"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("invalid PORT: %w", err)
		}
		port = parsed
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://ingest:ingest@localhost:5432/ingest?sslmode=disable"
	}

	publicKeyPEM, err := loadPublicKeyPEM()
	if err != nil {
		return Config{}, err
	}

	return Config{
		Port:         port,
		DatabaseURL:  dbURL,
		PublicKeyPEM: publicKeyPEM,
	}, nil
}

func loadPublicKeyPEM() ([]byte, error) {
	if inline := os.Getenv("INGEST_PUBLIC_KEY"); inline != "" {
		return []byte(inline), nil
	}

	path := os.Getenv("INGEST_PUBLIC_KEY_FILE")
	if path == "" {
		path = "keys/public.pem"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && os.Getenv("INGEST_PUBLIC_KEY_FILE") == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("read public key: %w", err)
	}
	return data, nil
}
