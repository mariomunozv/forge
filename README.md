# Forge

> Go web framework designed for AI-assisted development. Convention over configuration, generator-first.

Forge is a Rails-inspired Go web framework built for the way people actually build software today — fast iterations, AI agents writing scaffolding, humans focusing on business logic. It gives you a strong set of conventions, a powerful CLI, and a codebase that's easy to reason about whether you're a developer or an AI agent.

```bash
forge new myapp
cd myapp
forge server
```

---

## Why Forge

Most Go frameworks give you a router and get out of the way. That's fine — but it means every project reinvents the same structure, the same generators, the same migration conventions. Forge has opinions:

- **Rails-style routing** — `"users#index"` maps to `UsersController.Index`, always
- **Generator-first** — `forge g resource Post title:string body:text` scaffolds model + controller + views in one command
- **Agent-friendly** — conventions so predictable that an AI agent can navigate and extend your codebase without reading every file
- **templ components** — type-safe, slot-aware HTML components that catch errors at compile time

---

## Install

```bash
# Install Go 1.22+
brew install go

# Install templ
go install github.com/a-h/templ/cmd/templ@latest

# Install air (hot reload)
go install github.com/air-verse/air@latest

# Install Forge CLI
go install github.com/mariomunozv/forge/cmd/forge@latest
```

---

## Quick Start

```bash
forge new blog
cd blog
go mod tidy
forge server
```

`forge server` starts the app, watches `.templ` files and reloads on changes.

---

## Routing

```go
app := forge.New()
app.Register("posts", &controllers.PostsController{})

// individual routes
app.GET("/posts",     "posts#index")
app.GET("/posts/:id", "posts#show")
app.POST("/posts",    "posts#create")

// or all 5 REST routes at once
app.Resources("posts")
```

Routes use `"controller#action"` strings. The controller is resolved by name from the registry, the action by reflection. No magic, no decorators.

---

## Controllers

```go
type PostsController struct{}

func (c *PostsController) Index(ctx *forge.Context) error {
    posts, _ := db.QueryAll[Post](database, "SELECT * FROM posts ORDER BY created_at DESC")
    return ctx.Respond(posts, views.PostsIndex(posts))  // JSON or HTML based on Accept header
}

func (c *PostsController) Show(ctx *forge.Context) error {
    id := ctx.Param("id")
    post, err := db.QueryOne[Post](database, "SELECT * FROM posts WHERE id = $1", id)
    if err != nil {
        return ctx.Error(http.StatusNotFound, "post not found")
    }
    return ctx.Respond(post, views.PostsShow(post))
}

func (c *PostsController) Create(ctx *forge.Context) error {
    var input struct {
        Title string `json:"title"`
        Body  string `json:"body"`
    }
    if err := ctx.Bind(&input); err != nil {
        return ctx.Error(http.StatusBadRequest, "invalid request body")
    }
    // ...
    return ctx.Created(post)
}
```

### Context API

```go
ctx.Param("id")                   // URL param
ctx.Query("page")                 // query string
ctx.Bind(&input)                  // decode JSON body

ctx.Respond(data, component)      // auto-negotiate JSON vs HTML ← primary method
ctx.Component(views.Index(data))  // explicit templ render
ctx.Success(v)                    // 200 {"data": v}
ctx.Created(v)                    // 201 {"data": v}
ctx.Error(status, message)        // {"error": {"message": "...", "code": 422}}
ctx.JSON(status, v)               // raw JSON
ctx.Text(status, body)            // plain text
ctx.Redirect(status, url)         // HTTP redirect
ctx.Status(code)                  // status only, no body
ctx.Validate(model)               // writes 422 if model.Validate() returns errors

ctx.SignIn(userID)                 // set session cookie
ctx.SignOut()                      // clear session cookie
ctx.CurrentUserID()                // (int64, bool)
ctx.WantsJSON()                    // true if client expects JSON
```

### JSON responses

All JSON responses use a consistent envelope:

```json
{ "data": { "id": 1, "title": "Hello" } }
{ "error": { "message": "not found", "code": 404 } }
```

---

## Templates (templ)

Forge uses [`templ`](https://templ.guide) for type-safe, component-based HTML. Errors are caught at compile time. Slots work like modern JS frameworks.

```go
// views/layouts/application.templ
templ Application(title string) {
    <!DOCTYPE html>
    <html>
        <head><title>{ title }</title></head>
        <body>{ children... }</body>
    </html>
}

// views/posts/index.templ
templ Index(posts []models.Post) {
    @layouts.Application("Posts") {
        for _, post := range posts {
            @components.Card(components.CardProps{Title: post.Title}) {
                <p>{ post.Body }</p>
                @components.Button(components.ButtonProps{
                    Label: "Read more",
                    Href:  "/posts/" + strconv.Itoa(post.ID),
                })
            }
        }
    }
}
```

`forge server` runs `templ generate --watch` automatically.

---

## Database

Forge ships a lightweight `db` package for PostgreSQL. No ORM — just helpers that cut the boilerplate while keeping SQL visible.

```go
import "github.com/mariomunozv/forge/db"

// connect
database, err := db.Open() // reads DATABASE_URL

// query
post, err := db.QueryOne[Post](database, "SELECT * FROM posts WHERE id = $1", id)
posts, err := db.QueryAll[Post](database, "SELECT * FROM posts ORDER BY created_at DESC")

// exec
err := db.Insert(database, "posts", map[string]any{
    "title": "Hello",
    "body":  "World",
})
err := db.Update(database, "posts", map[string]any{"title": "Updated"}, "id = $1", id)
err := db.Delete(database, "posts", "id = $1", id)
```

Struct fields map to columns via `db:` tags:

```go
type Post struct {
    ID    int    `db:"id"    json:"id"`
    Title string `db:"title" json:"title"`
    Body  string `db:"body"  json:"body"`
}
```

### Migrations

```bash
forge g migration create_posts        # generates db/migrations/20260424120000_create_posts.sql
forge db migrate                      # run pending migrations
forge db rollback                     # roll back last migration
forge db status                       # show migration status
forge db create                       # create database from DATABASE_URL
forge db drop                         # drop database
```

Each migration file has `-- up` and `-- down` sections:

```sql
-- up
CREATE TABLE posts (
    id         SERIAL PRIMARY KEY,
    title      TEXT NOT NULL,
    body       TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- down
DROP TABLE posts;
```

---

## Middleware

```go
app.Use(middleware.Logger())         // method, path, status, duration
app.Use(middleware.Recovery())       // catch panics → 500
app.Use(middleware.CORS())           // configurable CORS headers
app.Use(middleware.Auth())           // reads session, sets current_user_id
app.Use(middleware.DevErrors())      // rich HTML error page with stack trace (dev only)
```

Protect specific routes:

```go
app.Use(middleware.RequireAuth())    // returns 401 for unauthenticated requests
```

---

## Sessions & Auth

```go
// sign in
ctx.SignIn(user.ID)

// sign out
ctx.SignOut()

// read current user
userID, ok := ctx.CurrentUserID()
```

Sessions use a signed cookie (`_forge_session`) with HMAC-SHA256. Set `SESSION_SECRET` in your environment.

Generate a full auth scaffold:

```bash
forge g auth   # User model + login/logout controller + session middleware wiring
```

---

## Background Jobs

Jobs are stored in PostgreSQL — no Redis required.

```go
// define a job
type WelcomeEmailJob struct {
    UserID int64 `json:"user_id"`
}

func (j *WelcomeEmailJob) Perform(ctx context.Context) error {
    // send email...
    return nil
}

func init() {
    jobs.Register("WelcomeEmailJob", func() jobs.Job { return &WelcomeEmailJob{} })
}

// enqueue from a controller
jobs.Enqueue(database, "WelcomeEmailJob", WelcomeEmailJob{UserID: user.ID})
```

```bash
forge g job WelcomeEmail    # generates jobs/welcome_email_job.go
forge jobs work             # start worker process
```

---

## Testing

The `forgetest` package gives you a fluent API designed to be easy to write by hand or by an AI agent:

```go
func TestPostsIndex(t *testing.T) {
    app := forgetest.New(t)
    app.Register("posts", &PostsController{})
    app.Resources("posts")

    app.Request("GET", "/posts").
        AsJSON().
        Do().
        AssertOK().
        AssertJSONPath("data.0.title", "Hello Forge").
        AssertJSONPathExists("data.0.id")
}

func TestCreatePost(t *testing.T) {
    app.Request("POST", "/posts").
        WithBody(forge.M{"title": "New Post", "body": "Content"}).
        Do().
        AssertCreated().
        AssertJSONPath("data.title", "New Post")
}
```

```bash
go test ./...
```

---

## CLI Reference

```bash
# App
forge new myapp                           # scaffold a new app

# Development
forge server                              # start dev server with hot reload
forge s                                   # alias

# Introspection
forge routes                              # list all routes
forge routes --json                       # routes as JSON
forge describe                            # full app structure as JSON
forge describe --md                       # generate FORGE.md (great for AI context)

# Generators
forge g resource Post title:string body:text published:bool
forge g controller posts index show create update destroy
forge g model Post title:string body:text
forge g view posts index show
forge g migration create_posts
forge g auth
forge g job WelcomeEmail
forge g r   # alias for resource
forge g c   # alias for controller
forge g m   # alias for model
forge g v   # alias for view

# Database
forge db create
forge db drop
forge db migrate
forge db rollback
forge db status

# Jobs
forge jobs work
```

Field types: `string`, `int`, `int64`, `float`, `bool`, `time`

---

## Agent-Friendly Design

Forge is built to work well with AI agents. Several features exist specifically for this:

**`forge describe`** dumps the full app structure as JSON — routes, controllers, models, views. Feed it directly to an agent as context:

```bash
forge describe | pbcopy    # copy to clipboard, paste into your AI context
forge describe --md        # generates FORGE.md — a self-documenting project map
```

**Predictable conventions** mean an agent that knows the framework can navigate any Forge app without reading every file. If it knows there's a `PostsController`, it knows the file is `app/controllers/posts_controller.go` and the actions are `Index`, `Show`, `Create`, `Update`, `Destroy`.

**Generator-first** means agents scaffold instead of write from scratch, reducing hallucination surface area.

---

## Project Status

Forge is under active development. The core is stable and usable. Things that work:

- [x] Router, controllers, middleware
- [x] Content negotiation (JSON / HTML)
- [x] templ integration with layouts and components
- [x] Database layer (PostgreSQL)
- [x] Migrations
- [x] Sessions and auth
- [x] Background jobs (PostgreSQL-backed)
- [x] CLI generators (resource, controller, model, view, migration, auth, job)
- [x] `forge describe` for AI introspection
- [x] `forgetest` testing package

Coming next:

- [ ] `forge console` — interactive REPL with app context
- [ ] `forge deploy` — production checklist and Dockerfile generation

---

## License

MIT
