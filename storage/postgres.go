package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"airbnb-scraper-w3e/config"
	"airbnb-scraper-w3e/models"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(cfg config.Config) (*PostgresStore, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBSSLMode,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}

	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	store := &PostgresStore{db: db}
	schemaCtx, schemaCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer schemaCancel()
	if err := store.ensureSchema(schemaCtx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}

func (s *PostgresStore) SaveResults(ctx context.Context, results []models.CityResult) (int, error) {
	if len(results) == 0 {
		return 0, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO listings (city, title, price, location, rating, url, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (url) DO UPDATE
		SET
			city = EXCLUDED.city,
			title = EXCLUDED.title,
			price = EXCLUDED.price,
			location = EXCLUDED.location,
			rating = EXCLUDED.rating,
			description = EXCLUDED.description,
			updated_at = NOW()`)
	if err != nil {
		return 0, fmt.Errorf("prepare insert statement: %w", err)
	}
	defer stmt.Close()

	total := 0
	for _, cityResult := range results {
		if cityResult.Err != nil {
			continue
		}
		for _, listing := range cityResult.Listings {
			if listing.URL == "" {
				continue
			}
			if _, err = stmt.ExecContext(
				ctx,
				cityResult.City,
				listing.Title,
				listing.Price,
				listing.Location,
				listing.Rating,
				listing.URL,
				listing.Description,
			); err != nil {
				return 0, fmt.Errorf("insert listing %q: %w", listing.URL, err)
			}
			total++
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	return total, nil
}

func (s *PostgresStore) ensureSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS listings (
			id BIGSERIAL PRIMARY KEY,
			city TEXT NOT NULL,
			title TEXT NOT NULL,
			price REAL NOT NULL DEFAULT 0,
			location TEXT NOT NULL DEFAULT '',
			rating REAL NOT NULL DEFAULT 0,
			url TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_listings_city ON listings(city);
	`)
	if err != nil {
		return fmt.Errorf("ensure schema: %w", err)
	}
	return nil
}
