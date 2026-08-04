package handler

import (
	"net/http"
)

// Home redirects the root path to the Heute view.
func Home(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/today", http.StatusFound)
}
