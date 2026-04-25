package forge

import (
	"testing"
)

func TestRoutesInfo(t *testing.T) {
	app := New()
	app.Register("users", &UsersController{})
	app.Resources("users")
	app.GET("/", "home#index")

	routes := app.router.Routes()

	// Resources genera 7 rutas + 1 manual = 8
	if len(routes) != 8 {
		t.Fatalf("expected 8 routes, got %d", len(routes))
	}

	cases := []struct {
		method     string
		path       string
		controller string
		action     string
	}{
		{"GET", "/users", "users", "index"},
		{"GET", "/users/new", "users", "new"},
		{"POST", "/users", "users", "create"},
		{"GET", "/users/:id", "users", "show"},
		{"GET", "/users/:id/edit", "users", "edit"},
		{"PUT", "/users/:id", "users", "update"},
		{"DELETE", "/users/:id", "users", "destroy"},
		{"GET", "/", "home", "index"},
	}

	for i, tc := range cases {
		r := routes[i]
		if r.Method != tc.method || r.Path != tc.path || r.Controller != tc.controller || r.Action != tc.action {
			t.Errorf("route[%d]: got {%s %s %s#%s}, want {%s %s %s#%s}",
				i, r.Method, r.Path, r.Controller, r.Action,
				tc.method, tc.path, tc.controller, tc.action)
		}
	}
}
