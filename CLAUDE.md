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
app.Resources("users")               // generates 7 REST routes (index, new, create, show, edit, update, destroy)
app.Member("users", "POST", "ban")   // POST /users/:id/ban → users#ban
app.Collection("users", "GET", "active") // GET /users/active → users#active
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
app.Member("users", "POST", "ban")        // POST /users/:id/ban → users#ban
app.Collection("users", "GET", "active")  // GET /users/active → users#active
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

// Session / auth
ctx.SignIn(userID int64)       // write signed session cookie
ctx.SignOut()                  // clear session cookie
ctx.CurrentUserID() (int64, bool) // read & verify session cookie

// Per-request values set by middleware
ctx.Values["current_user_id"]  // set by middleware.Auth()
```

### Session middleware
```go
app.Use(middleware.Auth())        // sets ctx.Values["current_user_id"] if logged in
app.Use(middleware.RequireAuth()) // 401 if no valid session
```

Reads `SESSION_SECRET` env var for HMAC key. Falls back to a dev default (log a warning in prod).

### Background jobs
```go
// Enqueue from anywhere with a DB connection
id, err := jobs.Enqueue(db, "welcome_email", map[string]any{"user_id": 42})

// Define a job (generated by forge g job)
type WelcomeEmailJob struct {
    UserID int64 `json:"user_id"`
}
func (j *WelcomeEmailJob) Perform(ctx context.Context) error { ... }
func init() { jobs.Register("welcome_email", func() jobs.Job { return &WelcomeEmailJob{} }) }

// Start workers (cmd/worker/main.go, generated by forge g job)
jobs.Work(ctx, db, 5)
```

```bash
forge jobs work             # start workers (reads DATABASE_URL)
forge jobs work --workers 10
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
forge g auth                    # users table + User model (bcrypt) + sessions controller + login view
forge g job WelcomeEmail        # app/jobs/welcome_email_job.go + cmd/worker/main.go

# all generators have short aliases: g r, g c, g m, g v, g j
```

Field types: `string/str/text`, `int/integer`, `int64`, `float/float64/decimal`, `bool/boolean`, `time/datetime/timestamp`

## What's built ✓
- [x] CLI: `forge new`, `forge server`, `forge routes`, `forge routes --json`, `forge version`, `forge setup`
- [x] Router: method routing, URL params (`:id`), `"controller#action"` convention
- [x] `app.Resources()` — generates 7 REST routes (adds `new` and `edit`)
- [x] `app.Member()` / `app.Collection()` — custom routes on resource instances / collections
- [x] `middleware.MethodOverride()` — PUT/DELETE from HTML forms via `_method` field
- [x] Context: full response API, `Values map[string]any` for middleware→handler data
- [x] JSON envelope: `{"data": ...}` / `{"error": {...}}`
- [x] Content negotiation: `ctx.WantsJSON()` + `ctx.Respond()`
- [x] templ integration: `ctx.Component()`, `forge server` auto-watches
- [x] `forgetest` package: fluent test API with JSON path assertions
- [x] Generators: `controller`, `model`, `view`, `resource`, `migration`, `auth`, `job`
- [x] `forge new` scaffold: dark welcome page (design system colors), `.env.example` with format comment + `SESSION_SECRET`, correct setup instructions (`db create` before `db migrate`)
- [x] `forge server` uses `air` for hot reload if available, falls back to `go run .`; prints branded yellow ASCII banner, filters air's own banner
- [x] All forge commands auto-load `.env` from the current directory (no need to `export` vars manually)
- [x] `middleware` package: `Logger`, `Recovery`, `CORS`, `DevErrors`, `Auth`, `RequireAuth`
- [x] `db` package: `Open()`, `Migrate()`, `Rollback()`, `Status()` — PostgreSQL via `lib/pq`
- [x] `forge db migrate` / `forge db rollback` / `forge db status` / `forge db create` / `forge db drop`
- [x] `forge g migration [name]` — generates `.sql` with timestamp and up/down sections
- [x] `db.QueryOne[T]` / `db.QueryAll[T]` — generic helpers with `db:` tag scanning
- [x] `db.Insert()`, `db.Update()`, `db.Delete()` — CRUD helpers with safe identifier quoting
- [x] Model validations: `email` and `url` field types; `ctx.Validate(model)` writes 422
- [x] `middleware.DevErrors()` — HTML error page with source highlighting, stack trace, request info
- [x] `forge describe` — app structure as JSON + `--md` generates `FORGE.md`
- [x] Authentication: `ctx.SignIn(userID)`, `ctx.SignOut()`, `ctx.CurrentUserID()` — HMAC-SHA256 signed cookies
- [x] `forge g auth` — generates users migration, User model (bcrypt), sessions controller, login view
- [x] `jobs` package: PostgreSQL-backed queue, `Register`/`Enqueue`/`Work`, `FOR UPDATE SKIP LOCKED`, exponential backoff, max 3 attempts
- [x] `forge jobs work` — starts N workers reading from `DATABASE_URL` (`--workers` flag)
- [x] `forge g job [name]` — generates job file with `init()` registration + `cmd/worker/main.go`

## What's next (pending — in recommended order)

### 1. `forge console` — interactive REPL
Launch a Go shell with app context loaded (DB connected, models available). Similar to `rails console`. Complex — Go has no native REPL. Options: `yaegi` interpreter or a generated script approach.

### 2. `forge deploy` — production checklist
Pre-deploy validation: runs tests, checks for missing env vars, verifies migrations are up to date, builds the binary. Optional Dockerfile generation.

## Database philosophy

### SQL (PostgreSQL)
Forge owns the connection and migration layer via the `db` package. `database/sql` is a stable Go standard interface — it doesn't change when drivers update.

### MongoDB (and other NoSQL)
Forge does **not** wrap MongoDB. The driver API (`go.mongodb.org/mongo-driver/v2`) is the interface — there is no `database/sql` equivalent. Wrapping it would leak the driver version into forge's public API, making upgrades (v2→v3) break framework users anyway.

**Recommended pattern** for MongoDB in a forge app:

```go
// config/database.go
package config

import (
    "context"
    "log"
    "os"
    "time"

    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"
)

var MongoDB *mongo.Client

func ConnectMongo() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(options.Client().ApplyURI(os.Getenv("MONGODB_URL")))
    if err != nil {
        log.Fatalf("mongodb: %v", err)
    }
    if err := client.Ping(ctx, nil); err != nil {
        log.Fatalf("mongodb ping: %v", err)
    }
    MongoDB = client
}
```

```go
// main.go
func main() {
    config.ConnectMongo()
    defer config.MongoDB.Disconnect(context.Background())

    app := forge.New()
    config.Setup(app)
    app.Start(":8080")
}
```

This keeps the driver version entirely in the app — no forge upgrade can break it.

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
