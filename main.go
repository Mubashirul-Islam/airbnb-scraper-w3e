package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"airbnb-scraper-w3e/config"
	"airbnb-scraper-w3e/services"
	"airbnb-scraper-w3e/utils"
)

func main() {
	cfg := config.Default()

	log.Printf("╔═══════════════════════════════════════════════════╗")
	log.Printf("║      Airbnb Concurrent Multi-City Scraper         ║")
	log.Printf("╚═══════════════════════════════════════════════════╝")
	log.Printf("Cities   : %s", strings.Join(cfg.Cities, ", "))
	log.Printf("Workers  : %d (one per city, running concurrently)", len(cfg.Cities))
	log.Printf("Pages    : %d per city", cfg.MaxPages)
	log.Printf("Output   : %s", cfg.OutFile)

	// One allocator → one Chrome process shared across all city tabs.
	allocCtx, cancelAlloc := utils.NewAllocator(context.Background(), cfg)
	defer cancelAlloc()

	rootCtx, cancelRoot := context.WithTimeout(allocCtx, cfg.GlobalTimeout)
	defer cancelRoot()

	results := services.RunAll(rootCtx, cfg)

	total, err := utils.WriteJSON(cfg.OutFile, results)
	if err != nil {
		log.Fatalf("✗ Failed to write JSON: %v", err)
	}

	log.Printf("═══════════════════════════════════════════════════")
	log.Printf("  DONE — %d total listings → %s", total, cfg.OutFile)
	for _, r := range results {
		status := fmt.Sprintf("%d listings", len(r.Listings))
		if r.Err != nil {
			status = "ERROR: " + r.Err.Error()
		}
		log.Printf("    %-14s %s", r.City+":", status)
	}
	log.Printf("═══════════════════════════════════════════════════")
}
