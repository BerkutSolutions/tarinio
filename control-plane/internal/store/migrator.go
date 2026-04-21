package store

import (
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func RunMigrations(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("migration db is nil")
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			filename TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	type migration struct {
		version  int64
		filename string
		body     string
	}
	items := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version, err := parseVersion(entry.Name())
		if err != nil {
			return err
		}
		raw, err := migrationFS.ReadFile(filepath.ToSlash(filepath.Join("migrations", entry.Name())))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		items = append(items, migration{
			version:  version,
			filename: entry.Name(),
			body:     string(raw),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].version < items[j].version })
	for _, item := range items {
		var applied bool
		if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, item.version).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", item.filename, err)
		}
		if applied {
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", item.filename, err)
		}
		if _, err := tx.Exec(item.body); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", item.filename, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version, filename) VALUES ($1, $2)`, item.version, item.filename); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("register migration %s: %w", item.filename, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", item.filename, err)
		}
	}
	return nil
}

func parseVersion(filename string) (int64, error) {
	prefix := filename
	if idx := strings.Index(prefix, "_"); idx >= 0 {
		prefix = prefix[:idx]
	}
	version, err := strconv.ParseInt(prefix, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse migration version from %s: %w", filename, err)
	}
	return version, nil
}
