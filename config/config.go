package config

import (
	"time"
)

// Config holds all runtime configuration for the scraper.
type Config struct {
	Cities               []string
	Workers              int
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
		Workers:              5,
		MaxPages:             2,
		MaxPropertiesPerPage: 3,
		OutFile:              "all_listings.json",
		Headless:             "new",
		UserAgent:            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",

		DetailDelay:   10 * time.Second,
		PageDelay:     10 * time.Second,
		DetailTimeout: 1 * time.Minute,
		GlobalTimeout: 15 * time.Minute,

		DBHost:     "localhost",
		DBPort:     5433,
		DBUser:     "airbnb",
		DBPassword: "airbnb",
		DBName:     "airbnb_scraper",
		DBSSLMode:  "disable",
	}
}
