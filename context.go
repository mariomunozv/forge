package forge

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/a-h/templ"
)

// Context holds the request/response state for a single HTTP request.
type Context struct {
	Request  *http.Request
	Response http.ResponseWriter
	Params   map[string]string
	Values   map[string]any // per-request store for middleware → handler communication
}

// NewContext creates a new Context. Useful for testing middleware directly.
func NewContext(w http.ResponseWriter, r *http.Request, params map[string]string) *Context {
	return newContext(w, r, params)
}

func newContext(w http.ResponseWriter, r *http.Request, params map[string]string) *Context {
	p := make(map[string]string, len(params))
	for k, v := range params {
		p[k] = v
	}
	return &Context{Request: r, Response: w, Params: p, Values: make(map[string]any)}
}

// Param returns a URL parameter by name (e.g. ":id" → ctx.Param("id")).
func (c *Context) Param(key string) string {
	return c.Params[key]
}

// Query returns a query string value by name.
func (c *Context) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

// WantsJSON returns true when the client expects a JSON response.
// Checks (in order): Accept header, Content-Type header, ?format=json query param.
func (c *Context) WantsJSON() bool {
	if c.Query("format") == "json" {
		return true
	}
	if strings.Contains(c.Request.Header.Get("Accept"), "application/json") {
		return true
	}
	if strings.Contains(c.Request.Header.Get("Content-Type"), "application/json") {
		return true
	}
	return false
}

// serveNotFound writes a styled 404 response — JSON for API clients, HTML for browsers.
func serveNotFound(ctx *Context) error {
	if ctx.WantsJSON() {
		return ctx.Error(http.StatusNotFound, "not found")
	}
	ctx.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	ctx.Response.WriteHeader(http.StatusNotFound)
	fmt.Fprint(ctx.Response, notFoundHTML)
	return nil
}

const notFoundHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
  <title>404 — Not Found</title>
</head>
<body style="box-sizing:border-box;margin:0;padding:0;background:#0f0f0f;color:#F0F0F0;font-family:ui-monospace,'SF Mono',Menlo,monospace;min-height:100vh;display:flex;align-items:center;justify-content:center">
  <div style="max-width:560px;width:100%;padding:32px">
    <div style="color:#E8FF00;font-size:11px;letter-spacing:4px;margin-bottom:24px;opacity:.7">// 404</div>
    <pre style="color:#E8FF00;font-size:11px;line-height:1.3;margin-bottom:32px;text-shadow:0 0 20px rgba(232,255,0,.3)">  ██╗  ██╗ ██████╗ ██╗  ██╗
  ██║  ██║██╔═══██╗██║  ██║
  ███████║██║   ██║███████║
  ╚════██║██║   ██║╚════██║
       ██║╚██████╔╝     ██║
       ╚═╝ ╚═════╝      ╚═╝</pre>
    <div style="border-top:1px solid #252525;padding-top:24px;margin-bottom:24px">
      <div style="font-size:20px;font-weight:700;color:#F0F0F0;margin-bottom:8px">Not Found<span style="color:#E8FF00">_</span></div>
      <div style="color:#888;font-size:13px;line-height:1.6">The page you&#39;re looking for doesn&#39;t exist.</div>
    </div>
    <div style="font-size:12px;color:#555">← go back</div>
  </div>
</body>
</html>`

// Component renders a templ component as an HTML response.
//
//	return ctx.Component(views.UsersIndex(users))
func (c *Context) Component(component templ.Component) error {
	c.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	return component.Render(c.Request.Context(), c.Response)
}

// Respond is the primary response method. It auto-negotiates between JSON and HTML:
//   - If the client wants JSON → returns {"data": ...} envelope
//   - Otherwise → renders the templ component
//
// Usage:
//
//	return ctx.Respond(users, views.UsersIndex(users))
func (c *Context) Respond(data any, component templ.Component) error {
	if c.WantsJSON() {
		return c.jsonData(http.StatusOK, data)
	}
	return c.Component(component)
}

// JSON writes a raw JSON response with full control over the body.
func (c *Context) JSON(status int, v any) error {
	c.Response.Header().Set("Content-Type", "application/json")
	c.Response.WriteHeader(status)
	return json.NewEncoder(c.Response).Encode(v)
}

// Success writes a 200 JSON envelope: {"data": v}.
//
//	return ctx.Success(user)
func (c *Context) Success(v any) error {
	return c.jsonData(http.StatusOK, v)
}

// Created writes a 201 JSON envelope: {"data": v}.
func (c *Context) Created(v any) error {
	return c.jsonData(http.StatusCreated, v)
}

// Error writes a JSON error envelope: {"error": {"message": ..., "code": ...}}.
//
//	return ctx.Error(http.StatusNotFound, "user not found")
func (c *Context) Error(status int, message string) error {
	return c.JSON(status, envelope{
		Error: &apiError{Message: message, Code: status},
	})
}

// Text writes a plain text response.
func (c *Context) Text(status int, body string) error {
	c.Response.Header().Set("Content-Type", "text/plain")
	c.Response.WriteHeader(status)
	_, err := c.Response.Write([]byte(body))
	return err
}

// Validate calls v.Validate() and writes a 422 error response if there are errors.
// Returns the error so the handler can return it directly.
//
//	if err := ctx.Validate(&post); err != nil {
//	    return err
//	}
func (c *Context) Validate(v interface{ Validate() []string }) error {
	if errs := v.Validate(); len(errs) > 0 {
		return c.Error(http.StatusUnprocessableEntity, strings.Join(errs, "; "))
	}
	return nil
}

// Status writes only a status code with no body.
func (c *Context) Status(code int) error {
	c.Response.WriteHeader(code)
	return nil
}

// Redirect sends an HTTP redirect.
func (c *Context) Redirect(status int, url string) error {
	http.Redirect(c.Response, c.Request, url, status)
	return nil
}

// Bind decodes the JSON request body into v.
func (c *Context) Bind(v any) error {
	defer c.Request.Body.Close()
	return json.NewDecoder(c.Request.Body).Decode(v)
}

// --- JSON envelope internals ---

type envelope struct {
	Data  any       `json:"data,omitempty"`
	Error *apiError `json:"error,omitempty"`
}

type apiError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (c *Context) jsonData(status int, v any) error {
	return c.JSON(status, envelope{Data: v})
}
