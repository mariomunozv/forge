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
		syncForgeMdIfNeeded()
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

// syncForgeMdIfNeeded regenerates FORGE.md if it is missing or was generated
// by a different CLI version. Runs silently on every forge command.
func syncForgeMdIfNeeded() {
	current := strings.TrimPrefix(forgeVersion(), "forge ")

	data, err := os.ReadFile("FORGE.md")
	if err == nil {
		firstLine := strings.SplitN(string(data), "\n", 2)[0]
		inner := strings.TrimPrefix(firstLine, "<!-- forge:")
		fileVersion := strings.SplitN(inner, " ", 2)[0]
		if fileVersion == current {
			return // already up to date
		}
	} else if !os.IsNotExist(err) {
		return // unreadable for another reason, skip silently
	}

	tmplData := struct{ Version string }{Version: current}
	f := scaffoldFile{path: "FORGE.md", tmpl: forgeMdTmpl}
	writeTemplate(".", f, tmplData) //nolint:errcheck — best-effort, not in a forge app dir
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
