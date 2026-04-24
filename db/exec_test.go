package db

import (
	"testing"
)

func TestInsertUpdateDelete_Integration(t *testing.T) {
	conn, err := Open()
	if err != nil {
		t.Skipf("skipping integration test: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Exec(`CREATE TEMP TABLE _forge_exec_test (
		id    SERIAL PRIMARY KEY,
		name  TEXT NOT NULL,
		email TEXT NOT NULL
	)`); err != nil {
		t.Fatal(err)
	}

	// Insert
	id, err := Insert(conn, "_forge_exec_test", map[string]any{
		"name":  "Alice",
		"email": "alice@example.com",
	})
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero id")
	}

	// Verify inserted row
	user, err := QueryOne[struct {
		ID    int    `db:"id"`
		Name  string `db:"name"`
		Email string `db:"email"`
	}](conn, "SELECT * FROM _forge_exec_test WHERE id = $1", id)
	if err != nil {
		t.Fatalf("QueryOne after insert: %v", err)
	}
	if user.Name != "Alice" || user.Email != "alice@example.com" {
		t.Errorf("unexpected row: %+v", user)
	}

	// Update
	if err := Update(conn, "_forge_exec_test", id, map[string]any{"name": "Alice Updated"}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := QueryOne[struct {
		Name string `db:"name"`
	}](conn, "SELECT name FROM _forge_exec_test WHERE id = $1", id)
	if err != nil {
		t.Fatalf("QueryOne after update: %v", err)
	}
	if updated.Name != "Alice Updated" {
		t.Errorf("expected 'Alice Updated', got %q", updated.Name)
	}

	// Delete
	if err := Delete(conn, "_forge_exec_test", id); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = QueryOne[struct{ ID int `db:"id"` }](conn, "SELECT id FROM _forge_exec_test WHERE id = $1", id)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestQuoteIdent(t *testing.T) {
	cases := []struct{ input, want string }{
		{"users", `"users"`},
		{"my table", `"my table"`},
		{`has"quote`, `"has""quote"`},
	}
	for _, c := range cases {
		if got := quoteIdent(c.input); got != c.want {
			t.Errorf("quoteIdent(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
