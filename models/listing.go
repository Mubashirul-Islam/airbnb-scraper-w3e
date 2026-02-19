package models

// Listing holds all scraped data for a single Airbnb property.
type Listing struct {
	Title       string  `json:"title"`
	Price       float32 `json:"price"`
	Location    string  `json:"location"`
	Rating      float32 `json:"rating"`
	URL         string  `json:"url"`
	Description string  `json:"description"`
}

// CityResult is sent back from each worker goroutine.
type CityResult struct {
	City     string
	Index    int // original position in cities slice — preserves output order
	Listings []Listing
	Err      error
}

// DetailClickResult captures the JS evaluation result when clicking a listing card.
type DetailClickResult struct {
	OK   bool   `json:"ok"`
	Href string `json:"href"`
}
