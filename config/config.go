package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the scraper.
type Config struct {
	Cities               []string
	MaxPages             int
	MaxPropertiesPerPage int
	OutFile              string
	Headless             any
	UserAgent            string

	// Timing
	DetailDelay   time.Duration
	PageDelay     time.Duration
	DetailTimeout time.Duration
	GlobalTimeout time.Duration

	// PostgreSQL
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
}

// Default returns a Config populated with sensible defaults.
func Default() Config {
	return Config{
		Cities: []string{
			"New York",
			"Paris",
			"Bangkok",
			"Tokyo",
			"Sydney",
		},
		MaxPages:             2,
		MaxPropertiesPerPage: 3,
		OutFile:              "all_listings.json",
		Headless:             false,
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
			"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",

		DetailDelay:   1500 * time.Millisecond,
		PageDelay:     2 * time.Second,
		DetailTimeout: 35 * time.Second,
		GlobalTimeout: 90 * time.Minute,

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 5433),
		DBUser:     getEnv("DB_USER", "airbnb"),
		DBPassword: getEnv("DB_PASSWORD", "airbnb"),
		DBName:     getEnv("DB_NAME", "airbnb_scraper"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

func getEnv(key string, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}
