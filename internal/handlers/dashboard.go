package handlers

import (
	"fmt"
	"net/http"

	"github.com/connorpodea/splitit/internal/models"
)

// GetInitialView serves the root page: renders the login screen for unauthenticated or
// deactivated users, and the full dashboard for valid active sessions.
func (h *Handler) GetInitialView(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_user_id")
	if err != nil || cookie.Value == "" {
		writeHTML(w, loginHTML())
		return
	}
	user, err := h.store.GetUser(cookie.Value)
	if err != nil || !user.IsActive {
		writeHTML(w, loginHTML())
		return
	}

	// Dynamic database queries with explicit error recovery fallbacks
	friends, err := h.store.ListFriends(user.ID)
	if err != nil {
		friends = []models.Profile{}
	}
	installments, err := h.store.ListInstallments(user.ID)
	if err != nil {
		installments = []models.Installment{}
	}
	overdueInstallments, err := h.store.ListOverdueInstallments(user.ID)
	if err != nil {
		overdueInstallments = []models.Installment{}
	}
	incomingRequests, err := h.store.ListIncomingPaymentRequests(user.ID)
	if err != nil {
		incomingRequests = []models.PaymentRequest{}
	}
	friendRequests, err := h.store.ListIncomingFriendRequests(user.ID)
	if err != nil {
		friendRequests = []models.FriendRequest{}
	}
	notifications, err := h.store.ListNotifications(user.ID)
	if err != nil {
		notifications = []models.Notification{}
	}

	writeHTML(w, dashboardHTML(user, friends, installments, overdueInstallments, incomingRequests, friendRequests, notifications))
}

// GetDashboardView re-renders the full dashboard HTML for an authenticated active user,
// used by HTMX partial swaps after state-changing actions.
func (h *Handler) GetDashboardView(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_user_id")
	if err != nil || cookie.Value == "" {
		writeHTML(w, loginHTML())
		return
	}
	user, err := h.store.GetUser(cookie.Value)
	if err != nil || !user.IsActive {
		writeHTML(w, loginHTML())
		return
	}

	friends, err := h.store.ListFriends(user.ID)
	if err != nil {
		friends = []models.Profile{}
	}
	installments, err := h.store.ListInstallments(user.ID)
	if err != nil {
		installments = []models.Installment{}
	}
	overdueInstallments, err := h.store.ListOverdueInstallments(user.ID)
	if err != nil {
		overdueInstallments = []models.Installment{}
	}
	incomingRequests, err := h.store.ListIncomingPaymentRequests(user.ID)
	if err != nil {
		incomingRequests = []models.PaymentRequest{}
	}
	friendRequests, err := h.store.ListIncomingFriendRequests(user.ID)
	if err != nil {
		friendRequests = []models.FriendRequest{}
	}
	notifications, err := h.store.ListNotifications(user.ID)
	if err != nil {
		notifications = []models.Notification{}
	}

	writeHTML(w, dashboardHTML(user, friends, installments, overdueInstallments, incomingRequests, friendRequests, notifications))
}

// dashboardHTML assembles the complete dashboard shell HTML including the topbar, sidebar,
// all view sections, the bottom tab bar, action sheets, and the embedded client script.
func dashboardHTML(user *models.User, friends []models.Profile, installments []models.Installment, overdueInstallments []models.Installment, incomingRequests []models.PaymentRequest, friendRequests []models.FriendRequest, notifications []models.Notification) string {
	name := displayName(user)
	avatar := initials(name)
	handle := user.ID
	email := user.Email
	phone := user.PhoneNumber
	if email == "" {
		email = handle + "@splitit.app"
	}

	var outstandingBNPLCents int
	for _, inst := range installments {
		if !inst.IsPaid {
			outstandingBNPLCents += inst.AmountCents
		}
	}
	availableCreditCents := user.CreditLimitCents - outstandingBNPLCents
	if availableCreditCents < 0 {
		availableCreditCents = 0
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
        <div class="promo-sub">` + scoreLabel + ` · BNPL limit $` + fmt.Sprintf("%d", user.CreditLimitCents/100) + `</div>
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

` + paySheetHTML(friends, availableCreditCents) + `
` + requestSheetHTML(friends) + `
` + addFriendSheetHTML() + `
` + sessionContextScript(handle, friends) + `
` + dashboardScript() + `
</div>`
}

// dashboardStyles returns the full embedded CSS stylesheet for the dashboard shell as a style tag string.
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
// dashboardScript returns the embedded JavaScript for all client-side dashboard interactions
// including view routing, API calls, sheet management, and real-time UI updates.
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
    var receiverID   = document.getElementById('pay-receiver-sel').value;
    var amountCents  = Math.round(parseFloat(document.getElementById('pay-amount-inp').value || '0') * 100);
    var note         = document.getElementById('pay-note-inp').value;
    if (!receiverID) { showToast('Please select a recipient', 'warn'); return; }
    if (amountCents <= 0) { showToast('Please enter a valid amount', 'warn'); return; }
    apiPost('/payments/pay',
      { receiver_id: receiverID, amount_cents: amountCents, note: note, payment_type: 'direct' },
      function() { closeSheet('pay'); showToast('Payment sent'); setTimeout(function() { location.reload(); }, 1200); },
      function(e) { showToast(e, 'warn'); }
    );
  }

  // --- Create BNPL loan ---------------------------------------------------
  function submitBNPL() {
    var receiverID  = document.getElementById('bnpl-receiver-sel').value;
    var amountCents = Math.round(parseFloat(document.getElementById('bnpl-amount-inp').value || '0') * 100);
    var note        = document.getElementById('bnpl-note-inp').value;
    var plan        = parseInt(document.getElementById('bnpl-plan-sel').value || '4', 10);
    if (!receiverID) { showToast('Please select a recipient', 'warn'); return; }
    if (amountCents <= 0) { showToast('Please enter a valid amount', 'warn'); return; }
    apiPost('/bnpl/loan/create',
      { receiver_id: receiverID, amount_cents: amountCents, note: note, payment_type: 'bnpl', total_installments: plan },
      function() { closeSheet('pay'); showToast('BNPL plan approved'); setTimeout(function() { location.reload(); }, 1200); },
      function(e) { showToast(e, 'warn'); }
    );
  }

  // --- Request money ------------------------------------------------------
  function submitRequest() {
    var payerID     = document.getElementById('req-payer-sel').value;
    var amountCents = Math.round(parseFloat(document.getElementById('req-amount-inp').value || '0') * 100);
    var note        = document.getElementById('req-note-inp').value;
    if (!payerID) { showToast('Please select a friend', 'warn'); return; }
    if (amountCents <= 0) { showToast('Please enter a valid amount', 'warn'); return; }
    apiPost('/payments/request/create',
      { payer_id: payerID, amount_cents: amountCents, note: note },
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
  function acceptFriendRequest(requestID) {
    apiPost('/friends/request/accept',
      { request_id: requestID },
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
  function payInstallment(installmentID, paymentID, amountCents) {
    apiPost('/bnpl/installment/pay',
      { installment_id: installmentID, payment_id: paymentID, amount_cents: amountCents },
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
      var iid        = row.getAttribute('data-installment-id');
      var pid        = row.getAttribute('data-payment-id');
      var amtCents   = parseInt(row.getAttribute('data-amount-cents') || '0', 10);
      fetch('/bnpl/installment/pay', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
        body: JSON.stringify({ installment_id: iid, payment_id: pid, amount_cents: amtCents })
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
