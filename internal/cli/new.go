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
		{path: "CLAUDE.md", tmpl: claudeMdTmpl},
		{path: "FORGE.md", tmpl: forgeMdTmpl},
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

	data := struct {
		AppName string
		Version string
	}{AppName: appName, Version: forgeVersion()}

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
		env   []string
		warn  string
		fatal bool
	}{
		{
			label: "Fetching forge dependency...",
			name:  "go",
			args:  []string{"get", "github.com/mariomunozv/forge@latest"},
			env:   []string{"GOPROXY=direct"},
			warn:  "could not fetch forge — run 'GOPROXY=direct go get github.com/mariomunozv/forge@latest' manually",
			fatal: true,
		},
		{
			// templ generate must run before go mod tidy so _templ.go files exist
			// when Go resolves local package imports.
			label: "Generating templ files...",
			name:  "templ",
			args:  []string{"generate"},
			warn:  "templ not found — install with: go install github.com/a-h/templ/cmd/templ@latest\n   then run 'templ generate' in your app directory",
		},
		{
			label: "Running go mod tidy...",
			name:  "go",
			args:  []string{"mod", "tidy"},
			env:   []string{"GONOSUMDB=*"},
			warn:  "go mod tidy failed — run 'go mod tidy' manually inside your app directory",
			fatal: true,
		},
	}

	for _, s := range steps {
		fmt.Printf("=> %s\n", s.label)
		c := exec.Command(s.name, s.args...)
		c.Dir = appName
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Env = append(os.Environ(), s.env...)
		if err := c.Run(); err != nil {
			fmt.Printf("\n   ✗ %s\n\n", s.warn)
			if s.fatal {
				return fmt.Errorf("setup failed at step: %s", s.label)
			}
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
		<body style="box-sizing:border-box;margin:0;padding:0;background:#0f0f0f;color:#F0F0F0;font-family:ui-monospace,'SF Mono',Menlo,monospace;min-height:100vh">
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

var forgeLogo = "  ███████╗ ██████╗ ██████╗  ██████╗ ███████╗\n  ██╔════╝██╔═══██╗██╔══██╗██╔════╝ ██╔════╝\n  █████╗  ██║   ██║██████╔╝██║  ███╗█████╗\n  ██╔══╝  ██║   ██║██╔══██╗██║   ██║██╔══╝\n  ██║     ╚██████╔╝██║  ██║╚██████╔╝███████╗\n  ╚═╝      ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚══════╝"

templ Index(data IndexData) {
	@layouts.Application("Welcome · " + data.AppName) {
		<div style="display:flex;align-items:center;justify-content:center;min-height:90vh;padding:32px">
			<div style="max-width:560px;width:100%">
				<div style="color:#E8FF00;font-size:11px;letter-spacing:4px;margin-bottom:24px;opacity:.7">
					{ "// FORGE FRAMEWORK" }
				</div>
				<pre style="color:#E8FF00;font-size:13px;line-height:1.3;letter-spacing:1px;text-shadow:0 0 20px rgba(232,255,0,.4);margin-bottom:32px">{ forgeLogo }</pre>
				<div style="border-top:1px solid #252525;padding-top:24px;margin-bottom:24px">
					<div style="font-size:20px;font-weight:700;color:#F0F0F0;margin-bottom:8px">
						{ data.AppName }<span style="color:#E8FF00">_</span>
					</div>
					<div style="color:#888;font-size:13px;line-height:1.6">
						Your Forge app is running.<br/>
						Go build something.
					</div>
				</div>
				<div style="display:flex;gap:24px;font-size:12px">
					<span><span style="color:#39FF5A">✓</span> <span style="color:#888">engine online</span></span>
					<span><span style="color:#00D4FF">→</span> <span style="color:#888">forge g resource Post title:string</span></span>
				</div>
			</div>
		</div>
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
FORGE.md
`

var initialMigrationTmpl = `-- migrate:up


-- migrate:down
`

var airTomlTmpl = `root = "."
tmp_dir = "tmp"

[build]
  cmd = "templ generate && go build -o tmp/main ."
  entrypoint = "tmp/main"
  delay = 200
  exclude_dir = ["tmp", "vendor", "db"]
  include_ext = ["go", "templ"]
  exclude_regex = ["_test\\.go", "_templ\\.go"]

[log]
  time = false

[color]
  main = "magenta"
  watcher = "cyan"
  build = "yellow"
  runner = "green"
`

var claudeMdTmpl = `# {{.AppName}}

@FORGE.md

## Project structure
` + "```" + `
app/
  controllers/   # one file per controller, e.g. posts_controller.go
  models/        # structs + DB helpers
  views/
    layouts/     # application.templ — shared HTML shell
    <resource>/  # index.templ, show.templ, new.templ, edit.templ
config/
  app.go         # route registration — add routes here
  database.go    # DB connection
db/
  migrations/    # .sql files — run with: forge db migrate
main.go
` + "```" + `

## Views (templ) — module path for this app
` + "```" + `go
// app/views/posts/index.templ
package posts

import "github.com/{{.AppName}}/app/views/layouts"

type IndexData struct{ Posts []models.Post }

templ Index(data IndexData) {
    @layouts.Application("Posts") {
        for _, p := range data.Posts {
            <div>{ p.Title }</div>
        }
    }
}
` + "```" + `
`

var forgeMdTmpl = `<!-- forge:{{.Version}} — run ` + "`forge sync`" + ` to update -->

# Forge conventions

## Stack
- **Go** + **Forge** framework — Rails-like conventions, no magic
- **templ** — type-safe HTML components (compile-time errors, not runtime)
- **PostgreSQL** via ` + "`lib/pq`" + `
- **air** — hot reload (` + "`forge server`" + ` starts it automatically)

## Generators
` + "```" + `bash
forge g resource Post title:string body:string published:bool
# → model, controller, all views (index/show/new/edit), migration

forge g controller Comments index show
forge g model Comment post_id:int body:string
forge g view comments index show
forge g migration add_published_to_posts
forge g auth          # users table + bcrypt model + sessions controller + login view
forge g job SendEmail # background job
` + "```" + `

Field types: ` + "`string`" + `, ` + "`int`" + `, ` + "`int64`" + `, ` + "`float64`" + `, ` + "`bool`" + `, ` + "`time`" + `

## Routing — config/app.go
` + "```" + `go
app.Register("posts", &controllers.PostsController{})
app.Resources("posts")                           // 7 routes: index, new, create, show, edit, update, destroy
app.Member("posts", "POST", "publish")           // POST /posts/:id/publish → posts#publish
app.Collection("posts", "GET", "drafts")         // GET /posts/drafts → posts#drafts
app.GET("/about", "home#about")
app.Use(middleware.Auth())                        // sets ctx.Values["current_user_id"] if session valid
app.Use(middleware.RequireAuth())                 // 401 if no valid session
app.Use(middleware.MethodOverride())              // PUT/DELETE from HTML forms via _method field
` + "```" + `

## Controllers
` + "```" + `go
type PostsController struct{}

func (c *PostsController) Index(ctx *forge.Context) error {
    posts, err := models.AllPosts(config.DB)
    if err != nil {
        return ctx.Error(500, "could not load posts")
    }
    return ctx.Respond(posts, views.PostsIndex(posts)) // JSON or HTML based on Accept header
}

func (c *PostsController) Create(ctx *forge.Context) error {
    var input struct { Title string ` + "`json:\"title\"`" + ` }
    if err := ctx.Bind(&input); err != nil {
        return ctx.Error(400, "invalid input")
    }
    return ctx.Created(post) // 201 {"data": post}
}
` + "```" + `

## Context API
` + "```" + `go
ctx.Param("id")               // URL param
ctx.Query("page")             // query string
ctx.Bind(&input)              // decode JSON or form body into struct
ctx.WantsJSON()               // true if client wants JSON

ctx.Respond(data, component)  // auto: JSON for API clients, HTML for browser
ctx.Component(component)      // render a templ component (HTML only)
ctx.Success(v)                // 200 {"data": v}
ctx.Created(v)                // 201 {"data": v}
ctx.Error(status, "msg")      // {"error": {"message": ..., "code": status}}
ctx.Redirect(302, "/path")

ctx.SignIn(userID)            // write signed session cookie
ctx.SignOut()                 // clear session cookie
ctx.CurrentUserID()           // (int64, bool) — read & verify session
ctx.Values["current_user_id"] // set by middleware.Auth()
` + "```" + `

## Database
` + "```" + `go
// Migrations: db/migrations/<timestamp>_name.sql
// -- migrate:up
// CREATE TABLE posts (...);
// -- migrate:down
// DROP TABLE posts;

post, err  := db.QueryOne[Post](config.DB, "SELECT * FROM posts WHERE id=$1", id)
posts, err := db.QueryAll[Post](config.DB, "SELECT * FROM posts ORDER BY created_at DESC")

id, err := db.Insert(config.DB, "posts", map[string]any{"title": "Hello"})
err     = db.Update(config.DB, "posts", id, map[string]any{"title": "Updated"})
err     = db.Delete(config.DB, "posts", id)
` + "```" + `

Struct tags for scanning:
` + "```" + `go
type Post struct {
    ID        int64     ` + "`db:\"id\"`" + `
    Title     string    ` + "`db:\"title\"`" + `
    CreatedAt time.Time ` + "`db:\"created_at\"`" + `
}
` + "```" + `

## Auth / sessions
` + "```" + `bash
forge g auth   # generates users migration, User model (bcrypt), sessions controller, login view
` + "```" + `

## Development commands
` + "```" + `bash
forge server          # start dev server with hot reload
forge routes          # list all registered routes
forge db migrate      # run pending migrations
forge db rollback     # roll back last migration
forge db status       # show migration state
` + "```" + `
`
