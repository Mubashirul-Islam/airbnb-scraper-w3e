package utils

import (
	"encoding/json"
	"os"

	"airbnb-scraper-w3e/models"
)

// WriteJSON writes all successful city results into a single flat JSON array.
// Returns the total number of listings written.
func WriteJSON(filename string, results []models.CityResult) (int, error) {
	all := make([]models.Listing, 0)
	for _, r := range results {
		if r.Err != nil {
			continue
		}
		all = append(all, r.Listings...)
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
