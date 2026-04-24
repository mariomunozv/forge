package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/mariomunozv/forge"
)

// Recovery catches panics, logs the stack trace, and returns a 500 response.
func Recovery() forge.MiddlewareFunc {
	return func(next forge.HandlerFunc) forge.HandlerFunc {
		return func(ctx *forge.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic: %v\n%s", r, debug.Stack())
					http.Error(ctx.Response, "Internal Server Error", http.StatusInternalServerError)
					err = nil
				}
			}()
			return next(ctx)
		}
	}
}
