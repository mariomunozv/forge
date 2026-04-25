package middleware

import (
	"net/http"

	"github.com/mariomunozv/forge"
)

// MethodOverride reads the _method form field on POST requests and overrides
// the HTTP method to support PUT/DELETE from HTML forms.
// Add a hidden input: <input type="hidden" name="_method" value="DELETE"/>
func MethodOverride() forge.MiddlewareFunc {
	return func(next forge.HandlerFunc) forge.HandlerFunc {
		return func(ctx *forge.Context) error {
			if ctx.Request.Method == http.MethodPost {
				m := ctx.Request.FormValue("_method")
				if m == http.MethodPut || m == http.MethodPatch || m == http.MethodDelete {
					ctx.Request.Method = m
				}
			}
			return next(ctx)
		}
	}
}
