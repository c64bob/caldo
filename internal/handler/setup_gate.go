package handler

import "net/http"

// SetupGateMiddleware blocks normal routes until setup is completed.
func SetupGateMiddleware(state *SetupState) func(http.Handler) http.Handler {
	allowedWhenIncomplete := map[string]struct{}{
		routeKey(http.MethodGet, "/setup"):               {},
		routeKey(http.MethodGet, "/setup/"):              {},
		routeKey(http.MethodPost, "/setup/caldav"):       {},
		routeKey(http.MethodGet, "/setup/calendars"):     {},
		routeKey(http.MethodPost, "/setup/calendars"):    {},
		routeKey(http.MethodPost, "/setup/import"):       {},
		routeKey(http.MethodGet, "/setup/import/events"): {},
		routeKey(http.MethodPost, "/setup/complete"):     {},
		routeKey(http.MethodGet, "/health"):              {},
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if state != nil && state.IsComplete() {
				next.ServeHTTP(w, r)
				return
			}

			if _, ok := allowedWhenIncomplete[routeKey(r.Method, r.URL.Path)]; ok {
				next.ServeHTTP(w, r)
				return
			}

			http.Redirect(w, r, "/setup", http.StatusFound)
		})
	}
}

func routeKey(method string, path string) string {
	return method + " " + path
}
