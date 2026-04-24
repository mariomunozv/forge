# Forge — Go web framework with Rails vibes

## What this is

A Go web framework being built from scratch. The goal is Rails-like DX (fast to develop, conventions over configuration, great generators) but with Go's performance and type safety. It's a personal project — "for the meme of it" — not trying to replace existing frameworks.

Module path: `github.com/mariomunozv/forge`

## Project structure

```
forge/              ← core framework package (public API)
  forge.go          # App struct, New(), routing methods, Resources(), Start()
  router.go         # route matching, "controller#action" resolution via reflection
  context.go        # Context with JSON/HTML response helpers
  controller.go     # Controller interface + registry
  renderer.go       # Renderer interface for pluggable template engines
forgetest/          ← testing helpers package
  forgetest.go      # TestApp wrapper
  request.go        # RequestBuilder (fluent API)
  response.go       # Response assertions
internal/cli/       ← forge CLI (cobra-based)
  root.go           # banner + root command
  new.go            # forge new [app-name]
  server.go         # forge server / forge s
  routes.go         # forge routes [--json]
  generate.go       # forge generate (parent) + naming helpers + field parser
  generate_controller.go
  generate_model.go
  generate_view.go
  generate_resource.go
cmd/forge/main.go   ← CLI entrypoint
example/views/      ← example templ components showing conventions
```

## Core design decisions

### Routing
Routes use Rails-style `"controller#action"` strings:
```go
app.GET("/users", "users#index")
app.GET("/users/:id", "users#show")
app.Resources("users")  // generates all 5 REST routes
```
Controllers are registered by name and actions are resolved via reflection. Controller must be registered before routes are served (not at definition time).

### Response negotiation
`ctx.WantsJSON()` checks (in order): `?format=json` param → `Accept: application/json` header → `Content-Type: application/json` header.

`ctx.Respond(data, component)` auto-negotiates:
- JSON client → `{"data": ...}` envelope
- HTML client → renders the templ component

All JSON responses use an envelope:
```json
{ "data": {...} }           // Success, Created, Respond
{ "error": { "message": "...", "code": 422 } }  // Error
```

### Template engine: templ (not html/template)
We chose `github.com/a-h/templ` over stdlib `html/template` because:
- Type-safe components (errors at compile time, not runtime)
- Real slots via `{ children... }`
- Component composition like modern JS frameworks
- `forge server` runs `templ generate --watch` automatically

Layout + component + slot pattern:
```go
// layouts/application.templ
templ Application(title string) {
  <html><body>{ children... }</body></html>
}

// views/posts/index.templ
templ Index(data IndexData) {
  @layouts.Application("Posts") {
    for _, post := range data.Posts {
      @components.Card(components.CardProps{Title: post.Title}) {
        <p>slot content here</p>
      }
    }
  }
}
```

### FORGE_CMD mechanism
`forge routes` works by running the app binary with `FORGE_CMD=routes`, which causes `App.Start()` to print routes and exit instead of starting the server. This is how all introspection commands work — the CLI talks to the compiled app, not to source files.

## Key APIs

### forge package
```go
app := forge.New()
app.Register("users", &UsersController{})
app.GET("/users", "users#index")
app.Resources("users")
app.Use(myMiddleware)
app.Start(":8080")
```

### Context methods
```go
ctx.Param("id")           // URL param
ctx.Query("page")         // query string
ctx.Bind(&input)          // decode JSON body
ctx.WantsJSON()           // content negotiation

ctx.Respond(data, component)  // auto-negotiate JSON vs HTML (primary method)
ctx.Component(component)      // explicit HTML via templ
ctx.Success(v)                // 200 {"data": v}
ctx.Created(v)                // 201 {"data": v}
ctx.Error(status, message)    // {"error": {...}}
ctx.JSON(status, v)           // raw JSON, no envelope
ctx.Text(status, body)        // plain text
ctx.Redirect(status, url)     // HTTP redirect
ctx.Status(code)              // status only, no body
```

### forgetest package
```go
app := forgetest.New(t)
app.Register("posts", &PostsController{})
app.Resources("posts")

app.Request("GET", "/posts/1").
    AsJSON().
    Do().
    AssertOK().
    AssertJSONPath("data.title", "Hello Forge").
    AssertJSONPathExists("data.id")
```

### CLI generators
```bash
forge g resource Post title:string body:string published:bool
# generates: model + controller + views (index, show, new, edit)

forge g controller posts index show create update destroy
forge g model Post title:string body:string
forge g view posts index show

# all generators have short aliases: g r, g c, g m, g v
```

Field types: `string/str/text`, `int/integer`, `int64`, `float/float64/decimal`, `bool/boolean`, `time/datetime/timestamp`

## What's built ✓
- [x] CLI: `forge new`, `forge server`, `forge routes`, `forge routes --json`
- [x] Router: method routing, URL params (`:id`), `"controller#action"` convention
- [x] `app.Resources()` — generates 5 REST routes
- [x] Context: full response API
- [x] JSON envelope: `{"data": ...}` / `{"error": {...}}`
- [x] Content negotiation: `ctx.WantsJSON()` + `ctx.Respond()`
- [x] templ integration: `ctx.Component()`, `forge server` auto-watches
- [x] `forgetest` package: fluent test API with JSON path assertions
- [x] Generators: `controller`, `model`, `view`, `resource`, `migration`
- [x] `forge new` generates proper `.templ` scaffold, runs `go get` + `templ generate` post-scaffold
- [x] `forge server` uses `air` for hot reload if available, falls back to `go run .`
- [x] `middleware` package: `Logger`, `Recovery`, `CORS`
- [x] `db` package: `Open()`, `Migrate()`, `Rollback()`, `Status()` — PostgreSQL via `lib/pq`
- [x] `forge db migrate` / `forge db rollback` / `forge db status`
- [x] `forge g migration [name]` — generates timestamped `.sql` file with up/down sections

## What's next (pending — in recommended order)

### 1. `forge describe` — AI agent introspection
A command that outputs the full app structure as JSON: routes, controllers, models, views, migrations. Meant to be fed as context to an AI agent working on the codebase. Also generates a `FORGE.md` (like this CLAUDE.md but for the app being built).

### 2. `forge new` wires database by default
Update the scaffold to include `config/database.go` that calls `db.Open()` on startup, and a sample migration. Makes `forge new myapp && forge db migrate` work end-to-end.

### 3. Query helpers
Thin helpers on top of `database/sql` for common patterns:
- `db.QueryOne`, `db.QueryAll` — scan rows into structs using `db:` tags
- No full ORM — just reduce boilerplate for standard queries

## Dependencies
- `github.com/spf13/cobra` — CLI
- `github.com/a-h/templ` — template engine
- `github.com/tidwall/gjson` — JSON path for forgetest assertions

## Setup on a new machine
```bash
# 1. Install Go 1.22+
brew install go

# 2. Install templ CLI
go install github.com/a-h/templ/cmd/templ@latest

# 3. Install dependencies
go mod tidy

# 4. Build the CLI binary
go build -o forge-bin ./cmd/forge/

# 5. Run tests to verify everything works
go test ./...
```

The module path is `github.com/mariomunozv/forge` — this is hardcoded in go.mod and in generated app scaffolds. If you fork to a different GitHub account/org, update go.mod and the import references in `internal/cli/new.go`.

## Known inconsistency to fix
`forge new` currently generates `.html` files (html/template) in the scaffold, but the framework uses `templ`. The scaffold needs to be updated to generate `.templ` files. This is the first thing to fix before `forge new` is actually usable end-to-end.

## Testing
```bash
go test ./...                    # run all tests
go test ./forge/... -v           # framework core tests
go test ./forgetest/... -v       # testing helper tests
```

22 tests, all passing.
