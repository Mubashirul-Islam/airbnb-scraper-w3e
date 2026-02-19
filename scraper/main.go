package main

import (
	"context"
	"encoding/json"
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
	Price       float32 `json:"price"`
	Location    string `json:"location"`
	Rating      float32 `json:"rating"`
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

type detailClickResult struct {
	OK   bool   `json:"ok"`
	Href string `json:"href"`
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
	cities := defaultCities
	maxPages := 2
	outFile := "all_listings.json"
	headless := true

	log.Printf("╔═══════════════════════════════════════════════════╗")
	log.Printf("║      Airbnb Concurrent Multi-City Scraper         ║")
	log.Printf("╚═══════════════════════════════════════════════════╝")
	log.Printf("Cities   : %s", strings.Join(cities, ", "))
	log.Printf("Workers  : %d (one per city, running concurrently)", len(cities))
	log.Printf("Pages    : %d per city", maxPages)
	log.Printf("Output   : %s", outFile)

	// ── Shared allocator ──────────────────────────────────────────────────────
	// One allocator → one Chrome process.
	// Each city goroutine gets its own tab via chromedp.NewContext(allocCtx).
	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
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
			listings, err := scrapeCity(tabCtx, c, maxPages)
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

	// ── Write single JSON ─────────────────────────────────────────────────────
	total, err := writeJSON(outFile, ordered)
	if err != nil {
		log.Fatalf("✗ Failed to write JSON: %v", err)
	}

	// ── Summary ───────────────────────────────────────────────────────────────
	log.Printf("═══════════════════════════════════════════════════")
	log.Printf("  DONE — %d total listings → %s", total, outFile)
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

// scrapeCity scrapes maxPages of search results for one city then fetches
// each listing's detail page. It uses tabCtx — an isolated browser tab.
func scrapeCity(tabCtx context.Context, city string, maxPages int) ([]Listing, error) {
	var all []Listing

	for page := 1; page <= maxPages; page++ {
		log.Printf("[%s] search page %d/%d", city, page, maxPages)

		got, err := scrapeSearchPage(tabCtx, city, page)
		if err != nil {
			log.Printf("[%s] ⚠ page %d: %v", city, page, err)
			continue
		}

		var pageDetailed []Listing
		for i := range got {
			log.Printf("[%s] detail %d/%d from search page %d", city, i+1, len(got), page)
			if err := fillDetailPageFromSearch(tabCtx, &got[i], i); err != nil {
				log.Printf("[%s] ⚠ detail error: %v", city, err)
				time.Sleep(1500 * time.Millisecond)
				continue
			}
			if strings.TrimSpace(got[i].URL) != "" || strings.TrimSpace(got[i].Title) != "" {
				pageDetailed = append(pageDetailed, got[i])
			}
			time.Sleep(1500 * time.Millisecond)
		}
		all = append(all, pageDetailed...)
		log.Printf("[%s] page %d → %d detailed listings (running total: %d)", city, page, len(pageDetailed), len(all))
		

		if page < maxPages {
			time.Sleep(2 * time.Second)
		}
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("no listings found")
	}

	return all, nil
}

// ── Search page scraping ──────────────────────────────────────────────────────

func scrapeSearchPage(ctx context.Context, city string, page int) ([]Listing, error) {
	encoded := strings.ReplaceAll(city, " ", "%20")
	propertySelector := `.c965t3n.atm_9s_11p5wf0.atm_dz_1osqo2v.dir.dir-ltr`
	if page == 1 {
		searchURL := fmt.Sprintf("https://www.airbnb.com/s/%s/homes", encoded)
		if err := chromedp.Run(ctx,
			chromedp.Navigate(searchURL),
			chromedp.WaitVisible(
				propertySelector,
				chromedp.ByQuery,
			),
			chromedp.Sleep(2*time.Second),
		); err != nil {
			return nil, fmt.Errorf("navigate %s: %w", searchURL, err)
		}
	} else {
		nextSelector := `.l1ovpqvx.atm_npmupv_14b5rvc_10saat9.atm_4s4swg_18xq13z_10saat9.atm_u9em2p_1r3889l_10saat9.atm_1ezpcqw_1u41vd9_10saat9.atm_fyjbsv_c4n71i_10saat9.atm_1rna0z7_1uk391_10saat9.c1ytbx3a.atm_mk_h2mmj6.atm_9s_1txwivl.atm_h_1h6ojuz.atm_fc_1h6ojuz.atm_bb_idpfg4.atm_26_1j28jx2.atm_3f_glywfm.atm_7l_b0j8a8.atm_gi_idpfg4.atm_l8_idpfg4.atm_uc_oxy5qq.atm_kd_glywfm.atm_gz_opxopj.atm_uc_glywfm__1rrf6b5.atm_26_ppd4by_1rqz0hn_uv4tnr.atm_tr_kv3y6q_csw3t1.atm_26_ppd4by_1ul2smo.atm_3f_glywfm_jo46a5.atm_l8_idpfg4_jo46a5.atm_gi_idpfg4_jo46a5.atm_3f_glywfm_1icshfk.atm_kd_glywfm_19774hq.atm_70_glywfm_1w3cfyq.atm_uc_1wx0j5_9xuho3.atm_70_1j5h5ka_9xuho3.atm_26_ppd4by_9xuho3.atm_uc_glywfm_9xuho3_1rrf6b5.atm_7l_156bl0x_1o5j5ji.atm_9j_13gfvf7_1o5j5ji.atm_26_1j28jx2_154oz7f.atm_92_1yyfdc7_vmtskl.atm_9s_1ulexfb_vmtskl.atm_mk_stnw88_vmtskl.atm_tk_1ssbidh_vmtskl.atm_fq_1ssbidh_vmtskl.atm_tr_pryxvc_vmtskl.atm_vy_1vi7ecw_vmtskl.atm_e2_1vi7ecw_vmtskl.atm_5j_1ssbidh_vmtskl.atm_mk_h2mmj6_1ko0jae.dir.dir-ltr`
		if err := chromedp.Run(ctx,
			chromedp.WaitVisible(nextSelector, chromedp.ByQuery),
			chromedp.Click(nextSelector, chromedp.ByQuery),
			chromedp.WaitVisible(
				`[data-testid="card-container"], [itemprop="itemListElement"], .cy5jw6o`,
				chromedp.ByQuery,
			),
			chromedp.Sleep(2*time.Second),
		); err != nil {
			return nil, fmt.Errorf("click next page button: %w", err)
		}
	}

	out := make([]Listing, 2)
	return out, nil
}

// ── Detail page scraping ──────────────────────────────────────────────────────

// fillDetailPageFromSearch opens a listing by clicking its search-card property
// element, scrapes details, then returns to the search page.
func fillDetailPageFromSearch(ctx context.Context, l *Listing, listingIndex int) error {
	detailCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()

	propertySelector := `.c965t3n.atm_9s_11p5wf0.atm_dz_1osqo2v.dir.dir-ltr`

	var searchURL string
	if err := chromedp.Run(detailCtx, chromedp.Location(&searchURL)); err != nil {
		return fmt.Errorf("read search url: %w", err)
	}

	var clickRes detailClickResult
	if err := chromedp.Run(detailCtx, chromedp.Evaluate(fmt.Sprintf(`
		(() => {
			const nodes = Array.from(document.querySelectorAll(%q));
			const idx = %d;
			if (nodes.length <= idx) return { ok: false, href: '' };
			const el = nodes[idx];
			const anchor = el.tagName === 'A' ? el : (el.closest('a[href]') || el.querySelector('a[href]'));
			const href = anchor ? (anchor.href || '') : '';
			el.click();
			return { ok: true, href };
		})();
	`, propertySelector, listingIndex), &clickRes)); err != nil {
		return fmt.Errorf("click property on search page: %w", err)
	}

	if !clickRes.OK {
		return fmt.Errorf("property selector not found for listing index %d", listingIndex)
	}

	var currentURL string
	_ = chromedp.Run(detailCtx, chromedp.Location(&currentURL))
	if currentURL == "" || currentURL == searchURL {
		if strings.TrimSpace(clickRes.Href) == "" {
			return fmt.Errorf("click did not navigate and href is empty")
		}
		if err := chromedp.Run(detailCtx, chromedp.Navigate(clickRes.Href)); err != nil {
			return fmt.Errorf("navigate to clicked listing href: %w", err)
		}
	}

	if err := chromedp.Run(detailCtx,
		chromedp.WaitVisible(`h1, [data-section-id="OVERVIEW_DEFAULT"]`, chromedp.ByQuery),
		 chromedp.WaitVisible(`span.u1opajno, span.u174bpcy`, chromedp.ByQuery),
        chromedp.Sleep(2*time.Second),
	); err != nil {
		return fmt.Errorf("wait detail page: %w", err)
	}

	var detail map[string]interface{}
    if err := chromedp.Run(detailCtx, chromedp.Evaluate(`
        (() => {
            const titleEl = document.querySelector('h1');
            const title = titleEl ? titleEl.textContent : '';

            const priceEl = document.querySelector('span.u1opajno, span.u174bpcy');
            const price = priceEl ? parseFloat(priceEl.textContent.replace(/[^0-9.]/g, '')) : 0;

            const locationEl = document.querySelector('h2');
            const location = locationEl ? locationEl.textContent : '';

            const ratingEl = document.querySelector('div[data-testid="pdp-reviews-highlight-banner-host-rating"] div[aria-hidden="true"], .r1lcxetl.atm_c8_o7aogt.atm_c8_l52nlx__oggzyc');
            const rating = ratingEl ? parseFloat(ratingEl.textContent.trim()) : 0;

            const descEl = document.querySelector('span .l1h825yc.atm_kd_adww2_24z95b');
            const description = descEl ? descEl.textContent : '';

            return { title, price, location, rating, description };
        })();
    `, &detail)); err != nil {
        return fmt.Errorf("extract detail page fields: %w", err)
    }

    _ = chromedp.Run(detailCtx, chromedp.Location(&l.URL))

    if v, ok := detail["title"].(string); ok {
        l.Title = strings.TrimSpace(v)
    }
    if v, ok := detail["price"].(float64); ok {
        l.Price = float32(v)
    }
    if v, ok := detail["location"].(string); ok {
        l.Location = strings.TrimSpace(v)
    }
    if v, ok := detail["rating"].(float64); ok {
        l.Rating = float32(v)
    }
    if v, ok := detail["description"].(string); ok {
        l.Description = strings.TrimSpace(v)
    }

	if err := chromedp.Run(detailCtx,
		chromedp.Navigate(searchURL),
		chromedp.WaitVisible(
			`[data-testid="card-container"], [itemprop="itemListElement"], .cy5jw6o`,
			chromedp.ByQuery,
		),
		chromedp.Sleep(1200*time.Millisecond),
	); err != nil {
		return fmt.Errorf("return to search page: %w", err)
	}

	return nil
}

// ── JSON writer ───────────────────────────────────────────────────────────────

// writeJSON writes all results from every city into one flat JSON array.
// Returns the total number of listings written.
func writeJSON(filename string, results []cityResult) (int, error) {
	all := make([]Listing, 0)
	for _, r := range results {
		if r.err != nil {
			continue
		}
		all = append(all, r.listings...)
	}

	f, err := os.Create(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(all); err != nil {
		return 0, err
	}

	return len(all), nil
}

