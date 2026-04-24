package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const createMigrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version    TEXT        PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`

// MigrationStatus describes one migration file and whether it has been applied.
type MigrationStatus struct {
	Version   string
	Name      string
	Applied   bool
	AppliedAt time.Time
}

// Migrate runs all pending migrations from dir in version order.
func Migrate(conn *sql.DB, dir string) error {
	if err := ensureMigrationsTable(conn); err != nil {
		return err
	}

	files, err := migrationFiles(dir)
	if err != nil {
		return err
	}

	applied, err := appliedVersions(conn)
	if err != nil {
		return err
	}

	ran := 0
	for _, f := range files {
		version := versionFromFilename(f)
		if applied[version] {
			continue
		}

		up, _, err := parseMigration(filepath.Join(dir, f))
		if err != nil {
			return fmt.Errorf("parse %s: %w", f, err)
		}
		if up == "" {
			continue
		}

		if err := runInTx(conn, up); err != nil {
			return fmt.Errorf("migrate %s: %w", f, err)
		}
		if _, err := conn.Exec(`INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}

		fmt.Printf("  migrate  %s\n", f)
		ran++
	}

	if ran == 0 {
		fmt.Println("  nothing to migrate")
	}
	return nil
}

// Rollback rolls back the most recently applied migration.
func Rollback(conn *sql.DB, dir string) error {
	if err := ensureMigrationsTable(conn); err != nil {
		return err
	}

	var version string
	err := conn.QueryRow(`SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&version)
	if err == sql.ErrNoRows {
		fmt.Println("  nothing to rollback")
		return nil
	}
	if err != nil {
		return err
	}

	files, err := migrationFiles(dir)
	if err != nil {
		return err
	}

	var filename string
	for _, f := range files {
		if versionFromFilename(f) == version {
			filename = f
			break
		}
	}
	if filename == "" {
		return fmt.Errorf("migration file for version %s not found in %s", version, dir)
	}

	_, down, err := parseMigration(filepath.Join(dir, filename))
	if err != nil {
		return fmt.Errorf("parse %s: %w", filename, err)
	}
	if down == "" {
		return fmt.Errorf("migration %s has no down section", filename)
	}

	if err := runInTx(conn, down); err != nil {
		return fmt.Errorf("rollback %s: %w", filename, err)
	}
	if _, err := conn.Exec(`DELETE FROM schema_migrations WHERE version = $1`, version); err != nil {
		return err
	}

	fmt.Printf("  rollback  %s\n", filename)
	return nil
}

// Status returns the applied/pending state of all migrations in dir.
func Status(conn *sql.DB, dir string) ([]MigrationStatus, error) {
	if err := ensureMigrationsTable(conn); err != nil {
		return nil, err
	}

	files, err := migrationFiles(dir)
	if err != nil {
		return nil, err
	}

	rows, err := conn.Query(`SELECT version, applied_at FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	appliedAt := map[string]time.Time{}
	for rows.Next() {
		var v string
		var t time.Time
		if err := rows.Scan(&v, &t); err != nil {
			return nil, err
		}
		appliedAt[v] = t
	}

	statuses := make([]MigrationStatus, 0, len(files))
	for _, f := range files {
		version := versionFromFilename(f)
		name := nameFromFilename(f)
		t, applied := appliedAt[version]
		statuses = append(statuses, MigrationStatus{
			Version:   version,
			Name:      name,
			Applied:   applied,
			AppliedAt: t,
		})
	}
	return statuses, nil
}

// --- helpers ---

func ensureMigrationsTable(conn *sql.DB) error {
	_, err := conn.Exec(createMigrationsTable)
	return err
}

func migrationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("migrations directory not found: %s", dir)
		}
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}

func appliedVersions(conn *sql.DB) (map[string]bool, error) {
	rows, err := conn.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, nil
}

func parseMigration(path string) (up, down string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	content := string(data)

	upMarker := "-- migrate:up"
	downMarker := "-- migrate:down"

	upIdx := strings.Index(content, upMarker)
	downIdx := strings.Index(content, downMarker)

	if upIdx >= 0 {
		start := upIdx + len(upMarker)
		end := len(content)
		if downIdx > upIdx {
			end = downIdx
		}
		up = strings.TrimSpace(content[start:end])
	}

	if downIdx >= 0 {
		start := downIdx + len(downMarker)
		down = strings.TrimSpace(content[start:])
	}

	return up, down, nil
}

func runInTx(conn *sql.DB, sql string) error {
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(sql); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func versionFromFilename(name string) string {
	base := filepath.Base(name)
	parts := strings.SplitN(base, "_", 2)
	return parts[0]
}

func nameFromFilename(name string) string {
	base := filepath.Base(name)
	base = strings.TrimSuffix(base, ".sql")
	parts := strings.SplitN(base, "_", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return base
}
