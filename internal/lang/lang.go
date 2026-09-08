// Package lang answers which language a request is served in.
package lang

import (
	"context"
	"net/http"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/respheader"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

// Middleware belongs inside the session resolver, since a member's own choice
// outranks everything the request carries.
func Middleware(bundle *i18n.Bundle) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			serve := bundle.Negotiate(chosen(ctx), r.Header.Get(i18n.AcceptHeader), siteDefault(ctx))
			next.ServeHTTP(w, r.WithContext(i18n.WithLanguage(ctx, serve)))
		})
		return respheader.VaryLanguage(inner)
	}
}

func chosen(ctx context.Context) string {
	if user := auth.FromContext(ctx); user != nil {
		return user.Language
	}
	return ""
}

func siteDefault(ctx context.Context) string {
	if current := site.FromContext(ctx); current != nil {
		return current.Language
	}
	return ""
}
