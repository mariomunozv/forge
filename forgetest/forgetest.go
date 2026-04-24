// Package forgetest provides testing helpers for Forge applications.
// It wraps net/http/httptest with a clean, readable API designed to be
// used by both humans and AI agents writing tests.
//
// Usage:
//
//	func TestUsersIndex(t *testing.T) {
//	    app := forgetest.New(t)
//	    app.Register("users", &UsersController{})
//	    app.GET("/users", "users#index")
//
//	    res := app.Request("GET", "/users").AsJSON().Do()
//
//	    res.AssertStatus(200)
//	    res.AssertJSONPath("data.users.0", "alice")
//	}
package forgetest

import (
	"net/http/httptest"
	"testing"

	"github.com/mariomunozv/forge/forge"
)

// TestApp wraps a forge.App for testing.
type TestApp struct {
	t   *testing.T
	app *forge.App
}

// New creates a TestApp ready to use in tests.
func New(t *testing.T) *TestApp {
	t.Helper()
	return &TestApp{t: t, app: forge.New()}
}

// Register maps a controller name to its instance.
func (ta *TestApp) Register(name string, c forge.Controller) *TestApp {
	ta.app.Register(name, c)
	return ta
}

// GET registers a GET route.
func (ta *TestApp) GET(path, handler string) *TestApp {
	ta.app.GET(path, handler)
	return ta
}

// POST registers a POST route.
func (ta *TestApp) POST(path, handler string) *TestApp {
	ta.app.POST(path, handler)
	return ta
}

// PUT registers a PUT route.
func (ta *TestApp) PUT(path, handler string) *TestApp {
	ta.app.PUT(path, handler)
	return ta
}

// PATCH registers a PATCH route.
func (ta *TestApp) PATCH(path, handler string) *TestApp {
	ta.app.PATCH(path, handler)
	return ta
}

// DELETE registers a DELETE route.
func (ta *TestApp) DELETE(path, handler string) *TestApp {
	ta.app.DELETE(path, handler)
	return ta
}

// Resources registers standard RESTful routes.
func (ta *TestApp) Resources(name string) *TestApp {
	ta.app.Resources(name)
	return ta
}

// Request starts building a test request.
func (ta *TestApp) Request(method, path string) *RequestBuilder {
	return newRequestBuilder(ta.t, ta.app, method, path)
}

// serve executes a built request and returns the recorded response.
func (ta *TestApp) serve(rb *RequestBuilder) *Response {
	ta.t.Helper()
	w := httptest.NewRecorder()
	ta.app.ServeHTTP(w, rb.build())
	return &Response{t: ta.t, recorder: w}
}
