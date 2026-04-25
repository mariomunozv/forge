package cli

import (
	"bytes"
	"fmt"
	"io"
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
	printServerBanner(serverPort)

	// Run templ generate --watch in the background.
	templ := exec.Command("templ", "generate", "--watch")
	templ.Stdout = os.Stdout
	templ.Stderr = os.Stderr
	if err := templ.Start(); err != nil {
		fmt.Println("=> templ watcher skipped (not installed)")
	}

	// Run with air (hot reload) if available, else go run .
	var server *exec.Cmd
	if _, err := exec.LookPath("air"); err == nil {
		server = exec.Command("air")
	} else {
		server = exec.Command("go", "run", ".")
	}
	server.Env = append(os.Environ(), fmt.Sprintf("PORT=%s", serverPort))
	server.Stdout = &airBannerFilter{w: os.Stdout}
	server.Stderr = os.Stderr
	if err := server.Start(); err != nil {
		return fmt.Errorf("could not start server: %w", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	fmt.Println("\n\033[90m=> shutting down...\033[0m")
	if templ.Process != nil {
		templ.Process.Kill()
	}
	if server.Process != nil {
		server.Process.Kill()
	}
	return nil
}

// printServerBanner prints the Forge logo and server status.
func printServerBanner(port string) {
	version := forgeVersion()

	yellow := "\033[93m"
	cyan := "\033[96m"
	green := "\033[92m"
	dim := "\033[90m"
	reset := "\033[0m"

	fmt.Println()
	fmt.Printf("%s  ███████╗ ██████╗ ██████╗  ██████╗ ███████╗%s\n", yellow, reset)
	fmt.Printf("%s  ██╔════╝██╔═══██╗██╔══██╗██╔════╝ ██╔════╝%s\n", yellow, reset)
	fmt.Printf("%s  █████╗  ██║   ██║██████╔╝██║  ███╗█████╗  %s\n", yellow, reset)
	fmt.Printf("%s  ██╔══╝  ██║   ██║██╔══██╗██║   ██║██╔══╝  %s\n", yellow, reset)
	fmt.Printf("%s  ██║     ╚██████╔╝██║  ██║╚██████╔╝███████╗%s\n", yellow, reset)
	fmt.Printf("%s  ╚═╝      ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚══════╝%s\n", yellow, reset)
	fmt.Println()
	fmt.Printf("%s  ─────────────────────────────────────────────%s\n", cyan, reset)
	fmt.Printf("  %s%-24s%s %s✓%s ENGINE ONLINE    %s→%s http://localhost:%s\n",
		yellow, version, reset,
		green, reset,
		cyan, reset, port,
	)
	fmt.Printf("%s  ─────────────────────────────────────────────%s\n", cyan, reset)
	fmt.Printf("  %sPress Ctrl+C to stop%s\n", dim, reset)
	fmt.Println()
}

// airBannerFilter buffers output line by line and drops air's ASCII art banner.
type airBannerFilter struct {
	w   io.Writer
	buf []byte
}

func (f *airBannerFilter) Write(p []byte) (int, error) {
	f.buf = append(f.buf, p...)
	for {
		idx := bytes.IndexByte(f.buf, '\n')
		if idx < 0 {
			break
		}
		line := string(f.buf[:idx+1])
		f.buf = f.buf[idx+1:]
		if !isAirBannerLine(line) {
			fmt.Fprint(f.w, line)
		}
	}
	return len(p), nil
}

func isAirBannerLine(line string) bool {
	fragments := []string{
		"/ /\\", "/_/--\\", "| |_)", "| |_| \\_",
		"__    _   ___", "__    _",
	}
	for _, f := range fragments {
		if strings.Contains(line, f) {
			return true
		}
	}
	return false
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
		fmt.Printf("\033[93m=> warning:\033[0m missing tools: %s\n", strings.Join(missing, ", "))
		fmt.Println("   run \033[96mforge setup\033[0m to install them")
		fmt.Println()
	}
}
