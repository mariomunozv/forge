package cli

import (
	"fmt"
	"runtime/debug"

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
		return "forge (unknown version)"
	}
	v := info.Main.Version
	if v == "" || v == "(devel)" {
		return "forge (dev build)"
	}
	return "forge " + v
}
