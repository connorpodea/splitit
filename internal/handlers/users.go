package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/connorpodea/splitit/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	// Initialize a custom, empty struct to hold the incoming data
	type RegistrationInput struct {
		ID          string `json:"id"`
		Password    string `json:"password"`
		Name        string `json:"name"`
		Email       string `json:"email"`
		PhoneNumber string `json:"phone_number"`
	}
	var input RegistrationInput

	// Read the JSON text out of the web request body and decode it into the Go struct
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil || input.Password == "" || input.ID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting or missing account properties"})
		return
	}
	input.ID = strings.ToLower(strings.TrimSpace(input.ID))
	input.Name = strings.ToLower(strings.TrimSpace(input.Name))
	if !regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(input.ID) {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Username may only contain letters, numbers, and underscores"})
		return
	}

	// Turn plain text password into a non-reversible cryptographic hash (bcrypt.DefaultCost tells it to scramble the password 14 times)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to protect account credentials securely"})
		return
	}

	// Map the form input data over to the structural Database model
	new_user := models.User{
		ID:           input.ID,
		PasswordHash: string(hashedPassword),
		Name:         input.Name,
		Email:        input.Email,
		PhoneNumber:  input.PhoneNumber,
		Balance:      0.00,
		CreditScore:  50,
		CreditLimit:  1000.00,
	}

	// Pass the populated struct down to the database engine
	err = h.store.CreateUser(&new_user)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusCreated, map[string]string{"status": "ledger user allocation committed successfully", "id": new_user.ID})
}

func (h *Handler) LoginUser(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	// Initialize a custom, empty struct to hold the incoming data
	type LoginInput struct {
		ID       string `json:"id"`
		Password string `json:"password"`
	}
	var input LoginInput

	// Read the JSON text out of the web request body and decode it into the Go struct
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	input.ID = strings.ToLower(strings.TrimSpace(input.ID))

	// Look up the user by ID
	user, err := h.store.GetUser(input.ID)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Invalid User ID or password parameters"})
		return
	}

	// Compare the password to the currently stored password hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password))
	if err != nil {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid User ID or password parameters"})
		return
	}

	// Issue Cookie Wristband on absolute match success
	cookie := &http.Cookie{
		Name:     "session_user_id",
		Value:    user.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	// Return a production-grade structural JSON status acknowledgement tracking packet
	WriteJSON(w, http.StatusOK, map[string]any{"authenticated": true, "user_id": user.ID, "name": user.Name})
}

// Logout clears the session cookie, signing the user out.
// Browsers persist the cookie until we explicitly expire it with MaxAge=-1.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	expired := &http.Cookie{
		Name:     "session_user_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, expired)

	WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}

	// Verify session context parameter state natively through current session validation
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Query the database engine for this user using our secure identity
	user, err := h.store.GetUser(userID)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	// Package the returned data struct into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	// Confirm user active session validity state before parsing database records
	_, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Query the database engine for all users
	users, err := h.store.ListUsers()
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Package the returned data struct into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, users)
}

func (h *Handler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	// Confirm user active session validity state before parsing database records
	_, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Query the database engine for all profiles
	profiles, err := h.store.ListProfiles()
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Package the returned data struct into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, profiles)
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	// Confirm user active session validity state before parsing database records
	_, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Extract the target profile identifier field from the URL query strings parameter bucket
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL query parameter: 'user_id'"})
		return
	}

	// Query the database engine for this profile
	profile, err := h.store.GetProfile(userID)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	// Package the returned data struct into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, profile)
}

func (h *Handler) GetRegistrationView(w http.ResponseWriter, r *http.Request) {
	writeHTML(w, registerHTML())
}

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

func viewProfile(user *models.User, avatar, name, handle, email, phone string, friends []models.Profile, installments []models.Installment) string {
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

func profileDisplayName(p *models.Profile) string {
	if strings.TrimSpace(p.Name) != "" {
		return p.Name
	}
	return p.ID
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

// UpdatePassword processes a credential rotation request, hashing the incoming
// plaintext password before delegating the digest to the store layer.
func (h *Handler) UpdatePassword(w http.ResponseWriter, r *http.Request) {}

// UpdateEmail processes a request to update the authenticated user's email address on record.
func (h *Handler) UpdateEmail(w http.ResponseWriter, r *http.Request) {}

// UpdatePhoneNumber processes a request to update the authenticated user's phone number on record.
func (h *Handler) UpdatePhoneNumber(w http.ResponseWriter, r *http.Request) {}

// UpdateDisplayName processes a request to update the authenticated user's display name.
func (h *Handler) UpdateDisplayName(w http.ResponseWriter, r *http.Request) {}

// DeactivateAccount processes a soft-delete deactivation request, transitioning the
// account to an inactive state without purging underlying ledger records.
func (h *Handler) DeactivateAccount(w http.ResponseWriter, r *http.Request) {}

func avatarClass(i int) string {
	classes := []string{"av-indigo", "av-amber", "av-emerald", "av-violet", "av-cyan", "av-pink"}
	return classes[i%len(classes)]
}
