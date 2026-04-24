package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mariomunozv/forge"
	"github.com/mariomunozv/forge/middleware"
)

func makeCtx(method, path string) (*forge.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, path, nil)
	ctx := forge.NewContext(w, r, nil)
	return ctx, w
}

// --- Logger ---

func TestLogger_PassesThrough(t *testing.T) {
	called := false
	handler := func(ctx *forge.Context) error {
		called = true
		return ctx.Text(http.StatusOK, "ok")
	}

	ctx, w := makeCtx("GET", "/test")
	wrapped := middleware.Logger()(handler)
	if err := wrapped(ctx); err != nil {
		t.Fatal(err)
	}

	if !called {
		t.Error("handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestLogger_CapturesStatus(t *testing.T) {
	handler := func(ctx *forge.Context) error {
		return ctx.Status(http.StatusCreated)
	}

	ctx, w := makeCtx("POST", "/items")
	wrapped := middleware.Logger()(handler)
	if err := wrapped(ctx); err != nil {
		t.Fatal(err)
	}

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

// --- Recovery ---

func TestRecovery_CatchesPanic(t *testing.T) {
	handler := func(ctx *forge.Context) error {
		panic("something went wrong")
	}

	ctx, w := makeCtx("GET", "/boom")
	wrapped := middleware.Recovery()(handler)
	if err := wrapped(ctx); err != nil {
		t.Fatalf("expected no error after recovery, got: %v", err)
	}

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestRecovery_PassesThroughNoPanic(t *testing.T) {
	handler := func(ctx *forge.Context) error {
		return ctx.Text(http.StatusOK, "fine")
	}

	ctx, w := makeCtx("GET", "/ok")
	wrapped := middleware.Recovery()(handler)
	if err := wrapped(ctx); err != nil {
		t.Fatal(err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- CORS ---

func TestCORS_SetsDefaultHeaders(t *testing.T) {
	handler := func(ctx *forge.Context) error {
		return ctx.Status(http.StatusOK)
	}

	ctx, w := makeCtx("GET", "/")
	wrapped := middleware.CORS(middleware.CORSConfig{})(handler)
	if err := wrapped(ctx); err != nil {
		t.Fatal(err)
	}

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected *, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("Access-Control-Allow-Methods not set")
	}
}

func TestCORS_PreflightReturns204(t *testing.T) {
	handler := func(ctx *forge.Context) error {
		t.Error("handler should not be called for preflight")
		return nil
	}

	ctx, w := makeCtx("OPTIONS", "/api/users")
	wrapped := middleware.CORS(middleware.CORSConfig{})(handler)
	if err := wrapped(ctx); err != nil {
		t.Fatal(err)
	}

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestCORS_CustomOrigin(t *testing.T) {
	handler := func(ctx *forge.Context) error {
		return ctx.Status(http.StatusOK)
	}

	ctx, w := makeCtx("GET", "/")
	wrapped := middleware.CORS(middleware.CORSConfig{
		AllowOrigins: []string{"https://example.com"},
	})(handler)
	if err := wrapped(ctx); err != nil {
		t.Fatal(err)
	}

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("expected https://example.com, got %q", got)
	}
}
