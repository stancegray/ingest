package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port        int
	DatabaseURL string
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

	return Config{
		Port:        port,
		DatabaseURL: dbURL,
	}, nil
}
