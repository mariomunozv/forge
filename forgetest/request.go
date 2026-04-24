package forgetest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/mariomunozv/forge/forge"
)

// RequestBuilder builds and executes a test HTTP request.
type RequestBuilder struct {
	t       *testing.T
	app     *forge.App
	method  string
	path    string
	headers map[string]string
	params  url.Values
	body    []byte
}

func newRequestBuilder(t *testing.T, app *forge.App, method, path string) *RequestBuilder {
	t.Helper()
	return &RequestBuilder{
		t:       t,
		app:     app,
		method:  method,
		path:    path,
		headers: make(map[string]string),
		params:  make(url.Values),
	}
}

// AsJSON sets Accept and Content-Type headers to application/json.
// This makes ctx.WantsJSON() return true in the handler.
func (rb *RequestBuilder) AsJSON() *RequestBuilder {
	rb.headers["Accept"] = "application/json"
	rb.headers["Content-Type"] = "application/json"
	return rb
}

// AsHTML sets the Accept header to text/html.
func (rb *RequestBuilder) AsHTML() *RequestBuilder {
	rb.headers["Accept"] = "text/html"
	return rb
}

// WithHeader adds a request header.
func (rb *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	rb.headers[key] = value
	return rb
}

// WithBody serializes v as JSON and sets it as the request body.
func (rb *RequestBuilder) WithBody(v any) *RequestBuilder {
	rb.t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		rb.t.Fatalf("forgetest: failed to marshal request body: %v", err)
	}
	rb.body = data
	rb.headers["Content-Type"] = "application/json"
	return rb
}

// WithParam adds a query string parameter.
func (rb *RequestBuilder) WithParam(key, value string) *RequestBuilder {
	rb.params.Set(key, value)
	return rb
}

// Do executes the request and returns the Response.
func (rb *RequestBuilder) Do() *Response {
	rb.t.Helper()
	w := httptest.NewRecorder()
	rb.app.ServeHTTP(w, rb.build())
	return &Response{t: rb.t, recorder: w}
}

func (rb *RequestBuilder) build() *http.Request {
	rb.t.Helper()

	path := rb.path
	if len(rb.params) > 0 {
		path += "?" + rb.params.Encode()
	}

	var bodyReader *bytes.Reader
	if rb.body != nil {
		bodyReader = bytes.NewReader(rb.body)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(rb.method, path, bodyReader)
	for k, v := range rb.headers {
		req.Header.Set(k, v)
	}
	return req
}
