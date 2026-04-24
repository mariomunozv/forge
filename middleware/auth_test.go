package middleware_test

import (
	"net/http"
	"testing"

	"github.com/mariomunozv/forge"
	"github.com/mariomunozv/forge/middleware"
)

func TestAuth_SetsCurrentUserID(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret")

	// Build a context that already has a session cookie
	ctx, w := makeCtx("GET", "/dashboard")
	ctx.SignIn(55)
	resp := w.Result()

	ctx2, _ := makeCtx("GET", "/dashboard")
	for _, c := range resp.Cookies() {
		ctx2.Request.AddCookie(c)
	}

	handler := func(ctx *forge.Context) error {
		id, ok := ctx.Values["current_user_id"]
		if !ok {
			t.Error("expected current_user_id in Values")
		}
		if id.(int64) != 55 {
			t.Errorf("expected 55, got %v", id)
		}
		return ctx.Status(http.StatusOK)
	}

	if err := middleware.Auth()(handler)(ctx2); err != nil {
		t.Fatal(err)
	}
}

func TestAuth_PassesThroughWithoutCookie(t *testing.T) {
	called := false
	handler := func(ctx *forge.Context) error {
		called = true
		_, hasID := ctx.Values["current_user_id"]
		if hasID {
			t.Error("should not have current_user_id without cookie")
		}
		return ctx.Status(http.StatusOK)
	}

	ctx, _ := makeCtx("GET", "/")
	if err := middleware.Auth()(handler)(ctx); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("handler should have been called")
	}
}

func TestRequireAuth_Blocks(t *testing.T) {
	handler := func(ctx *forge.Context) error {
		return ctx.Status(http.StatusOK)
	}

	ctx, w := makeCtx("GET", "/protected")
	if err := middleware.RequireAuth()(handler)(ctx); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRequireAuth_PassesWithSession(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret")

	signIn, w := makeCtx("POST", "/login")
	signIn.SignIn(12)
	cookies := w.Result().Cookies()

	ctx, w2 := makeCtx("GET", "/protected")
	for _, c := range cookies {
		ctx.Request.AddCookie(c)
	}

	handler := func(ctx *forge.Context) error {
		return ctx.Status(http.StatusOK)
	}

	if err := middleware.RequireAuth()(handler)(ctx); err != nil {
		t.Fatal(err)
	}
	if w2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w2.Code)
	}
}
