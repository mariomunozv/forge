package cli

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the forge version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(forgeVersion())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func forgeVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "forge (unknown)"
	}
	v := info.Main.Version
	if v == "" || v == "(devel)" {
		return "forge (dev)"
	}
	return "forge " + shortVersion(v)
}

// shortVersion converts "v0.0.0-20260425031158-b3959d833a3c" → "v0.0.0-b3959d8"
func shortVersion(v string) string {
	parts := strings.Split(v, "-")
	// pseudo-version: base-timestamp-hash
	if len(parts) == 3 && len(parts[1]) == 14 {
		hash := parts[2]
		if len(hash) > 7 {
			hash = hash[:7]
		}
		return parts[0] + "-" + hash
	}
	return v
}
