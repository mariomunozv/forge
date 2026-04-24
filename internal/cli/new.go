package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new [app-name]",
	Short: "Create a new Forge application",
	Args:  cobra.ExactArgs(1),
	RunE:  runNew,
}

func init() {
	rootCmd.AddCommand(newCmd)
}

func runNew(cmd *cobra.Command, args []string) error {
	appName := args[0]

	if _, err := os.Stat(appName); !os.IsNotExist(err) {
		return fmt.Errorf("directory '%s' already exists", appName)
	}

	fmt.Printf("Creating new Forge app: %s\n\n", appName)

	dirs := []string{
		"app/controllers",
		"app/models",
		"app/views/layouts",
		"config",
		"db/migrations",
		"public",
	}

	for _, dir := range dirs {
		path := filepath.Join(appName, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
		fmt.Printf("  create  %s\n", path)
	}

	files := []scaffoldFile{
		{path: "go.mod", tmpl: goModTmpl},
		{path: "main.go", tmpl: mainGoTmpl},
		{path: "config/app.go", tmpl: configAppTmpl},
		{path: "app/controllers/home_controller.go", tmpl: homeControllerTmpl},
		{path: "app/views/layouts/application.html", tmpl: layoutTmpl},
		{path: "app/views/home/index.html", tmpl: homeViewTmpl},
	}

	data := struct{ AppName string }{AppName: appName}

	for _, f := range files {
		if err := writeTemplate(appName, f, data); err != nil {
			return err
		}
	}

	fmt.Printf("\nDone! Your Forge app is ready.\n\n")
	fmt.Printf("  cd %s\n", appName)
	fmt.Printf("  go mod tidy\n")
	fmt.Printf("  forge server\n\n")

	return nil
}

type scaffoldFile struct {
	path string
	tmpl string
}

func writeTemplate(appName string, f scaffoldFile, data any) error {
	path := filepath.Join(appName, f.path)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	tmpl, err := template.New("").Parse(f.tmpl)
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Printf("  create  %s\n", path)
	return tmpl.Execute(file, data)
}

var goModTmpl = `module github.com/{{.AppName}}

go 1.22

require github.com/mariomunozv/forge latest
`

var mainGoTmpl = `package main

import (
	"github.com/{{.AppName}}/config"
	"github.com/mariomunozv/forge"
)

func main() {
	app := forge.New()
	config.Routes(app)
	app.Start(":8080")
}
`

var configAppTmpl = `package config

import "github.com/mariomunozv/forge"

func Routes(app *forge.App) {
	app.GET("/", "home#index")
}
`

var homeControllerTmpl = `package controllers

import "github.com/mariomunozv/forge"

type HomeController struct{}

func (c *HomeController) Index(ctx *forge.Context) error {
	return ctx.Render("home/index", forge.M{
		"title": "Welcome to Forge",
	})
}
`

var layoutTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>{{"{{"}} .title {{"}}"}}</title>
</head>
<body>
  {{"{{"}} yield {{"}}"}}
</body>
</html>
`

var homeViewTmpl = `<h1>{{"{{"}} .title {{"}}"}}</h1>
<p>Your Forge app is running. Go build something.</p>
`
