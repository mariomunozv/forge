package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
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
	warnMissingTools()

	fmt.Printf("=> Forge server starting on http://localhost:%s\n", serverPort)
	fmt.Println("=> Press Ctrl+C to stop")
	fmt.Println()

	// Run templ generate --watch in the background.
	templ := exec.Command("templ", "generate", "--watch")
	templ.Stdout = os.Stdout
	templ.Stderr = os.Stderr
	if err := templ.Start(); err != nil {
		fmt.Println("=> templ watcher skipped (not installed)")
	} else {
		fmt.Println("=> templ watching for changes...")
	}

	// Run with air (hot reload) if available, else go run .
	var server *exec.Cmd
	if _, err := exec.LookPath("air"); err == nil {
		fmt.Println("=> hot reload enabled (air)")
		server = exec.Command("air")
	} else {
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

// warnMissingTools checks for templ and air and suggests forge setup if missing.
func warnMissingTools() {
	missing := []string{}
	if !isInstalled("templ version") {
		missing = append(missing, "templ")
	}
	if !isInstalled("air -v") {
		missing = append(missing, "air")
	}
	if len(missing) > 0 {
		fmt.Printf("=> warning: missing tools: %s\n", strings.Join(missing, ", "))
		fmt.Println("   run `forge setup` to install them")
		fmt.Println()
	}
}
