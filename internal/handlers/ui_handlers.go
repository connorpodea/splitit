package handlers

import (
	"net/http"
	"strings"

	"github.com/connorpodea/splitit/internal/models"
)

// GetInitialView is the entry route the master shell hits on page load.
// It checks the session cookie and either returns the login form or the dashboard.
func (h *Handler) GetInitialView(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_user_id")
	if err != nil || cookie.Value == "" {
		writeHTML(w, loginHTML())
		return
	}

	user, err := h.store.GetUser(cookie.Value)
	if err != nil {
		// Session points at a missing user (e.g. db reset). Fall back to login.
		writeHTML(w, loginHTML())
		return
	}
	writeHTML(w, dashboardHTML(user))
}

// GetRegistrationView renders the standalone signup screen.
func (h *Handler) GetRegistrationView(w http.ResponseWriter, r *http.Request) {
	writeHTML(w, registerHTML())
}

// GetDashboardView is hit after a successful login.
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
	writeHTML(w, dashboardHTML(user))
}

// writeHTML is a tiny helper so each handler isn't 4 boilerplate lines.
func writeHTML(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(body))
}

// initials derives "JD" from "Jane Doe" — first letter of first name,
// first letter of last name. Single-word names fall back to the first two
// characters. Empty inputs return a neutral "?".
func initials(fullName string) string {
	clean := strings.TrimSpace(fullName)
	if clean == "" {
		return "?"
	}
	parts := strings.Fields(clean)
	if len(parts) == 1 {
		// Single-word name: take first two characters if available.
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

// displayName picks the most natural label for a user — their real name if set,
// otherwise their handle.
func displayName(u *models.User) string {
	if strings.TrimSpace(u.Name) != "" {
		return u.Name
	}
	return u.ID
}

// ---------------------------------------------------------------------------
// LOGIN SCREEN
// Kept close to the original since it tested well — small polish only.
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
    <div style="font-size: 34px; font-weight: 800; letter-spacing: -0.04em; color: #f8fafc; margin-bottom: 4px;">
      split<span style="color: #6366f1;">it</span>
    </div>
    <div style="font-size: 13px; color: #64748b;">Send money. Split bills. Pay later.</div>
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
    <div style="font-size: 34px; font-weight: 800; letter-spacing: -0.04em; color: #f8fafc; margin-bottom: 4px;">split<span style="color: #6366f1;">it</span></div>
    <div style="font-size: 13px; color: #64748b;">Join millions splitting smarter</div>
  </div>

  <div style="background: #0f172a; border: 1px solid #1e293b; border-radius: 20px; padding: 28px 24px;">
    <div style="font-size: 18px; font-weight: 700; color: #f1f5f9; margin-bottom: 6px;">Create your account</div>
    <div style="font-size: 13px; color: #64748b; margin-bottom: 24px;">Free to join. No hidden fees.</div>

    <form id="reg-form"
          hx-post="/users/create"
          hx-ext="json-enc"
          hx-target="#reg-msg"
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
      <div style="display:grid; grid-template-columns:1fr 1fr; gap:12px;">
        <div><label class="lbl">User ID</label><input class="inp-plain" type="text" name="id" placeholder="your_handle" required /></div>
        <div><label class="lbl">Full name</label><input class="inp-plain" type="text" name="name" placeholder="Jane Doe" required /></div>
      </div>
      <div><label class="lbl">Email</label><input class="inp-plain" type="email" name="email" placeholder="you@email.com" required /></div>
      <div><label class="lbl">Phone</label><input class="inp-plain" type="tel" name="phone_number" placeholder="555-0199" required /></div>
      <div><label class="lbl">Password</label><input class="inp-plain" type="password" name="password" placeholder="Min 8 characters" required /></div>
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
// DASHBOARD
// Polished, responsive, multi-view shell that addresses the feedback:
//   - Two main actions only (Pay, Request). BNPL is a tab inside Pay.
//   - Splitit Score (renamed from Credit Score) lives on the Profile view,
//     not the main dashboard — Home stays uncluttered.
//   - Real bottom tab bar on mobile, sidebar on desktop.
//   - Working sign-out via /users/logout.
//   - Profile button opens the Profile tab (friend count + scrollable list).
//   - Avatar uses FL initials from the user's real name.
//   - Realistic mock data for activity, payments, installments, friends.
//   - Empty states wired up for overdue, friends, and activity.
//
// ---------------------------------------------------------------------------
func dashboardHTML(user *models.User) string {
	name := displayName(user)
	avatar := initials(name)
	handle := user.ID
	email := user.Email
	if email == "" {
		email = handle + "@splitit.app"
	}

	return dashboardStyles() + `
<div id="app-root" class="app-shell">
  <div id="toast"></div>

  <!-- ============== TOP BAR ============== -->
  <header class="topbar">
    <div class="topbar-inner">
      <button class="brand-btn" data-tab="home" onclick="goView('home')" aria-label="Home">
        <span class="brand">split<span>it</span></span>
      </button>
      <div class="topbar-actions">
        <button class="icon-btn" aria-label="Notifications" onclick="showToast('No new notifications')">
          <svg width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>
          <span class="dot"></span>
        </button>
        <button class="avatar-btn" id="profile-btn" onclick="goView('profile')" aria-label="Profile">` + avatar + `</button>
      </div>
    </div>
  </header>

  <div class="app-body">
    <!-- ============== SIDEBAR (desktop only) ============== -->
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
        <button class="side-link" data-tab="friends" onclick="goView('friends')">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
          Friends
        </button>
        <button class="side-link" data-tab="profile" onclick="goView('profile')">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
          Profile
        </button>
      </nav>
      <div class="side-promo">
        <div class="promo-title">splitit Score</div>
        <div class="promo-score"><span class="mono">76</span><span class="promo-small">/100</span></div>
        <div class="promo-sub">Excellent · BNPL limit $1,500</div>
      </div>
    </aside>

    <!-- ============== MAIN CONTENT ============== -->
    <main class="content">
` + viewHome(name) + `
` + viewActivity() + `
` + viewFriends() + `
` + viewProfile(user, avatar, name, handle, email) + `
    </main>
  </div>

  <!-- ============== BOTTOM TAB BAR (mobile/tablet) ============== -->
  <nav class="tabbar">
    <button class="tab active" data-tab="home" onclick="goView('home')">
      <svg width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M3 12l9-9 9 9"/><path d="M5 10v10h14V10"/></svg>
      <span>Home</span>
    </button>
    <button class="tab" data-tab="activity" onclick="goView('activity')">
      <svg width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M3 12h4l3-9 4 18 3-9h4"/></svg>
      <span>Activity</span>
    </button>
    <button class="tab" data-tab="friends" onclick="goView('friends')">
      <svg width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
      <span>Friends</span>
    </button>
    <button class="tab" data-tab="profile" onclick="goView('profile')">
      <svg width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
      <span>Profile</span>
    </button>
  </nav>

` + paySheetHTML() + `
` + requestSheetHTML() + `
` + dashboardScript() + `
</div>`
}

// dashboardStyles bundles all CSS for the dashboard into a single <style> block.
// Splitting CSS across files would force a Tailwind/build step which this server
// intentionally doesn't have.
func dashboardStyles() string {
	return `
<style>
  @import url('https://fonts.googleapis.com/css2?family=Onest:wght@400;500;600;700;800&family=JetBrains+Mono:wght@500;600;700&display=swap');

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

  /* Reset on the master canvas to get rid of the default p-4 flex centering */
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

  /* --------- TOPBAR --------- */
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
  .brand { font-size: 22px; font-weight: 800; letter-spacing: -0.04em; color: var(--text); }
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

  /* --------- BODY GRID (sidebar + content) --------- */
  .app-body {
    max-width: 1280px; margin: 0 auto;
    display: grid; grid-template-columns: 1fr;
    min-height: calc(100vh - 60px);
  }
  .sidebar { display: none; }

  /* --------- CONTENT --------- */
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
  @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }

  /* --------- BALANCE HERO --------- */
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

  /* --------- ACTION ROW (2 buttons: Pay, Request) --------- */
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

  /* --------- QUICK STATS STRIP --------- */
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

  /* --------- SECTION LABEL --------- */
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

  /* --------- CARD + LIST --------- */
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

  /* Avatar color presets (used inline by data) */
  .av-indigo { background: var(--indigo-soft); color: var(--indigo-hi); }
  .av-amber  { background: #1c0a00; color: var(--amber-hi); }
  .av-emerald { background: var(--emerald-soft); color: var(--emerald-hi); }
  .av-rose { background: var(--rose-soft); color: var(--rose-hi); }
  .av-violet { background: #1a0533; color: #c084fc; }
  .av-cyan { background: #042f2e; color: #5eead4; }
  .av-pink { background: #4a044e; color: #f0abfc; }
  .av-slate { background: var(--surface-2); color: var(--text-dim); border: 1px solid var(--border); }

  /* --------- ACTIVITY VIEW (sub-tabs) --------- */
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

  /* --------- OVERDUE TREATMENT --------- */
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
  .pill.muted { background: var(--surface-3); color: var(--text-dim); }

  .progress {
    height: 4px; border-radius: 2px;
    background: var(--surface-3); overflow: hidden;
    margin-top: 8px;
  }
  .progress > span { display: block; height: 100%; background: var(--indigo); border-radius: 2px; }
  .progress.warn > span { background: var(--rose); }

  /* --------- FRIENDS VIEW --------- */
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

  /* --------- PROFILE VIEW --------- */
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

  /* --------- EMPTY STATES --------- */
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

  /* --------- BOTTOM TAB BAR --------- */
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

  /* --------- SHEETS (Pay + Request) --------- */
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

  /* --------- TOAST --------- */
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

  /* ===================================================================
     RESPONSIVE BREAKPOINTS
     - Mobile (default): single column, bottom tab bar.
     - Tablet (≥ 640px): wider content, 2-col stat layouts.
     - Desktop (≥ 1024px): left sidebar, no bottom tab bar.
     =================================================================== */
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
// HOME VIEW — balance hero, two primary actions, quick stats, recent activity.
// Splitit Score is intentionally absent here (lives on Profile per the IA brief).
// ---------------------------------------------------------------------------
func viewHome(name string) string {
	firstName := strings.Fields(name)
	greet := "there"
	if len(firstName) > 0 {
		greet = firstName[0]
	}
	return `
<section class="view active" data-view="home">
  <div class="greeting" style="font-size:14px; color:var(--text-mute); margin-bottom:14px;">Welcome back, <b style="color:var(--text);">` + greet + `</b></div>

  <!-- Balance hero -->
  <div class="hero">
    <div class="hero-label">Available balance</div>
    <div class="hero-amount mono">$1,247<span class="cents">.30</span></div>
    <div class="hero-meta">
      <span><b>$696.00</b> outstanding</span>
      <span style="opacity:.4;">·</span>
      <span><b>3</b> active splits</span>
    </div>
  </div>

  <!-- Primary actions (just two now: Pay and Request). BNPL lives inside Pay. -->
  <div class="actions">
    <button class="action pay" onclick="openSheet('pay')">
      <span class="ico"><svg width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg></span>
      <span class="lbl-stack"><b>Pay</b><small>Send · BNPL</small></span>
    </button>
    <button class="action req" onclick="openSheet('request')">
      <span class="ico"><svg width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M12 5v14"/><path d="M5 12h14"/></svg></span>
      <span class="lbl-stack"><b>Request</b><small>Ask a friend</small></span>
    </button>
  </div>

  <!-- Quick stats: gently surface urgency (overdue is red) -->
  <div class="quick-strip">
    <button class="qstat warn" onclick="goView('activity', 'overdue')">
      <div>
        <div class="qstat-label">Overdue</div>
        <div class="qstat-val mono">2 bills</div>
      </div>
      <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M9 18l6-6-6-6"/></svg>
    </button>
    <button class="qstat" onclick="goView('activity', 'active')">
      <div>
        <div class="qstat-label">Due this week</div>
        <div class="qstat-val mono">$127.25</div>
      </div>
      <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M9 18l6-6-6-6"/></svg>
    </button>
  </div>

  <!-- Recent activity (preview of 5; full list in Activity tab) -->
  <div class="section-row">
    <h2>Recent activity</h2>
    <button class="linklike" onclick="goView('activity')">See all</button>
  </div>
  <div class="card">
    <div class="row">
      <div class="row-avatar av-indigo">MH</div>
      <div class="row-body">
        <div class="row-title"><b>Marcus</b> paid <b>you</b></div>
        <div class="row-sub">Brewery tab</div>
      </div>
      <div class="row-right"><div class="row-amt pos mono">+$42.50</div><div class="row-time">18m ago</div></div>
    </div>
    <div class="row">
      <div class="row-avatar av-amber">PR</div>
      <div class="row-body">
        <div class="row-title"><b>Priya</b> requested from <b>you</b></div>
        <div class="row-sub">Lyft to airport</div>
      </div>
      <div class="row-right"><div class="row-amt req mono">$28.75</div><div class="row-time">2h ago</div></div>
    </div>
    <div class="row">
      <div class="row-avatar av-emerald">SP</div>
      <div class="row-body">
        <div class="row-title"><b>You</b> paid <b>Sweetgreen</b></div>
        <div class="row-sub">Lunch order</div>
      </div>
      <div class="row-right"><div class="row-amt neg mono">$18.40</div><div class="row-time">Yesterday</div></div>
    </div>
    <div class="row">
      <div class="row-avatar av-violet">EK</div>
      <div class="row-body">
        <div class="row-title"><b>Elena</b> paid <b>you</b></div>
        <div class="row-sub">Coffee + bagels</div>
      </div>
      <div class="row-right"><div class="row-amt pos mono">+$19.25</div><div class="row-time">Yesterday</div></div>
    </div>
    <div class="row">
      <div class="row-avatar av-cyan">BS</div>
      <div class="row-body">
        <div class="row-title"><b>BNPL</b> · Bose QC Ultra <span class="pill">Pay-in-4</span></div>
        <div class="row-sub">Installment 2 of 4 charged</div>
      </div>
      <div class="row-right"><div class="row-amt bnpl mono">$94.75</div><div class="row-time">3d ago</div></div>
    </div>
  </div>
</section>
`
}

// ---------------------------------------------------------------------------
// ACTIVITY VIEW — sub-tabs for Payments / Active installments / Overdue.
// Overdue uses warning treatment; empty states render a friendly success card.
// ---------------------------------------------------------------------------
func viewActivity() string {
	return `
<section class="view" data-view="activity">
  <div class="section-row" style="margin-top:0;">
    <h2 style="font-size:22px; font-weight:800; letter-spacing:-0.02em;">Activity</h2>
  </div>

  <div class="subtabs" role="tablist">
    <button class="subtab active" data-pane="all" onclick="goPane(this)">Payments</button>
    <button class="subtab" data-pane="active" onclick="goPane(this)">Active</button>
    <button class="subtab" data-pane="overdue" onclick="goPane(this)">Overdue <span class="badge mono">2</span></button>
  </div>

  <!-- ============ PAYMENTS PANE ============ -->
  <div data-pane-content="all" class="pane-content">
    <div class="card">
      <div class="row">
        <div class="row-avatar av-indigo">MH</div>
        <div class="row-body"><div class="row-title"><b>Marcus Holloway</b> paid you</div><div class="row-sub">Brewery tab</div></div>
        <div class="row-right"><div class="row-amt pos mono">+$42.50</div><div class="row-time">18m ago</div></div>
      </div>
      <div class="row">
        <div class="row-avatar av-amber">PR</div>
        <div class="row-body"><div class="row-title"><b>Priya Raman</b> requested</div><div class="row-sub">Lyft to airport · Pending</div></div>
        <div class="row-right"><div class="row-amt req mono">$28.75</div><div class="row-time">2h ago</div></div>
      </div>
      <div class="row">
        <div class="row-avatar av-emerald">SG</div>
        <div class="row-body"><div class="row-title">You paid <b>Sweetgreen</b></div><div class="row-sub">Lunch order</div></div>
        <div class="row-right"><div class="row-amt neg mono">$18.40</div><div class="row-time">Yesterday</div></div>
      </div>
      <div class="row">
        <div class="row-avatar av-violet">EK</div>
        <div class="row-body"><div class="row-title"><b>Elena Kowalski</b> paid you</div><div class="row-sub">Coffee + bagels</div></div>
        <div class="row-right"><div class="row-amt pos mono">+$19.25</div><div class="row-time">Yesterday</div></div>
      </div>
      <div class="row">
        <div class="row-avatar av-cyan">DF</div>
        <div class="row-body"><div class="row-title">You paid <b>Diego Fernández</b></div><div class="row-sub">Concert tickets · split 3 ways</div></div>
        <div class="row-right"><div class="row-amt neg mono">$156.00</div><div class="row-time">2d ago</div></div>
      </div>
      <div class="row">
        <div class="row-avatar av-emerald">BO</div>
        <div class="row-body"><div class="row-title"><b>BNPL</b> · Bose QC Ultra</div><div class="row-sub">Installment charged</div></div>
        <div class="row-right"><div class="row-amt bnpl mono">$94.75</div><div class="row-time">3d ago</div></div>
      </div>
      <div class="row">
        <div class="row-avatar av-pink">AM</div>
        <div class="row-body"><div class="row-title"><b>Aisha Mansour</b> paid you</div><div class="row-sub">Birthday gift split</div></div>
        <div class="row-right"><div class="row-amt pos mono">+$67.30</div><div class="row-time">4d ago</div></div>
      </div>
      <div class="row">
        <div class="row-avatar av-amber">RS</div>
        <div class="row-body"><div class="row-title">You paid <b>Reuben Sterling</b></div><div class="row-sub">Festival pass</div></div>
        <div class="row-right"><div class="row-amt neg mono">$215.00</div><div class="row-time">1w ago</div></div>
      </div>
      <div class="row">
        <div class="row-avatar av-violet">HC</div>
        <div class="row-body"><div class="row-title"><b>Hannah Choi</b> paid you</div><div class="row-sub">Groceries split</div></div>
        <div class="row-right"><div class="row-amt pos mono">+$38.15</div><div class="row-time">1w ago</div></div>
      </div>
      <div class="row">
        <div class="row-avatar av-cyan">IK</div>
        <div class="row-body"><div class="row-title"><b>BNPL</b> · IKEA furniture</div><div class="row-sub">Installment 3 of 4 charged</div></div>
        <div class="row-right"><div class="row-amt bnpl mono">$187.25</div><div class="row-time">2w ago</div></div>
      </div>
    </div>
  </div>

  <!-- ============ ACTIVE INSTALLMENTS PANE ============ -->
  <div data-pane-content="active" class="pane-content" style="display:none;">
    <div class="card">
      <div class="row">
        <div class="row-avatar av-cyan">BO</div>
        <div class="row-body">
          <div class="row-title"><b>Bose QC Ultra headphones</b> <span class="pill">Pay-in-4</span></div>
          <div class="row-sub">Next due Jun 14 · 2 of 4 paid</div>
          <div class="progress"><span style="width:50%;"></span></div>
        </div>
        <div class="row-right"><div class="row-amt bnpl mono">$94.75</div><div class="row-time">/installment</div></div>
      </div>
      <div class="row">
        <div class="row-avatar av-emerald">AL</div>
        <div class="row-body">
          <div class="row-title"><b>Allbirds Wool Runners</b> <span class="pill">Pay-in-4</span></div>
          <div class="row-sub">Next due Jun 22 · 1 of 4 paid</div>
          <div class="progress"><span style="width:25%;"></span></div>
        </div>
        <div class="row-right"><div class="row-amt bnpl mono">$32.50</div><div class="row-time">/installment</div></div>
      </div>
      <div class="row">
        <div class="row-avatar av-amber">IK</div>
        <div class="row-body">
          <div class="row-title"><b>IKEA furniture set</b> <span class="pill">Pay-in-4</span></div>
          <div class="row-sub">Next due Jul 5 · 3 of 4 paid</div>
          <div class="progress"><span style="width:75%;"></span></div>
        </div>
        <div class="row-right"><div class="row-amt bnpl mono">$187.25</div><div class="row-time">/installment</div></div>
      </div>
    </div>
  </div>

  <!-- ============ OVERDUE PANE ============ -->
  <div data-pane-content="overdue" class="pane-content" style="display:none;">
    <!-- Banner with urgency treatment -->
    <div class="overdue-banner">
      <svg width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
      <div class="ob-text">
        <strong>$39.94 overdue across 2 bills</strong>
        <div>Late fees may apply. Pay now to keep your Splitit Score intact.</div>
      </div>
      <button onclick="showToast('Opening pay-all flow…')">Pay all</button>
    </div>

    <div class="card">
      <div class="row overdue-row">
        <div class="row-avatar av-rose">SD</div>
        <div class="row-body">
          <div class="row-title"><b>Steam Deck cover</b> <span class="pill warn">3 days late</span></div>
          <div class="row-sub">Due May 28 · Installment 2 of 4</div>
          <div class="progress warn"><span style="width:25%;"></span></div>
        </div>
        <div class="row-right"><div class="row-amt mono" style="color:var(--rose-hi);">$24.99</div><div class="row-time">+ $5 late fee</div></div>
      </div>
      <div class="row overdue-row">
        <div class="row-avatar av-rose">AU</div>
        <div class="row-body">
          <div class="row-title"><b>Audible subscription</b> <span class="pill warn">1 day late</span></div>
          <div class="row-sub">Due May 30 · Installment 1 of 1</div>
          <div class="progress warn"><span style="width:0%;"></span></div>
        </div>
        <div class="row-right"><div class="row-amt mono" style="color:var(--rose-hi);">$14.95</div><div class="row-time">no grace fee</div></div>
      </div>
    </div>

    <!-- Empty state for when there's nothing overdue.
         Hidden by default; the JS swaps in this card if the populated list above is removed.
         Kept in the DOM so the toggle is instant. -->
    <div class="empty" style="display:none;" data-empty="overdue">
      <div class="empty-icon success">
        <svg width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg>
      </div>
      <div class="empty-title">You're all caught up</div>
      <div class="empty-sub">No overdue bills. Your Splitit Score is safe.</div>
    </div>
  </div>
</section>
`
}

// ---------------------------------------------------------------------------
// FRIENDS VIEW — count header, search, scrollable list, per-friend actions.
// Includes a hidden empty-state card for the brand-new-user scenario.
// ---------------------------------------------------------------------------
func viewFriends() string {
	return `
<section class="view" data-view="friends">
  <div class="friends-header">
    <div class="friends-count">12 <small>friends</small></div>
    <button class="friend-add" onclick="showToast('Add friend modal coming soon')">
      <svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path d="M12 5v14"/><path d="M5 12h14"/></svg>
      Add friend
    </button>
  </div>

  <div class="search-wrap">
    <svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
    <input class="search-inp" type="search" placeholder="Search friends" oninput="filterFriends(this.value)" />
  </div>

  <div class="card" id="friend-list">
` + friendsRows() + `
  </div>

  <!-- Empty state shown if user has zero friends. -->
  <div class="empty" id="friends-empty" style="display:none;">
    <div class="empty-icon">
      <svg width="24" height="24" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><line x1="20" y1="8" x2="20" y2="14"/><line x1="23" y1="11" x2="17" y2="11"/></svg>
    </div>
    <div class="empty-title">No friends yet</div>
    <div class="empty-sub">Add friends to split bills, send payments, and request money in seconds.</div>
    <button onclick="showToast('Add friend modal coming soon')">Add your first friend</button>
  </div>
</section>
`
}

func friendsRows() string {
	type fr struct {
		Initials string
		Class    string
		Name     string
		Handle   string
		Last     string
	}
	all := []fr{
		{"MH", "av-indigo", "Marcus Holloway", "@mholloway", "Last paid you · Yesterday"},
		{"PR", "av-amber", "Priya Raman", "@priya.r", "Last paid you · 3d ago"},
		{"DF", "av-emerald", "Diego Fernández", "@diego.fz", "You paid · 1w ago"},
		{"EK", "av-violet", "Elena Kowalski", "@ekowalski", "Last paid you · 1w ago"},
		{"AM", "av-pink", "Aisha Mansour", "@aisha.m", "Last paid you · 2w ago"},
		{"TD", "av-cyan", "Tomás Delacroix", "@tdelacroix", "No activity yet"},
		{"HC", "av-indigo", "Hannah Choi", "@hchoi", "Last paid you · 3w ago"},
		{"YW", "av-amber", "Yuki Watanabe", "@yuki.w", "You paid · 1mo ago"},
		{"RS", "av-emerald", "Reuben Sterling", "@rsterling", "You paid · 1mo ago"},
		{"NB", "av-violet", "Naomi Blackwood", "@nblackwood", "Last paid you · 2mo ago"},
		{"SM", "av-pink", "Sofia Martinelli", "@smartinelli", "You paid · 3mo ago"},
		{"KA", "av-cyan", "Kwame Asante", "@kwame.a", "No activity yet"},
	}
	out := ""
	for _, f := range all {
		out += `
    <div class="friend-row" data-name="` + strings.ToLower(f.Name+" "+f.Handle) + `">
      <div class="row-avatar ` + f.Class + `">` + f.Initials + `</div>
      <div class="row-body" style="flex:1; min-width:0;">
        <b>` + f.Name + `</b>
        <div>` + f.Handle + ` · ` + f.Last + `</div>
      </div>
      <div class="friend-actions">
        <button title="Pay" onclick="openSheet('pay'); showToast('Selected ` + f.Name + `');">
          <svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
        </button>
        <button class="remove" title="Remove" onclick="removeFriend(this, '` + f.Name + `')">
          <svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-2 14a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2L5 6"/></svg>
        </button>
      </div>
    </div>`
	}
	return out
}

// ---------------------------------------------------------------------------
// PROFILE VIEW — user info, Splitit Score gauge, BNPL limit, stats, menu, sign out.
// The Splitit Score sits here (not on Home) per the IA brief.
// ---------------------------------------------------------------------------
func viewProfile(_ *models.User, avatar, name, handle, email string) string {
	return `
<section class="view" data-view="profile">
  <div class="profile-hero">
    <div class="avatar-xl">` + avatar + `</div>
    <div class="profile-name">` + name + `</div>
    <div class="profile-handle">@` + handle + `</div>
    <div class="profile-email">` + email + `</div>
  </div>

  <!-- Splitit Score (the renamed Credit Score) -->
  <div class="score-card">
    <div class="score-header">
      <div class="score-title">Splitit Score</div>
      <div class="score-pill">Excellent</div>
    </div>
    <div class="score-val mono">76<small>/100</small></div>
    <div class="score-track"><span style="width:76%;"></span></div>
    <div class="score-track-labels"><span>Poor</span><span>Fair</span><span>Good</span><span>Excellent</span></div>
    <div style="display:flex; justify-content:space-between; margin-top:14px; padding-top:14px; border-top:1px solid rgba(255,255,255,0.06);">
      <div>
        <div style="font-size:11px; color:var(--text-mute); text-transform:uppercase; letter-spacing:0.05em; font-weight:600;">BNPL limit</div>
        <div class="mono" style="font-size:18px; font-weight:700; color:var(--emerald-hi); margin-top:4px;">$1,500</div>
      </div>
      <div style="text-align:right;">
        <div style="font-size:11px; color:var(--text-mute); text-transform:uppercase; letter-spacing:0.05em; font-weight:600;">Available</div>
        <div class="mono" style="font-size:18px; font-weight:700; color:var(--text); margin-top:4px;">$804.00</div>
      </div>
    </div>
  </div>

  <div class="stat-grid">
    <div class="stat-tile"><div class="stat-lbl">Balance</div><div class="stat-val green mono">$1,247</div></div>
    <div class="stat-tile"><div class="stat-lbl">Friends</div><div class="stat-val mono">12</div></div>
    <div class="stat-tile"><div class="stat-lbl">Splits</div><div class="stat-val indigo mono">3</div></div>
  </div>

  <div class="menu-list">
    <button class="menu-item" onclick="goView('friends')">
      <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
      Manage friends
      <svg class="chev" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M9 18l6-6-6-6"/></svg>
    </button>
    <button class="menu-item" onclick="showToast('Payment methods coming soon')">
      <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><rect x="1" y="4" width="22" height="16" rx="2"/><line x1="1" y1="10" x2="23" y2="10"/></svg>
      Payment methods
      <svg class="chev" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M9 18l6-6-6-6"/></svg>
    </button>
    <button class="menu-item" onclick="showToast('Edit profile coming soon')">
      <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
      Edit profile
      <svg class="chev" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M9 18l6-6-6-6"/></svg>
    </button>
    <button class="menu-item" onclick="showToast('Settings coming soon')">
      <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
      Settings
      <svg class="chev" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M9 18l6-6-6-6"/></svg>
    </button>
    <button class="menu-item danger" onclick="signOut()">
      <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
      Sign out
    </button>
  </div>
</section>
`
}

// ---------------------------------------------------------------------------
// PAY SHEET — Send and BNPL as TABS within a single flow.
// (Per feedback: BNPL is no longer a separate top-level action.)
// ---------------------------------------------------------------------------
func paySheetHTML() string {
	return `
<div class="sheet-backdrop" id="pay-sheet" onclick="if(event.target===this) closeSheet('pay')">
  <div class="sheet">
    <div class="sheet-handle"></div>
    <div class="sheet-title">New payment</div>
    <div class="sheet-tabs" role="tablist">
      <button class="sheet-tab active" data-paytab="send" onclick="payTab(this)">Send to friend</button>
      <button class="sheet-tab" data-paytab="bnpl" onclick="payTab(this)">Buy now, pay later</button>
    </div>

    <!-- ============ SEND PANE ============ -->
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
        <select class="modal-sel">
          <option value="">Select a friend…</option>
          <option>Marcus Holloway · @mholloway</option>
          <option>Priya Raman · @priya.r</option>
          <option>Diego Fernández · @diego.fz</option>
          <option>Elena Kowalski · @ekowalski</option>
          <option>Aisha Mansour · @aisha.m</option>
        </select>
      </div>
      <div>
        <label class="modal-lbl">Amount</label>
        <input class="modal-inp" type="number" inputmode="decimal" placeholder="0.00" oninput="document.getElementById('send-amt').textContent = parseFloat(this.value || 0).toFixed(2)" />
      </div>
      <div>
        <label class="modal-lbl">Note</label>
        <input class="modal-inp" type="text" placeholder="Dinner, rent, etc." />
      </div>
      <button class="submit-btn" onclick="submitPayment('Payment sent successfully')">Send payment</button>
    </div>

    <!-- ============ BNPL PANE (nested inside Pay) ============ -->
    <div class="sheet-pane" data-pay-pane="bnpl">
      <div class="info-card">
        <span class="label">Available credit</span>
        <span class="val mono">$804.00</span>
      </div>
      <div>
        <label class="modal-lbl">Merchant or seller</label>
        <select class="modal-sel">
          <option value="">Select merchant…</option>
          <option>Apple Store</option>
          <option>Best Buy</option>
          <option>Wayfair</option>
          <option>Diego Fernández · @diego.fz</option>
          <option>Marcus Holloway · @mholloway</option>
        </select>
      </div>
      <div>
        <label class="modal-lbl">Purchase amount</label>
        <input class="modal-inp" type="number" inputmode="decimal" placeholder="0.00" />
      </div>
      <div>
        <label class="modal-lbl">Plan</label>
        <select class="modal-sel">
          <option>Pay-in-4 (0% APR · 4 payments over 6 weeks)</option>
          <option>Pay-in-6 (12.99% APR · 6 monthly payments)</option>
          <option>Pay-in-12 (15.99% APR · 12 monthly payments)</option>
        </select>
      </div>
      <div>
        <label class="modal-lbl">What are you buying?</label>
        <input class="modal-inp" type="text" placeholder="Item name" />
      </div>
      <button class="submit-btn emerald" onclick="submitPayment('BNPL plan approved')">Approve plan</button>
    </div>
  </div>
</div>`
}

// ---------------------------------------------------------------------------
// REQUEST SHEET — single pane, asks a friend for money.
// ---------------------------------------------------------------------------
func requestSheetHTML() string {
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
        <select class="modal-sel">
          <option value="">Select a friend…</option>
          <option>Marcus Holloway · @mholloway</option>
          <option>Priya Raman · @priya.r</option>
          <option>Diego Fernández · @diego.fz</option>
          <option>Elena Kowalski · @ekowalski</option>
          <option>Aisha Mansour · @aisha.m</option>
        </select>
      </div>
      <div>
        <label class="modal-lbl">Amount</label>
        <input class="modal-inp" type="number" inputmode="decimal" placeholder="0.00" oninput="document.getElementById('req-amt').textContent = parseFloat(this.value || 0).toFixed(2)" />
      </div>
      <div>
        <label class="modal-lbl">Note</label>
        <input class="modal-inp" type="text" placeholder="What's it for?" />
      </div>
      <button class="submit-btn amber" onclick="submitPayment('Request sent')">Send request</button>
    </div>
  </div>
</div>`
}

// ---------------------------------------------------------------------------
// CLIENT SCRIPT — view router, sheets, sign-out via real /users/logout endpoint.
// Note: no JS template literals (Go raw strings can't nest backticks).
// ---------------------------------------------------------------------------
func dashboardScript() string {
	return `
<script>
  // --- View router --------------------------------------------------------
  function goView(name, subpane) {
    document.querySelectorAll('[data-view]').forEach(function(el) {
      el.classList.toggle('active', el.getAttribute('data-view') === name);
    });
    document.querySelectorAll('.tab[data-tab], .side-link[data-tab]').forEach(function(btn) {
      btn.classList.toggle('active', btn.getAttribute('data-tab') === name);
    });
    if (subpane && name === 'activity') {
      // Wait one frame so the view is visible before clicking the sub-tab.
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

  // --- Amount chip helpers (Send + Request) ------------------------------
  function setSendAmt(v) { document.getElementById('send-amt').textContent = v.toFixed(2); }
  function setReqAmt(v)  { document.getElementById('req-amt').textContent  = v.toFixed(2); }

  // --- Submit (mocked — closes sheet, fires toast) -----------------------
  function submitPayment(msg) {
    closeSheet('pay');
    closeSheet('request');
    showToast(msg || 'Done');
  }

  // --- Toast --------------------------------------------------------------
  var toastTimer = null;
  function showToast(msg, kind) {
    var t = document.getElementById('toast');
    t.textContent = msg;
    t.classList.remove('warn');
    if (kind === 'warn') t.classList.add('warn');
    t.classList.add('show');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(function() { t.classList.remove('show'); }, 2400);
  }

  // --- Friends ------------------------------------------------------------
  function filterFriends(q) {
    q = (q || '').toLowerCase().trim();
    var rows = document.querySelectorAll('#friend-list .friend-row');
    var visible = 0;
    rows.forEach(function(r) {
      var match = !q || r.getAttribute('data-name').indexOf(q) !== -1;
      r.style.display = match ? '' : 'none';
      if (match) visible++;
    });
    // If filter wipes the list to zero, show a soft "no results" — toggle list
    // visibility but keep the dedicated empty state for the no-friends case.
    var list = document.getElementById('friend-list');
    list.style.opacity = visible === 0 && q ? '0.4' : '1';
  }
  function removeFriend(btn, name) {
    var row = btn.closest('.friend-row');
    row.style.transition = 'opacity .2s, transform .2s';
    row.style.opacity = '0';
    row.style.transform = 'translateX(-20px)';
    setTimeout(function() {
      row.remove();
      showToast(name + ' removed');
      // If we drained the list, flip to the empty-state card.
      var remaining = document.querySelectorAll('#friend-list .friend-row').length;
      if (remaining === 0) {
        document.getElementById('friend-list').style.display = 'none';
        document.getElementById('friends-empty').style.display = 'block';
      }
    }, 200);
  }

  // --- Sign out (real call to /users/logout) -----------------------------
  function signOut() {
    fetch('/users/logout', { method: 'POST', credentials: 'same-origin' })
      .then(function() { window.location.reload(); })
      .catch(function() {
        // Even if the call fails (network), clear the cookie client-side
        // and reload — failing closed.
        document.cookie = 'session_user_id=; Max-Age=0; path=/';
        window.location.reload();
      });
  }

  // --- Esc closes any open sheet -----------------------------------------
  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') {
      closeSheet('pay'); closeSheet('request');
    }
  });
</script>
`
}
