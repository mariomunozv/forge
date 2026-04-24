package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var generateViewCmd = &cobra.Command{
	Use:     "view [name] [actions...]",
	Aliases: []string{"v"},
	Short:   "Generate templ view files",
	Example: "  forge g view users index show",
	Args:    cobra.MinimumNArgs(1),
	RunE:    runGenerateView,
}

func init() {
	generateCmd.AddCommand(generateViewCmd)
}

type viewData struct {
	Package   string
	Name      string // "Users"
	SnakeName string // "users"
	ModelName string // "User"
	Action    string // current action being generated
}

func runGenerateView(cmd *cobra.Command, args []string) error {
	name := args[0]
	actions := args[1:]

	if len(actions) == 0 {
		actions = []string{"index"}
	}

	for _, action := range actions {
		data := viewData{
			Package:   snake(name),
			Name:      pascal(name),
			SnakeName: snake(name),
			ModelName: pascal(singular(name)),
			Action:    action,
		}

		path := fmt.Sprintf("app/views/%s/%s.templ", data.SnakeName, action)
		tmpl := viewTemplate(action)

		if err := writeGeneratedFile(path, tmpl, data); err != nil {
			return err
		}
	}

	fmt.Println("\nDone! Run templ generate (or forge server) to compile the templates.")
	return nil
}

func viewTemplate(action string) string {
	switch action {
	case "index":
		return viewIndexTmpl
	case "show":
		return viewShowTmpl
	case "new", "create":
		return viewFormTmpl
	case "edit", "update":
		return viewFormTmpl
	default:
		return viewBlankTmpl
	}
}

var viewIndexTmpl = `package {{.Package}}

import "github.com/mariomunozv/forge/example/views/layouts"

type IndexData struct {
	{{.Name}} []{{.ModelName}}
}

type {{.ModelName}} struct {
	ID int
}

templ Index(data IndexData) {
	@layouts.Application("{{.Name}}") {
		<div class="container">
			<div class="page-header">
				<h1>{{.Name}}</h1>
				<a href="/{{.SnakeName}}/new" class="btn btn-primary">New {{.ModelName}}</a>
			</div>
			<ul>
				for _, item := range data.{{.Name}} {
					<li>
						<a href={ templ.SafeURL("/{{.SnakeName}}/" + itoa(item.ID)) }>
							Item { itoa(item.ID) }
						</a>
					</li>
				}
			</ul>
		</div>
	}
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
`

var viewShowTmpl = `package {{.Package}}

import "github.com/mariomunozv/forge/example/views/layouts"

type ShowData struct {
	{{.ModelName}} {{.ModelName}}
}

type {{.ModelName}} struct {
	ID int
}

templ Show(data ShowData) {
	@layouts.Application("{{.ModelName}}") {
		<div class="container">
			<h1>{{.ModelName}} { itoa(data.{{.ModelName}}.ID) }</h1>
			<a href="/{{.SnakeName}}">Back to list</a>
		</div>
	}
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
`

var viewFormTmpl = `package {{.Package}}

import "github.com/mariomunozv/forge/example/views/layouts"

type FormData struct {
	Action string // "new" or "edit"
}

templ Form(data FormData) {
	@layouts.Application(data.Action + " {{.ModelName}}") {
		<div class="container">
			<h1>{ data.Action } {{.ModelName}}</h1>
			<form method="POST" action="/{{.SnakeName}}">
				<button type="submit">Save</button>
			</form>
		</div>
	}
}
`

var viewBlankTmpl = `package {{.Package}}

import "github.com/mariomunozv/forge/example/views/layouts"

templ {{pascal .Action}}() {
	@layouts.Application("{{.Name}} — {{pascal .Action}}") {
		<div class="container">
			<h1>{{.Name}}: {{pascal .Action}}</h1>
		</div>
	}
}
`
