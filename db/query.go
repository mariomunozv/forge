package db

import (
	"database/sql"
	"errors"
	"reflect"
	"strings"
)

// ErrNotFound is returned by QueryOne when no rows match the query.
var ErrNotFound = errors.New("db: record not found")

// Queryable is satisfied by *sql.DB and *sql.Tx, so helpers work in transactions.
type Queryable interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

// QueryOne runs query and scans the first row into T using db: struct tags.
// Returns ErrNotFound if no rows match.
//
//	user, err := db.QueryOne[models.User](conn, "SELECT * FROM users WHERE id = $1", id)
func QueryOne[T any](q Queryable, query string, args ...any) (T, error) {
	var zero T

	rows, err := q.Query(query, args...)
	if err != nil {
		return zero, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return zero, err
		}
		return zero, ErrNotFound
	}

	result, err := scanRow[T](rows)
	if err != nil {
		return zero, err
	}
	return result, nil
}

// QueryAll runs query and scans all rows into []T using db: struct tags.
//
//	users, err := db.QueryAll[models.User](conn, "SELECT * FROM users ORDER BY name")
func QueryAll[T any](q Queryable, query string, args ...any) ([]T, error) {
	rows, err := q.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		row, err := scanRow[T](rows)
		if err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// scanRow scans the current row into a new T by matching column names to db: tags.
func scanRow[T any](rows *sql.Rows) (T, error) {
	var result T

	cols, err := rows.Columns()
	if err != nil {
		return result, err
	}

	rv := reflect.ValueOf(&result).Elem()
	rt := rv.Type()

	// Build map: db tag → field value
	fieldMap := make(map[string]reflect.Value, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}
		tag = strings.SplitN(tag, ",", 2)[0]
		fieldMap[tag] = rv.Field(i)
	}

	// Build scan destinations — discard columns with no matching field.
	dests := make([]any, len(cols))
	for i, col := range cols {
		if fv, ok := fieldMap[col]; ok {
			dests[i] = fv.Addr().Interface()
		} else {
			var discard any
			dests[i] = &discard
		}
	}

	return result, rows.Scan(dests...)
}
