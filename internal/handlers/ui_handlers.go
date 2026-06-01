package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/connorpodea/splitit/internal/models"
)

func (h *Handler) GetInitialView(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_user_id")
	if err != nil || cookie.Value == "" {
		writeHTML(w, loginHTML())
		return
	}
	user, err := h.store.GetUser(cookie.Value)
	if err != nil {
		writeHTML(w, loginHTML())
		return
	}

	// Dynamic database queries with explicit error recovery fallbacks
	friends, err := h.store.ListFriends(user.ID)
	if err != nil {
		friends = []*models.Profile{}
	}
	installments, err := h.store.ListInstallments(user.ID)
	if err != nil {
		installments = []*models.Installment{}
	}
	overdueInstallments, err := h.store.ListOverdueInstallments(user.ID)
	if err != nil {
		overdueInstallments = []*models.Installment{}
	}
	incomingRequests, err := h.store.ListIncomingPaymentRequests(user.ID)
	if err != nil {
		incomingRequests = []*models.PaymentRequest{}
	}
	friendRequests, err := h.store.ListIncomingFriendRequests(user.ID)
	if err != nil {
		friendRequests = []*models.FriendRequest{}
	}
	notifications, err := h.store.ListNotifications(user.ID)
	if err != nil {
		notifications = []*models.Notification{}
	}

	writeHTML(w, dashboardHTML(user, friends, installments, overdueInstallments, incomingRequests, friendRequests, notifications))
}

func (h *Handler) GetRegistrationView(w http.ResponseWriter, r *http.Request) {
	writeHTML(w, registerHTML())
}

func (h *Handler) GetDashboardView(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_user_id")
	if err != nil || cookie.Value == "" {
		writeHTML(w, loginHTML())
		return
	}
	user, err := h.store.GetUser(cookie.Value)
	if err != nil {
		writeHTML(w, loginHTML())
		return
	}

	friends, err := h.store.ListFriends(user.ID)
	if err != nil {
		friends = []*models.Profile{}
	}
	installments, err := h.store.ListInstallments(user.ID)
	if err != nil {
		installments = []*models.Installment{}
	}
	overdueInstallments, err := h.store.ListOverdueInstallments(user.ID)
	if err != nil {
		overdueInstallments = []*models.Installment{}
	}
	incomingRequests, err := h.store.ListIncomingPaymentRequests(user.ID)
	if err != nil {
		incomingRequests = []*models.PaymentRequest{}
	}
	friendRequests, err := h.store.ListIncomingFriendRequests(user.ID)
	if err != nil {
		friendRequests = []*models.FriendRequest{}
	}
	notifications, err := h.store.ListNotifications(user.ID)
	if err != nil {
		notifications = []*models.Notification{}
	}

	writeHTML(w, dashboardHTML(user, friends, installments, overdueInstallments, incomingRequests, friendRequests, notifications))
}

func writeHTML(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(body))
}

func initials(fullName string) string {
	clean := strings.TrimSpace(fullName)
	if clean == "" {
		return "?"
	}
	parts := strings.Fields(clean)
	if len(parts) == 1 {
		runes := []rune(parts[0])
		if len(runes) >= 2 {
			return strings.ToUpper(string(runes[0])) + strings.ToUpper(string(runes[1]))
		}
		return strings.ToUpper(string(runes[0]))
	}
	first := []rune(parts[0])
	last := []rune(parts[len(parts)-1])
	return strings.ToUpper(string(first[0])) + strings.ToUpper(string(last[0]))
}

func displayName(u *models.User) string {
	if strings.TrimSpace(u.Name) != "" {
		return u.Name
	}
	return u.ID
}

func creditScoreLabel(score uint8) string {
	switch {
	case score >= 76:
		return "Excellent"
	case score >= 51:
		return "Good"
	case score >= 31:
		return "Fair"
	default:
		return "Poor"
	}
}

func avatarClass(i int) string {
	classes := []string{"av-indigo", "av-amber", "av-emerald", "av-violet", "av-cyan", "av-pink"}
	return classes[i%len(classes)]
}

func profileDisplayName(p *models.Profile) string {
	if strings.TrimSpace(p.Name) != "" {
		return p.Name
	}
	return p.ID
}

// ---------------------------------------------------------------------------
// DASHBOARD SHELL
// ---------------------------------------------------------------------------

func dashboardHTML(user *models.User, friends []*models.Profile, installments []*models.Installment, overdueInstallments []*models.Installment, incomingRequests []*models.PaymentRequest, friendRequests []*models.FriendRequest, notifications []*models.Notification) string {
	name := displayName(user)
	avatar := initials(name)
	handle := user.ID
	email := user.Email
	phone := user.PhoneNumber
	if email == "" {
		email = handle + "@splitit.app"
	}

	var outstandingBNPL float64
	for _, inst := range installments {
		if !inst.IsPaid {
			outstandingBNPL += inst.Amount
		}
	}
	availableCredit := user.CreditLimit - outstandingBNPL
	if availableCredit < 0 {
		availableCredit = 0
	}
	scoreLabel := creditScoreLabel(user.CreditScore)
	friendReqCount := len(friendRequests)
	_ = friendReqCount
	notifCount := len(notifications)

	return dashboardStyles() + `
<div id="app-root" class="app-shell" data-uid="` + handle + `">
  <div id="toast"></div>

  <header class="topbar">
    <div class="topbar-inner">
      <button class="brand-btn" data-tab="home" onclick="goView('home')" aria-label="Home">
        <span class="brand">split<span>it</span></span>
      </button>
      <div class="topbar-actions">
        <button class="icon-btn" aria-label="Notifications" onclick="goView('notifications')">
          <svg width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>
          ` + func() string {
		if notifCount > 0 {
			return `<span style="position:absolute;top:6px;right:6px;width:16px;height:16px;border-radius:50%;background:#ef4444;border:2px solid #0f172a;display:flex;align-items:center;justify-content:center;font-size:9px;font-weight:700;color:#fff;font-family:'JetBrains Mono',monospace;">` + fmt.Sprintf("%d", notifCount) + `</span>`
		}
		return `<span class="dot"></span>`
	}() + `
        </button>
        <button class="avatar-btn" id="profile-btn" onclick="goView('profile')" aria-label="Profile">` + avatar + `</button>
      </div>
    </div>
  </header>

  <div class="app-body">
    <aside class="sidebar">
      <nav class="side-nav">
        <button class="side-link active" data-tab="home" onclick="goView('home')">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M3 12l9-9 9 9"/><path d="M5 10v10h14V10"/></svg>
          Home
        </button>
        <button class="side-link" data-tab="activity" onclick="goView('activity')">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M3 12h4l3-9 4 18 3-9h4"/></svg>
          Activity
        </button>
        <button class="side-link" data-tab="social" onclick="goView('social')">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
          Social
        </button>
        <button class="side-link" data-tab="analytics" onclick="goView('analytics')">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 0 1 3 19.875v-6.75Z" /><path stroke-linecap="round" stroke-linejoin="round" d="M9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V8.625Z" /><path stroke-linecap="round" stroke-linejoin="round" d="M16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V4.125Z" /></svg>
          Analytics
        </button>
        <button class="side-link" data-tab="wallet" onclick="goView('wallet')">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M21 12a2.25 2.25 0 0 0-2.25-2.25H15a3 3 0 1 1-6 0H5.25A2.25 2.25 0 0 0 3 12m18 0v6A2.25 2.25 0 0 1 18.75 20H5.25A2.25 2.25 0 0 1 3 18v-6m18 0V9A2.25 2.25 0 0 0 18.75 6.75H5.25A2.25 2.25 0 0 0 3 9v3M16.5 10.5a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5Z" /></svg>
          Wallet
        </button>
        <button class="side-link" data-tab="profile" onclick="goView('profile')">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
          Profile
        </button>
        <button class="side-link" data-tab="settings" onclick="goView('settings')">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.43l-1.003.767a1.123 1.123 0 0 0-.417 1.03c.004.074.006.148.006.222 0 .074-.002.148-.006.222a1.123 1.123 0 0 0 .417 1.03l1.003.767a1.125 1.125 0 0 1 .26 1.43l-1.296 2.247a1.125 1.125 0 0 1-1.37.49l-1.216-.456a1.125 1.125 0 0 0-1.075.124c-.073.044-.146.087-.22.128c-.332.183-.582.495-.645.869l-.213 1.28c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281a1.125 1.125 0 0 0-.646-.87c-.074-.04-.148-.083-.22-.127a1.124 1.124 0 0 0-1.075-.124l-1.217.456a1.125 1.125 0 0 1-1.37-.49l-1.296-2.247a1.125 1.125 0 0 1 .26-1.43l1.003-.767a1.122 1.122 0 0 0 .417-1.03c-.004-.074-.006-.148-.006-.222 0-.074.002-.148.006-.222a1.122 1.122 0 0 0-.417-1.03l-1.003-.767a1.125 1.125 0 0 1-.26-1.43l1.296-2.247a1.125 1.125 0 0 1 1.37-.49l1.216.456c.356.133.751.072 1.076-.124.072-.041.146-.084.218-.128.332-.183.582-.495.646-.869l.213-1.28Z" /><circle cx="12" cy="12" r="3" /></svg>
          Settings
        </button>
        <button class="side-link" onclick="signOut()" style="color: #fca5a5;">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
          Sign out
        </button>
      </nav>
      <div class="side-promo">
        <div class="promo-title">splitit Score</div>
        <div class="promo-score"><span class="mono">` + fmt.Sprintf("%d", user.CreditScore) + `</span><span class="promo-small">/100</span></div>
        <div class="promo-sub">` + scoreLabel + ` · BNPL limit $` + fmt.Sprintf("%.0f", user.CreditLimit) + `</div>
      </div>
    </aside>

    <main class="content">
` + viewHome(name, user, overdueInstallments, installments, incomingRequests) + `
` + viewActivity(installments, overdueInstallments, incomingRequests) + `
` + viewFriends(friends, friendRequests) + `
` + viewNotifications(notifications) + `
` + viewProfile(user, avatar, name, handle, email, phone, friends, installments) + `
    </main>
  </div>

  <nav class="tabbar">
    <button class="tab active" data-tab="home" onclick="goView('home')">
      <svg width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M3 12l9-9 9 9"/><path d="M5 10v10h14V10"/></svg>
      <span>Home</span>
    </button>
    <button class="tab" data-tab="activity" onclick="goView('activity')">
      <svg width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M3 12h4l3-9 4 18 3-9h4"/></svg>
      <span>Activity</span>
    </button>
    <button class="tab" data-tab="social" onclick="goView('social')">
      <svg width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
      <span>Social</span>
    </button>
    <button class="tab" data-tab="profile" onclick="goView('profile')">
      <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
      <span>Profile</span>
    </button>
  </nav>

` + paySheetHTML(friends, availableCredit) + `
` + requestSheetHTML(friends) + `
` + addFriendSheetHTML() + `
` + sessionContextScript(handle, friends) + `
` + dashboardScript() + `
</div>`
}

// ---------------------------------------------------------------------------
// LOGIN SCREEN
// ---------------------------------------------------------------------------
func loginHTML() string {
	return `
<div class="w-full max-w-sm mx-auto px-4 py-8 auth-anim">
  <style>
    @import url('https://fonts.googleapis.com/css2?family=Onest:wght@400;500;600;700;800&family=JetBrains+Mono:wght@500;600;700&display=swap');
    .auth-anim { animation: fadeSlideUp 0.4s cubic-bezier(0.16, 1, 0.3, 1) both; font-family: 'Onest', system-ui, sans-serif; }
    @keyframes fadeSlideUp { from { opacity: 0; transform: translateY(16px); } to { opacity: 1; transform: translateY(0); } }
    .field-wrap { position: relative; }
    .field-icon { position: absolute; right: 14px; top: 50%; transform: translateY(-50%); cursor: pointer; color: #64748b; display: flex; align-items: center; }
    .field-icon:hover { color: #94a3b8; }
    .inp { width: 100%; background: #0f172a; border: 1px solid #1e293b; border-radius: 12px; padding: 12px 44px 12px 16px; font-size: 14px; color: #f1f5f9; outline: none; transition: border-color 0.2s; box-sizing: border-box; font-family: inherit; }
    .inp:focus { border-color: #6366f1; }
    .inp::placeholder { color: #475569; }
    .btn-main { width: 100%; background: #6366f1; color: #fff; border: none; border-radius: 12px; padding: 13px; font-size: 15px; font-weight: 600; cursor: pointer; transition: background 0.2s, transform 0.1s; letter-spacing: -0.01em; font-family: inherit; }
    .btn-main:hover { background: #4f46e5; }
    .btn-main:active { transform: scale(0.98); }
    .lbl { display: block; font-size: 11px; color: #64748b; margin-bottom: 6px; font-weight: 600; letter-spacing: 0.06em; text-transform: uppercase; }
  </style>

  <div style="text-align:center; margin-bottom: 36px;">
    <div style="font-size: 54px; font-weight: 800; letter-spacing: -0.04em; color: #f8fafc; margin-bottom: 4px;">
      split<span style="color: #6366f1;">it</span>
    </div>
    <div style="font-size: 16px; color: #64748b;">Send money. Split bills. Pay later.</div>
  </div>

  <div style="background: #0f172a; border: 1px solid #1e293b; border-radius: 20px; padding: 28px 24px;">
    <div style="font-size: 18px; font-weight: 700; color: #f1f5f9; margin-bottom: 6px;">Welcome back</div>
    <div style="font-size: 13px; color: #64748b; margin-bottom: 24px;">Sign in to your account</div>

    <form hx-post="/users/login"
          hx-ext="json-enc"
          hx-target="#auth-msg"
          hx-on::response-error="
            const d = JSON.parse(event.detail.xhr.responseText);
            document.getElementById('auth-msg').innerHTML = '<div style=\'color:#f87171;background:#1c0a0a;border:1px solid #450a0a;border-radius:10px;padding:10px 14px;font-size:13px\'>' + (d.error || 'Sign in failed') + '</div>';
          "
          hx-on::after-request="if(event.detail.successful) { document.getElementById('auth-msg').innerHTML=''; window.location.reload(); }"
          style="display:flex; flex-direction:column; gap:16px;">
      <div>
        <label class="lbl">User ID</label>
        <div class="field-wrap">
          <input class="inp" type="text" name="id" placeholder="your_handle" required autocomplete="username" />
          <span class="field-icon"><svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg></span>
        </div>
      </div>
      <div>
        <label class="lbl">Password</label>
        <div class="field-wrap">
          <input class="inp" type="password" id="login-pw" name="password" placeholder="••••••••" required autocomplete="current-password" />
          <span class="field-icon" onclick="
            const i = document.getElementById('login-pw');
            i.type = i.type === 'password' ? 'text' : 'password';
          "><svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg></span>
        </div>
      </div>
      <div id="auth-msg"></div>
      <button class="btn-main" type="submit">Sign in</button>
    </form>
  </div>

  <div style="text-align:center; margin-top: 20px;">
    <span style="color: #64748b; font-size: 14px;">Don't have an account? </span>
    <button hx-get="/ui/register-view" hx-target="#main-application-viewport" hx-swap="innerHTML" style="background:none; border:none; color:#6366f1; font-size:14px; font-weight:600; cursor:pointer; padding:0;">Create account</button>
  </div>
</div>`
}

// ---------------------------------------------------------------------------
// REGISTRATION SCREEN
// ---------------------------------------------------------------------------
func registerHTML() string {
	return `
<div class="w-full max-w-sm mx-auto px-4 py-8 auth-anim">
  <style>
    @import url('https://fonts.googleapis.com/css2?family=Onest:wght@400;500;600;700;800&display=swap');
    .auth-anim { animation: fadeSlideUp 0.4s cubic-bezier(0.16, 1, 0.3, 1) both; font-family: 'Onest', system-ui, sans-serif; }
    @keyframes fadeSlideUp { from { opacity: 0; transform: translateY(16px); } to { opacity: 1; transform: translateY(0); } }
    .inp, .inp-plain { width: 100%; background: #0f172a; border: 1px solid #1e293b; border-radius: 12px; padding: 12px 16px; font-size: 14px; color: #f1f5f9; outline: none; transition: border-color 0.2s; box-sizing: border-box; font-family: inherit; }
    .inp:focus, .inp-plain:focus { border-color: #6366f1; }
    .inp::placeholder, .inp-plain::placeholder { color: #475569; }
    .btn-main { width: 100%; background: #10b981; color: #fff; border: none; border-radius: 12px; padding: 13px; font-size: 15px; font-weight: 600; cursor: pointer; transition: background 0.2s, transform 0.1s; font-family: inherit; }
    .btn-main:hover { background: #059669; }
    .btn-main:active { transform: scale(0.98); }
    .lbl { display: block; font-size: 11px; color: #64748b; margin-bottom: 6px; font-weight: 600; letter-spacing: 0.06em; text-transform: uppercase; }
    #reg-toast { display: none; position: fixed; top: 24px; left: 50%; transform: translateX(-50%); background: #052e16; border: 1px solid #166534; color: #4ade80; border-radius: 12px; padding: 12px 20px; font-size: 14px; font-weight: 600; z-index: 999; white-space: nowrap; box-shadow: 0 8px 24px rgba(0,0,0,0.4); }
  </style>

  <div id="reg-toast">Account created successfully!</div>

  <div style="text-align:center; margin-bottom: 32px;">
    <div style="font-size: 54px; font-weight: 800; letter-spacing: -0.04em; color: #f8fafc; margin-bottom: 4px;">split<span style="color: #6366f1;">it</span></div>
    <div style="font-size: 16px; color: #64748b;">Join millions splitting smarter</div>
  </div>

  <div style="background: #0f172a; border: 1px solid #1e293b; border-radius: 20px; padding: 28px 24px;">
    <div style="font-size: 18px; font-weight: 700; color: #f1f5f9; margin-bottom: 6px;">Create your account</div>
    <div style="font-size: 13px; color: #64748b; margin-bottom: 24px;">Free to join. No hidden fees.</div>

    <form id="reg-form"
          hx-post="/users/create"
          hx-ext="json-enc"
          hx-target="#reg-msg"
          hx-on::before-request="
            var showErr = function(msg) {
              document.getElementById('reg-msg').innerHTML = '<div style=\'color:#f87171;background:#1c0a0a;border:1px solid #450a0a;border-radius:10px;padding:10px 14px;font-size:13px;font-weight:500\'>' + msg + '</div>';
            };

            var uid = document.getElementById('reg-uid').value.trim();
            if (!uid) { event.preventDefault(); showErr('Please choose a username.'); return; }
            if (!/^[a-zA-Z0-9_]+$/.test(uid)) { event.preventDefault(); showErr('Username may only contain letters, numbers, and underscores (_).'); return; }

            var f = document.getElementById('reg-fname').value.trim();
            var l = document.getElementById('reg-lname').value.trim();
            if (!f || !l) { event.preventDefault(); showErr('Please enter your first and last name.'); return; }
            document.getElementById('reg-fullname').value = f + ' ' + l;

            var phoneEl = document.querySelector('[name=phone_number]');
            if (phoneEl) phoneEl.value = phoneEl.value.replace(/\D/g, '');

            var pw  = document.getElementById('reg-pw').value;
            var cpw = document.getElementById('reg-cpw').value;
            if (pw.length < 8)  { event.preventDefault(); showErr('Password must be at least 8 characters.'); return; }
            if (pw !== cpw)     { event.preventDefault(); showErr('Passwords don’t match — please try again.'); return; }
          "
          hx-on::response-error="
            const d = JSON.parse(event.detail.xhr.responseText);
            document.getElementById('reg-msg').innerHTML = '<div style=\'color:#f87171;background:#1c0a0a;border:1px solid #450a0a;border-radius:10px;padding:10px 14px;font-size:13px\'>' + (d.error || 'Could not create account') + '</div>';
          "
          hx-on::after-request="
            if(event.detail.successful) {
              const toast = document.getElementById('reg-toast');
              toast.style.display = 'block';
              document.getElementById('reg-form').style.opacity = '0.4';
              document.getElementById('reg-form').style.pointerEvents = 'none';
              document.getElementById('reg-back-btn').style.display = 'block';
              setTimeout(() => { toast.style.display = 'none'; }, 4000);
            }
          "
          style="display:flex; flex-direction:column; gap:14px;">
      <input type="hidden" name="name" id="reg-fullname" />
      <div>
        <label class="lbl">User ID</label>
        <input class="inp-plain" type="text" name="id" id="reg-uid" placeholder="your_handle" required autocomplete="username"
               oninput="
                 var ok = /^[a-zA-Z0-9_]*$/.test(this.value);
                 this.style.borderColor = (!ok && this.value) ? '#ef4444' : '';
                 var h = document.getElementById('uid-hint');
                 if (h) h.style.display = (!ok && this.value) ? 'block' : 'none';
               " />
        <div id="uid-hint" style="display:none; color:#f87171; font-size:12px; margin-top:5px; font-weight:500;">Only letters, numbers, and _ are allowed</div>
      </div>
      <div><label class="lbl">First name</label><input class="inp-plain" type="text" id="reg-fname" placeholder="First" autocomplete="given-name" /></div>
      <div><label class="lbl">Last name</label><input class="inp-plain" type="text" id="reg-lname" placeholder="Last" autocomplete="family-name" /></div>
      <div><label class="lbl">Email</label><input class="inp-plain" type="email" name="email" placeholder="you@email.com" required autocomplete="email" /></div>
      <div><label class="lbl">Phone</label><input class="inp-plain" type="tel" name="phone_number" placeholder="123-456-7890" autocomplete="tel" /></div>
      <div><label class="lbl">Password</label><input class="inp-plain" type="password" id="reg-pw" name="password" placeholder="Min 8 characters" autocomplete="new-password" /></div>
      <div><label class="lbl">Confirm password</label><input class="inp-plain" type="password" id="reg-cpw" placeholder="Re-enter password" autocomplete="new-password" /></div>
      <div id="reg-msg"></div>
      <button class="btn-main" type="submit">Create account</button>
    </form>

    <button id="reg-back-btn" hx-get="/ui/initial-view" hx-target="#main-application-viewport" hx-swap="innerHTML" style="display:none; width:100%; background:none; border:1px solid #1e293b; border-radius:12px; color:#94a3b8; font-size:14px; font-weight:600; padding:13px; cursor:pointer; margin-top:12px;">Sign in to your new account →</button>
  </div>

  <div style="text-align:center; margin-top: 20px;">
    <span style="color: #64748b; font-size: 14px;">Already have an account? </span>
    <button hx-get="/ui/initial-view" hx-target="#main-application-viewport" hx-swap="innerHTML" style="background:none; border:none; color:#6366f1; font-size:14px; font-weight:600; cursor:pointer; padding:0;">Sign in</button>
  </div>
</div>`
}

// ---------------------------------------------------------------------------
// HOME VIEW
// ---------------------------------------------------------------------------

func viewHome(name string, user *models.User, overdueInstallments []*models.Installment, installments []*models.Installment, incomingRequests []*models.PaymentRequest) string {
	firstName := strings.Fields(name)
	greet := "there"
	if len(firstName) > 0 {
		greet = firstName[0]
	}

	balanceWhole := int(user.Balance)
	balanceCents := int((user.Balance-float64(balanceWhole))*100 + 0.5)

	var outstandingBNPL float64
	splitIDs := make(map[string]struct{})
	for _, inst := range installments {
		if !inst.IsPaid {
			outstandingBNPL += inst.Amount
			splitIDs[inst.PaymentID] = struct{}{}
		}
	}
	activeSplits := len(splitIDs)

	overdueCount := len(overdueInstallments)
	overdueStatClass := "qstat"
	overdueStatVal := "0 bills"
	if overdueCount > 0 {
		overdueStatClass = "qstat warn"
		overdueStatVal = fmt.Sprintf("%d bills", overdueCount)
	}

	activeCount := 0
	for _, inst := range installments {
		if !inst.IsPaid {
			isOverdue := false
			for _, ov := range overdueInstallments {
				if ov.ID == inst.ID {
					isOverdue = true
					break
				}
			}
			if !isOverdue {
				activeCount++
			}
		}
	}

	// Build recent activity rows — incoming requests first, then installments, cap at 5.
	recentRows := ""
	count := 0
	for i, req := range incomingRequests {
		if count >= 5 {
			break
		}
		cls := avatarClass(i)
		ini := strings.ToUpper(req.RequesterID)
		if len([]rune(ini)) > 2 {
			ini = string([]rune(ini)[:2])
		}
		date := req.CreatedAt
		if len(date) >= 10 {
			date = date[:10]
		}
		recentRows += `
    <div class="row">
      <div class="row-avatar ` + cls + `">` + ini + `</div>
      <div class="row-body">
        <div class="row-title"><b>@` + req.RequesterID + `</b> requested from <b>you</b></div>
        <div class="row-sub">` + req.Note + `</div>
      </div>
      <div class="row-right"><div class="row-amt req mono">$` + fmt.Sprintf("%.2f", req.Amount) + `</div><div class="row-time">` + date + `</div></div>
    </div>`
		count++
	}
	for i, inst := range installments {
		if count >= 5 {
			break
		}
		cls := avatarClass(i)
		statusLabel := "Due " + inst.DueDate
		amtClass := "bnpl"
		if inst.IsPaid {
			statusLabel = "Paid"
			amtClass = "pos"
		}
		recentRows += `
    <div class="row">
      <div class="row-avatar ` + cls + `">BN</div>
      <div class="row-body">
        <div class="row-title"><b>BNPL</b> installment <span class="pill">Pay-in-4</span></div>
        <div class="row-sub">` + statusLabel + `</div>
      </div>
      <div class="row-right"><div class="row-amt ` + amtClass + ` mono">$` + fmt.Sprintf("%.2f", inst.Amount) + `</div><div class="row-time">` + inst.DueDate + `</div></div>
    </div>`
		count++
	}
	if recentRows == "" {
		recentRows = `
    <div style="text-align:center; padding:28px 16px; color:var(--text-mute); font-size:13px;">No recent activity yet.</div>`
	}

	return `
<section class="view active" data-view="home">
  <div class="greeting" style="font-size:14px; color:var(--text-mute); margin-bottom:14px;">Welcome back, <b style="color:var(--text);">` + greet + `</b></div>

  <div class="hero">
    <div class="hero-label">Available balance</div>
    <div class="hero-amount mono">$` + fmt.Sprintf("%d", balanceWhole) + `<span class="cents">.` + fmt.Sprintf("%02d", balanceCents) + `</span></div>
    <div class="hero-meta">
      <span><b>$` + fmt.Sprintf("%.2f", outstandingBNPL) + `</b> outstanding</span>
      <span style="opacity:.4;">·</span>
      <span><b>` + fmt.Sprintf("%d", activeSplits) + `</b> active splits</span>
    </div>
  </div>

  <div class="actions">
    <button class="action req" onclick="openSheet('request')">
      <span class="ico"><svg width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M12 5v14"/><path d="M5 12h14"/></svg></span>
      <span class="lbl-stack"><b>Request</b><small>Ask a friend</small></span>
    </button>
    <button class="action pay" onclick="openSheet('pay')">
      <span class="ico"><svg width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg></span>
      <span class="lbl-stack"><b>Pay</b><small>Send · BNPL</small></span>
    </button>
  </div>

  <div class="quick-strip">
    <button class="` + overdueStatClass + `" onclick="goView('activity', 'overdue')">
      <div>
        <div class="qstat-label">Overdue</div>
        <div class="qstat-val mono">` + overdueStatVal + `</div>
      </div>
      <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M9 18l6-6-6-6"/></svg>
    </button>
    <button class="qstat" onclick="goView('activity', 'active')">
      <div>
        <div class="qstat-label">Active plans</div>
        <div class="qstat-val mono">` + fmt.Sprintf("%d", activeCount) + `</div>
      </div>
      <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M9 18l6-6-6-6"/></svg>
    </button>
  </div>

  <div class="section-row">
    <h2>Recent activity</h2>
    <button class="linklike" onclick="goView('activity')">See all</button>
  </div>
  <div class="card">` + recentRows + `</div>
</section>
`
}

// ---------------------------------------------------------------------------
// ACTIVITY VIEW
// ---------------------------------------------------------------------------

func viewActivity(installments []*models.Installment, overdueInstallments []*models.Installment, incomingRequests []*models.PaymentRequest) string {
	// Build overdue ID set for fast lookup.
	overdueIDs := make(map[string]bool)
	for _, inst := range overdueInstallments {
		overdueIDs[inst.ID] = true
	}

	// --- Payments pane: incoming payment requests ---
	paymentsRows := ""
	for i, req := range incomingRequests {
		cls := avatarClass(i)
		ini := strings.ToUpper(req.RequesterID)
		if len([]rune(ini)) > 2 {
			ini = string([]rune(ini)[:2])
		}
		date := req.CreatedAt
		if len(date) >= 10 {
			date = date[:10]
		}
		statusLabel := req.Note
		if statusLabel == "" {
			statusLabel = req.Status
		} else {
			statusLabel += " · " + req.Status
		}
		paymentsRows += `
      <div class="row">
        <div class="row-avatar ` + cls + `">` + ini + `</div>
        <div class="row-body"><div class="row-title"><b>@` + req.RequesterID + `</b> requested</div><div class="row-sub">` + statusLabel + `</div></div>
        <div class="row-right"><div class="row-amt req mono">$` + fmt.Sprintf("%.2f", req.Amount) + `</div><div class="row-time">` + date + `</div></div>
      </div>`
	}
	if paymentsRows == "" {
		paymentsRows = `<div style="text-align:center; padding:28px 16px; color:var(--text-mute); font-size:13px;">No incoming payment requests.</div>`
	}

	// --- Active pane: non-overdue, unpaid installments ---
	activeRows := ""
	activeIdx := 0
	for _, inst := range installments {
		if inst.IsPaid || overdueIDs[inst.ID] {
			continue
		}
		cls := avatarClass(activeIdx)
		activeRows += `
      <div class="row">
        <div class="row-avatar ` + cls + `">BN</div>
        <div class="row-body">
          <div class="row-title"><b>BNPL installment</b> <span class="pill">Pay-in-4</span></div>
          <div class="row-sub">Due ` + inst.DueDate + `</div>
          <div class="progress"><span style="width:50%;"></span></div>
        </div>
        <div class="row-right"><div class="row-amt bnpl mono">$` + fmt.Sprintf("%.2f", inst.Amount) + `</div><div class="row-time">/installment</div></div>
      </div>`
		activeIdx++
	}
	if activeRows == "" {
		activeRows = `
      <div style="text-align:center; padding:28px 16px; color:var(--text-mute); font-size:13px;">No active BNPL plans.</div>`
	}

	// --- Overdue pane ---
	overdueSubtabBadge := ""
	if len(overdueInstallments) > 0 {
		overdueSubtabBadge = ` <span class="badge mono">` + fmt.Sprintf("%d", len(overdueInstallments)) + `</span>`
	}

	overdueContent := ""
	if len(overdueInstallments) == 0 {
		overdueContent = `
    <div class="empty">
      <div class="empty-icon success">
        <svg width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg>
      </div>
      <div class="empty-title">You're all caught up</div>
      <div class="empty-sub">No overdue bills. Your Splitit Score is safe.</div>
    </div>`
	} else {
		var overdueTotal float64
		for _, inst := range overdueInstallments {
			overdueTotal += inst.Amount
		}
		overdueRows := ""
		for i, inst := range overdueInstallments {
			_ = i
			overdueRows += `
        <div class="row overdue-row" data-installment-id="` + inst.ID + `" data-payment-id="` + inst.PaymentID + `" data-amount="` + fmt.Sprintf("%.2f", inst.Amount) + `">
          <div class="row-avatar av-rose">OD</div>
          <div class="row-body">
            <div class="row-title"><b>BNPL installment</b> <span class="pill warn">Overdue</span></div>
            <div class="row-sub">Due ` + inst.DueDate + `</div>
            <div class="progress warn"><span style="width:25%;"></span></div>
          </div>
          <div class="row-right">
            <button onclick="payInstallment('` + inst.ID + `','` + inst.PaymentID + `',` + fmt.Sprintf("%.2f", inst.Amount) + `)"
              style="background:var(--rose);color:#fff;border:none;border-radius:8px;padding:6px 12px;font-size:12px;font-weight:700;cursor:pointer;font-family:inherit;">
              Pay $` + fmt.Sprintf("%.2f", inst.Amount) + `
            </button>
          </div>
        </div>`
		}
		overdueContent = `
    <div class="overdue-banner">
      <svg width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
      <div class="ob-text">
        <strong>$` + fmt.Sprintf("%.2f", overdueTotal) + ` overdue across ` + fmt.Sprintf("%d", len(overdueInstallments)) + ` bills</strong>
        <div>Late fees may apply. Pay now to keep your Splitit Score intact.</div>
      </div>
      <button onclick="payAllOverdue()">Pay all</button>
    </div>
    <div class="card">` + overdueRows + `</div>`
	}

	return `
<section class="view" data-view="activity">
  <div class="section-row" style="margin-top:0;">
    <h2 style="font-size:22px; font-weight:800; letter-spacing:-0.02em;">Activity</h2>
  </div>

  <div class="subtabs" role="tablist">
    <button class="subtab active" data-pane="all" onclick="goPane(this)">Payments</button>
    <button class="subtab" data-pane="active" onclick="goPane(this)">Active</button>
    <button class="subtab" data-pane="overdue" onclick="goPane(this)">Overdue` + overdueSubtabBadge + `</button>
  </div>

  <div data-pane-content="all" class="pane-content">
    <div class="card">` + paymentsRows + `</div>
  </div>

  <div data-pane-content="active" class="pane-content" style="display:none;">
    <div class="card">` + activeRows + `</div>
  </div>

  <div data-pane-content="overdue" class="pane-content" style="display:none;">
    ` + overdueContent + `
  </div>
</section>
`
}

// ---------------------------------------------------------------------------
// FRIENDS / SOCIAL VIEW
// ---------------------------------------------------------------------------

func viewFriends(friends []*models.Profile, friendRequests []*models.FriendRequest) string {
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
          <button onclick="acceptFriendRequest('` + req.ID + `','` + req.SenderID + `')"
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

func friendsRows(friends []*models.Profile) string {
	out := ""
	for i, f := range friends {
		cls := avatarClass(i)
		dname := profileDisplayName(f)
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

// ---------------------------------------------------------------------------
// NOTIFICATIONS VIEW
// ---------------------------------------------------------------------------

func viewNotifications(notifications []*models.Notification) string {
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

// ---------------------------------------------------------------------------
// PROFILE VIEW
// ---------------------------------------------------------------------------

func viewProfile(user *models.User, avatar, name, handle, email, phone string, friends []*models.Profile, installments []*models.Installment) string {
	score := user.CreditScore
	scoreLabel := creditScoreLabel(score)
	scoreWidth := fmt.Sprintf("%d%%", score)

	var outstandingBNPL float64
	activeSplits := make(map[string]struct{})
	for _, inst := range installments {
		if !inst.IsPaid {
			outstandingBNPL += inst.Amount
			activeSplits[inst.PaymentID] = struct{}{}
		}
	}
	availableCredit := user.CreditLimit - outstandingBNPL
	if availableCredit < 0 {
		availableCredit = 0
	}

	balanceWhole := int(user.Balance)

	return `
<section class="view" data-view="profile">
  <div class="profile-hero">
    <div class="avatar-xl">` + avatar + `</div>
    <div class="profile-name">` + name + `</div>
    <div class="profile-handle">@` + handle + `</div>
    <div class="profile-email">` + email + `</div>
    <div class="profile-phone_number">` + phone + `</div>
  </div>

  <div class="score-card">
    <div class="score-header">
      <div class="score-title">Splitit Score</div>
      <div class="score-pill">` + scoreLabel + `</div>
    </div>
    <div class="score-val mono">` + fmt.Sprintf("%d", score) + `<small>/100</small></div>
    <div class="score-track"><span style="width:` + scoreWidth + `;"></span></div>
    <div class="score-track-labels"><span>Poor</span><span>Fair</span><span>Good</span><span>Excellent</span></div>
    <div style="display:flex; justify-content:space-between; margin-top:14px; padding-top:14px; border-top:1px solid rgba(255,255,255,0.06);">
      <div>
        <div style="font-size:11px; color:var(--text-mute); text-transform:uppercase; letter-spacing:0.05em; font-weight:600;">BNPL limit</div>
        <div class="mono" style="font-size:18px; font-weight:700; color:var(--emerald-hi); margin-top:4px;">$` + fmt.Sprintf("%.2f", user.CreditLimit) + `</div>
      </div>
      <div style="text-align:right;">
        <div style="font-size:11px; color:var(--text-mute); text-transform:uppercase; letter-spacing:0.05em; font-weight:600;">Available</div>
        <div class="mono" style="font-size:18px; font-weight:700; color:var(--text); margin-top:4px;">$` + fmt.Sprintf("%.2f", availableCredit) + `</div>
      </div>
    </div>
  </div>

  <div class="stat-grid">
    <div class="stat-tile"><div class="stat-lbl">Balance</div><div class="stat-val green mono">$` + fmt.Sprintf("%d", balanceWhole) + `</div></div>
    <div class="stat-tile"><div class="stat-lbl">Friends</div><div class="stat-val mono">` + fmt.Sprintf("%d", len(friends)) + `</div></div>
    <div class="stat-tile"><div class="stat-lbl">Splits</div><div class="stat-val indigo mono">` + fmt.Sprintf("%d", len(activeSplits)) + `</div></div>
  </div>

  <div class="menu-list">
    <button class="menu-item mobile-only" onclick="goView('analytics')">
      <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 0 1 3 19.875v-6.75Z" /><path stroke-linecap="round" stroke-linejoin="round" d="M9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V8.625Z" /><path stroke-linecap="round" stroke-linejoin="round" d="M16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V4.125Z" /></svg>
      Analytics
      <svg class="chev" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M9 18l6-6-6-6"/></svg>
    </button>
    <button class="menu-item mobile-only" onclick="goView('wallet')">
      <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M21 12a2.25 2.25 0 0 0-2.25-2.25H15a3 3 0 1 1-6 0H5.25A2.25 2.25 0 0 0 3 12m18 0v6A2.25 2.25 0 0 1 18.75 20H5.25A2.25 2.25 0 0 1 3 18v-6m18 0V9A2.25 2.25 0 0 0 18.75 6.75H5.25A2.25 2.25 0 0 0 3 9v3M16.5 10.5a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5Z" /></svg>
      Wallet
      <svg class="chev" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M9 18l6-6-6-6"/></svg>
    </button>
    <button class="menu-item mobile-only" onclick="goView('settings')">
      <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
      Settings
      <svg class="chev" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M9 18l6-6-6-6"/></svg>
    </button>
    <button class="menu-item danger mobile-only" onclick="signOut()">
      <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
      Sign out
    </button>
  </div>
</section>
`
}

// ---------------------------------------------------------------------------
// PAY SHEET  (dynamic friend dropdowns + real credit display)
// ---------------------------------------------------------------------------

func paySheetHTML(friends []*models.Profile, availableCredit float64) string {
	friendOpts := `<option value="">Select a friend…</option>`
	for _, f := range friends {
		dname := profileDisplayName(f)
		friendOpts += `<option value="` + f.ID + `">` + dname + ` · @` + f.ID + `</option>`
	}

	return `
<div class="sheet-backdrop" id="pay-sheet" onclick="if(event.target===this) closeSheet('pay')">
  <div class="sheet">
    <div class="sheet-handle"></div>
    <div class="sheet-title">New payment</div>
    <div class="sheet-tabs" role="tablist">
      <button class="sheet-tab active" data-paytab="send" onclick="payTab(this)">Send to friend</button>
      <button class="sheet-tab" data-paytab="bnpl" onclick="payTab(this)">Buy now, pay later</button>
    </div>

    <div class="sheet-pane active" data-pay-pane="send">
      <div class="amount-big mono"><span class="dollar">$</span><span id="send-amt">0.00</span></div>
      <div class="chip-row" style="justify-content:center;">
        <button type="button" class="chip" onclick="setSendAmt(10)">$10</button>
        <button type="button" class="chip" onclick="setSendAmt(25)">$25</button>
        <button type="button" class="chip" onclick="setSendAmt(50)">$50</button>
        <button type="button" class="chip" onclick="setSendAmt(100)">$100</button>
      </div>
      <div>
        <label class="modal-lbl">To</label>
        <select class="modal-sel" id="pay-receiver-sel">` + friendOpts + `</select>
      </div>
      <div>
        <label class="modal-lbl">Amount</label>
        <input class="modal-inp" id="pay-amount-inp" type="number" inputmode="decimal" placeholder="0.00"
               oninput="document.getElementById('send-amt').textContent=parseFloat(this.value||0).toFixed(2)" />
      </div>
      <div>
        <label class="modal-lbl">Note</label>
        <input class="modal-inp" id="pay-note-inp" type="text" placeholder="Dinner, rent, etc." />
      </div>
      <button class="submit-btn" onclick="submitSendPayment()">Send payment</button>
    </div>

    <div class="sheet-pane" data-pay-pane="bnpl">
      <div class="info-card">
        <span class="label">Available credit</span>
        <span class="val mono">$` + fmt.Sprintf("%.2f", availableCredit) + `</span>
      </div>
      <div>
        <label class="modal-lbl">Recipient</label>
        <select class="modal-sel" id="bnpl-receiver-sel">` + friendOpts + `</select>
      </div>
      <div>
        <label class="modal-lbl">Purchase amount</label>
        <input class="modal-inp" id="bnpl-amount-inp" type="number" inputmode="decimal" placeholder="0.00" />
      </div>
      <div>
        <label class="modal-lbl">Plan</label>
        <select class="modal-sel" id="bnpl-plan-sel">
          <option value="4">Pay-in-4 (0% APR · 4 payments over 6 weeks)</option>
          <option value="6">Pay-in-6 (12.99% APR · 6 monthly payments)</option>
          <option value="12">Pay-in-12 (15.99% APR · 12 monthly payments)</option>
        </select>
      </div>
      <div>
        <label class="modal-lbl">Note</label>
        <input class="modal-inp" id="bnpl-note-inp" type="text" placeholder="Item or purchase description" />
      </div>
      <button class="submit-btn emerald" onclick="submitBNPL()">Approve plan</button>
    </div>
  </div>
</div>`
}

// ---------------------------------------------------------------------------
// REQUEST SHEET  (dynamic friend dropdowns)
// ---------------------------------------------------------------------------

func requestSheetHTML(friends []*models.Profile) string {
	friendOpts := `<option value="">Select a friend…</option>`
	for _, f := range friends {
		dname := profileDisplayName(f)
		friendOpts += `<option value="` + f.ID + `">` + dname + ` · @` + f.ID + `</option>`
	}

	return `
<div class="sheet-backdrop" id="request-sheet" onclick="if(event.target===this) closeSheet('request')">
  <div class="sheet">
    <div class="sheet-handle"></div>
    <div class="sheet-title">Request money</div>
    <div class="sheet-pane active" style="display:flex;">
      <div class="amount-big mono"><span class="dollar">$</span><span id="req-amt">0.00</span></div>
      <div class="chip-row" style="justify-content:center;">
        <button type="button" class="chip" onclick="setReqAmt(10)">$10</button>
        <button type="button" class="chip" onclick="setReqAmt(25)">$25</button>
        <button type="button" class="chip" onclick="setReqAmt(50)">$50</button>
        <button type="button" class="chip" onclick="setReqAmt(100)">$100</button>
      </div>
      <div>
        <label class="modal-lbl">Request from</label>
        <select class="modal-sel" id="req-payer-sel">` + friendOpts + `</select>
      </div>
      <div>
        <label class="modal-lbl">Amount</label>
        <input class="modal-inp" id="req-amount-inp" type="number" inputmode="decimal" placeholder="0.00"
               oninput="document.getElementById('req-amt').textContent=parseFloat(this.value||0).toFixed(2)" />
      </div>
      <div>
        <label class="modal-lbl">Note</label>
        <input class="modal-inp" id="req-note-inp" type="text" placeholder="What's it for?" />
      </div>
      <button class="submit-btn amber" onclick="submitRequest()">Send request</button>
    </div>
  </div>
</div>`
}

// ---------------------------------------------------------------------------
// ADD FRIEND SHEET  (new)
// ---------------------------------------------------------------------------

// sessionContextScript embeds the current user's ID and friend set as JS
// globals so the client-side search can exclude them without a round-trip.
func sessionContextScript(userID string, friends []*models.Profile) string {
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

// ---------------------------------------------------------------------------
// DASHBOARD STYLES  (unchanged — kept here for reference)
// ---------------------------------------------------------------------------
func dashboardStyles() string {
	return `
<style>
  @import url('https://fonts.googleapis.com/css2?family=Onest:wght@400;500;600;700;800&family=JetBrains+Mono:wght@500;600;700&display=swap');

  html { scroll-padding-top: 62px; }

  :root {
    --bg: #020617;
    --surface: #0b1220;
    --surface-2: #0f172a;
    --surface-3: #1e293b;
    --border: #1e293b;
    --border-soft: #131c30;
    --text: #f1f5f9;
    --text-dim: #94a3b8;
    --text-mute: #64748b;
    --text-faint: #475569;
    --indigo: #6366f1;
    --indigo-soft: #1e1b4b;
    --indigo-hi: #818cf8;
    --emerald: #10b981;
    --emerald-soft: #052e16;
    --emerald-hi: #34d399;
    --amber: #f59e0b;
    --amber-hi: #fbbf24;
    --rose: #ef4444;
    --rose-soft: #1c0a0a;
    --rose-hi: #f87171;
  }

  .desktop-only { display: flex !important; }
  .mobile-only { display: none !important; }

  @media (max-width: 1023px) {
    .desktop-only { display: none !important; }
    .mobile-only { display: flex !important; }
  }

  #main-application-viewport { max-width: 100% !important; padding: 0 !important; }
  body { background: var(--bg) !important; padding: 0 !important; display: block !important; min-height: 100vh; align-items: initial !important; justify-content: initial !important; }

  .app-shell {
    font-family: 'Onest', system-ui, -apple-system, sans-serif;
    color: var(--text);
    background: var(--bg);
    min-height: 100vh;
    min-height: 100dvh;
    -webkit-font-smoothing: antialiased;
  }
  .mono { font-family: 'JetBrains Mono', monospace; font-feature-settings: 'tnum' 1; }

  .topbar {
    position: sticky; top: 0; z-index: 40;
    background: rgba(2, 6, 23, 0.85);
    backdrop-filter: blur(14px);
    -webkit-backdrop-filter: blur(14px);
    border-bottom: 1px solid var(--border-soft);
  }
  .topbar-inner {
    max-width: 1280px; margin: 0 auto;
    display: flex; align-items: center; justify-content: space-between;
    padding: 14px 20px;
  }
  .brand-btn { background: none; border: none; cursor: pointer; padding: 0; }
  .brand { font-size: 38px; font-weight: 800; letter-spacing: -0.04em; color: var(--text); }
  .brand span { color: var(--indigo); }
  .topbar-actions { display: flex; align-items: center; gap: 10px; }
  .icon-btn {
    position: relative;
    width: 38px; height: 38px; border-radius: 50%;
    background: var(--surface-2); border: 1px solid var(--border);
    color: var(--text-dim); cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    transition: background .15s, color .15s, transform .15s;
  }
  .icon-btn:hover { background: var(--surface-3); color: var(--text); }
  .icon-btn .dot { position: absolute; top: 9px; right: 10px; width: 7px; height: 7px; border-radius: 50%; background: var(--emerald-hi); border: 2px solid var(--surface-2); }
  .avatar-btn {
    width: 38px; height: 38px; border-radius: 50%;
    background: linear-gradient(135deg, var(--indigo) 0%, #4f46e5 100%);
    border: 2px solid #312e81;
    cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    font-size: 13px; font-weight: 700; color: #fff;
    font-family: inherit;
    transition: transform .15s;
  }
  .avatar-btn:hover { transform: scale(1.06); }

  .app-body {
    max-width: 1280px; margin: 0 auto;
    display: grid; grid-template-columns: 1fr;
    min-height: calc(100vh - 60px);
  }
  .sidebar { display: none; }

  .content {
    padding: 18px 16px 96px;
    width: 100%;
    max-width: 720px;
    margin: 0 auto;
    box-sizing: border-box;
  }
  .view { display: none; animation: fadeUp .35s cubic-bezier(0.16, 1, 0.3, 1) both; }
  .view.active { display: block; }
  @keyframes fadeUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

  .hero {
    position: relative;
    background: linear-gradient(135deg, #1e1b4b 0%, #0f172a 60%, #0b1220 100%);
    border: 1px solid #312e81;
    border-radius: 20px;
    padding: 22px 22px 24px;
    overflow: hidden;
  }
  .hero::before {
    content: ""; position: absolute; inset: -40% -10% auto auto; width: 280px; height: 280px;
    background: radial-gradient(circle, rgba(99,102,241,0.35), transparent 60%);
    pointer-events: none;
  }
  .hero-label { position: relative; font-size: 11px; color: var(--indigo-hi); font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; }
  .hero-amount { position: relative; font-size: clamp(34px, 7vw, 44px); font-weight: 800; color: #f8fafc; letter-spacing: -0.035em; margin-top: 6px; line-height: 1; }
  .hero-amount .cents { font-size: 0.55em; color: #94a3b8; font-weight: 700; }
  .hero-meta { position: relative; display: flex; gap: 18px; margin-top: 16px; font-size: 13px; color: var(--text-dim); }
  .hero-meta b { color: var(--text); font-weight: 600; }

  .actions {
    display: grid; grid-template-columns: 1fr 1fr; gap: 12px;
    margin: 18px 0;
  }
  .action {
    display: flex; align-items: center; gap: 12px;
    background: var(--surface-2);
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 16px 18px;
    cursor: pointer;
    text-align: left;
    color: var(--text);
    font-family: inherit; font-size: 15px; font-weight: 600;
    transition: border-color .2s, transform .1s, background .2s;
  }
  .action:hover { border-color: var(--indigo); background: #131c30; }
  .action:active { transform: scale(0.98); }
  .action .ico {
    width: 38px; height: 38px; border-radius: 12px;
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0;
  }
  .action.pay .ico { background: var(--indigo-soft); color: var(--indigo-hi); }
  .action.req .ico { background: #1c0a00; color: var(--amber-hi); }
  .action .lbl-stack { display: flex; flex-direction: column; }
  .action .lbl-stack b { font-size: 15px; font-weight: 700; }
  .action .lbl-stack small { font-size: 12px; color: var(--text-mute); font-weight: 500; margin-top: 2px; }

  .quick-strip {
    display: grid; grid-template-columns: 1fr 1fr;
    gap: 10px; margin-bottom: 18px;
  }
  .qstat {
    background: var(--surface-2); border: 1px solid var(--border);
    border-radius: 14px; padding: 12px 14px;
    display: flex; align-items: center; justify-content: space-between; gap: 8px;
    cursor: pointer;
    transition: border-color .2s, background .2s;
  }
  .qstat:hover { border-color: var(--surface-3); }
  .qstat-label { font-size: 12px; color: var(--text-mute); font-weight: 600; }
  .qstat-val { font-size: 16px; color: var(--text); font-weight: 700; }
  .qstat.warn { border-color: rgba(239, 68, 68, 0.35); background: rgba(239, 68, 68, 0.06); }
  .qstat.warn .qstat-val { color: var(--rose-hi); }
  .qstat.warn .qstat-label { color: #fca5a5; }

  .section-row {
    display: flex; align-items: center; justify-content: space-between;
    margin: 22px 0 10px;
  }
  .section-row h2 { font-size: 15px; font-weight: 700; color: var(--text); margin: 0; letter-spacing: -0.01em; }
  .section-row a, .section-row button.linklike {
    font-size: 13px; color: var(--indigo-hi); text-decoration: none; font-weight: 600;
    background: none; border: none; cursor: pointer; font-family: inherit;
  }
  .section-row a:hover, .section-row button.linklike:hover { color: var(--indigo); }

  .card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 16px; overflow: hidden; }

  .row { display: flex; align-items: center; gap: 12px; padding: 14px 16px; border-bottom: 1px solid var(--border-soft); transition: background .15s; }
  .row:last-child { border-bottom: none; }
  .row:hover { background: var(--surface-3); }
  .row-avatar {
    width: 40px; height: 40px; border-radius: 50%;
    display: flex; align-items: center; justify-content: center;
    font-size: 13px; font-weight: 700; flex-shrink: 0;
    font-family: inherit;
  }
  .row-body { flex: 1; min-width: 0; }
  .row-title { font-size: 14px; color: var(--text); font-weight: 500; line-height: 1.35; }
  .row-title b { font-weight: 700; }
  .row-sub { font-size: 12px; color: var(--text-faint); margin-top: 2px; }
  .row-right { text-align: right; flex-shrink: 0; }
  .row-amt { font-size: 15px; font-weight: 700; letter-spacing: -0.01em; }
  .row-amt.pos { color: var(--emerald-hi); }
  .row-amt.neg { color: var(--text); }
  .row-amt.req { color: var(--amber-hi); }
  .row-amt.bnpl { color: var(--indigo-hi); }
  .row-time { color: var(--text-faint); font-size: 11px; margin-top: 2px; }

  .av-indigo { background: var(--indigo-soft); color: var(--indigo-hi); }
  .av-amber  { background: #1c0a00; color: var(--amber-hi); }
  .av-emerald { background: var(--emerald-soft); color: var(--emerald-hi); }
  .av-rose { background: var(--rose-soft); color: var(--rose-hi); }
  .av-violet { background: #1a0533; color: #c084fc; }
  .av-cyan { background: #042f2e; color: #5eead4; }
  .av-pink { background: #4a044e; color: #f0abfc; }

  .subtabs {
    display: flex; gap: 4px;
    background: var(--surface-2); border: 1px solid var(--border);
    border-radius: 12px; padding: 4px;
    margin-bottom: 16px;
  }
  .subtab {
    flex: 1; background: none; border: none; cursor: pointer;
    padding: 9px 8px; border-radius: 8px;
    font-size: 13px; font-weight: 600; color: var(--text-mute);
    font-family: inherit;
    transition: background .15s, color .15s;
    display: flex; align-items: center; justify-content: center; gap: 6px;
  }
  .subtab:hover { color: var(--text-dim); }
  .subtab.active { background: var(--surface-3); color: var(--text); }
  .subtab .badge {
    display: inline-flex; align-items: center; justify-content: center;
    min-width: 18px; height: 18px; padding: 0 5px;
    border-radius: 9px;
    background: var(--rose); color: #fff;
    font-size: 10px; font-weight: 700;
    font-family: 'JetBrains Mono', monospace;
  }

  .overdue-row { background: rgba(239, 68, 68, 0.06); }
  .overdue-row:hover { background: rgba(239, 68, 68, 0.12); }
  .overdue-row .row-title b { color: var(--rose-hi); }
  .overdue-banner {
    background: linear-gradient(135deg, rgba(239, 68, 68, 0.12), rgba(239, 68, 68, 0.04));
    border: 1px solid rgba(239, 68, 68, 0.35);
    border-radius: 14px;
    padding: 14px 16px;
    margin-bottom: 14px;
    display: flex; align-items: center; gap: 12px;
  }
  .overdue-banner svg { flex-shrink: 0; color: var(--rose-hi); }
  .overdue-banner .ob-text { flex: 1; }
  .overdue-banner .ob-text strong { color: var(--rose-hi); font-weight: 700; font-size: 14px; }
  .overdue-banner .ob-text div { color: #fca5a5; font-size: 12px; margin-top: 2px; }
  .overdue-banner button {
    background: var(--rose); color: #fff;
    border: none; border-radius: 10px; padding: 9px 14px;
    font-size: 13px; font-weight: 700; cursor: pointer; font-family: inherit;
    transition: background .2s;
  }
  .overdue-banner button:hover { background: #dc2626; }

  .pill {
    display: inline-flex; align-items: center;
    padding: 2px 8px; border-radius: 99px;
    font-size: 11px; font-weight: 600;
    background: var(--indigo-soft); color: var(--indigo-hi);
    margin-left: 8px;
  }
  .pill.warn { background: rgba(239, 68, 68, 0.15); color: var(--rose-hi); }
  .pill.ok { background: var(--emerald-soft); color: var(--emerald-hi); }

  .progress {
    height: 4px; border-radius: 2px;
    background: var(--surface-3); overflow: hidden;
    margin-top: 8px;
  }
  .progress > span { display: block; height: 100%; background: var(--indigo); border-radius: 2px; }
  .progress.warn > span { background: var(--rose); }

  .friends-header {
    display: flex; align-items: center; justify-content: space-between;
    margin-bottom: 14px;
  }
  .friends-count { font-size: 24px; font-weight: 800; letter-spacing: -0.02em; }
  .friends-count small { font-size: 14px; color: var(--text-mute); font-weight: 600; margin-left: 6px; }
  .friend-add {
    background: var(--indigo); color: #fff;
    border: none; border-radius: 10px;
    padding: 9px 14px; font-size: 13px; font-weight: 700;
    cursor: pointer; font-family: inherit;
    display: flex; align-items: center; gap: 6px;
    transition: background .2s;
  }
  .friend-add:hover { background: #4f46e5; }
  .search-inp {
    width: 100%; box-sizing: border-box;
    background: var(--surface-2); border: 1px solid var(--border);
    border-radius: 12px;
    padding: 11px 14px 11px 38px;
    font-size: 14px; color: var(--text);
    outline: none; font-family: inherit;
    transition: border-color .2s;
  }
  .search-inp:focus { border-color: var(--indigo); }
  .search-inp::placeholder { color: var(--text-faint); }
  .search-wrap { position: relative; margin-bottom: 14px; }
  .search-wrap svg { position: absolute; left: 12px; top: 50%; transform: translateY(-50%); color: var(--text-faint); pointer-events: none; }

  .friend-row { display: flex; align-items: center; gap: 12px; padding: 12px 14px; border-bottom: 1px solid var(--border-soft); }
  .friend-row:last-child { border-bottom: none; }
  .friend-row .row-body b { font-size: 14px; color: var(--text); font-weight: 600; }
  .friend-row .row-body div { font-size: 12px; color: var(--text-faint); margin-top: 2px; }
  .friend-actions { display: flex; gap: 6px; }
  .friend-actions button {
    background: var(--surface-3); border: 1px solid var(--border);
    color: var(--text-dim); cursor: pointer;
    width: 32px; height: 32px; border-radius: 8px;
    display: flex; align-items: center; justify-content: center;
    transition: background .15s, color .15s, border-color .15s;
  }
  .friend-actions button:hover { background: var(--surface-2); color: var(--text); border-color: var(--indigo); }
  .friend-actions button.remove:hover { color: var(--rose-hi); border-color: var(--rose); }

  .profile-hero {
    background: var(--surface-2);
    border: 1px solid var(--border);
    border-radius: 20px;
    padding: 24px; margin-bottom: 16px;
    text-align: center;
  }
  .avatar-xl {
    width: 80px; height: 80px; border-radius: 50%;
    background: linear-gradient(135deg, var(--indigo) 0%, #4f46e5 100%);
    border: 2px solid #312e81;
    margin: 0 auto 14px;
    display: flex; align-items: center; justify-content: center;
    font-size: 28px; font-weight: 700; color: #fff;
    font-family: inherit;
  }
  .profile-name { font-size: 22px; font-weight: 800; color: var(--text); letter-spacing: -0.02em; }
  .profile-handle { font-size: 13px; color: var(--text-mute); margin-top: 4px; }
  .profile-email { font-size: 13px; color: var(--text-faint); margin-top: 2px; }
  .profile-phone_number { font-size: 13px; color: var(--text-faint); margin-top: 2px; }

  .score-card {
    background: linear-gradient(135deg, #1e1b4b 0%, #0f172a 100%);
    border: 1px solid #312e81;
    border-radius: 20px;
    padding: 22px;
    margin-bottom: 16px;
  }
  .score-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 14px; }
  .score-title { font-size: 13px; color: var(--indigo-hi); font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase; }
  .score-pill { background: rgba(52, 211, 153, 0.12); color: var(--emerald-hi); padding: 3px 10px; border-radius: 99px; font-size: 11px; font-weight: 700; letter-spacing: 0.03em; }
  .score-val { font-size: 56px; font-weight: 800; line-height: 1; color: var(--text); letter-spacing: -0.04em; }
  .score-val small { font-size: 18px; color: var(--text-mute); font-weight: 600; margin-left: 4px; }
  .score-track { height: 8px; border-radius: 4px; background: rgba(255,255,255,0.08); overflow: hidden; margin: 16px 0 8px; }
  .score-track > span { display: block; height: 100%; background: linear-gradient(90deg, var(--rose), var(--amber), var(--emerald-hi)); border-radius: 4px; }
  .score-track-labels { display: flex; justify-content: space-between; font-size: 10px; color: var(--text-faint); font-weight: 600; letter-spacing: 0.05em; text-transform: uppercase; }

  .stat-grid {
    display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; margin-bottom: 16px;
  }
  .stat-tile {
    background: var(--surface-2); border: 1px solid var(--border);
    border-radius: 14px; padding: 14px;
    text-align: left;
  }
  .stat-tile .stat-lbl { font-size: 11px; color: var(--text-mute); font-weight: 600; letter-spacing: 0.05em; text-transform: uppercase; margin-bottom: 6px; }
  .stat-tile .stat-val { font-size: 18px; font-weight: 700; color: var(--text); letter-spacing: -0.02em; }
  .stat-tile .stat-val.green { color: var(--emerald-hi); }
  .stat-tile .stat-val.indigo { color: var(--indigo-hi); }

  .menu-list { background: var(--surface-2); border: 1px solid var(--border); border-radius: 14px; overflow: hidden; }
  .menu-item {
    width: 100%; background: none; border: none;
    display: flex; align-items: center; gap: 12px;
    padding: 14px 16px; cursor: pointer;
    color: var(--text); font-size: 14px; font-weight: 500;
    font-family: inherit; text-align: left;
    border-bottom: 1px solid var(--border-soft);
    transition: background .15s;
  }
  .menu-item:last-child { border-bottom: none; }
  .menu-item:hover { background: var(--surface-3); }
  .menu-item svg { color: var(--text-mute); flex-shrink: 0; }
  .menu-item.danger { color: var(--rose-hi); }
  .menu-item.danger svg { color: var(--rose-hi); }
  .menu-item .chev { margin-left: auto; color: var(--text-faint); }

  .empty {
    text-align: center;
    padding: 36px 20px;
    background: var(--surface-2); border: 1px dashed var(--border);
    border-radius: 16px;
  }
  .empty-icon {
    width: 56px; height: 56px; border-radius: 50%;
    background: var(--surface-3); color: var(--text-dim);
    margin: 0 auto 12px;
    display: flex; align-items: center; justify-content: center;
  }
  .empty-icon.success { background: var(--emerald-soft); color: var(--emerald-hi); }
  .empty-title { font-size: 15px; font-weight: 700; color: var(--text); margin-bottom: 4px; }
  .empty-sub { font-size: 13px; color: var(--text-mute); max-width: 280px; margin: 0 auto; }
  .empty button {
    margin-top: 14px;
    background: var(--indigo); color: #fff;
    border: none; border-radius: 10px;
    padding: 9px 16px; font-size: 13px; font-weight: 700;
    cursor: pointer; font-family: inherit;
  }
  .empty button:hover { background: #4f46e5; }

  .tabbar {
    position: fixed; bottom: 0; left: 0; right: 0; z-index: 30;
    background: rgba(2, 6, 23, 0.92);
    backdrop-filter: blur(14px);
    -webkit-backdrop-filter: blur(14px);
    border-top: 1px solid var(--border-soft);
    display: grid; grid-template-columns: repeat(4, 1fr);
    padding: 6px 6px max(6px, env(safe-area-inset-bottom));
  }
  .tab {
    background: none; border: none; cursor: pointer;
    display: flex; flex-direction: column; align-items: center; gap: 2px;
    padding: 8px 0;
    color: var(--text-faint); font-family: inherit; font-size: 11px; font-weight: 600;
    transition: color .15s;
    border-radius: 10px;
  }
  .tab span { letter-spacing: 0.01em; }
  .tab.active { color: var(--indigo-hi); }
  .tab:active { background: var(--surface-2); }

  .sheet-backdrop {
    display: none;
    position: fixed; inset: 0; z-index: 60;
    background: rgba(0, 0, 0, 0.7);
    align-items: flex-end; justify-content: center;
  }
  .sheet-backdrop.open { display: flex; animation: fadeIn .2s both; }
  .sheet {
    background: var(--surface-2);
    border: 1px solid var(--border);
    border-radius: 24px 24px 0 0;
    width: 100%; max-width: 540px;
    padding: 0 0 max(28px, env(safe-area-inset-bottom));
    animation: slideUp .35s cubic-bezier(0.34, 1.36, 0.64, 1) both;
    max-height: 92vh; max-height: 92dvh;
    overflow-y: auto;
  }
  @keyframes slideUp { from { transform: translateY(100%); } to { transform: translateY(0); } }
  .sheet-handle { width: 40px; height: 4px; background: var(--surface-3); border-radius: 2px; margin: 12px auto 0; }
  .sheet-title { font-size: 18px; font-weight: 700; color: var(--text); padding: 14px 22px 4px; }
  .sheet-tabs { display: flex; padding: 0 22px; border-bottom: 1px solid var(--border-soft); margin-top: 8px; }
  .sheet-tab {
    flex: 1; background: none; border: none; cursor: pointer;
    padding: 12px 4px; font-size: 14px; font-weight: 600;
    color: var(--text-mute); font-family: inherit;
    border-bottom: 2px solid transparent;
    transition: color .2s, border-color .2s;
  }
  .sheet-tab.active { color: var(--indigo); border-bottom-color: var(--indigo); }
  .sheet-pane { padding: 20px 22px; display: none; flex-direction: column; gap: 16px; }
  .sheet-pane.active { display: flex; }

  .modal-inp, .modal-sel {
    width: 100%; box-sizing: border-box;
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 12px 14px; font-size: 14px; color: var(--text);
    outline: none; font-family: inherit;
    transition: border-color .2s;
  }
  .modal-inp:focus, .modal-sel:focus { border-color: var(--indigo); }
  .modal-inp::placeholder { color: var(--text-faint); }
  .modal-lbl { display: block; font-size: 11px; color: var(--text-mute); margin-bottom: 6px; font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase; }

  .chip-row { display: flex; gap: 6px; flex-wrap: wrap; }
  .chip {
    background: var(--bg); border: 1px solid var(--border);
    border-radius: 20px; padding: 6px 14px;
    color: var(--text-dim); font-size: 13px; font-weight: 600;
    cursor: pointer; font-family: inherit;
    transition: border-color .15s, color .15s, background .15s;
  }
  .chip:hover { color: var(--text); border-color: var(--surface-3); }
  .chip.active { background: var(--indigo-soft); border-color: var(--indigo); color: var(--indigo-hi); }

  .amount-big {
    font-family: 'JetBrains Mono', monospace;
    font-size: 44px; font-weight: 700; color: var(--text);
    letter-spacing: -0.03em; text-align: center;
    padding: 8px 0 4px;
  }
  .amount-big .dollar { color: var(--text-faint); margin-right: 4px; }

  .submit-btn {
    width: 100%; background: var(--indigo); color: #fff;
    border: none; border-radius: 12px; padding: 14px;
    font-size: 15px; font-weight: 700; cursor: pointer;
    font-family: inherit;
    transition: background .2s, transform .1s;
    margin-top: 4px;
  }
  .submit-btn:hover { background: #4f46e5; }
  .submit-btn:active { transform: scale(0.99); }
  .submit-btn.amber { background: var(--amber); }
  .submit-btn.amber:hover { background: #d97706; }
  .submit-btn.emerald { background: var(--emerald); }
  .submit-btn.emerald:hover { background: #059669; }

  .info-card {
    background: var(--bg);
    border: 1px solid var(--border-soft);
    border-radius: 12px;
    padding: 12px 14px;
    display: flex; justify-content: space-between; align-items: center;
    font-size: 13px;
  }
  .info-card .label { color: var(--text-mute); font-weight: 600; }
  .info-card .val { color: var(--emerald-hi); font-weight: 700; font-family: 'JetBrains Mono', monospace; }

  #toast {
    display: none;
    position: fixed; top: 72px; left: 50%; transform: translateX(-50%);
    background: var(--emerald-soft); border: 1px solid #166534; color: #4ade80;
    border-radius: 12px; padding: 11px 20px;
    font-size: 13px; font-weight: 600; font-family: inherit;
    z-index: 200; white-space: nowrap;
    box-shadow: 0 8px 24px rgba(0,0,0,0.4);
  }
  #toast.show { display: block; animation: fadeUp .2s both; }
  #toast.warn { background: var(--rose-soft); border-color: #7f1d1d; color: var(--rose-hi); }

  @media (min-width: 640px) {
    .content { padding: 28px 24px 96px; max-width: 760px; }
    .actions { gap: 14px; }
    .stat-grid { gap: 12px; }
    .hero { padding: 26px 28px 28px; border-radius: 24px; }
  }

  @media (min-width: 1024px) {
    .app-body { grid-template-columns: 240px 1fr; gap: 28px; padding: 0 20px; }
    .sidebar {
      display: block;
      padding: 28px 0;
      position: sticky; top: 60px;
      align-self: start;
    }
    .side-nav { display: flex; flex-direction: column; gap: 4px; }
    .side-link {
      display: flex; align-items: center; gap: 12px;
      width: 100%; padding: 10px 14px;
      background: none; border: none; border-radius: 10px;
      color: var(--text-dim); font-family: inherit; font-size: 14px; font-weight: 600;
      cursor: pointer; text-align: left;
      transition: background .15s, color .15s;
    }
    .side-link:hover { background: var(--surface-2); color: var(--text); }
    .side-link.active { background: var(--indigo-soft); color: var(--indigo-hi); }
    .side-link svg { flex-shrink: 0; }
    .side-promo {
      margin-top: 24px;
      background: linear-gradient(135deg, #1e1b4b 0%, #0f172a 100%);
      border: 1px solid #312e81;
      border-radius: 16px;
      padding: 18px;
    }
    .promo-title { font-size: 11px; color: var(--indigo-hi); font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase; }
    .promo-score { font-size: 32px; font-weight: 800; color: var(--text); margin-top: 6px; letter-spacing: -0.03em; }
    .promo-score .promo-small { font-size: 14px; color: var(--text-mute); font-weight: 600; margin-left: 4px; }
    .promo-sub { font-size: 12px; color: var(--text-dim); margin-top: 4px; }
    .content { padding: 28px 0 60px; max-width: none; margin: 0; }
    .tabbar { display: none; }
    .topbar-inner { padding: 14px 28px; }
  }
  .side-promo { display: none; }
  @media (min-width: 1024px) { .side-promo { display: block; } }
</style>
`
}

// ---------------------------------------------------------------------------
// CLIENT SCRIPT  (all actions are real API calls)
// ---------------------------------------------------------------------------
func dashboardScript() string {
	return `
<script>
  // Prevent the browser from restoring a stale scroll position on refresh,
  // which would push content up behind the sticky topbar.
  if (history.scrollRestoration) history.scrollRestoration = 'manual';
  window.scrollTo(0, 0);

  // --- View router --------------------------------------------------------
  function goView(name, subpane) {
    document.querySelectorAll('[data-view]').forEach(function(el) {
      el.classList.toggle('active', el.getAttribute('data-view') === name);
    });
    document.querySelectorAll('.tab[data-tab], .side-link[data-tab]').forEach(function(btn) {
      btn.classList.toggle('active', btn.getAttribute('data-tab') === name);
    });
    if (subpane && name === 'activity') {
      requestAnimationFrame(function() {
        var btn = document.querySelector('[data-pane="' + subpane + '"]');
        if (btn) btn.click();
      });
    }
    window.scrollTo({ top: 0, behavior: 'instant' });
  }

  // --- Activity sub-pane router ------------------------------------------
  function goPane(btn) {
    var target = btn.getAttribute('data-pane');
    document.querySelectorAll('.subtab').forEach(function(b) { b.classList.remove('active'); });
    btn.classList.add('active');
    document.querySelectorAll('[data-pane-content]').forEach(function(c) {
      c.style.display = c.getAttribute('data-pane-content') === target ? 'block' : 'none';
    });
  }

  // --- Sheets -------------------------------------------------------------
  function openSheet(which) {
    document.getElementById(which + '-sheet').classList.add('open');
  }
  function closeSheet(which) {
    document.getElementById(which + '-sheet').classList.remove('open');
  }
  function payTab(btn) {
    var target = btn.getAttribute('data-paytab');
    document.querySelectorAll('.sheet-tab').forEach(function(b) { b.classList.remove('active'); });
    btn.classList.add('active');
    document.querySelectorAll('[data-pay-pane]').forEach(function(p) {
      p.classList.toggle('active', p.getAttribute('data-pay-pane') === target);
    });
  }

  // --- Amount chip helpers ------------------------------------------------
  function setSendAmt(v) { document.getElementById('send-amt').textContent = v.toFixed(2); document.getElementById('pay-amount-inp').value = v; }
  function setReqAmt(v)  { document.getElementById('req-amt').textContent  = v.toFixed(2); document.getElementById('req-amount-inp').value = v; }

  // --- Open pay sheet pre-filled for a specific friend -------------------
  function openPayToFriend(friendID) {
    openSheet('pay');
    requestAnimationFrame(function() {
      var sel = document.getElementById('pay-receiver-sel');
      if (sel) sel.value = friendID;
    });
  }

  // --- Toast --------------------------------------------------------------
  var toastTimer = null;
  function showToast(msg, kind) {
    var t = document.getElementById('toast');
    t.textContent = msg;
    t.className = kind === 'warn' ? 'show warn' : 'show';
    clearTimeout(toastTimer);
    toastTimer = setTimeout(function() { t.className = ''; }, 2400);
  }

  // --- API helper ---------------------------------------------------------
  function apiPost(url, body, onOk, onErr) {
    fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'same-origin',
      body: JSON.stringify(body)
    })
    .then(function(res) {
      return res.json().then(function(d) { return { ok: res.ok, data: d }; });
    })
    .then(function(r) {
      if (r.ok) { onOk(r.data); } else { onErr(r.data.error || 'Request failed'); }
    })
    .catch(function() { onErr('Network error'); });
  }

  // --- Send payment -------------------------------------------------------
  function submitSendPayment() {
    var receiverID = document.getElementById('pay-receiver-sel').value;
    var amount     = parseFloat(document.getElementById('pay-amount-inp').value || '0');
    var note       = document.getElementById('pay-note-inp').value;
    if (!receiverID) { showToast('Please select a recipient', 'warn'); return; }
    if (amount <= 0) { showToast('Please enter a valid amount', 'warn'); return; }
    apiPost('/payments/pay',
      { receiver_id: receiverID, amount: amount, note: note, payment_type: 'direct' },
      function() { closeSheet('pay'); showToast('Payment sent'); setTimeout(function() { location.reload(); }, 1200); },
      function(e) { showToast(e, 'warn'); }
    );
  }

  // --- Create BNPL loan ---------------------------------------------------
  function submitBNPL() {
    var receiverID = document.getElementById('bnpl-receiver-sel').value;
    var amount     = parseFloat(document.getElementById('bnpl-amount-inp').value || '0');
    var note       = document.getElementById('bnpl-note-inp').value;
    var plan       = parseInt(document.getElementById('bnpl-plan-sel').value || '4', 10);
    if (!receiverID) { showToast('Please select a recipient', 'warn'); return; }
    if (amount <= 0) { showToast('Please enter a valid amount', 'warn'); return; }
    apiPost('/bnpl/loan/create',
      { receiver_id: receiverID, amount: amount, note: note, payment_type: 'bnpl', total_installments: plan },
      function() { closeSheet('pay'); showToast('BNPL plan approved'); setTimeout(function() { location.reload(); }, 1200); },
      function(e) { showToast(e, 'warn'); }
    );
  }

  // --- Request money ------------------------------------------------------
  function submitRequest() {
    var payerID = document.getElementById('req-payer-sel').value;
    var amount  = parseFloat(document.getElementById('req-amount-inp').value || '0');
    var note    = document.getElementById('req-note-inp').value;
    if (!payerID) { showToast('Please select a friend', 'warn'); return; }
    if (amount <= 0) { showToast('Please enter a valid amount', 'warn'); return; }
    apiPost('/payments/request/create',
      { payer_id: payerID, amount: amount, note: note },
      function() { closeSheet('request'); showToast('Request sent'); },
      function(e) { showToast(e, 'warn'); }
    );
  }

  // --- Add friend: live profile search -----------------------------------
  var _allProfiles = null;
  var _profilesLoading = false;

  function initProfileSearch() {
    if (_allProfiles !== null || _profilesLoading) return;
    _profilesLoading = true;
    document.getElementById('addfriend-results').innerHTML =
      '<div style="color:var(--text-mute);font-size:13px;padding:8px 0;">Loading users…</div>';
    fetch('/profiles/list', { credentials: 'same-origin' })
      .then(function(r) { return r.json(); })
      .then(function(data) {
        _allProfiles = Array.isArray(data) ? data : [];
        _profilesLoading = false;
        searchUsersToAdd(document.getElementById('addfriend-search-inp').value);
      })
      .catch(function() {
        _profilesLoading = false;
        document.getElementById('addfriend-results').innerHTML =
          '<div style="color:var(--rose-hi);font-size:13px;padding:8px 0;">Could not load users.</div>';
      });
  }

  function searchUsersToAdd(q) {
    var container = document.getElementById('addfriend-results');
    if (!_allProfiles) { initProfileSearch(); return; }
    q = (q || '').toLowerCase().trim();
    var colors = ['av-indigo','av-amber','av-emerald','av-violet','av-cyan','av-pink'];
    var matches = _allProfiles.filter(function(p) {
      if (p.id === window._selfID) return false;
      if (window._friendIDs && window._friendIDs.has(p.id)) return false;
      if (!q) return true;
      var qDigits = q.replace(/\D/g, '');
      var phoneDigits = (p.phone_number || '').replace(/\D/g, '');
      return (p.id || '').toLowerCase().indexOf(q) !== -1 ||
             (p.name || '').toLowerCase().indexOf(q) !== -1 ||
             (qDigits.length >= 3 && phoneDigits.indexOf(qDigits) !== -1);
    }).slice(0, 10);
    if (matches.length === 0) {
      container.innerHTML = '<div style="color:var(--text-mute);font-size:13px;padding:8px 0;">No users found.</div>';
      return;
    }
    container.innerHTML = matches.map(function(p, i) {
      var dname = p.name || p.id;
      var r = Array.from(dname);
      var ini = r.length >= 2 ? (r[0] + r[1]).toUpperCase() : r[0].toUpperCase();
      var cls = colors[i % colors.length];
      var safeName = dname.replace(/\\/g, '').replace(/'/g, '');
      return '<div style="display:flex;align-items:center;gap:12px;padding:10px 12px;background:var(--bg);border:1px solid var(--border);border-radius:12px;">' +
        '<div class="row-avatar ' + cls + '" style="width:36px;height:36px;font-size:12px;flex-shrink:0;">' + ini + '</div>' +
        '<div style="flex:1;min-width:0;">' +
          '<div style="font-size:14px;font-weight:600;color:var(--text);">' + dname + '</div>' +
          '<div style="font-size:12px;color:var(--text-faint);">@' + p.id + '</div>' +
        '</div>' +
        '<button onclick="sendFriendRequestTo(\'' + p.id + '\',\'' + safeName + '\')" ' +
          'style="background:var(--indigo);color:#fff;border:none;border-radius:8px;padding:6px 12px;font-size:12px;font-weight:700;cursor:pointer;font-family:inherit;flex-shrink:0;">Add</button>' +
      '</div>';
    }).join('');
  }

  function sendFriendRequestTo(id, name) {
    apiPost('/friends/request/send',
      { receiver_id: id },
      function() {
        closeSheet('addfriend');
        document.getElementById('addfriend-search-inp').value = '';
        document.getElementById('addfriend-results').innerHTML = '';
        showToast('Friend request sent to ' + name);
      },
      function(e) { showToast(e, 'warn'); }
    );
  }

  // Reset search state when sheet closes so it re-fetches on next open
  var _origCloseSheet = closeSheet;
  closeSheet = function(which) {
    _origCloseSheet(which);
    if (which === 'addfriend') {
      var inp = document.getElementById('addfriend-search-inp');
      var res = document.getElementById('addfriend-results');
      if (inp) inp.value = '';
      if (res) res.innerHTML = '';
    }
  };

  // --- Notifications ------------------------------------------------------
  function dismissNotif(notifID, linkView, row) {
    apiPost('/notifications/seen', { notif_id: notifID },
      function() {
        row.style.transition = 'opacity .2s, transform .2s';
        row.style.opacity = '0';
        row.style.transform = 'translateX(12px)';
        setTimeout(function() {
          row.remove();
          var list = document.getElementById('notif-list');
          if (list && list.querySelectorAll('.row').length === 0) {
            list.outerHTML = '<div class="empty"><div class="empty-icon success"><svg width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg></div><div class="empty-title">All caught up</div><div class="empty-sub">No new notifications.</div></div>';
          }
          // Update bell badge count
          var badge = document.querySelector('.icon-btn span[style*="ef4444"]');
          if (badge) {
            var cur = parseInt(badge.textContent || '1', 10) - 1;
            if (cur <= 0) { badge.remove(); }
            else { badge.textContent = cur; }
          }
        }, 200);
        if (linkView) { goView(linkView); }
      },
      function(e) { showToast(e, 'warn'); }
    );
  }

  function clearAllNotifs() {
    apiPost('/notifications/clear', {},
      function() {
        var list = document.getElementById('notif-list');
        if (list) {
          list.outerHTML = '<div class="empty"><div class="empty-icon success"><svg width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg></div><div class="empty-title">All caught up</div><div class="empty-sub">No new notifications.</div></div>';
        }
        var badge = document.querySelector('.icon-btn span[style*="ef4444"]');
        if (badge) badge.remove();
        document.querySelector('[data-tab="notifications"] span[style*="ef4444"]') && document.querySelector('[data-tab="notifications"] span[style*="ef4444"]').remove();
        showToast('All notifications cleared');
      },
      function(e) { showToast(e, 'warn'); }
    );
  }

  // --- Accept / Decline friend requests -----------------------------------
  function acceptFriendRequest(requestID, senderID) {
    apiPost('/friends/request/accept',
      { request_id: requestID, sender_id: senderID },
      function() { showToast('Friend request accepted'); setTimeout(function() { location.reload(); }, 800); },
      function(e) { showToast(e, 'warn'); }
    );
  }

  function declineFriendRequest(requestID, btn) {
    apiPost('/friends/request/decline',
      { request_id: requestID },
      function() {
        var row = document.querySelector('[data-req-row="' + requestID + '"]');
        if (row) {
          row.style.transition = 'opacity .2s';
          row.style.opacity = '0';
          setTimeout(function() { row.remove(); }, 200);
        }
        showToast('Request declined');
      },
      function(e) { showToast(e, 'warn'); }
    );
  }

  // --- Remove friend (optimistic UI + real API) ---------------------------
  function removeFriend(btn, friendID, name) {
    apiPost('/friends/remove',
      { friend_id: friendID },
      function() {
        var row = btn.closest('.friend-row');
        row.style.transition = 'opacity .2s, transform .2s';
        row.style.opacity = '0';
        row.style.transform = 'translateX(-20px)';
        setTimeout(function() {
          row.remove();
          showToast(name + ' removed');
          var remaining = document.querySelectorAll('#friend-list .friend-row').length;
          if (remaining === 0) {
            document.getElementById('friend-list').style.display = 'none';
            document.getElementById('friends-empty').style.display = 'block';
          }
        }, 200);
      },
      function(e) { showToast(e, 'warn'); }
    );
  }

  // --- Pay single installment ---------------------------------------------
  function payInstallment(installmentID, paymentID, amount) {
    apiPost('/bnpl/installment/pay',
      { installment_id: installmentID, payment_id: paymentID, amount: amount },
      function() { showToast('Installment paid'); setTimeout(function() { location.reload(); }, 1200); },
      function(e) { showToast(e, 'warn'); }
    );
  }

  // --- Pay all overdue installments ---------------------------------------
  function payAllOverdue() {
    var rows = document.querySelectorAll('[data-installment-id]');
    if (!rows.length) { showToast('No overdue installments'); return; }
    var pending = rows.length, failed = 0;
    rows.forEach(function(row) {
      var iid = row.getAttribute('data-installment-id');
      var pid = row.getAttribute('data-payment-id');
      var amt = parseFloat(row.getAttribute('data-amount') || '0');
      fetch('/bnpl/installment/pay', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
        body: JSON.stringify({ installment_id: iid, payment_id: pid, amount: amt })
      })
      .then(function(res) { if (!res.ok) failed++; })
      .catch(function() { failed++; })
      .finally(function() {
        pending--;
        if (pending === 0) {
          if (failed > 0) { showToast(failed + ' payment(s) failed', 'warn'); }
          else { showToast('All overdue installments paid'); }
          setTimeout(function() { location.reload(); }, 1200);
        }
      });
    });
  }

  // --- Friends search filter ----------------------------------------------
  function filterFriends(q) {
    q = (q || '').toLowerCase().trim();
    var rows = document.querySelectorAll('#friend-list .friend-row');
    var visible = 0;
    rows.forEach(function(r) {
      var match = !q || r.getAttribute('data-name').indexOf(q) !== -1;
      r.style.display = match ? '' : 'none';
      if (match) visible++;
    });
    document.getElementById('friend-list').style.opacity = (visible === 0 && q) ? '0.4' : '1';
  }

  // --- Sign out -----------------------------------------------------------
  function signOut() {
    fetch('/users/logout', { method: 'POST', credentials: 'same-origin' })
      .then(function() { location.reload(); })
      .catch(function() {
        document.cookie = 'session_user_id=; Max-Age=0; path=/';
        location.reload();
      });
  }

  // --- Esc closes any open sheet ------------------------------------------
  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') {
      closeSheet('pay'); closeSheet('request'); closeSheet('addfriend');
    }
  });
</script>
`
}
