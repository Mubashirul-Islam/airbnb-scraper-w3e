package config

import "time"

// Config holds all runtime configuration for the scraper.
type Config struct {
	Cities               []string
	MaxPages             int
	MaxPropertiesPerPage int
	OutFile              string
	Headless             bool
	UserAgent            string

	// Timing
	DetailDelay   time.Duration
	PageDelay     time.Duration
	DetailTimeout time.Duration
	GlobalTimeout time.Duration
}

// Default returns a Config populated with sensible defaults.
func Default() Config {
	return Config{
		Cities: []string{
			"New York",
			"Paris",
			//"Bangkok",
			//"Tokyo",
			"Sydney",
		},
		MaxPages:             2,
		MaxPropertiesPerPage: 2,
		OutFile:              "all_listings.json",
		Headless:             true,
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
			"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",

		DetailDelay:   1500 * time.Millisecond,
		PageDelay:     2 * time.Second,
		DetailTimeout: 35 * time.Second,
		GlobalTimeout: 90 * time.Minute,
	}
}
