package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVersionFromFilename(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"20260424153045_create_users.sql", "20260424153045"},
		{"20260101000000_initial.sql", "20260101000000"},
		{"db/migrations/20260424153045_create_posts.sql", "20260424153045"},
	}

	for _, c := range cases {
		if got := versionFromFilename(c.input); got != c.want {
			t.Errorf("versionFromFilename(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestNameFromFilename(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"20260424153045_create_users.sql", "create_users"},
		{"20260424153045_add_email_to_users.sql", "add_email_to_users"},
	}

	for _, c := range cases {
		if got := nameFromFilename(c.input); got != c.want {
			t.Errorf("nameFromFilename(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestParseMigration(t *testing.T) {
	content := `-- migrate:up
CREATE TABLE users (id SERIAL PRIMARY KEY);

-- migrate:down
DROP TABLE users;
`
	f := filepath.Join(t.TempDir(), "test.sql")
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	up, down, err := parseMigration(f)
	if err != nil {
		t.Fatal(err)
	}

	wantUp := "CREATE TABLE users (id SERIAL PRIMARY KEY);"
	wantDown := "DROP TABLE users;"

	if up != wantUp {
		t.Errorf("up = %q, want %q", up, wantUp)
	}
	if down != wantDown {
		t.Errorf("down = %q, want %q", down, wantDown)
	}
}

func TestParseMigration_UpOnly(t *testing.T) {
	content := "-- migrate:up\nCREATE INDEX idx_email ON users(email);\n"

	f := filepath.Join(t.TempDir(), "test.sql")
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	up, down, err := parseMigration(f)
	if err != nil {
		t.Fatal(err)
	}

	if up == "" {
		t.Error("expected non-empty up section")
	}
	if down != "" {
		t.Errorf("expected empty down section, got %q", down)
	}
}

func TestMigrationFiles_Sorted(t *testing.T) {
	dir := t.TempDir()
	names := []string{
		"20260424153045_create_posts.sql",
		"20260101000000_create_users.sql",
		"20260301120000_add_email.sql",
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := migrationFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
	if files[0] != "20260101000000_create_users.sql" {
		t.Errorf("expected first file to be create_users, got %s", files[0])
	}
	if files[2] != "20260424153045_create_posts.sql" {
		t.Errorf("expected last file to be create_posts, got %s", files[2])
	}
}
