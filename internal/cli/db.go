package cli

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/mariomunozv/forge/db"
	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database commands (create, drop, migrate, rollback, status)",
}

var dbCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create the database",
	RunE:  runDBCreate,
}

var dbDropCmd = &cobra.Command{
	Use:   "drop",
	Short: "Drop the database",
	RunE:  runDBDrop,
}

var dbMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run all pending migrations",
	RunE:  runDBMigrate,
}

var dbRollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback the last applied migration",
	RunE:  runDBRollback,
}

var dbStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show applied and pending migrations",
	RunE:  runDBStatus,
}

func init() {
	dbCmd.AddCommand(dbCreateCmd, dbDropCmd, dbMigrateCmd, dbRollbackCmd, dbStatusCmd)
	rootCmd.AddCommand(dbCmd)
}

func runDBCreate(cmd *cobra.Command, args []string) error {
	adminURL, dbName, err := parseAdminURL()
	if err != nil {
		return err
	}
	conn, err := db.OpenURL(adminURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.Exec(`CREATE DATABASE ` + pgQuoteIdent(dbName)); err != nil {
		return fmt.Errorf("create database: %w", err)
	}
	fmt.Printf("=> Created database %q\n", dbName)
	return nil
}

func runDBDrop(cmd *cobra.Command, args []string) error {
	adminURL, dbName, err := parseAdminURL()
	if err != nil {
		return err
	}
	conn, err := db.OpenURL(adminURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Terminate active connections so the DROP doesn't fail.
	_, _ = conn.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1`, dbName)

	if _, err := conn.Exec(`DROP DATABASE IF EXISTS ` + pgQuoteIdent(dbName)); err != nil {
		return fmt.Errorf("drop database: %w", err)
	}
	fmt.Printf("=> Dropped database %q\n", dbName)
	return nil
}

// parseAdminURL returns the admin connection URL (pointing to "postgres" DB)
// and the target database name extracted from DATABASE_URL.
func parseAdminURL() (adminURL, dbName string, err error) {
	raw := os.Getenv("DATABASE_URL")
	if raw == "" {
		return "", "", fmt.Errorf("DATABASE_URL is not set")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", fmt.Errorf("invalid DATABASE_URL: %w", err)
	}
	dbName = strings.TrimPrefix(u.Path, "/")
	u.Path = "/postgres"
	return u.String(), dbName, nil
}

// pgQuoteIdent safely quotes a PostgreSQL identifier.
func pgQuoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func runDBMigrate(cmd *cobra.Command, args []string) error {
	conn, err := db.Open()
	if err != nil {
		return err
	}
	defer conn.Close()

	fmt.Println("=> Running migrations...")
	return db.Migrate(conn, "db/migrations")
}

func runDBRollback(cmd *cobra.Command, args []string) error {
	conn, err := db.Open()
	if err != nil {
		return err
	}
	defer conn.Close()

	fmt.Println("=> Rolling back...")
	return db.Rollback(conn, "db/migrations")
}

func runDBStatus(cmd *cobra.Command, args []string) error {
	conn, err := db.Open()
	if err != nil {
		return err
	}
	defer conn.Close()

	statuses, err := db.Status(conn, "db/migrations")
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "STATUS\tVERSION\tNAME")
	fmt.Fprintln(w, "------\t-------\t----")
	for _, s := range statuses {
		status := "pending"
		if s.Applied {
			status = "applied"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", status, s.Version, s.Name)
	}
	w.Flush()
	return nil
}
