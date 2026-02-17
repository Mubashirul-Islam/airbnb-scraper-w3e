package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// ── Data models ───────────────────────────────────────────────────────────────

// Listing holds all scraped data for a single Airbnb property.
type Listing struct {
	Title       string `json:"title"`
	Price       string `json:"price"`
	Location    string `json:"location"`
	Rating      string `json:"rating"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// cityResult is sent back from each worker goroutine.
type cityResult struct {
	city     string
	index    int // original position in cities slice — preserves output order
	listings []Listing
	err      error
}

// ── Defaults ──────────────────────────────────────────────────────────────────

var defaultCities = []string{
	"New York",
	"Paris",
	"Bangkok",
	"Tokyo",
	"Sydney",
}

// ── Entry point ───────────────────────────────────────────────────────────────

func main() {
	locationsFlag := flag.String("locations", "",
		"Comma-separated city list (default: New York,Paris,Bangkok,Tokyo,Sydney)")
	maxPages := flag.Int("pages", 2,
		"Search-result pages to scrape per city")
	detailScrape := flag.Bool("details", true,
		"Fetch each listing's detail page for extended metadata")
	outFile := flag.String("out", "all_listings.csv",
		"Output CSV filename")
	headless := flag.Bool("headless", true,
		"Run Chrome headless (false = visible window)")
	flag.Parse()

	cities := defaultCities
	if *locationsFlag != "" {
		cities = splitTrim(*locationsFlag, ",")
	}

	log.Printf("╔═══════════════════════════════════════════════════╗")
	log.Printf("║      Airbnb Concurrent Multi-City Scraper         ║")
	log.Printf("╚═══════════════════════════════════════════════════╝")
	log.Printf("Cities   : %s", strings.Join(cities, ", "))
	log.Printf("Workers  : %d (one per city, running concurrently)", len(cities))
	log.Printf("Pages    : %d per city", *maxPages)
	log.Printf("Details  : %v", *detailScrape)
	log.Printf("Output   : %s", *outFile)

	// ── Shared allocator ──────────────────────────────────────────────────────
	// One allocator → one Chrome process.
	// Each city goroutine gets its own tab via chromedp.NewContext(allocCtx).
	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", *headless),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.UserAgent(
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "+
				"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		),
		chromedp.WindowSize(1440, 900),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	defer cancelAlloc()

	// Global timeout covers the full concurrent run.
	rootCtx, cancelRoot := context.WithTimeout(allocCtx, 90*time.Minute)
	defer cancelRoot()

	// ── Launch one goroutine per city ─────────────────────────────────────────
	resultCh := make(chan cityResult, len(cities))
	var wg sync.WaitGroup

	for i, city := range cities {
		wg.Add(1)
		go func(idx int, c string) {
			defer wg.Done()

			// Each goroutine owns its own browser tab (child context of allocCtx).
			tabCtx, cancelTab := chromedp.NewContext(rootCtx,
				chromedp.WithLogf(func(format string, args ...interface{}) {
					log.Printf("[%s] "+format, append([]interface{}{c}, args...)...)
				}),
			)
			defer cancelTab()

			log.Printf("[%s] ▶ starting", c)
			listings, err := scrapeCity(tabCtx, c, *maxPages, *detailScrape)
			if err != nil {
				log.Printf("[%s] ✗ %v", c, err)
			} else {
				log.Printf("[%s] ✓ %d listings collected", c, len(listings))
			}
			resultCh <- cityResult{city: c, index: idx, listings: listings, err: err}
		}(i, city)
	}

	// Close the channel once all goroutines finish.
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// ── Collect results in original city order ────────────────────────────────
	ordered := make([]cityResult, len(cities))
	for r := range resultCh {
		ordered[r.index] = r
	}

	// ── Write single CSV ──────────────────────────────────────────────────────
	total, err := writeCSV(*outFile, ordered)
	if err != nil {
		log.Fatalf("✗ Failed to write CSV: %v", err)
	}

	// ── Summary ───────────────────────────────────────────────────────────────
	log.Printf("═══════════════════════════════════════════════════")
	log.Printf("  DONE — %d total listings → %s", total, *outFile)
	for _, r := range ordered {
		status := fmt.Sprintf("%d listings", len(r.listings))
		if r.err != nil {
			status = "ERROR: " + r.err.Error()
		}
		log.Printf("    %-14s %s", r.city+":", status)
	}
	log.Printf("═══════════════════════════════════════════════════")
}

// ── City-level orchestration ──────────────────────────────────────────────────

// scrapeCity scrapes maxPages of search results for one city then optionally
// fetches each listing's detail page. It uses tabCtx — an isolated browser tab.
func scrapeCity(tabCtx context.Context, city string, maxPages int, fetchDetails bool) ([]Listing, error) {
	var all []Listing

	for page := 1; page <= maxPages; page++ {
		log.Printf("[%s] search page %d/%d", city, page, maxPages)

		got, err := scrapeSearchPage(tabCtx, city, page)
		if err != nil {
			log.Printf("[%s] ⚠ page %d: %v", city, page, err)
			continue
		}
		all = append(all, got...)
		log.Printf("[%s] page %d → %d listings (running total: %d)", city, page, len(got), len(all))

		if page < maxPages {
			time.Sleep(2 * time.Second)
		}
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("no listings found")
	}

	if fetchDetails {
		for i := range all {
			if all[i].URL == "" {
				continue
			}
			log.Printf("[%s] detail %d/%d — %s", city, i+1, len(all), truncate(all[i].Title, 45))
			if err := fillDetailPage(tabCtx, &all[i]); err != nil {
				log.Printf("[%s] ⚠ detail error: %v", city, err)
			}
			time.Sleep(1500 * time.Millisecond)
		}
	}

	return all, nil
}

// ── Search page scraping ──────────────────────────────────────────────────────

func scrapeSearchPage(ctx context.Context, city string, page int) ([]Listing, error) {
	encoded := strings.ReplaceAll(city, " ", "%20")
	var searchURL string
	if page == 1 {
		searchURL = fmt.Sprintf("https://www.airbnb.com/s/%s/homes", encoded)
	} else {
		offset := (page - 1) * 20
		searchURL = fmt.Sprintf(
			"https://www.airbnb.com/s/%s/homes?items_offset=%d&items_per_grid=20",
			encoded, offset,
		)
	}

	if err := chromedp.Run(ctx,
		chromedp.Navigate(searchURL),
		chromedp.WaitVisible(
			`[data-testid="card-container"], [itemprop="itemListElement"], .cy5jw6o`,
			chromedp.ByQuery,
		),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return nil, fmt.Errorf("navigate %s: %w", searchURL, err)
	}

	var cards []map[string]string
	if err := chromedp.Run(ctx, chromedp.Evaluate(`
		Array.from(document.querySelectorAll(
			'[data-testid="card-container"], [itemprop="itemListElement"], div[itemprop="itemListElement"], .cy5jw6o'
		)).map(card => {
			const txt = (el) => (el && (el.innerText || '').trim()) || '';
			const title = txt(card.querySelector('[data-testid="listing-card-title"], .t1jojoys, [id^="title_"]'));
			let price = '';
			const priceNodes = Array.from(card.querySelectorAll('[data-testid="price-availability-row"] span, ._tyxjp1, .pquyp81, span[aria-hidden="true"]._1y74zjx, [data-testid="price"]'));
			for (const n of priceNodes) {
				const v = txt(n).split('\n')[0];
				if (/\$|€|£|¥|₹|฿|A\$/.test(v) || /night|nuit|mois|月/.test(v.toLowerCase())) {
					price = v;
					break;
				}
			}
			const location = txt(card.querySelector('[data-testid="listing-card-subtitle"], .s1cjsi4j, .fb4nyux')).split('\n')[0];
			const ratingEl = card.querySelector('[aria-label*="rating"], [aria-label*="stars"], .r1dxllyb, span.ru0q88m');
			const rating = ((ratingEl && ratingEl.getAttribute('aria-label')) || txt(ratingEl)).trim();
			const a = card.querySelector('a[href*="/rooms/"], a[href*="airbnb.com/rooms/"]');
			let url = '';
			if (a) {
				const h = a.getAttribute('href') || '';
				url = h.startsWith('http') ? h : (h ? 'https://www.airbnb.com' + h : '');
			}
			return { title, price, location, rating, url };
		}).filter(x => x.url || x.title)
	`, &cards)); err != nil {
		return nil, fmt.Errorf("extract cards: %w", err)
	}

	if len(cards) == 0 {
		return nil, fmt.Errorf("zero listings extracted (selectors may need updating)")
	}

	out := make([]Listing, len(cards))
	for i := range out {
		location := strings.TrimSpace(cards[i]["location"])
		if location == "" {
			location = city
		}
		out[i] = Listing{
			Title:    strings.TrimSpace(cards[i]["title"]),
			Price:    strings.TrimSpace(cards[i]["price"]),
			Location: location,
			Rating:   strings.TrimSpace(cards[i]["rating"]),
			URL:      strings.TrimSpace(cards[i]["url"]),
		}
	}
	return out, nil
}

// ── Detail page scraping ──────────────────────────────────────────────────────

// fillDetailPage visits a listing URL and mutates the Listing in place with
// extended metadata.
func fillDetailPage(ctx context.Context, l *Listing) error {
	if err := chromedp.Run(ctx,
		chromedp.Navigate(l.URL),
		chromedp.WaitVisible(`h1, [data-section-id="OVERVIEW_DEFAULT"]`, chromedp.ByQuery),
		chromedp.Sleep(1500*time.Millisecond),
	); err != nil {
		return fmt.Errorf("navigate: %w", err)
	}

	var description string
	chromedp.Run(ctx, chromedp.Evaluate(`
		(document.querySelector('[data-section-id="DESCRIPTION_DEFAULT"] span, .ll4r2nl') || {innerText:''}).innerText.trim()
	`, &description))
	l.Description = description

	return nil
}

// ── CSV writer ────────────────────────────────────────────────────────────────

// writeCSV writes all results from every city into one flat CSV file.
// Rows are grouped by city in the original cities order.
// Returns the total number of data rows written.
func writeCSV(filename string, results []cityResult) (int, error) {
	f, err := os.Create(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	if err := w.Write([]string{
		"title", "price", "location", "rating", "url", "description",
	}); err != nil {
		return 0, err
	}

	total := 0
	for _, r := range results {
		if r.err != nil {
			continue
		}
		for _, l := range r.listings {
			row := []string{
				l.Title,
				l.Price,
				l.Location,
				l.Rating,
				l.URL,
				l.Description,
			}
			if err := w.Write(row); err != nil {
				return total, err
			}
			total++
		}
	}
	return total, w.Error()
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func splitTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
