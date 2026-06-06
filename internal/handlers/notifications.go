package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/connorpodea/splitit/internal/models"
)

// ListNotifications returns all unseen notifications for the authenticated user.
func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}
	notifs, err := h.store.ListNotifications(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	WriteJSON(w, http.StatusOK, notifs)
}

// CountUnseenNotifications returns the integer count of unseen notifications for the
// authenticated user, intended for driving badge indicators in the UI.
func (h *Handler) CountUnseenNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}
	count, err := h.store.CountUnseenNotifications(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]int{"unseen_count": count})
}

// MarkNotificationSeen marks a single notification as seen for the authenticated user.
func (h *Handler) MarkNotificationSeen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}
	type Input struct {
		NotifID string `json:"notif_id"`
	}
	var input Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.NotifID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing notif_id"})
		return
	}
	if err := h.store.MarkNotificationSeen(input.NotifID, userID); err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ClearAllNotifications marks every notification for the authenticated user as seen in one batch.
func (h *Handler) ClearAllNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}
	if err := h.store.ClearAllNotifications(userID); err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// viewNotifications renders the notifications section HTML, including per-type icons,
// dismiss-on-click rows, and an empty state if there are no unseen notifications.
func viewNotifications(notifications []models.Notification) string {
	typeIcon := map[string]string{
		"friend_request":   `<svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><line x1="20" y1="8" x2="20" y2="14"/><line x1="23" y1="11" x2="17" y2="11"/></svg>`,
		"friend_accepted":  `<svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><polyline points="16 11 18 13 22 9"/></svg>`,
		"payment_received": `<svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>`,
		"payment_request":  `<svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M12 5v14"/><path d="M5 12h14"/></svg>`,
	}
	typeColor := map[string]string{
		"friend_request":   "av-indigo",
		"friend_accepted":  "av-emerald",
		"payment_received": "av-emerald",
		"payment_request":  "av-amber",
	}

	rows := ""
	for _, n := range notifications {
		icon := typeIcon[n.Type]
		if icon == "" {
			icon = typeIcon["payment_request"]
		}
		cls := typeColor[n.Type]
		if cls == "" {
			cls = "av-indigo"
		}
		date := n.CreatedAt
		if len(date) >= 10 {
			date = date[:10]
		}
		rows += `
    <div class="row" style="cursor:pointer;" onclick="dismissNotif('` + n.ID + `','` + n.LinkView + `',this)">
      <div class="row-avatar ` + cls + `">` + icon + `</div>
      <div class="row-body">
        <div class="row-title"><b>` + n.Title + `</b></div>
        <div class="row-sub">` + n.Body + `</div>
      </div>
      <div class="row-right"><div class="row-time">` + date + `</div></div>
    </div>`
	}

	emptyBlock := ""
	if len(notifications) == 0 {
		emptyBlock = `
  <div class="empty">
    <div class="empty-icon success">
      <svg width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg>
    </div>
    <div class="empty-title">All caught up</div>
    <div class="empty-sub">No new notifications.</div>
  </div>`
	}

	clearBtn := ""
	if len(notifications) > 0 {
		clearBtn = `<button class="linklike" onclick="clearAllNotifs()">Clear all</button>`
	}

	return `
<section class="view" data-view="notifications">
  <div class="section-row" style="margin-top:0;">
    <h2 style="font-size:22px; font-weight:800; letter-spacing:-0.02em;">Notifications</h2>
    ` + clearBtn + `
  </div>
  ` + func() string {
		if len(notifications) > 0 {
			return `<div class="card" id="notif-list">` + rows + `</div>`
		}
		return emptyBlock
	}() + `
</section>
`
}
