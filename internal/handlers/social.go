package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/connorpodea/splitit/internal/models"
)

func (h *Handler) SendFriendRequest(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	// Acquire dynamic tracking criteria natively through current session validation
	senderID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Initialize an empty FriendRequest model struct to hold the incoming data
	var input models.FriendRequest

	// Read the JSON text out of the web request body and decode it into the Go struct
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// Hard-bind the origin verification identity using internal session elements instead of raw user variables
	input.SenderID = senderID

	// Pass the populated struct down to the database engine
	err = h.store.SendFriendRequest(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusCreated, input)
}

func (h *Handler) ListIncomingFriendRequests(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	// Validate target context parameters natively through current session validation
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Query the database engine for this user's incoming friend requests
	requests, err := h.store.ListIncomingFriendRequests(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, requests)
}

func (h *Handler) ListOutgoingFriendRequests(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	// Validate target context parameters natively through current session validation
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Query the database engine for this user's outgoing friend requests
	requests, err := h.store.ListOutgoingFriendRequests(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, requests)
}

func (h *Handler) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST methods are permitted to this route"})
		return
	}

	// Secure user validation using credentials fetched directly from cookie memory storage structures
	receiverID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	type Input struct {
		RequestID string `json:"request_id"`
	}
	var input Input

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil || input.RequestID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// senderID is derived inside the store from the database — not trusted from the client
	err = h.store.AcceptFriendRequest(input.RequestID, receiverID)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusOK, input)
}

func (h *Handler) DeclineFriendRequest(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint on accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST methods are permitted to this route"})
		return
	}

	// Confirm user active session validation parameter state before deletion routines run
	_, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Initialize a custom, empty struct to hold the incoming data
	type Input struct {
		RequestID string `json:"request_id"`
	}
	var input Input

	// Read the JSON text out of the web request body and decode it into the variable
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// Pass the variable down to the database engine
	err = h.store.DeclineFriendRequest(input.RequestID)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusOK, input)
}

func (h *Handler) RemoveFriendMutual(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint on accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST methods are permitted to this route"})
		return
	}

	// Extract core verified sender profile constraints from active session data layers
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Initialize a custom, empty struct to hold the incoming data
	type Input struct {
		FriendID string `json:"friend_id"`
	}
	var input Input

	// Read the JSON text out of the web request body and decode it into the variable
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// Pass the verified identity along with request payload details down to the storage layers
	err = h.store.RemoveFriendMutual(userID, input.FriendID)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusOK, input)
}

func (h *Handler) ListFriends(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET methods are permitted to this route"})
		return
	}

	// Secure user identity discovery tracking through current session mapping configurations
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Query the database engine for this users friends
	friends, err := h.store.ListFriends(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, friends)
}

func viewFriends(friends []models.Profile, friendRequests []models.FriendRequest) string {
	friendCount := len(friends)

	// --- Incoming friend requests banner ---
	requestsSection := ""
	if len(friendRequests) > 0 {
		plural := ""
		if len(friendRequests) > 1 {
			plural = "s"
		}
		rows := ""
		for i, req := range friendRequests {
			cls := avatarClass(i)
			ini := strings.ToUpper(req.SenderID)
			if len([]rune(ini)) > 2 {
				ini = string([]rune(ini)[:2])
			}
			rows += `
      <div style="display:flex;align-items:center;gap:12px;padding:12px 14px;border-bottom:1px solid var(--border-soft);" data-req-row="` + req.ID + `">
        <div class="row-avatar ` + cls + `">` + ini + `</div>
        <div style="flex:1;min-width:0;">
          <div style="font-size:14px;font-weight:600;color:var(--text);">@` + req.SenderID + `</div>
          <div style="font-size:12px;color:var(--text-faint);">Wants to be friends</div>
        </div>
        <div style="display:flex;gap:6px;">
          <button onclick="acceptFriendRequest('` + req.ID + `')"
            style="background:var(--emerald);color:#fff;border:none;border-radius:8px;padding:6px 12px;font-size:12px;font-weight:700;cursor:pointer;font-family:inherit;">
            Accept
          </button>
          <button onclick="declineFriendRequest('` + req.ID + `',this)"
            style="background:var(--surface-3);color:var(--text-dim);border:1px solid var(--border);border-radius:8px;padding:6px 12px;font-size:12px;font-weight:700;cursor:pointer;font-family:inherit;">
            Decline
          </button>
        </div>
      </div>`
		}
		requestsSection = `
  <div style="margin-bottom:16px;">
    <div style="font-size:11px;font-weight:700;color:var(--text-mute);text-transform:uppercase;letter-spacing:0.06em;margin-bottom:8px;">` + fmt.Sprintf("%d", len(friendRequests)) + ` pending request` + plural + `</div>
    <div class="card">` + rows + `</div>
  </div>`
	}

	listContent := ""
	if friendCount > 0 {
		listContent = `<div class="card" id="friend-list">` + friendsRows(friends) + `</div>`
	} else {
		listContent = `<div class="card" id="friend-list" style="display:none;"></div>`
	}

	emptyStyle := `style="display:none;"`
	if friendCount == 0 {
		emptyStyle = ""
	}

	return `
<section class="view" data-view="social">
  <div class="friends-header">
    <div class="friends-count">` + fmt.Sprintf("%d", friendCount) + ` <small>friends</small></div>
    <button class="friend-add" onclick="openSheet('addfriend')">
      <svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path d="M12 5v14"/><path d="M5 12h14"/></svg>
      Add friend
    </button>
  </div>
` + requestsSection + `
  <div class="search-wrap">
    <svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
    <input class="search-inp" type="search" placeholder="Search friends" oninput="filterFriends(this.value)" />
  </div>

  ` + listContent + `

  <div class="empty" id="friends-empty" ` + emptyStyle + `>
    <div class="empty-icon">
      <svg width="24" height="24" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><line x1="20" y1="8" x2="20" y2="14"/><line x1="23" y1="11" x2="17" y2="11"/></svg>
    </div>
    <div class="empty-title">No friends yet</div>
    <div class="empty-sub">Add friends to split bills, send payments, and request money in seconds.</div>
    <button onclick="openSheet('addfriend')">Add your first friend</button>
  </div>
</section>
`
}

func friendsRows(friends []models.Profile) string {
	out := ""
	for i, f := range friends {
		cls := avatarClass(i)
		dname := profileDisplayName(&f)
		ini := initials(dname)
		if ini == "?" {
			r := []rune(f.ID)
			if len(r) >= 2 {
				ini = strings.ToUpper(string(r[0])) + strings.ToUpper(string(r[1]))
			} else if len(r) == 1 {
				ini = strings.ToUpper(string(r[0]))
			}
		}
		searchKey := strings.ToLower(dname + " @" + f.ID)
		out += `
    <div class="friend-row" data-name="` + searchKey + `" data-friend-id="` + f.ID + `">
      <div class="row-avatar ` + cls + `">` + ini + `</div>
      <div class="row-body" style="flex:1; min-width:0;">
        <b>` + dname + `</b>
        <div>@` + f.ID + `</div>
      </div>
      <div class="friend-actions">
        <button title="Pay" onclick="openPayToFriend('` + f.ID + `')">
          <svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
        </button>
        <button class="remove" title="Remove" onclick="removeFriend(this,'` + f.ID + `','` + dname + `')">
          <svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-2 14a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2L5 6"/></svg>
        </button>
      </div>
    </div>`
	}
	return out
}

func addFriendSheetHTML() string {
	return `
<div class="sheet-backdrop" id="addfriend-sheet" onclick="if(event.target===this) closeSheet('addfriend')">
  <div class="sheet">
    <div class="sheet-handle"></div>
    <div class="sheet-title">Add friend</div>
    <div class="sheet-pane active" style="display:flex;">
      <div class="search-wrap" style="margin-bottom:0;">
        <svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
        <input class="search-inp" id="addfriend-search-inp" type="search" placeholder="Search by name or @handle…"
               oninput="searchUsersToAdd(this.value)"
               onfocus="initProfileSearch()"
               autocomplete="off" />
      </div>
      <div id="addfriend-results" style="display:flex;flex-direction:column;gap:8px;max-height:360px;overflow-y:auto;padding-top:4px;"></div>
    </div>
  </div>
</div>`
}

// sessionContextScript embeds the current user's ID and friend set as JS
// globals so the client-side search can exclude them without a round-trip.
func sessionContextScript(userID string, friends []models.Profile) string {
	safeID := strings.ReplaceAll(userID, `"`, ``)
	lit := "["
	for i, f := range friends {
		if i > 0 {
			lit += ","
		}
		lit += `"` + strings.ReplaceAll(f.ID, `"`, ``) + `"`
	}
	lit += "]"
	return `<script>window._selfID="` + safeID + `";window._friendIDs=new Set(` + lit + `);</script>`
}
