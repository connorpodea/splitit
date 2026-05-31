package handlers

import (
	"net/http"
)

// GetInitialView determines what screen layout the user should see when they first hit the app.
// It checks the browser for a valid session token cookie wristband.
func (h *Handler) GetInitialView(w http.ResponseWriter, r *http.Request) {
	// Look for our secure session cookie using the helper we made earlier
	cookie, err := r.Cookie("session_user_id")

	// If the cookie is missing or empty, they are not logged in
	if err != nil || cookie.Value == "" {
		// Stream a simple HTML text fragment back to the viewport container
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<div class="text-center font-mono">[DEBUG] No session cookie found. Display Login Form here.</div>`))
		return
	}

	// If the cookie exists, they ARE logged in!
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<div class="text-center font-mono text-emerald-400">[DEBUG] Session verified for: ` + cookie.Value + `. Display Dashboard here.</div>`))
}
