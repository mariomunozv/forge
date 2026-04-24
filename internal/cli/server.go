package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:     "server",
	Aliases: []string{"s"},
	Short:   "Start the development server",
	RunE:    runServer,
}

var serverPort string

func init() {
	serverCmd.Flags().StringVarP(&serverPort, "port", "p", "8080", "Port to listen on")
	rootCmd.AddCommand(serverCmd)
}

func runServer(cmd *cobra.Command, args []string) error {
	fmt.Printf("=> Forge server starting on http://localhost:%s\n", serverPort)
	fmt.Println("=> templ watching for changes...")
	fmt.Println("=> Press Ctrl+C to stop")
	fmt.Println()

	// Run templ generate --watch in the background so .templ files
	// are recompiled automatically as they change.
	templ := exec.Command("templ", "generate", "--watch")
	templ.Stdout = os.Stdout
	templ.Stderr = os.Stderr
	if err := templ.Start(); err != nil {
		fmt.Println("=> warning: templ not found in PATH, skipping template watcher")
		fmt.Println("   install with: go install github.com/a-h/templ/cmd/templ@latest")
	}

	// Run the Go server with air (hot reload) if available, else go run .
	var server *exec.Cmd
	if _, err := exec.LookPath("air"); err == nil {
		fmt.Println("=> air detected — hot reload enabled")
		server = exec.Command("air")
	} else {
		fmt.Println("=> air not found — using go run . (no hot reload)")
		fmt.Println("   install air: go install github.com/air-verse/air@latest")
		server = exec.Command("go", "run", ".")
	}
	server.Env = append(os.Environ(), fmt.Sprintf("PORT=%s", serverPort))
	server.Stdout = os.Stdout
	server.Stderr = os.Stderr
	if err := server.Start(); err != nil {
		return fmt.Errorf("could not start server: %w", err)
	}

	// Wait for Ctrl+C and clean up both processes.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	fmt.Println("\n=> Shutting down...")
	if templ.Process != nil {
		templ.Process.Kill()
	}
	if server.Process != nil {
		server.Process.Kill()
	}
	return nil
}
