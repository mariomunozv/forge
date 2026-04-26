package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Update FORGE.md with the current CLI version's conventions",
	RunE:  runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	data := struct {
		AppName string
		Version string
	}{Version: forgeVersion()}

	f := scaffoldFile{path: "FORGE.md", tmpl: forgeMdTmpl}
	if err := writeTemplate(".", f, data); err != nil {
		return err
	}
	fmt.Printf("\n\033[92m✓\033[0m FORGE.md updated to %s\n", forgeVersion())
	return nil
}
