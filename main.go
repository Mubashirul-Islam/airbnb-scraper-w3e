package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"airbnb-scraper-w3e/config"
	"airbnb-scraper-w3e/services"
	"airbnb-scraper-w3e/storage"
	"airbnb-scraper-w3e/utils"
)

func main() {
	cfg := config.Default()

	log.Printf("╔═══════════════════════════════════════════════════╗")
	log.Printf("║      Airbnb Multi-City Scraper (Concurrent)       ║")
	log.Printf("╚═══════════════════════════════════════════════════╝")
	log.Printf("Cities   : %s", strings.Join(cfg.Cities, ", "))
	log.Printf("Workers  : %d (cities processed concurrently)", cfg.Workers)
	log.Printf("Pages    : %d per city", cfg.MaxPages)
	log.Printf("Output   : %s", cfg.OutFile)
	log.Printf("Postgres : %s:%d/%s", cfg.DBHost, cfg.DBPort, cfg.DBName)

	rootCtx, cancelRoot := context.WithTimeout(context.Background(), cfg.GlobalTimeout)
	defer cancelRoot()

	results := services.RunAll(rootCtx, cfg)

	store, err := storage.NewPostgresStore(cfg)
	if err != nil {
		log.Fatalf("✗ Failed to connect to PostgreSQL: %v", err)
	}
	defer store.Close()

	total, err := utils.WriteJSON(cfg.OutFile, results)
	if err != nil {
		log.Fatalf("✗ Failed to write JSON: %v", err)
	}

	dbCtx, cancelDB := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelDB()
	savedCount, err := store.SaveResults(dbCtx, results)
	if err != nil {
		log.Fatalf("✗ Failed to store listings in PostgreSQL: %v", err)
	}

	log.Printf("═══════════════════════════════════════════════════")
	log.Printf("  DONE — %d total listings → %s", total, cfg.OutFile)
	log.Printf("  DB   — %d listings upserted → listings table", savedCount)
	for _, r := range results {
		status := fmt.Sprintf("%d listings", len(r.Listings))
		if r.Err != nil {
			status = "ERROR: " + r.Err.Error()
		}
		log.Printf("    %-14s %s", r.City+":", status)
	}

	stats := utils.BuildSummaryStats(results)
	log.Printf("  STATS")
	log.Printf("    Total Listings Scraped : %d", stats.TotalListings)
	log.Printf("    Average Price          : %.2f", stats.AveragePrice)
	log.Printf("    Minimum Price          : %.2f", stats.MinimumPrice)
	log.Printf("    Maximum Price          : %.2f", stats.MaximumPrice)
	if stats.TotalListings > 0 {
		log.Printf("    Most Expensive Property: %s | $%.2f",
			stats.MostExpensiveProperty.Title,
			stats.MostExpensiveProperty.Price,
		)
	}

	log.Printf("    Listings per City")
	for _, cityStat := range stats.ListingsPerCity {
		log.Printf("      - %s: %d", cityStat.City, cityStat.Count)
	}

	log.Printf("    Top 5 Highest Rated Properties")
	for i, property := range stats.TopRatedProperties {
		log.Printf("      %d) %.2f★ | %s",
			i+1,
			property.Rating,
			property.Title,
		)
	}
	log.Printf("═══════════════════════════════════════════════════")
}
