package config

import (
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	Port          string
	SessionTTL    time.Duration
	CookieSecure  bool
	AdminName     string
	AdminEmail    string
	AdminPassword string
}

func Load() (*Config, error) {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("Error loading .env file")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, errors.New("DATABASE_URL is not set on .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ttl := 12 * time.Hour
	if s := os.Getenv("SESSION_TTL"); s != "" {
		d, err := time.ParseDuration(s)
		if err != nil || d <= 0 {
			return nil, errors.New("SESSION_TTL must be a duration like 12h or 30m")
		}
		ttl = d
	}

	secure, _ := strconv.ParseBool(os.Getenv("COOKIE_SECURE"))

	adminName := os.Getenv("ADMIN_NAME")
	if adminName == "" {
		adminName = "Administrador"
	}

	return &Config{
		DatabaseURL:   dbURL,
		Port:          port,
		SessionTTL:    ttl,
		CookieSecure:  secure,
		AdminName:     adminName,
		AdminEmail:    os.Getenv("ADMIN_EMAIL"),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
	}, nil
}
