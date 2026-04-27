package middleware_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/mariomunozv/forge"
	"github.com/mariomunozv/forge/middleware"
)

func TestDevErrors_CatchesPanic(t *testing.T) {
	handler := func(ctx *forge.Context) error {
		panic("something exploded")
	}

	ctx, w := makeCtx("GET", "/boom")
	wrapped := middleware.DevErrors()(handler)
	if err := wrapped(ctx); err != nil {
		t.Fatalf("expected no error after recovery, got: %v", err)
	}

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "something exploded") {
		t.Error("expected error message in HTML body")
	}
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("expected HTML response")
	}
	if !strings.Contains(body, "GET") {
		t.Error("expected request method in HTML body")
	}
}

func TestDevErrors_CatchesHandlerError(t *testing.T) {
	handler := func(ctx *forge.Context) error {
		return ctx.Error(http.StatusUnprocessableEntity, "validation failed")
	}

	ctx, w := makeCtx("POST", "/users")
	wrapped := middleware.DevErrors()(handler)
	if err := wrapped(ctx); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// 4xx errors pass through normally — DevErrors only intercepts 5xx.
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

func TestDevErrors_Intercepts5xx(t *testing.T) {
	handler := func(ctx *forge.Context) error {
		return ctx.Error(http.StatusInternalServerError, "database is down")
	}

	ctx, w := makeCtx("GET", "/crash")
	wrapped := middleware.DevErrors()(handler)
	if err := wrapped(ctx); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "database is down") {
		t.Errorf("expected error message in dev error page, got: %s", body)
	}
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("expected HTML dev error page")
	}
}

func TestDevErrors_PassesThroughSuccess(t *testing.T) {
	handler := func(ctx *forge.Context) error {
		return ctx.Text(http.StatusOK, "all good")
	}

	ctx, w := makeCtx("GET", "/ok")
	wrapped := middleware.DevErrors()(handler)
	if err := wrapped(ctx); err != nil {
		t.Fatal(err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "all good" {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}
