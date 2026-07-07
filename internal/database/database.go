package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS domains (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    source_id TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    token TEXT NOT NULL UNIQUE,
    username TEXT NOT NULL,
    expires_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS app_migration (
    id INTEGER PRIMARY KEY,
    migration TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS sources (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL,
    parser_type TEXT NOT NULL DEFAULT 'raw',
    enabled INTEGER NOT NULL DEFAULT 1,
    update_interval INTEGER NOT NULL DEFAULT 3600,
    last_update DATETIME,
    last_error TEXT NOT NULL DEFAULT '',
    domain_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS snapshots (
    id TEXT PRIMARY KEY,
    build_time DATETIME NOT NULL,
    total_domains INTEGER NOT NULL DEFAULT 0,
    total_sources INTEGER NOT NULL DEFAULT 0,
    total_fetched INTEGER NOT NULL DEFAULT 0,
    total_parsed INTEGER NOT NULL DEFAULT 0,
    duplicates INTEGER NOT NULL DEFAULT 0,
    errors TEXT NOT NULL DEFAULT '',
    build_time_ms INTEGER NOT NULL DEFAULT 0,
    domains_json TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS update_metadata (
    source_id TEXT PRIMARY KEY,
    etag TEXT NOT NULL DEFAULT '',
    last_modified TEXT NOT NULL DEFAULT '',
    last_fetched DATETIME NOT NULL,
    content_hash TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE
);
`

func Init(path string) (*sql.DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	log.Printf("database.Init: path=%q dir=%q", path, dir)

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA busy_timeout = 30000"); err != nil {
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM app_migration WHERE id = 1").Scan(&count); err != nil {
		if _, err := db.Exec("INSERT INTO app_migration (id, migration) VALUES (1, 'initial')"); err != nil {
			return fmt.Errorf("insert initial migration: %w", err)
		}
	}

	// Migration: add source_id column to domains table
	var colCount int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('domains') WHERE name='source_id'
	`).Scan(&colCount)
	if err == nil && colCount == 0 {
		db.Exec("ALTER TABLE domains ADD COLUMN source_id TEXT")
	}

	return nil
}
