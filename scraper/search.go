package scraper

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"

	"airbnb-scraper-w3e/models"
)

// SearchPage navigates to (or advances to) the given search-results page for
// a city and returns a slice of stub Listings ready for detail enrichment.
func SearchPage(ctx context.Context, city string, page int, pageDelay time.Duration, maxPropertiesPerPage int) ([]models.Listing, error) {
	encoded := strings.ReplaceAll(city, " ", "%20")

	if page == 1 {
		searchURL := fmt.Sprintf("https://www.airbnb.com/s/%s/homes", encoded)
		if err := chromedp.Run(ctx,
			chromedp.Navigate(searchURL),
			chromedp.WaitVisible(PropertyCardSelector, chromedp.ByQuery),
			chromedp.Sleep(pageDelay),
		); err != nil {
			return nil, fmt.Errorf("navigate %s: %w", searchURL, err)
		}
	} else {
		if err := chromedp.Run(ctx,
			chromedp.WaitVisible(NextPageSelector, chromedp.ByQuery),
			chromedp.Click(NextPageSelector, chromedp.ByQuery),
			chromedp.WaitVisible(CardContainerFallback, chromedp.ByQuery),
			chromedp.Sleep(pageDelay),
		); err != nil {
			return nil, fmt.Errorf("advance to page %d: %w", page, err)
		}
	}

	// Count cards visible on page so callers know how many details to fetch.
	var cardCount int
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(
			fmt.Sprintf(`document.querySelectorAll(%q).length`, PropertyCardSelector),
			&cardCount,
		),
	); err != nil || cardCount == 0 {
		// Fallback: return 2 stubs (preserves original behaviour).
		return make([]models.Listing, 2), nil
	}

	if maxPropertiesPerPage > 0 && cardCount > maxPropertiesPerPage {
		cardCount = maxPropertiesPerPage
	}

	return make([]models.Listing, cardCount), nil
}
