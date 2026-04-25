package cli

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Install required tools (templ, air) and verify environment",
	RunE:  runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

type tool struct {
	name    string
	pkg     string
	check   string // command to verify it's installed
	purpose string
}

var requiredTools = []tool{
	{
		name:    "templ",
		pkg:     "github.com/a-h/templ/cmd/templ@latest",
		check:   "templ version",
		purpose: "compile .templ components to Go",
	},
	{
		name:    "air",
		pkg:     "github.com/air-verse/air@latest",
		check:   "air -v",
		purpose: "hot reload on file changes",
	},
}

func runSetup(cmd *cobra.Command, args []string) error {
	fmt.Println("Forge setup")
	fmt.Println()

	// 1. check Go version
	if err := checkGo(); err != nil {
		return err
	}

	// 2. install tools
	for _, t := range requiredTools {
		installTool(t)
	}

	fmt.Println("\nAll done. You're ready to build with Forge.")
	fmt.Println("\n  forge new myapp")
	fmt.Println("  cd myapp")
	fmt.Println("  forge server")

	return nil
}

func checkGo() error {
	out, err := exec.Command("go", "version").Output()
	if err != nil {
		return fmt.Errorf("Go not found — install from https://go.dev/dl/")
	}

	version := strings.TrimSpace(string(out))
	fmt.Printf("  ✓ %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
	return nil
}

func installTool(t tool) {
	// check if already installed
	if isInstalled(t.check) {
		out, _ := exec.Command(strings.Fields(t.check)[0], strings.Fields(t.check)[1:]...).Output()
		fmt.Printf("  ✓ %s already installed — %s\n", t.name, strings.TrimSpace(string(out)))
		return
	}

	fmt.Printf("  installing %s (%s)...", t.name, t.purpose)
	if err := exec.Command("go", "install", t.pkg).Run(); err != nil {
		fmt.Printf(" failed\n    run manually: go install %s\n", t.pkg)
		return
	}
	fmt.Println(" done")
}

func isInstalled(checkCmd string) bool {
	parts := strings.Fields(checkCmd)
	if len(parts) == 0 {
		return false
	}
	_, err := exec.LookPath(parts[0])
	return err == nil
}
