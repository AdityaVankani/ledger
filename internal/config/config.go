package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Port          string
	DatabaseURL   string
	SessionTTL    time.Duration
	AllowedOrigin string
}

func Load() (Config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	sessionTTL := 7 * 24 * time.Hour
	if rawTTL := os.Getenv("SESSION_TTL"); rawTTL != "" {
		var err error
		sessionTTL, err = time.ParseDuration(rawTTL)
		if err != nil || sessionTTL <= 0 {
			return Config{}, fmt.Errorf("SESSION_TTL must be a positive Go duration")
		}
	}

	return Config{Port: port, DatabaseURL: databaseURL, SessionTTL: sessionTTL, AllowedOrigin: os.Getenv("ALLOWED_ORIGIN")}, nil
}
