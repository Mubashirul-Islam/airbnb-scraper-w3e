package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"airbnb-scraper-w3e/config"
	"airbnb-scraper-w3e/models"
	"airbnb-scraper-w3e/scraper"
)

// ScrapeCity fetches up to cfg.MaxPages of search results for one city,
// then enriches each listing with its detail page.
// It uses tabCtx — an isolated browser tab context.
func ScrapeCity(tabCtx context.Context, city string, cfg config.Config) ([]models.Listing, error) {
	var all []models.Listing

	for page := 1; page <= cfg.MaxPages; page++ {
		log.Printf("[%s] search page %d/%d", city, page, cfg.MaxPages)

		stubs, err := scraper.SearchPage(tabCtx, city, page, config.RandomDelay(), cfg.MaxPropertiesPerPage)
		if err != nil {
			log.Printf("[%s] ⚠ page %d: %v", city, page, err)
			continue
		}

		var pageListings []models.Listing
		for i := range stubs {
			log.Printf("[%s] detail %d/%d (search page %d)", city, i+1, len(stubs), page)

			if err := scraper.FillDetailPage(tabCtx, &stubs[i], i, cfg); err != nil {
				log.Printf("[%s] ⚠ detail error: %v", city, err)
				time.Sleep(config.RandomDelay())
				continue
			}

			if strings.TrimSpace(stubs[i].URL) != "" || strings.TrimSpace(stubs[i].Title) != "" {
				pageListings = append(pageListings, stubs[i])
			}
			time.Sleep(config.RandomDelay())
		}

		all = append(all, pageListings...)
		log.Printf("[%s] page %d → %d listings (running total: %d)",
			city, page, len(pageListings), len(all))

		if page < cfg.MaxPages {
			time.Sleep(config.RandomDelay())
		}
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("no listings found")
	}

	return all, nil
}
