package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/connorpodea/splitit/internal/models"
)

// SendFriendRequest handles a request to send a friend invitation, binding the sender
// identity from the session cookie before delegating to the store.
func (h *Handler) SendFriendRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	senderID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	var input models.FriendRequest
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	input.SenderID = senderID

	err = h.store.SendFriendRequest(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusCreated, input)
}

// ListIncomingFriendRequests returns all pending friend requests received by the authenticated user.
func (h *Handler) ListIncomingFriendRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	requests, err := h.store.ListIncomingFriendRequests(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, requests)
}

// ListOutgoingFriendRequests returns all pending friend requests sent by the authenticated user.
func (h *Handler) ListOutgoingFriendRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	requests, err := h.store.ListOutgoingFriendRequests(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, requests)
}

// AcceptFriendRequest accepts a pending friend request by ID, using the session cookie
// to confirm the caller is the intended receiver.
func (h *Handler) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST methods are permitted to this route"})
		return
	}

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
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, input)
}

// DeclineFriendRequest deletes a pending friend request by ID. The store enforces that the
// session user is the intended receiver — callers cannot decline other users' requests.
func (h *Handler) DeclineFriendRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST methods are permitted to this route"})
		return
	}

	callerID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	type Input struct {
		RequestID string `json:"request_id"`
	}
	var input Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.RequestID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	if err := h.store.DeclineFriendRequest(input.RequestID, callerID); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, input)
}

// RemoveFriendMutual severs a bidirectional friendship, binding the caller's identity
// from the session cookie so users can only remove their own friendships.
func (h *Handler) RemoveFriendMutual(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST methods are permitted to this route"})
		return
	}

	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	type Input struct {
		FriendID string `json:"friend_id"`
	}
	var input Input
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	err = h.store.RemoveFriendMutual(userID, input.FriendID)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, input)
}

// ListFriends returns the public profiles of all active friends for the authenticated user.
func (h *Handler) ListFriends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET methods are permitted to this route"})
		return
	}

	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	friends, err := h.store.ListFriends(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, friends)
}

// viewFriends renders the social section HTML, including pending incoming friend request cards
// and the searchable friends list.
func viewFriends(friends []models.Profile, friendRequests []models.FriendRequest) string {
	friendCount := len(friends)

	requestsSection := ""
	if len(friendRequests) > 0 {
		plural := ""
		if len(friendRequests) > 1 {
			plural = "s"
		}
		hexPalette := []string{"#4ade80", "#22d3ee", "#fb7185", "#facc15", "#c084fc", "#db2777"}
		var reqBuf strings.Builder
		for i, req := range friendRequests {
			hex := hexPalette[i%len(hexPalette)]
			ini := strings.ToUpper(req.SenderID)
			if len([]rune(ini)) > 2 {
				ini = string([]rune(ini)[:2])
			}
			reqBuf.WriteString(renderTmpl(friendRequestRowTmpl, friendRequestRowData{
				RequestID: req.ID,
				Color:     hex,
				Initials:  ini,
				SenderID:  req.SenderID,
			}))
		}
		requestsSection = `
  <div style="margin-bottom:16px;">
    <div style="font-size:11px;font-weight:700;color:var(--text-mute);text-transform:uppercase;letter-spacing:0.06em;margin-bottom:8px;">` + fmt.Sprintf("%d", len(friendRequests)) + ` pending request` + plural + `</div>
    <div class="card">` + reqBuf.String() + `</div>
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

// friendsRows renders the list of individual friend row HTML elements for the social view.
func friendsRows(friends []models.Profile) string {
	hexPalette := []string{"#4ade80", "#22d3ee", "#fb7185", "#facc15", "#c084fc", "#db2777"}
	var buf strings.Builder
	for i, f := range friends {
		hex := f.ProfileColor
		if hex == "" {
			hex = hexPalette[i%len(hexPalette)]
		}
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
		buf.WriteString(renderTmpl(friendListRowTmpl, friendListRowData{
			SearchKey: strings.ToLower(dname + " @" + f.ID),
			FriendID:  f.ID,
			Color:     hex,
			Initials:  ini,
			Name:      dname,
			Handle:    f.ID,
		}))
	}
	return buf.String()
}

// addFriendSheetHTML renders the bottom-sheet HTML for the add-friend search flow,
// with a live-search input that queries the profiles list endpoint.
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

// sessionContextScript embeds the current user's ID and friend set as JS globals so the
// client-side search can exclude them without a round-trip.
// template.JSEscapeString is used for all user-supplied values embedded in JS string literals.
func sessionContextScript(userID string, friends []models.Profile) string {
	lit := "["
	for i, f := range friends {
		if i > 0 {
			lit += ","
		}
		lit += `"` + template.JSEscapeString(f.ID) + `"`
	}
	lit += "]"
	return `<script>window._selfID="` + template.JSEscapeString(userID) + `";window._friendIDs=new Set(` + lit + `);</script>`
}
