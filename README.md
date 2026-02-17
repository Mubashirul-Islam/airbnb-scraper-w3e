# Airbnb Concurrent Multi-City Scraper (chromedp)

Scrapes **New York, Paris, Bangkok, Tokyo, and Sydney** in **parallel** — one goroutine
per city, each driving its own isolated browser tab — and merges everything into a
**single `all_listings.csv`**.

---

## How concurrency works

```
main()
 │
 ├── NewExecAllocator()   ← one shared Chrome process
 │
 ├── goroutine: New York  ── own tab (NewContext) ──► scrapeCity() ──► resultCh
 ├── goroutine: Paris     ── own tab (NewContext) ──► scrapeCity() ──► resultCh
 ├── goroutine: Bangkok   ── own tab (NewContext) ──► scrapeCity() ──► resultCh
 ├── goroutine: Tokyo     ── own tab (NewContext) ──► scrapeCity() ──► resultCh
 └── goroutine: Sydney    ── own tab (NewContext) ──► scrapeCity() ──► resultCh
          │
          └── collect via channel, sort by original city order
                    │
                    └── writeCSV()  →  all_listings.csv
```

All 5 cities run simultaneously. Total wall-clock time ≈ slowest single city,
not the sum of all cities.

Each goroutine:

1. Opens 2 search-result pages for its city
2. Extracts title / price / location / rating / URL from each card
3. Visits every listing's detail page and extracts a `description` field

---

## Output — single CSV

`all_listings.csv` columns:

| Column        | Source                                        |
| ------------- | --------------------------------------------- |
| `title`       | property name                                 |
| `price`       | per-night rate                                |
| `location`    | card subtitle                                 |
| `rating`      | star rating (if available)                    |
| `url`         | direct link to listing                        |
| `description` | listing description text from the detail page |

Rows are grouped city by city in the original order:
New York → Paris → Bangkok → Tokyo → Sydney.

---

## Prerequisites

1. **Go 1.21+** — https://go.dev/dl/
2. **Google Chrome** or Chromium installed:
   - macOS: `brew install --cask google-chrome`
   - Ubuntu/Debian: `sudo apt install chromium-browser`
   - Windows: https://google.com/chrome

---

## Setup & run

```bash
go mod tidy                        # fetch chromedp + dependencies
go build -o airbnb-scraper ./cmd/scraper

# Default: 5 cities, 2 pages each, details on → all_listings.csv
./airbnb-scraper

# Custom output filename
./airbnb-scraper -out my_results.csv

# Show the browser windows (useful for debugging)
./airbnb-scraper -headless=false

# Skip detail pages — much faster, search-card data only
./airbnb-scraper -details=false

# Custom cities
./airbnb-scraper -locations "London,Dubai,Singapore" -pages 2
```

### All flags

| Flag         | Default            | Description               |
| ------------ | ------------------ | ------------------------- |
| `-locations` | (5 defaults)       | Comma-separated city list |
| `-pages`     | `2`                | Search pages per city     |
| `-details`   | `true`             | Fetch detail pages        |
| `-out`       | `all_listings.csv` | Output filename           |
| `-headless`  | `true`             | Headless Chrome           |

---

## Project structure

```
airbnb-scraper/
├── cmd/scraper/
│   └── main.go      ← all logic: concurrency, scraping, CSV writer
├── go.mod
├── go.sum
└── README.md
```

---

## Disclaimer

Airbnb's [Terms of Service](https://www.airbnb.com/terms) prohibit automated scraping.
Use for personal, educational, or research purposes only.
