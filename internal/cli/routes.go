package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var routesCmd = &cobra.Command{
	Use:   "routes",
	Short: "List all registered routes",
	RunE:  runRoutes,
}

var routesJSON bool

func init() {
	routesCmd.Flags().BoolVar(&routesJSON, "json", false, "Output routes as JSON")
	rootCmd.AddCommand(routesCmd)
}

func runRoutes(cmd *cobra.Command, args []string) error {
	forgeCmd := "routes"
	if routesJSON {
		forgeCmd = "routes:json"
	}

	c := exec.Command("go", "run", ".")
	c.Env = append(os.Environ(), "FORGE_CMD="+forgeCmd)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return fmt.Errorf("could not load app routes: %w", err)
	}
	return nil
}
