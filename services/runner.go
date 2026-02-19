package services

import (
	"context"
	"log"

	"github.com/chromedp/chromedp"

	"airbnb-scraper-w3e/config"
	"airbnb-scraper-w3e/models"
	"airbnb-scraper-w3e/utils"
)

// RunAll processes cities sequentially and returns results in original order.
func RunAll(rootCtx context.Context, cfg config.Config) []models.CityResult {
	ordered := make([]models.CityResult, len(cfg.Cities))

	for i, city := range cfg.Cities {
		allocCtx, cancelAlloc := utils.NewAllocator(rootCtx, cfg)

		tabCtx, cancelTab := chromedp.NewContext(allocCtx,
			chromedp.WithLogf(func(format string, args ...interface{}) {
				log.Printf("[%s] "+format, append([]interface{}{city}, args...)...)
			}),
		)

		log.Printf("[%s] ▶ starting", city)
		listings, err := ScrapeCity(tabCtx, city, cfg)
		if err != nil {
			log.Printf("[%s] ✗ %v", city, err)
		} else {
			log.Printf("[%s] ✓ %d listings collected", city, len(listings))
		}

		cancelTab()
		cancelAlloc()

		ordered[i] = models.CityResult{City: city, Index: i, Listings: listings, Err: err}
	}

	return ordered
}
