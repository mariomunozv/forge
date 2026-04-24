package db

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// Execer is satisfied by *sql.DB and *sql.Tx.
type Execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// RowQueryer is satisfied by *sql.DB and *sql.Tx.
type RowQueryer interface {
	QueryRow(query string, args ...any) *sql.Row
}

// Insert inserts a row into table with the given column values and returns the new id.
// Requires the table to have an id SERIAL or BIGSERIAL column.
//
//	id, err := db.Insert(conn, "users", map[string]any{"name": "Alice", "email": "alice@example.com"})
func Insert(q RowQueryer, table string, values map[string]any) (int64, error) {
	cols, args := sortedKeysAndValues(values)

	placeholders := make([]string, len(cols))
	for i := range cols {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING id",
		quoteIdent(table),
		strings.Join(quoteIdents(cols), ", "),
		strings.Join(placeholders, ", "),
	)

	var id int64
	err := q.QueryRow(query, args...).Scan(&id)
	return id, err
}

// Update updates columns in table for the row with the given id.
//
//	err := db.Update(conn, "users", 1, map[string]any{"name": "Bob"})
func Update(q Execer, table string, id any, values map[string]any) error {
	cols, args := sortedKeysAndValues(values)

	sets := make([]string, len(cols))
	for i, col := range cols {
		sets[i] = fmt.Sprintf("%s = $%d", quoteIdent(col), i+1)
	}
	args = append(args, id)

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = $%d",
		quoteIdent(table),
		strings.Join(sets, ", "),
		len(args),
	)

	_, err := q.Exec(query, args...)
	return err
}

// Delete deletes the row with the given id from table.
//
//	err := db.Delete(conn, "users", 1)
func Delete(q Execer, table string, id any) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", quoteIdent(table))
	_, err := q.Exec(query, id)
	return err
}

func sortedKeysAndValues(m map[string]any) ([]string, []any) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	vals := make([]any, len(keys))
	for i, k := range keys {
		vals[i] = m[k]
	}
	return keys, vals
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func quoteIdents(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = quoteIdent(s)
	}
	return out
}
