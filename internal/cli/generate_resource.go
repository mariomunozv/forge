package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var generateResourceCmd = &cobra.Command{
	Use:     "resource [name] [fields...]",
	Aliases: []string{"r"},
	Short:   "Generate a full resource: model + controller + views",
	Example: "  forge g resource Post title:string body:text published:bool",
	Args:    cobra.MinimumNArgs(1),
	RunE:    runGenerateResource,
}

func init() {
	generateCmd.AddCommand(generateResourceCmd)
}

func runGenerateResource(cmd *cobra.Command, args []string) error {
	name := args[0]
	fields := args[1:]

	fmt.Printf("Generating resource: %s\n\n", pascal(name))

	// model
	fmt.Println("==> Model")
	modelArgs := append([]string{singular(name)}, fields...)
	if err := runGenerateModel(cmd, modelArgs); err != nil {
		return err
	}

	// controller
	fmt.Println("\n==> Controller")
	restActions := []string{"index", "show", "create", "update", "destroy"}
	if err := runGenerateController(cmd, append([]string{name}, restActions...)); err != nil {
		return err
	}

	// views
	fmt.Println("\n==> Views")
	viewActions := []string{"index", "show", "new", "edit"}
	if err := runGenerateView(cmd, append([]string{name}, viewActions...)); err != nil {
		return err
	}

	// summary
	fmt.Printf(`
Done! Add to config/app.go:

  app.Register(%q, &controllers.%sController{})
  app.Resources(%q)

`, snake(name), pascal(name), snake(name))

	return nil
}
