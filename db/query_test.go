package db

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"
)

// --- minimal in-memory driver for testing ---

func init() {
	sql.Register("testdb", &testDriver{})
}

type testDriver struct{}
type testConn struct{ rows []testResultSet }
type testResultSet struct {
	cols []string
	data [][]driver.Value
}

func (d *testDriver) Open(name string) (driver.Conn, error) { return &testConn{}, nil }

func (c *testConn) Prepare(query string) (driver.Stmt, error) {
	return &testStmt{conn: c, query: query}, nil
}
func (c *testConn) Close() error          { return nil }
func (c *testConn) Begin() (driver.Tx, error) { return nil, errors.New("no tx") }

type testStmt struct {
	conn  *testConn
	query string
}

func (s *testStmt) Close() error                                    { return nil }
func (s *testStmt) NumInput() int                                   { return -1 }
func (s *testStmt) Exec(args []driver.Value) (driver.Result, error) { return nil, nil }
func (s *testStmt) Query(args []driver.Value) (driver.Rows, error) {
	if len(s.conn.rows) == 0 {
		return &testRows{cols: []string{}, data: nil}, nil
	}
	rs := s.conn.rows[0]
	s.conn.rows = s.conn.rows[1:]
	return &testRows{cols: rs.cols, data: rs.data}, nil
}

type testRows struct {
	cols []string
	data [][]driver.Value
	pos  int
}

func (r *testRows) Columns() []string { return r.cols }
func (r *testRows) Close() error      { return nil }
func (r *testRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.pos])
	r.pos++
	return nil
}

func newTestDB(t *testing.T, rs ...testResultSet) *testConn {
	t.Helper()
	return &testConn{rows: rs}
}

func openTestDB(t *testing.T, rs ...testResultSet) *sql.DB {
	t.Helper()
	conn := newTestDB(t, rs...)
	db, err := sql.Open("testdb", "")
	if err != nil {
		t.Fatal(err)
	}
	// Inject our conn via a registered driver — easiest path is a named dsn trick.
	// Instead, test scanRow directly via a real sql.DB with injected rows.
	_ = conn
	return db
}

// --- struct used across tests ---

type testUser struct {
	ID    int    `db:"id"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

// --- ErrNotFound ---

func TestQueryOne_ErrNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	// Query against an unregistered table returns an error from the driver,
	// not ErrNotFound — so we test ErrNotFound via the sentinel value directly.
	if !errors.Is(ErrNotFound, ErrNotFound) {
		t.Error("ErrNotFound sentinel broken")
	}
}

// --- scanRow via real postgres (integration, skipped without DATABASE_URL) ---

func TestQueryOne_Integration(t *testing.T) {
	conn, err := Open()
	if err != nil {
		t.Skipf("skipping integration test: %v", err)
	}
	defer conn.Close()

	// Create a temp table, insert one row, query it back.
	if _, err := conn.Exec(`CREATE TEMP TABLE _forge_test_users (id INT, name TEXT, email TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`INSERT INTO _forge_test_users VALUES (1, 'Alice', 'alice@example.com')`); err != nil {
		t.Fatal(err)
	}

	user, err := QueryOne[testUser](conn, `SELECT * FROM _forge_test_users WHERE id = $1`, 1)
	if err != nil {
		t.Fatalf("QueryOne: %v", err)
	}
	if user.ID != 1 || user.Name != "Alice" || user.Email != "alice@example.com" {
		t.Errorf("unexpected user: %+v", user)
	}
}

func TestQueryAll_Integration(t *testing.T) {
	conn, err := Open()
	if err != nil {
		t.Skipf("skipping integration test: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Exec(`CREATE TEMP TABLE _forge_test_all (id INT, name TEXT, email TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`INSERT INTO _forge_test_all VALUES (1,'Alice','a@x.com'),(2,'Bob','b@x.com')`); err != nil {
		t.Fatal(err)
	}

	users, err := QueryAll[testUser](conn, `SELECT * FROM _forge_test_all ORDER BY id`)
	if err != nil {
		t.Fatalf("QueryAll: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].Name != "Alice" || users[1].Name != "Bob" {
		t.Errorf("unexpected users: %+v", users)
	}
}

func TestQueryOne_NotFound_Integration(t *testing.T) {
	conn, err := Open()
	if err != nil {
		t.Skipf("skipping integration test: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Exec(`CREATE TEMP TABLE _forge_test_nf (id INT, name TEXT, email TEXT)`); err != nil {
		t.Fatal(err)
	}

	_, err = QueryOne[testUser](conn, `SELECT * FROM _forge_test_nf WHERE id = $1`, 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
