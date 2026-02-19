# Airbnb Scraper (JSON + PostgreSQL)

This scraper now stores listings in both:

- `all_listings.json`
- PostgreSQL table `listings`

## Start PostgreSQL with Docker

```bash
docker compose up -d postgres
```

The DB is exposed on `localhost:5432` with default credentials:

- user: `airbnb`
- password: `airbnb`
- database: `airbnb_scraper`

## Configure DB connection (optional)

Copy env template and override if needed:

```bash
cp .env.example .env
```

Supported environment variables:

- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `DB_SSLMODE`

## Run scraper

```bash
go run .
```

## Check stored data

```bash
docker exec -it airbnb-scraper-postgres psql -U airbnb -d airbnb_scraper -c "SELECT city, title, price, url FROM listings ORDER BY id DESC LIMIT 10;"
```
