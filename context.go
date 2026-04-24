package forge

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/a-h/templ"
)

// Context holds the request/response state for a single HTTP request.
type Context struct {
	Request  *http.Request
	Response http.ResponseWriter
	Params   map[string]string
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
	return &Context{Request: r, Response: w, Params: p}
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
