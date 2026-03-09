package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port             string
	DatabaseURL      string
	JWTSecret        string
	JWTAccessExpiry  time.Duration
	JWTRefreshExpiry time.Duration
	// Admin — if not set, listing approval and deposit verification are skipped (auto-approved, for local dev)
	AdminTelegramChatID int64
	AdminTelebirr       string
}

func Load() *Config {
	cfg := &Config{
		Port:             getEnv("PORT", "8080"),
		DatabaseURL:      mustEnv("DATABASE_URL"),
		JWTSecret:        mustEnv("JWT_SECRET"),
		JWTAccessExpiry:  15 * time.Minute,
		JWTRefreshExpiry: 7 * 24 * time.Hour,
		AdminTelebirr:    os.Getenv("ADMIN_TELEBIRR"),
	}
	if v := os.Getenv("ADMIN_TELEGRAM_CHAT_ID"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.AdminTelegramChatID = id
		}
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required environment variable not set: " + key)
	}
	return v
}
