package cli

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/mariomunozv/forge/jobs"
	"github.com/spf13/cobra"
)

var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Background job commands",
}

var jobsWorkCmd = &cobra.Command{
	Use:   "work",
	Short: "Start background job workers",
	RunE:  runJobsWork,
}

var workersFlag int

func init() {
	jobsWorkCmd.Flags().IntVarP(&workersFlag, "workers", "w", 5, "number of concurrent workers")
	jobsCmd.AddCommand(jobsWorkCmd)
	rootCmd.AddCommand(jobsCmd)
}

func runJobsWork(_ *cobra.Command, _ []string) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer conn.Close()

	if err := conn.Ping(); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Printf("jobs: worker started (workers=%d)", workersFlag)
	return jobs.Work(ctx, conn, workersFlag)
}
