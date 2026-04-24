package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/mariomunozv/forge"
)

// Logger logs method, path, status code, and duration for each request.
func Logger() forge.MiddlewareFunc {
	return func(next forge.HandlerFunc) forge.HandlerFunc {
		return func(ctx *forge.Context) error {
			start := time.Now()

			rw := &captureWriter{ResponseWriter: ctx.Response, status: http.StatusOK}
			ctx.Response = rw

			err := next(ctx)

			log.Printf("%s %s %d %s", ctx.Request.Method, ctx.Request.URL.Path, rw.status, time.Since(start))
			return err
		}
	}
}

// captureWriter wraps http.ResponseWriter to capture the written status code.
type captureWriter struct {
	http.ResponseWriter
	status int
}

func (cw *captureWriter) WriteHeader(status int) {
	cw.status = status
	cw.ResponseWriter.WriteHeader(status)
}
