package forge

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func newTestContext(method, path string, headers map[string]string) (*Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	return newContext(w, req, nil), w
}

// stubComponent is a minimal templ.Component for tests.
func stubComponent(html string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, html)
		return err
	})
}

func TestWantsJSON_AcceptHeader(t *testing.T) {
	ctx, _ := newTestContext("GET", "/users", map[string]string{
		"Accept": "application/json",
	})
	if !ctx.WantsJSON() {
		t.Fatal("expected WantsJSON() = true for Accept: application/json")
	}
}

func TestWantsJSON_ContentType(t *testing.T) {
	ctx, _ := newTestContext("POST", "/users", map[string]string{
		"Content-Type": "application/json",
	})
	if !ctx.WantsJSON() {
		t.Fatal("expected WantsJSON() = true for Content-Type: application/json")
	}
}

func TestWantsJSON_FormatParam(t *testing.T) {
	ctx, _ := newTestContext("GET", "/users?format=json", nil)
	if !ctx.WantsJSON() {
		t.Fatal("expected WantsJSON() = true for ?format=json")
	}
}

func TestWantsJSON_False(t *testing.T) {
	ctx, _ := newTestContext("GET", "/users", map[string]string{
		"Accept": "text/html",
	})
	if ctx.WantsJSON() {
		t.Fatal("expected WantsJSON() = false for Accept: text/html")
	}
}

func TestSuccess_Envelope(t *testing.T) {
	ctx, w := newTestContext("GET", "/", nil)
	ctx.Success(M{"name": "alice"})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"data"`) {
		t.Fatalf("expected envelope with 'data' key, got: %s", body)
	}
	if !strings.Contains(body, `"alice"`) {
		t.Fatalf("expected 'alice' in body, got: %s", body)
	}
}

func TestCreated_Envelope(t *testing.T) {
	ctx, w := newTestContext("POST", "/", nil)
	ctx.Created(M{"id": 1})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"data"`) {
		t.Fatal("expected envelope with 'data' key")
	}
}

func TestError_Envelope(t *testing.T) {
	ctx, w := newTestContext("GET", "/", map[string]string{
		"Accept": "application/json",
	})
	ctx.Error(http.StatusNotFound, "user not found")

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"error"`) {
		t.Fatalf("expected envelope with 'error' key, got: %s", body)
	}
	if !strings.Contains(body, `"user not found"`) {
		t.Fatalf("expected message in body, got: %s", body)
	}
}

func TestError_HTML(t *testing.T) {
	ctx, w := newTestContext("GET", "/", map[string]string{
		"Accept": "text/html",
	})
	ctx.Error(http.StatusNotFound, "user not found")

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Not Found") {
		t.Fatalf("expected HTML error page, got: %s", body)
	}
	if !strings.Contains(body, "user not found") {
		t.Fatalf("expected message in HTML body, got: %s", body)
	}
}

func TestComponent(t *testing.T) {
	ctx, w := newTestContext("GET", "/", map[string]string{
		"Accept": "text/html",
	})
	ctx.Component(stubComponent("<h1>Hello Forge</h1>"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Hello Forge") {
		t.Fatalf("expected component output in body, got: %s", w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("expected text/html Content-Type, got: %s", ct)
	}
}

func TestRespond_JSON(t *testing.T) {
	ctx, w := newTestContext("GET", "/users", map[string]string{
		"Accept": "application/json",
	})
	ctx.Respond(M{"users": []string{"alice"}}, stubComponent("<p>ignored</p>"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"data"`) {
		t.Fatal("expected JSON envelope")
	}
}

func TestRespond_HTML(t *testing.T) {
	ctx, w := newTestContext("GET", "/users", map[string]string{
		"Accept": "text/html",
	})
	ctx.Respond(M{"users": []string{"alice"}}, stubComponent("<ul><li>alice</li></ul>"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "alice") {
		t.Fatalf("expected HTML in body, got: %s", w.Body.String())
	}
}
