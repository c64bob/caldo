package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

const sessionCookieName = "session_id"

const sessionIDKey contextKey = "session_id"

// SessionIDFromContext returns the UI session id and whether it was present.
func SessionIDFromContext(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(sessionIDKey).(string)
	return sessionID, ok
}

// SessionMiddleware ensures authenticated UI requests have a session_id cookie.
func SessionMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			sessionID := existingSessionID(r)
			if sessionID == "" {
				generated, err := generateSessionID()
				if err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				sessionID = generated
				http.SetCookie(w, sessionCookie(sessionID))
			}

			ctx := context.WithValue(r.Context(), sessionIDKey, sessionID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func existingSessionID(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	sessionID := strings.TrimSpace(cookie.Value)
	if !validSessionID(sessionID) {
		return ""
	}
	return sessionID
}

func generateSessionID() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func validSessionID(sessionID string) bool {
	if len(sessionID) != 64 {
		return false
	}
	_, err := hex.DecodeString(sessionID)
	return err == nil
}

func sessionCookie(sessionID string) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
}

func requestSessionID(r *http.Request) string {
	if sessionID, ok := SessionIDFromContext(r.Context()); ok && strings.TrimSpace(sessionID) != "" {
		return strings.TrimSpace(sessionID)
	}
	return "single-user-session"
}
