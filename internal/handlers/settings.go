package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/connorpodea/splitit/internal/models"
)

// GetUserSettings returns the preference and privacy configuration for the authenticated user.
// Returns a default settings struct if no row exists yet for this user.
func (h *Handler) GetUserSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	settings, err := h.store.GetUserSettings(userID)
	if err != nil {
		WriteJSON(w, http.StatusOK, models.UserSettings{
			UserID:             userID,
			EmailNotifications: true,
			IsDiscoverable:     true,
		})
		return
	}

	WriteJSON(w, http.StatusOK, settings)
}

// UpdateUserSettings persists updated preference and privacy settings for the authenticated user
// via a conflict-safe upsert operation.
func (h *Handler) UpdateUserSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	var settings models.UserSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload"})
		return
	}

	settings.UserID = userID

	if err := h.store.UpsertUserSettings(&settings); err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}
