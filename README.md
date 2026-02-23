# Airbnb Scraper

A concurrent, multi-city Airbnb scraper written in Go. It uses a headless Chromium browser (via `chromedp`) to scrape listing data and persists results to both a local JSON file and a PostgreSQL database.

---

## Features

- Scrapes multiple cities concurrently via a configurable worker pool
- Extracts title, price, location, rating, URL, and description for each listing
- Upserts results into PostgreSQL (no duplicates on re-run)
- Writes all results to `all_listings.json`
- Prints a summary with stats: total listings, average/min/max price, top-rated properties, and per-city counts

---

## Requirements

- [Go 1.21+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/) and Docker Compose (for PostgreSQL)
- Google Chrome or Chromium installed on the host (used by `chromedp`)

---

## Cloning the Repository

```bash
git clone https://github.com/Mubashirul-Islam/airbnb-scraper-w3e.git
cd airbnb-scraper-w3e
```

---

## Quick Start

After cloning, you can run the project with a single command:

```bash
chmod +x run.sh
./run.sh
```

This script runs the following steps sequentially:

1. `go mod download` — installs Go dependencies
2. `docker compose up -d postgres` — starts the PostgreSQL container
3. `go run .` — runs the scraper

---

## Setup

### 1. Install Go dependencies

```bash
go mod download
```

### 2. Start PostgreSQL with Docker

```bash
docker compose up -d postgres
```

The container is named `airbnb-scraper-postgres` and exposes PostgreSQL on **port 5433** (mapped from container port 5432).

Default credentials:

| Setting  | Value            |
| -------- | ---------------- |
| Host     | `localhost`      |
| Port     | `5433`           |
| User     | `airbnb`         |
| Password | `airbnb`         |
| Database | `airbnb_scraper` |

The `listings` table is created automatically on first start via the init script at `docker/postgres-init/001_create_listings.sql`.

### 3. Configure environment variables (optional)

All settings have sensible defaults and can be overridden with environment variables:

| Variable      | Default          | Description                          |
| ------------- | ---------------- | ------------------------------------ |
| `DB_HOST`     | `localhost`      | PostgreSQL host                      |
| `DB_PORT`     | `5433`           | PostgreSQL port                      |
| `DB_USER`     | `airbnb`         | PostgreSQL user                      |
| `DB_PASSWORD` | `airbnb`         | PostgreSQL password                  |
| `DB_NAME`     | `airbnb_scraper` | PostgreSQL database name             |
| `DB_SSLMODE`  | `disable`        | PostgreSQL SSL mode                  |
| `WORKERS`     | `3`              | Number of cities scraped in parallel |

---

## Running the Scraper

```bash
go run .
```

The scraper will process the configured cities (default: New York, Paris, Bangkok, Tokyo, Sydney), scrape up to 2 pages and 3 properties per page for each city, and then write results to `all_listings.json` and upsert them into the `listings` table.

---

## Check stored data

```bash
docker exec -it airbnb-scraper-postgres psql -U airbnb -d airbnb_scraper -c "SELECT * FROM listings;"
```

---

## Insight Report

```
═══════════════════════════════════════════════════
DONE — 30 total listings → all_listings.json
DB   — 30 listings upserted → listings table
  New York:      6 listings
  Paris:         6 listings
  Bangkok:       6 listings
  Tokyo:         6 listings
  Sydney:        6 listings
STATS
  Total Listings Scraped : 30
  Average Price          : 335.97
  Minimum Price          : 172.00
  Maximum Price          : 524.00
  Most Expensive Property: RESORT STYLE BY THE BAY | $524.00
  Listings per City
    - Bangkok: 6
    - New York: 6
    - Paris: 6
    - Sydney: 6
    - Tokyo: 6
  Top 5 Highest Rated Properties
    1) 5.00★ | Parisian nest in the heart of the 11th arrondissement of Paris
    2) 5.00★ | Pigcat Room near Em District & BTS Prom Phong
    3) 5.00★ | Light-Filled Queen Room with shared bathroom
    4) 5.00★ | Cozy room near BTS- Iconsiam B505
    5) 4.98★ | Home away from Home Apartment in Thonglor
 ══════════════════════════════════════════════════
```

---

## Project Structure

```
airbnb-scraper-w3e/
├── main.go                          # Entry point: orchestrates scraping, storage, and stats output
├── go.mod                           # Go module definition and dependencies
├── all_listings.json                # Scrape output (auto-generated)
│
├── config/
│   └── config.go                    # Runtime config with defaults and env var overrides
│
├── models/
│   └── listing.go                   # Data models: Listing, CityResult, DetailClickResult
│
├── scraper/
│   ├── search.go                    # Searches Airbnb for a city and collects listing URLs
│   ├── detail.go                    # Visits each listing URL and extracts full details
│   └── selectors.go                 # CSS/JS selectors used during scraping
│
├── services/
│   ├── runner.go                    # Concurrent worker pool — dispatches cities to goroutines
│   └── city_scraper.go              # Coordinates search + detail scraping for one city
│
├── storage/
│   └── postgres.go                  # PostgreSQL connection and upsert logic (pgx/v5)
│
├── utils/
│   ├── browser.go                   # chromedp allocator setup (headless, user-agent, etc.)
│   ├── json.go                      # Writes results to JSON file
│   └── stats.go                     # Computes summary statistics from scraped results
│
└── docker/
    └── postgres-init/
        └── 001_create_listings.sql  # Auto-run SQL: creates listings table and indexes
```
