package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/connorpodea/splitit/internal/models"
)

// CreateGroup creates a new group, binding the creator identity from the session cookie
// so the client can never nominate an arbitrary creator.
func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	// Acquire the caller's identity from the active session — never trust a client-supplied creator ID
	creatorID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Initialize an empty Group model struct to hold the incoming data
	var input models.Group

	// Read the JSON text out of the web request body and decode it into the Go struct
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	if input.Name == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required field: 'name'"})
		return
	}

	// Hard-bind the creator identity from the session to prevent client spoofing
	input.CreatorID = creatorID

	// Pass the populated struct down to the database engine
	err = h.store.CreateGroup(&input)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	// Return the created group record, now populated with its generated ID
	WriteJSON(w, http.StatusCreated, input)
}

// ListGroups returns all groups that the authenticated user is a member of.
func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}

	// Validate target context parameters natively through current session validation
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Query the database engine for all groups this user belongs to
	groups, err := h.store.ListGroups(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, groups)
}

// ListGroupMembers returns the public profile of every active member in the specified group.
func (h *Handler) ListGroupMembers(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}

	// Confirm the caller holds a valid session before exposing member roster data
	_, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Extract the target group identifier from the URL query parameters
	groupID := r.URL.Query().Get("group_id")
	if groupID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL query parameter: 'group_id'"})
		return
	}

	// Query the database engine for the member roster of this group
	members, err := h.store.ListGroupMembers(groupID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, members)
}

// SendGroupInvitation dispatches a group invitation from the authenticated user to a receiver,
// binding the sender identity from the session cookie to prevent spoofing.
func (h *Handler) SendGroupInvitation(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	// Acquire the caller's identity from the active session — never trust a client-supplied sender ID
	senderID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Initialize a custom struct to hold the two client-supplied fields
	type Input struct {
		GroupID    string `json:"group_id"`
		ReceiverID string `json:"receiver_id"`
	}
	var input Input

	// Read the JSON text out of the web request body and decode it into the struct
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	if input.GroupID == "" || input.ReceiverID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required fields: 'group_id' and 'receiver_id'"})
		return
	}

	// Pass the session-verified sender identity and client-supplied routing details to the store
	err = h.store.SendGroupInvitation(input.GroupID, senderID, input.ReceiverID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	// Return an acknowledgement with the submitted payload
	WriteJSON(w, http.StatusCreated, input)
}

// AcceptGroupInvitation accepts a pending group invitation and registers the caller as a group member.
// The caller identity is bound from the session cookie — the store derives the groupID from the database
// to prevent the client from spoofing membership into an arbitrary group.
func (h *Handler) AcceptGroupInvitation(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	// Secure user validation using credentials fetched directly from cookie memory storage structures
	callerID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	type Input struct {
		InvitationID string `json:"invitation_id"`
	}
	var input Input

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil || input.InvitationID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// callerID is verified by the session; groupID is resolved inside the store from the database
	err = h.store.AcceptGroupInvitation(input.InvitationID, callerID)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": friendlyError(err)})
		return
	}

	// Return a successful acknowledgement with the accepted invitation ID
	WriteJSON(w, http.StatusOK, input)
}

// DeclineGroupInvitation rejects a pending group invitation for the authenticated user.
// The store enforces that the caller must be the intended receiver via a dual-key constraint.
func (h *Handler) DeclineGroupInvitation(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	// Confirm user active session validation parameter state before deletion routines run
	callerID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	type Input struct {
		InvitationID string `json:"invitation_id"`
	}
	var input Input

	// Read the JSON text out of the web request body and decode it into the variable
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil || input.InvitationID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// callerID is session-bound; the store's WHERE receiver_id = ? ensures the caller owns the invitation
	err = h.store.DeclineGroupInvitation(input.InvitationID, callerID)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": friendlyError(err)})
		return
	}

	// Return a successful acknowledgement with the declined invitation ID
	WriteJSON(w, http.StatusOK, input)
}

// ListIncomingGroupInvitations returns all pending group invitations received by the authenticated user.
func (h *Handler) ListIncomingGroupInvitations(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}

	// Validate target context parameters natively through current session validation
	receiverID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Query the database engine for this user's pending incoming group invitations
	invitations, err := h.store.ListIncomingGroupInvitations(receiverID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, invitations)
}

// ListOutgoingGroupInvitations returns all pending group invitations dispatched by the authenticated user.
func (h *Handler) ListOutgoingGroupInvitations(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}

	// Validate target context parameters natively through current session validation
	senderID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Query the database engine for all group invitations this user has sent
	invitations, err := h.store.ListOutgoingGroupInvitations(senderID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, invitations)
}

// RemoveGroupMember forcibly removes a target user from a group's membership roster.
// This is a moderation action — the caller must be the group creator. The store enforces
// that the target must be an active member before the deletion proceeds.
func (h *Handler) RemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	// Bind the caller's identity from the session — the store will verify they are the group creator
	callerID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	type Input struct {
		GroupID      string `json:"group_id"`
		TargetUserID string `json:"target_user_id"`
	}
	var input Input

	// Read the JSON text out of the web request body and decode it into the struct
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	if input.GroupID == "" || input.TargetUserID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required fields: 'group_id' and 'target_user_id'"})
		return
	}

	// Pass the caller identity so the store can enforce creator-only access
	err = h.store.RemoveGroupMember(input.GroupID, input.TargetUserID, callerID)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": friendlyError(err)})
		return
	}

	// Return a successful acknowledgement with the submitted payload
	WriteJSON(w, http.StatusOK, input)
}

// ListGroupActivity returns all payments where both sender and receiver are members of the
// specified group, providing the activity log for the group detail view.
func (h *Handler) ListGroupActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}

	_, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	groupID := r.URL.Query().Get("group_id")
	if groupID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL query parameter: 'group_id'"})
		return
	}

	payments, err := h.store.ListGroupActivity(groupID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, payments)
}

// LeaveGroup removes the authenticated user from a group's membership roster voluntarily.
// The userID is bound from the session cookie so users can only leave on behalf of themselves.
func (h *Handler) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	// Extract core verified sender profile constraints from active session data layers
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	type Input struct {
		GroupID string `json:"group_id"`
	}
	var input Input

	// Read the JSON text out of the web request body and decode it into the variable
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil || input.GroupID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// Pass the session-verified user identity and the target group ID down to the storage layer
	err = h.store.LeaveGroup(input.GroupID, userID)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": friendlyError(err)})
		return
	}

	// Return a successful acknowledgement with the submitted group ID
	WriteJSON(w, http.StatusOK, input)
}
