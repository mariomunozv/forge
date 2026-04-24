package forge

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type UsersController struct{}

func (c *UsersController) Index(ctx *Context) error {
	return ctx.JSON(http.StatusOK, M{"users": []string{"alice", "bob"}})
}

func (c *UsersController) Show(ctx *Context) error {
	return ctx.JSON(http.StatusOK, M{"id": ctx.Param("id")})
}

func TestRouterGET(t *testing.T) {
	app := New()
	app.Register("users", &UsersController{})
	app.GET("/users", "users#index")

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	app.buildHandler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRouterParams(t *testing.T) {
	app := New()
	app.Register("users", &UsersController{})
	app.GET("/users/:id", "users#show")

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	w := httptest.NewRecorder()
	app.buildHandler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if body := w.Body.String(); body != `{"id":"42"}`+"\n" {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestRouterResources(t *testing.T) {
	app := New()
	app.Register("users", &UsersController{})
	app.Resources("users")

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/users"},
		{http.MethodGet, "/users/1"},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		app.buildHandler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("%s %s: expected 200, got %d", tc.method, tc.path, w.Code)
		}
	}
}

func TestRouterNotFound(t *testing.T) {
	app := New()

	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	w := httptest.NewRecorder()
	app.buildHandler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUnregisteredController(t *testing.T) {
	app := New()
	app.GET("/users", "users#index") // registered route but no controller

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	app.buildHandler().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
