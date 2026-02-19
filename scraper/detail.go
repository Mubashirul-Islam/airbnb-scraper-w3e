package scraper

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"

	"airbnb-scraper-w3e/config"
	"airbnb-scraper-w3e/models"
)

// detailJS is the JavaScript snippet that extracts fields from a detail page.
const detailJS = `
(() => {
	const titleEl   = document.querySelector('h1');
	const title     = titleEl ? titleEl.textContent : '';

	const priceEl   = document.querySelector('` + JSPriceSelector + `');
	const price     = priceEl ? parseFloat(priceEl.textContent.replace(/[^0-9.]/g, '')) : 0;

	const locationEl = document.querySelector('h2');
	const location   = locationEl ? locationEl.textContent : '';

	const ratingEl  = document.querySelector('` + JSRatingSelector + `');
	const rating    = ratingEl ? parseFloat(ratingEl.textContent.trim()) : 0;

	const descEl    = document.querySelector('` + JSDescSelector + `');
	const description = descEl ? descEl.textContent : '';

	return { title, price, location, rating, description };
})();
`

// FillDetailPage opens a listing card by index on the current search page,
// scrapes its detail page, populates l, then returns to the search results.
func FillDetailPage(ctx context.Context, l *models.Listing, listingIndex int, cfg config.Config) error {
	detailCtx, cancel := context.WithTimeout(ctx, cfg.DetailTimeout)
	defer cancel()

	var searchURL string
	if err := chromedp.Run(detailCtx, chromedp.Location(&searchURL)); err != nil {
		return fmt.Errorf("read search url: %w", err)
	}

	// Click the nth card and capture the href in one JS round-trip.
	var clickRes models.DetailClickResult
	clickScript := fmt.Sprintf(`
		(() => {
			const nodes = Array.from(document.querySelectorAll(%q));
			const idx   = %d;
			if (nodes.length <= idx) return { ok: false, href: '' };
			const el     = nodes[idx];
			const anchor = el.tagName === 'A'
				? el
				: (el.closest('a[href]') || el.querySelector('a[href]'));
			const href = anchor ? (anchor.href || '') : '';
			el.click();
			return { ok: true, href };
		})();
	`, PropertyCardSelector, listingIndex)

	if err := chromedp.Run(detailCtx, chromedp.Evaluate(clickScript, &clickRes)); err != nil {
		return fmt.Errorf("click property card %d: %w", listingIndex, err)
	}
	if !clickRes.OK {
		return fmt.Errorf("property card not found at index %d", listingIndex)
	}

	// If the click didn't navigate (SPA behaviour), navigate explicitly.
	var currentURL string
	_ = chromedp.Run(detailCtx, chromedp.Location(&currentURL))
	if currentURL == "" || currentURL == searchURL {
		if strings.TrimSpace(clickRes.Href) == "" {
			return fmt.Errorf("click did not navigate and href is empty")
		}
		if err := chromedp.Run(detailCtx, chromedp.Navigate(clickRes.Href)); err != nil {
			return fmt.Errorf("navigate to listing href: %w", err)
		}
	}

	// Wait for the detail page to be ready.
	if err := chromedp.Run(detailCtx,
		chromedp.WaitVisible(DetailReadySelector, chromedp.ByQuery),
		chromedp.WaitVisible(PriceSelector, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return fmt.Errorf("wait for detail page: %w", err)
	}

	// Extract fields.
	var raw map[string]interface{}
	if err := chromedp.Run(detailCtx, chromedp.Evaluate(detailJS, &raw)); err != nil {
		return fmt.Errorf("extract detail fields: %w", err)
	}
	_ = chromedp.Run(detailCtx, chromedp.Location(&l.URL))
	applyDetail(l, raw)

	// Return to the search page.
	if err := chromedp.Run(detailCtx,
		chromedp.Navigate(searchURL),
		chromedp.WaitVisible(CardContainerFallback, chromedp.ByQuery),
		chromedp.Sleep(1200*time.Millisecond),
	); err != nil {
		return fmt.Errorf("return to search page: %w", err)
	}

	return nil
}

// applyDetail maps JS-extracted values into a Listing, handling type assertions safely.
func applyDetail(l *models.Listing, raw map[string]interface{}) {
	if v, ok := raw["title"].(string); ok {
		l.Title = strings.TrimSpace(v)
	}
	if v, ok := raw["price"].(float64); ok {
		l.Price = float32(v)
	}
	if v, ok := raw["location"].(string); ok {
		l.Location = strings.TrimSpace(v)
	}
	if v, ok := raw["rating"].(float64); ok {
		l.Rating = float32(v)
	}
	if v, ok := raw["description"].(string); ok {
		l.Description = strings.TrimSpace(v)
	}
}
