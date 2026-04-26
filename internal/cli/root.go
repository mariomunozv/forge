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
		warnOutdatedForgeMd()
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

// warnOutdatedForgeMd checks if FORGE.md exists and its embedded version
// matches the running CLI. Prints a warning if they differ.
func warnOutdatedForgeMd() {
	data, err := os.ReadFile("FORGE.md")
	if err != nil {
		return
	}
	firstLine := strings.SplitN(string(data), "\n", 2)[0]
	if !strings.HasPrefix(firstLine, "<!-- forge:") {
		return
	}
	// extract version between "<!-- forge:" and " —"
	inner := strings.TrimPrefix(firstLine, "<!-- forge:")
	fileVersion := strings.SplitN(inner, " ", 2)[0]

	current := forgeVersion() // e.g. "forge v0.0.0-abc1234"
	currentVersion := strings.TrimPrefix(current, "forge ")

	if fileVersion != currentVersion {
		fmt.Printf("\033[93m=> warning:\033[0m FORGE.md was generated with %s, CLI is now %s\n", fileVersion, currentVersion)
		fmt.Println("   run \033[96mforge sync\033[0m to update it")
		fmt.Println()
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
