package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"

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
		"app/views/home",
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

	version := time.Now().UTC().Format("20060102150405")
	files := []scaffoldFile{
		{path: "go.mod", tmpl: goModTmpl},
		{path: "main.go", tmpl: mainGoTmpl},
		{path: ".air.toml", tmpl: airTomlTmpl},
		{path: ".env.example", tmpl: envExampleTmpl},
		{path: ".gitignore", tmpl: gitignoreTmpl},
		{path: "config/app.go", tmpl: configAppTmpl},
		{path: "config/database.go", tmpl: configDBTmpl},
		{path: "app/controllers/home_controller.go", tmpl: homeControllerTmpl},
		{path: "app/models/validate.go", tmpl: modelsValidateTmpl},
		{path: "app/views/layouts/application.templ", tmpl: layoutTmpl},
		{path: "app/views/home/index.templ", tmpl: homeViewTmpl},
		{path: fmt.Sprintf("db/migrations/%s_initial.sql", version), tmpl: initialMigrationTmpl},
	}

	data := struct{ AppName string }{AppName: appName}

	for _, f := range files {
		if err := writeTemplate(appName, f, data); err != nil {
			return err
		}
	}

	fmt.Println()
	if err := runPostScaffold(appName); err != nil {
		return err
	}

	fmt.Printf("\nDone! Your Forge app is ready.\n\n")
	fmt.Printf("  cd %s\n", appName)
	fmt.Printf("  cp .env.example .env   # edit DATABASE_URL with your postgres credentials\n")
	fmt.Printf("  forge db create        # create the database\n")
	fmt.Printf("  forge db migrate       # run migrations\n")
	fmt.Printf("  forge server\n\n")

	return nil
}

func runPostScaffold(appName string) error {
	steps := []struct {
		label string
		name  string
		args  []string
		warn  string
	}{
		{
			label: "Fetching forge dependency...",
			name:  "go",
			args:  []string{"get", "github.com/mariomunozv/forge@latest"},
			warn:  "could not fetch forge — run 'go get github.com/mariomunozv/forge@latest' manually",
		},
		{
			label: "Running go mod tidy...",
			name:  "go",
			args:  []string{"mod", "tidy"},
			warn:  "go mod tidy failed — run it manually inside your app directory",
		},
		{
			label: "Generating templ files...",
			name:  "templ",
			args:  []string{"generate"},
			warn:  "templ not found — install with: go install github.com/a-h/templ/cmd/templ@latest\n   then run 'templ generate' in your app directory",
		},
	}

	for _, s := range steps {
		fmt.Printf("=> %s\n", s.label)
		c := exec.Command(s.name, s.args...)
		c.Dir = appName
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			fmt.Printf("   warning: %s\n", s.warn)
		}
	}

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
`

var mainGoTmpl = `package main

import (
	"github.com/{{.AppName}}/config"
	"github.com/mariomunozv/forge"
)

func main() {
	config.ConnectDB()
	defer config.DB.Close()

	app := forge.New()
	config.Setup(app)
	app.Start(":8080")
}
`

var configAppTmpl = `package config

import (
	"github.com/{{.AppName}}/app/controllers"
	"github.com/mariomunozv/forge"
)

func Setup(app *forge.App) {
	app.Register("home", &controllers.HomeController{})
	app.GET("/", "home#index")
}
`

var homeControllerTmpl = `package controllers

import (
	home "github.com/{{.AppName}}/app/views/home"
	"github.com/mariomunozv/forge"
)

type HomeController struct{}

func (c *HomeController) Index(ctx *forge.Context) error {
	return ctx.Component(home.Index(home.IndexData{
		AppName: "{{.AppName}}",
	}))
}
`

var layoutTmpl = `package layouts

templ Application(title string) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>{ title }</title>
		</head>
		<body>
			{ children... }
		</body>
	</html>
}
`

var homeViewTmpl = `package home

import "github.com/{{.AppName}}/app/views/layouts"

type IndexData struct {
	AppName string
}

templ Index(data IndexData) {
	@layouts.Application("Welcome") {
		<h1>Welcome to { data.AppName }!</h1>
		<p>Your Forge app is running. Go build something.</p>
	}
}
`

var configDBTmpl = `package config

import (
	"database/sql"
	"log"

	"github.com/mariomunozv/forge/db"
)

// DB is the global database connection. Initialized by ConnectDB.
var DB *sql.DB

func ConnectDB() {
	conn, err := db.Open()
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	DB = conn
}
`

var envExampleTmpl = `# Format: postgres://user:password@host/dbname?sslmode=disable
# If running postgres locally with no password: postgres://localhost/{{.AppName}}_development?sslmode=disable
DATABASE_URL=postgres://localhost/{{.AppName}}_development?sslmode=disable

SESSION_SECRET=change-me-in-production
`

var gitignoreTmpl = `.env
tmp/
`

var initialMigrationTmpl = `-- migrate:up


-- migrate:down
`

var airTomlTmpl = `root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o tmp/main ."
  bin = "tmp/main"
  delay = 200
  exclude_dir = ["tmp", "vendor", "db"]
  include_ext = ["go"]
  exclude_regex = ["_test\\.go", "_templ\\.go"]

[log]
  time = false

[color]
  main = "magenta"
  watcher = "cyan"
  build = "yellow"
  runner = "green"
`
