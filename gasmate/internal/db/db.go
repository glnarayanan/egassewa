package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

//go:embed seed.sql
var seedSQL string

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite is single-writer
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	if err := seed(db); err != nil {
		return nil, fmt.Errorf("seed: %w", err)
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(schemaSQL)
	return err
}

func seed(db *sql.DB) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM gas_cylinders").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil // already seeded
	}
	log.Println("seeding initial data...")
	_, err := db.Exec(seedSQL)
	return err
}
