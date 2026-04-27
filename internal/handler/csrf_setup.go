package handler

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"caldo/internal/view"
)

const (
	csrfCookieName = "caldo_csrf"
	csrfHeaderName = "X-CSRF-Token"
)

var errInvalidCSRF = errors.New("invalid csrf token")

// SetupCSRFMiddleware enforces double-submit-cookie CSRF protection for mutating setup routes.
func SetupCSRFMiddleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ensureCSRFToken(w, r, secret)
			if token != "" {
				w.Header().Set(csrfHeaderName, token)
				r = r.WithContext(view.WithCSRFToken(r.Context(), token))
			}
			if isMutatingMethod(r.Method) {
				if err := validateCSRFToken(r, secret); err != nil {
					http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func ensureCSRFToken(w http.ResponseWriter, r *http.Request, secret []byte) string {
	cookie, err := r.Cookie(csrfCookieName)
	if err == nil && validateSignedCSRF(cookie.Value, secret) == nil {
		return cookie.Value
	}

	token, generateErr := generateSignedCSRF(secret)
	if generateErr != nil {
		return ""
	}

	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	return token
}

func validateCSRFToken(r *http.Request, secret []byte) error {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return errInvalidCSRF
	}

	headerValue := strings.TrimSpace(r.Header.Get(csrfHeaderName))
	if headerValue == "" || headerValue != cookie.Value {
		return errInvalidCSRF
	}

	if err := validateSignedCSRF(headerValue, secret); err != nil {
		return errInvalidCSRF
	}

	return nil
}

func generateSignedCSRF(secret []byte) (string, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate csrf nonce: %w", err)
	}

	nonceHex := hex.EncodeToString(nonce)
	sig := csrfSignature(secret, nonceHex)
	return nonceHex + "." + sig, nil
}

func validateSignedCSRF(token string, secret []byte) error {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return errInvalidCSRF
	}
	if len(parts[0]) != 64 {
		return errInvalidCSRF
	}

	expected := csrfSignature(secret, parts[0])
	if !hmac.Equal([]byte(parts[1]), []byte(expected)) {
		return errInvalidCSRF
	}

	return nil
}

func csrfSignature(secret []byte, nonceHex string) string {
	h := hmac.New(sha256.New, secret)
	_, _ = h.Write([]byte(nonceHex))
	return hex.EncodeToString(h.Sum(nil))
}

func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
