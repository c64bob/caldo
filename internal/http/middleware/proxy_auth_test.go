package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxyAuth_MissingHeader_ReturnsUnauthorized(t *testing.T) {
	h := ProxyAuth("X-Forwarded-User")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestProxyAuth_MultipleHeaders_ReturnsBadRequest(t *testing.T) {
	h := ProxyAuth("X-Forwarded-User")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req.Header.Add("X-Forwarded-User", "alice")
	req.Header.Add("X-Forwarded-User", "bob")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestProxyAuth_CommaSeparatedHeader_ReturnsBadRequest(t *testing.T) {
	h := ProxyAuth("X-Forwarded-User")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req.Header.Set("X-Forwarded-User", "alice,bob")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestProxyAuth_ValidHeader_SetsPrincipalAndCallsNext(t *testing.T) {
	called := false
	h := ProxyAuth("X-Forwarded-User")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		p, ok := PrincipalFromContext(r.Context())
		if !ok {
			t.Fatal("expected principal in context")
		}
		if p != "alice@example.com" {
			t.Fatalf("unexpected principal: %q", p)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req.Header.Set("X-Forwarded-User", " alice@example.com ")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if !called {
		t.Fatal("expected next handler to be called")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, rr.Code)
	}
}
