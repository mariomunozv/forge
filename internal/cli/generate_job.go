package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var generateJobCmd = &cobra.Command{
	Use:   "job [name]",
	Short: "Generate a background job",
	Args:  cobra.ExactArgs(1),
	RunE:  runGenerateJob,
}

func init() {
	generateCmd.AddCommand(generateJobCmd)
}

func runGenerateJob(_ *cobra.Command, args []string) error {
	name := args[0]
	modPath := readModulePath()

	data := struct {
		Name       string // pascal: "WelcomeEmail"
		SnakeName  string // snake: "welcome_email"
		ModulePath string
	}{
		Name:       pascal(name),
		SnakeName:  snake(name),
		ModulePath: modPath,
	}

	jobPath := fmt.Sprintf("app/jobs/%s_job.go", data.SnakeName)
	if err := writeGeneratedFile(jobPath, jobFileTmpl, data); err != nil {
		return err
	}

	workerPath := "cmd/worker/main.go"
	if err := ensureFile(workerPath, workerMainTmpl, data); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("Generated job: %s\n", jobPath)
	fmt.Println("Usage:")
	fmt.Printf("  jobs.Enqueue(db, \"%s\", payload)\n", data.SnakeName)
	fmt.Println("  forge jobs work")
	return nil
}

var jobFileTmpl = `package jobs

import (
	"context"

	"github.com/mariomunozv/forge/jobs"
)

func init() {
	jobs.Register("{{.SnakeName}}", func() jobs.Job { return &{{.Name}}Job{} })
}

type {{.Name}}Job struct {
	// Add payload fields here — they are populated from JSON when the job runs.
}

func (j *{{.Name}}Job) Perform(ctx context.Context) error {
	// TODO: implement
	return nil
}
`

var workerMainTmpl = `package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/mariomunozv/forge/jobs"
	_ "{{.ModulePath}}/app/jobs" // register all jobs
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer conn.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := jobs.Work(ctx, conn, 5); err != nil {
		log.Fatalf("jobs: %v", err)
	}
}
`
