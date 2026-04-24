package forgetest

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

// Response wraps httptest.ResponseRecorder with assertion helpers.
type Response struct {
	t        *testing.T
	recorder *httptest.ResponseRecorder
}

// Status returns the HTTP status code.
func (r *Response) Status() int {
	return r.recorder.Code
}

// Body returns the raw response body as a string.
func (r *Response) Body() string {
	return r.recorder.Body.String()
}

// Header returns the value of a response header.
func (r *Response) Header(key string) string {
	return r.recorder.Header().Get(key)
}

// AssertStatus fails the test if the status code does not match.
func (r *Response) AssertStatus(expected int) *Response {
	r.t.Helper()
	if r.recorder.Code != expected {
		r.t.Errorf("expected status %d, got %d\nbody: %s", expected, r.recorder.Code, r.Body())
	}
	return r
}

// AssertOK asserts status 200.
func (r *Response) AssertOK() *Response {
	r.t.Helper()
	return r.AssertStatus(200)
}

// AssertCreated asserts status 201.
func (r *Response) AssertCreated() *Response {
	r.t.Helper()
	return r.AssertStatus(201)
}

// AssertNotFound asserts status 404.
func (r *Response) AssertNotFound() *Response {
	r.t.Helper()
	return r.AssertStatus(404)
}

// AssertHeader fails the test if the header value does not match.
func (r *Response) AssertHeader(key, expected string) *Response {
	r.t.Helper()
	got := r.recorder.Header().Get(key)
	if got != expected {
		r.t.Errorf("header %q: expected %q, got %q", key, expected, got)
	}
	return r
}

// AssertBodyContains fails the test if the body does not contain the given string.
func (r *Response) AssertBodyContains(substr string) *Response {
	r.t.Helper()
	if !strings.Contains(r.Body(), substr) {
		r.t.Errorf("expected body to contain %q\nbody: %s", substr, r.Body())
	}
	return r
}

// AssertJSON fails the test if the response body does not match the expected JSON.
// Expected can be a string, number, bool, map, or slice.
//
//	res.AssertJSON(forge.M{"data": forge.M{"name": "alice"}})
func (r *Response) AssertJSON(expected any) *Response {
	r.t.Helper()

	expectedBytes, err := json.Marshal(expected)
	if err != nil {
		r.t.Fatalf("forgetest: failed to marshal expected JSON: %v", err)
	}

	var got, want any
	if err := json.Unmarshal([]byte(r.Body()), &got); err != nil {
		r.t.Fatalf("forgetest: response is not valid JSON: %v\nbody: %s", err, r.Body())
	}
	if err := json.Unmarshal(expectedBytes, &want); err != nil {
		r.t.Fatalf("forgetest: expected value is not valid JSON: %v", err)
	}

	gotStr := mustMarshal(got)
	wantStr := mustMarshal(want)
	if gotStr != wantStr {
		r.t.Errorf("JSON mismatch:\n  want: %s\n   got: %s", wantStr, gotStr)
	}
	return r
}

// AssertJSONPath fails the test if the value at the JSON path does not match.
// Uses gjson path syntax: "data.users.0.name", "data.total", etc.
//
//	res.AssertJSONPath("data.name", "alice")
//	res.AssertJSONPath("error.code", 404)
func (r *Response) AssertJSONPath(path string, expected any) *Response {
	r.t.Helper()

	result := gjson.Get(r.Body(), path)
	if !result.Exists() {
		r.t.Errorf("JSON path %q not found in body: %s", path, r.Body())
		return r
	}

	got := result.Value()
	wantStr := fmt.Sprintf("%v", expected)
	gotStr := fmt.Sprintf("%v", got)

	// gjson returns float64 for numbers — normalize int comparisons
	if gotStr != wantStr {
		r.t.Errorf("JSON path %q:\n  want: %v\n   got: %v", path, expected, got)
	}
	return r
}

// AssertJSONPathExists fails the test if the JSON path does not exist.
func (r *Response) AssertJSONPathExists(path string) *Response {
	r.t.Helper()
	if !gjson.Get(r.Body(), path).Exists() {
		r.t.Errorf("JSON path %q not found in body: %s", path, r.Body())
	}
	return r
}

func mustMarshal(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
