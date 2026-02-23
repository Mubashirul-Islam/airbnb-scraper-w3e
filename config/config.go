package config

import (
	"math/rand"
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

		DetailTimeout: 30 * time.Second,
		GlobalTimeout: 10 * time.Minute,

		DBHost:     "localhost",
		DBPort:     5433,
		DBUser:     "airbnb",
		DBPassword: "airbnb",
		DBName:     "airbnb_scraper",
		DBSSLMode:  "disable",
	}
}

// RandomDelay returns a random duration between 3 and 9 seconds.
func RandomDelay() time.Duration {
	return time.Duration(3+rand.Intn(7)) * time.Second
}
