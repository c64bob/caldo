package middleware

import (
	"context"
	"net/http"
	"strings"
)

type principalContextKey struct{}

func PrincipalFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(principalContextKey{}).(string)
	return v, ok
}

func ProxyAuth(headerName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headerValues := r.Header.Values(headerName)
			if len(headerValues) == 0 {
				http.Error(w, "Authentifizierungsheader fehlt", http.StatusUnauthorized)
				return
			}
			if len(headerValues) > 1 {
				http.Error(w, "Mehrdeutiger Benutzerheader", http.StatusBadRequest)
				return
			}

			principal := strings.TrimSpace(headerValues[0])
			if principal == "" || strings.Contains(principal, ",") {
				http.Error(w, "Mehrdeutiger Benutzerheader", http.StatusBadRequest)
				return
			}

			ctx := context.WithValue(r.Context(), principalContextKey{}, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
