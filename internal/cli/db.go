package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/mariomunozv/forge/db"
	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database commands (migrate, rollback, status)",
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
	dbCmd.AddCommand(dbMigrateCmd, dbRollbackCmd, dbStatusCmd)
	rootCmd.AddCommand(dbCmd)
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
