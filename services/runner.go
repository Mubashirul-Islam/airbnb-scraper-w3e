package services

import (
	"context"
	"log"
	"sync"

	"github.com/chromedp/chromedp"

	"airbnb-scraper-w3e/config"
	"airbnb-scraper-w3e/models"
	"airbnb-scraper-w3e/utils"
)

// RunAll processes cities concurrently and returns results in original order.
func RunAll(rootCtx context.Context, cfg config.Config) []models.CityResult {
	ordered := make([]models.CityResult, len(cfg.Cities))
	if len(cfg.Cities) == 0 {
		return ordered
	}

	workers := cfg.Workers
	if workers <= 0 {
		workers = 1
	}
	if workers > len(cfg.Cities) {
		workers = len(cfg.Cities)
	}

	type cityJob struct {
		index int
		city  string
	}

	jobs := make(chan cityJob)
	results := make(chan models.CityResult, len(cfg.Cities))

	var wg sync.WaitGroup
	for workerID := 0; workerID < workers; workerID++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				allocCtx, cancelAlloc := utils.NewAllocator(rootCtx, cfg)

				tabCtx, cancelTab := chromedp.NewContext(allocCtx,
					chromedp.WithLogf(func(format string, args ...interface{}) {
						log.Printf("[%s] "+format, append([]interface{}{job.city}, args...)...)
					}),
				)

				log.Printf("[%s] ▶ starting", job.city)
				listings, err := ScrapeCity(tabCtx, job.city, cfg)
				if err != nil {
					log.Printf("[%s] ✗ %v", job.city, err)
				} else {
					log.Printf("[%s] ✓ %d listings collected", job.city, len(listings))
				}

				cancelTab()
				cancelAlloc()

				results <- models.CityResult{City: job.city, Index: job.index, Listings: listings, Err: err}
			}
		}()
	}

	go func() {
		for i, city := range cfg.Cities {
			jobs <- cityJob{index: i, city: city}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	for result := range results {
		ordered[result.Index] = result
	}

	return ordered
}
