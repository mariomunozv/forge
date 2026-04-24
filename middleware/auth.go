package middleware

import (
	"net/http"

	"github.com/mariomunozv/forge"
)

// Auth reads the session cookie and, if valid, sets ctx.Values["current_user_id"].
// Does not block unauthenticated requests — use RequireAuth for that.
func Auth() forge.MiddlewareFunc {
	return func(next forge.HandlerFunc) forge.HandlerFunc {
		return func(ctx *forge.Context) error {
			if id, ok := ctx.CurrentUserID(); ok {
				ctx.Values["current_user_id"] = id
			}
			return next(ctx)
		}
	}
}

// RequireAuth blocks unauthenticated requests with 401.
func RequireAuth() forge.MiddlewareFunc {
	return func(next forge.HandlerFunc) forge.HandlerFunc {
		return func(ctx *forge.Context) error {
			if _, ok := ctx.CurrentUserID(); !ok {
				return ctx.Error(http.StatusUnauthorized, "authentication required")
			}
			return next(ctx)
		}
	}
}
