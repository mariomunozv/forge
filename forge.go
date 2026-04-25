package forge

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"
)

// M is a shorthand for template/response data maps.
type M map[string]any

// App is the main Forge application.
type App struct {
	router     *Router
	middleware []MiddlewareFunc
}

// New creates a new Forge application.
func New() *App {
	return &App{
		router: newRouter(),
	}
}

// Use registers global middleware.
func (a *App) Use(mw MiddlewareFunc) {
	a.middleware = append(a.middleware, mw)
}

// GET registers a GET route.
func (a *App) GET(path, handler string) {
	a.router.add(http.MethodGet, path, handler)
}

// POST registers a POST route.
func (a *App) POST(path, handler string) {
	a.router.add(http.MethodPost, path, handler)
}

// PUT registers a PUT route.
func (a *App) PUT(path, handler string) {
	a.router.add(http.MethodPut, path, handler)
}

// PATCH registers a PATCH route.
func (a *App) PATCH(path, handler string) {
	a.router.add(http.MethodPatch, path, handler)
}

// DELETE registers a DELETE route.
func (a *App) DELETE(path, handler string) {
	a.router.add(http.MethodDelete, path, handler)
}

// Register maps a controller name to its instance.
// Usage: app.Register("home", &HomeController{})
func (a *App) Register(name string, c Controller) {
	a.router.registry.set(name, c)
}

// Resources registers the 7 standard RESTful routes for a controller.
//
//	GET    /users          → users#index
//	GET    /users/new      → users#new
//	POST   /users          → users#create
//	GET    /users/:id      → users#show
//	GET    /users/:id/edit → users#edit
//	PUT    /users/:id      → users#update
//	DELETE /users/:id      → users#destroy
//
// /new and /edit are registered before /:id so exact segments take priority.
func (a *App) Resources(name string) {
	a.GET("/"+name, name+"#index")
	a.GET("/"+name+"/new", name+"#new")
	a.POST("/"+name, name+"#create")
	a.GET("/"+name+"/:id", name+"#show")
	a.GET("/"+name+"/:id/edit", name+"#edit")
	a.PUT("/"+name+"/:id", name+"#update")
	a.DELETE("/"+name+"/:id", name+"#destroy")
}

// Start runs the HTTP server on the given address.
// If the FORGE_CMD environment variable is set, it runs that command and exits
// instead of starting the server. This is how `forge routes` works.
func (a *App) Start(addr string) error {
	switch os.Getenv("FORGE_CMD") {
	case "routes":
		a.printRoutes(false)
		os.Exit(0)
	case "routes:json":
		a.printRoutes(true)
		os.Exit(0)
	}

	fmt.Printf("=> Forge listening on http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, a.buildHandler()); err != nil {
		fmt.Fprintf(os.Stderr, "\nforge: server error: %v\n", err)
		os.Exit(1)
	}
	return nil
}

// printRoutes prints all registered routes to stdout.
func (a *App) printRoutes(asJSON bool) {
	routes := a.router.Routes()

	if asJSON {
		json.NewEncoder(os.Stdout).Encode(routes)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "METHOD\tPATH\tCONTROLLER\tACTION")
	fmt.Fprintln(w, "------\t----\t----------\t------")
	for _, r := range routes {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Method, r.Path, r.Controller, r.Action)
	}
	w.Flush()
}

// ServeHTTP implements http.Handler, making App usable directly in tests.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.serve(w, r, a.middleware)
}

func (a *App) buildHandler() http.Handler {
	return http.HandlerFunc(a.ServeHTTP)
}
