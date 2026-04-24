package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var generateMigrationCmd = &cobra.Command{
	Use:     "migration [name]",
	Aliases: []string{"mig"},
	Short:   "Generate a migration file",
	Example: "  forge g migration create_users\n  forge g migration add_email_to_users",
	Args:    cobra.ExactArgs(1),
	RunE:    runGenerateMigration,
}

func init() {
	generateCmd.AddCommand(generateMigrationCmd)
}

func runGenerateMigration(cmd *cobra.Command, args []string) error {
	name := snake(args[0])
	version := time.Now().UTC().Format("20060102150405")
	filename := fmt.Sprintf("db/migrations/%s_%s.sql", version, name)

	if err := writeGeneratedFile(filename, migrationTmpl, nil); err != nil {
		return err
	}

	fmt.Printf("\nDone! Edit your migration at %s\n", filename)
	return nil
}

var migrationTmpl = `-- migrate:up


-- migrate:down

`
