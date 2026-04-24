package middleware

import (
	"net/http"
	"strings"

	"github.com/mariomunozv/forge"
)

// CORSConfig holds CORS configuration.
type CORSConfig struct {
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string
}

// CORS adds Cross-Origin Resource Sharing headers to every response.
// Preflight OPTIONS requests are answered with 204 and no further processing.
//
// Usage with defaults:
//
//	app.Use(middleware.CORS(middleware.CORSConfig{}))
//
// Usage with explicit config:
//
//	app.Use(middleware.CORS(middleware.CORSConfig{
//	    AllowOrigins: []string{"https://example.com"},
//	    AllowMethods: []string{"GET", "POST"},
//	    AllowHeaders: []string{"Content-Type", "Authorization"},
//	}))
func CORS(cfg CORSConfig) forge.MiddlewareFunc {
	origins := join(cfg.AllowOrigins, "*")
	methods := join(cfg.AllowMethods, "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	headers := join(cfg.AllowHeaders, "Content-Type, Authorization")

	return func(next forge.HandlerFunc) forge.HandlerFunc {
		return func(ctx *forge.Context) error {
			h := ctx.Response.Header()
			h.Set("Access-Control-Allow-Origin", origins)
			h.Set("Access-Control-Allow-Methods", methods)
			h.Set("Access-Control-Allow-Headers", headers)

			if ctx.Request.Method == http.MethodOptions {
				ctx.Response.WriteHeader(http.StatusNoContent)
				return nil
			}

			return next(ctx)
		}
	}
}

func join(values []string, fallback string) string {
	if len(values) == 0 {
		return fallback
	}
	return strings.Join(values, ", ")
}
