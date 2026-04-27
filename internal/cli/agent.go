package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// agentFiles maps agent names to their context file conventions.
var agentFiles = map[string]string{
	"claude":   "CLAUDE.md",
	"cursor":   ".cursorrules",
	"copilot":  ".github/copilot-instructions.md",
	"windsurf": ".windsurfrules",
}

var agentCmd = &cobra.Command{
	Use:   "agent [name]",
	Short: "Switch AI agent convention (renames the context file)",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgent,
}

func init() {
	rootCmd.AddCommand(agentCmd)
}

func runAgent(cmd *cobra.Command, args []string) error {
	target := args[0]
	newPath, ok := agentFiles[target]
	if !ok {
		names := make([]string, 0, len(agentFiles))
		for k := range agentFiles {
			names = append(names, k)
		}
		sort.Strings(names)
		return fmt.Errorf("unknown agent %q — supported: %s", target, strings.Join(names, ", "))
	}

	current := currentContextFile()
	if current == "" {
		return fmt.Errorf("no context file found in this directory")
	}
	if current == newPath {
		fmt.Printf("=> already using %s\n", newPath)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
		return err
	}

	if err := os.Rename(current, newPath); err != nil {
		return err
	}
	fmt.Printf("=> %s → %s\n", current, newPath)
	return nil
}

// currentContextFile returns the path of the existing agent context file.
func currentContextFile() string {
	// Check in a consistent order
	for _, agent := range []string{"claude", "cursor", "copilot", "windsurf"} {
		path := agentFiles[agent]
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
