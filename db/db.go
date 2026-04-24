// Package db provides database connection and migration helpers for Forge apps.
package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// Open opens a PostgreSQL connection from the DATABASE_URL environment variable.
func Open() (*sql.DB, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}
	return connect(url)
}

// OpenURL opens a PostgreSQL connection from the given URL.
func OpenURL(url string) (*sql.DB, error) {
	return connect(url)
}

func connect(url string) (*sql.DB, error) {
	conn, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}
	return conn, nil
}
