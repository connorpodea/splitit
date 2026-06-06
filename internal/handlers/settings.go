package handlers

import "net/http"

// CreateLinkedCard registers a new external funding card instrument to the
// authenticated user's account using a tokenized provider reference.
func (h *Handler) CreateLinkedCard(w http.ResponseWriter, r *http.Request) {}

// DeleteLinkedCard removes a linked card from the authenticated user's account.
func (h *Handler) DeleteLinkedCard(w http.ResponseWriter, r *http.Request) {}

// ListLinkedCards returns all external funding instruments registered to the
// authenticated user's account.
func (h *Handler) ListLinkedCards(w http.ResponseWriter, r *http.Request) {}

// GetLinkedCard returns a single linked card record by card identifier,
// scoped to the authenticated user.
func (h *Handler) GetLinkedCard(w http.ResponseWriter, r *http.Request) {}

// SetDefaultCard designates a specific linked card as the principal funding
// source for the authenticated user.
func (h *Handler) SetDefaultCard(w http.ResponseWriter, r *http.Request) {}

// GetUserSettings returns the preference and privacy configuration metadata
// for the authenticated user.
func (h *Handler) GetUserSettings(w http.ResponseWriter, r *http.Request) {}

// UpdateUserSettings persists updated preference and privacy settings for the
// authenticated user via an upsert operation.
func (h *Handler) UpdateUserSettings(w http.ResponseWriter, r *http.Request) {}
