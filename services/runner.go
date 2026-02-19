package services

import (
	"context"
	"log"
	"sync"

	"github.com/chromedp/chromedp"

	"airbnb-scraper-w3e/config"
	"airbnb-scraper-w3e/models"
)

// RunAll launches one goroutine per city, collects results in original order,
// and returns the ordered slice of CityResults.
func RunAll(rootCtx context.Context, cfg config.Config) []models.CityResult {
	resultCh := make(chan models.CityResult, len(cfg.Cities))
	var wg sync.WaitGroup

	for i, city := range cfg.Cities {
		wg.Add(1)
		go func(idx int, c string) {
			defer wg.Done()

			tabCtx, cancelTab := chromedp.NewContext(rootCtx,
				chromedp.WithLogf(func(format string, args ...interface{}) {
					log.Printf("[%s] "+format, append([]interface{}{c}, args...)...)
				}),
			)
			defer cancelTab()

			log.Printf("[%s] ▶ starting", c)
			listings, err := ScrapeCity(tabCtx, c, cfg)
			if err != nil {
				log.Printf("[%s] ✗ %v", c, err)
			} else {
				log.Printf("[%s] ✓ %d listings collected", c, len(listings))
			}

			resultCh <- models.CityResult{City: c, Index: idx, Listings: listings, Err: err}
		}(i, city)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	ordered := make([]models.CityResult, len(cfg.Cities))
	for r := range resultCh {
		ordered[r.Index] = r
	}
	return ordered
}
