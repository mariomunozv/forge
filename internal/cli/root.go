package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "forge",
	Short: "Forge — Go web framework with Rails vibes",
	Long: `
███████╗ ██████╗ ██████╗  ██████╗ ███████╗
██╔════╝██╔═══██╗██╔══██╗██╔════╝ ██╔════╝
█████╗  ██║   ██║██████╔╝██║  ███╗█████╗
██╔══╝  ██║   ██║██╔══██╗██║   ██║██╔══╝
██║     ╚██████╔╝██║  ██║╚██████╔╝███████╗
╚═╝      ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚══════╝

Go web framework with Rails vibes.
`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		loadDotEnv(".env")
		syncContextFileIfNeeded()
	},
}

// loadDotEnv reads key=value pairs from path and sets them as env vars,
// skipping keys that are already set in the environment.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		// strip optional surrounding quotes
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		if key != "" && os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

// syncContextFileIfNeeded updates the forge:conventions section of the
// project's AI context file if the embedded version differs from the CLI.
func syncContextFileIfNeeded() {
	path := currentContextFile()
	if path == "" {
		return
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}

	current := strings.TrimPrefix(forgeVersion(), "forge ")
	content := string(raw)

	// Check version stamp on first line: <!-- forge:VERSION -->
	firstLine := strings.SplitN(content, "\n", 2)[0]
	fileVersion := strings.TrimSuffix(strings.TrimPrefix(firstLine, "<!-- forge:"), " -->")
	if fileVersion == current {
		return
	}

	// Replace content between markers
	const startMarker = "<!-- forge:conventions:start -->"
	const endMarker = "<!-- forge:conventions:end -->"
	si := strings.Index(content, startMarker)
	ei := strings.Index(content, endMarker)
	if si < 0 || ei < 0 || ei <= si {
		return
	}

	newContent := content[:si] + startMarker + "\n" + contextConventions + endMarker + content[ei+len(endMarker):]

	// Update version stamp on first line
	nl := strings.IndexByte(newContent, '\n')
	if nl >= 0 {
		newContent = "<!-- forge:" + current + " -->" + newContent[nl:]
	}

	if err := os.WriteFile(path, []byte(newContent), 0644); err == nil {
		fmt.Printf("\033[90m=> context file updated to %s\033[0m\n\n", current)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
