package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var generateControllerCmd = &cobra.Command{
	Use:     "controller [name] [actions...]",
	Aliases: []string{"c"},
	Short:   "Generate a controller",
	Example: "  forge g controller users index show create update destroy",
	Args:    cobra.MinimumNArgs(1),
	RunE:    runGenerateController,
}

func init() {
	generateCmd.AddCommand(generateControllerCmd)
}

type controllerData struct {
	Package    string
	Name       string // "Users"
	SnakeName  string // "users"
	Actions    []string
	ModulePath string
}

func runGenerateController(cmd *cobra.Command, args []string) error {
	name := args[0]
	actions := args[1:]

	// default to RESTful actions if none specified
	if len(actions) == 0 {
		actions = []string{"index", "show", "create", "update", "destroy"}
	}

	data := controllerData{
		Package:   "controllers",
		Name:      pascal(name),
		SnakeName: snake(name),
		Actions:   actions,
	}

	path := fmt.Sprintf("app/controllers/%s_controller.go", data.SnakeName)
	if err := writeGeneratedFile(path, controllerTmpl, data); err != nil {
		return err
	}

	fmt.Printf("\nDone! Register your controller in config/app.go:\n")
	fmt.Printf("  app.Register(%q, &controllers.%sController{})\n", data.SnakeName, data.Name)
	fmt.Printf("  app.Resources(%q)\n", data.SnakeName)

	return nil
}

var controllerTmpl = `package controllers

import (
	"net/http"

	"github.com/mariomunozv/forge/forge"
)

type {{.Name}}Controller struct{}
{{range .Actions}}
func (c *{{$.Name}}Controller) {{pascal .}}(ctx *forge.Context) error {
	{{- if eq . "index"}}
	return ctx.Success(forge.M{"{{$.SnakeName}}": []any{}})
	{{- else if eq . "show"}}
	id := ctx.Param("id")
	return ctx.Success(forge.M{"id": id})
	{{- else if eq . "create"}}
	return ctx.Created(forge.M{"{{$.SnakeName}}": nil})
	{{- else if eq . "update"}}
	id := ctx.Param("id")
	return ctx.Success(forge.M{"id": id})
	{{- else if eq . "destroy"}}
	_ = ctx.Param("id")
	return ctx.Status(http.StatusNoContent)
	{{- else}}
	return ctx.Success(forge.M{})
	{{- end}}
}
{{end}}`
