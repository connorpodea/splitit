package handlers

import "html/template"

// Row-level html/template definitions for every view section that contains
// user-supplied data. All string fields are auto-escaped by the template
// engine in the correct context (HTML text, HTML attribute, CSS, or JS
// string literal). template.HTML fields are used only where server-controlled
// markup wraps user data that has already been explicitly escaped with
// template.HTMLEscapeString before the template.HTML cast.

// ---- Home view rows ----------------------------------------------------------

type homeRequestRowData struct {
	Color       string // validated or fallback hex
	Initials    string // derived from requester ID
	RequesterID string // user-supplied
	Note        string // user-supplied
	Amount      string // server-formatted "$X.XX"
	Date        string // server-formatted "YYYY-MM-DD"
}

var homeRequestRowTmpl = template.Must(template.New("home-req-row").Parse(
	`<div class="row">` +
		`<div class="row-avatar" style="background:{{.Color}};color:#fff">{{.Initials}}</div>` +
		`<div class="row-body">` +
		`<div class="row-title"><b>@{{.RequesterID}}</b> requested from <b>you</b></div>` +
		`<div class="row-sub">{{.Note}}</div>` +
		`</div>` +
		`<div class="row-right"><div class="row-amt req mono">{{.Amount}}</div><div class="row-time">{{.Date}}</div></div>` +
		`</div>`))

type homeBNPLRowData struct {
	Color    string
	PeerName string // user-supplied
	Sub      string // user-supplied note + server suffix
	AmtClass string // "bnpl" or "pos" — server-controlled
	Amount   string
	Date     string
}

var homeBNPLRowTmpl = template.Must(template.New("home-bnpl-row").Parse(
	`<div class="row">` +
		`<div class="row-avatar" style="background:{{.Color}};color:#fff">BN</div>` +
		`<div class="row-body">` +
		`<div class="row-title"><b>BNPL</b> to <b>{{.PeerName}}</b> <span class="pill">Pay-in-4</span></div>` +
		`<div class="row-sub">{{.Sub}}</div>` +
		`</div>` +
		`<div class="row-right"><div class="row-amt {{.AmtClass}} mono">{{.Amount}}</div><div class="row-time">{{.Date}}</div></div>` +
		`</div>`))

// ---- Activity view rows ------------------------------------------------------

type activityBNPLActiveRowData struct {
	Color    string
	PeerName string
	Sub      string
	Amount   string
}

var activityBNPLActiveRowTmpl = template.Must(template.New("activity-active-row").Parse(
	`<div class="row">` +
		`<div class="row-avatar" style="background:{{.Color}};color:#fff">BN</div>` +
		`<div class="row-body">` +
		`<div class="row-title"><b>BNPL</b> to <b>{{.PeerName}}</b> <span class="pill">Pay-in-4</span></div>` +
		`<div class="row-sub">{{.Sub}}</div>` +
		`<div class="progress"><span style="width:50%;"></span></div>` +
		`</div>` +
		`<div class="row-right"><div class="row-amt bnpl mono">{{.Amount}}</div><div class="row-time">/installment</div></div>` +
		`</div>`))

type activityBNPLOverdueRowData struct {
	Color         string
	PeerName      string
	Sub           string
	InstallmentID string
	PaymentID     string
	AmountCents   int
	Amount        string // pre-formatted "$X.XX"
}

var activityBNPLOverdueRowTmpl = template.Must(template.New("activity-overdue-row").Parse(
	`<div class="row overdue-row" data-installment-id="{{.InstallmentID}}" data-payment-id="{{.PaymentID}}" data-amount-cents="{{.AmountCents}}">` +
		`<div class="row-avatar" style="background:{{.Color}};color:#fff">OD</div>` +
		`<div class="row-body">` +
		`<div class="row-title"><b>BNPL</b> to <b>{{.PeerName}}</b> <span class="pill warn">Overdue</span></div>` +
		`<div class="row-sub">{{.Sub}}</div>` +
		`<div class="progress warn"><span style="width:25%;"></span></div>` +
		`</div>` +
		`<div class="row-right">` +
		`<button onclick="payInstallment('{{.InstallmentID}}','{{.PaymentID}}',{{.AmountCents}})" style="background:var(--rose);color:#fff;border:none;border-radius:8px;padding:6px 12px;font-size:12px;font-weight:700;cursor:pointer;font-family:inherit;">` +
		`Pay {{.Amount}}</button>` +
		`</div></div>`))

// activityFeedRowData is the unified activity feed row.
// Title is pre-assembled in Go: each user-data fragment is passed through
// template.HTMLEscapeString before being embedded, then the result is cast
// to template.HTML so the engine does not double-escape the <b> wrapper tags.
// ExtraHTML holds server-controlled button HTML produced by a sub-template.
type activityFeedRowData struct {
	Color     string
	Initials  string
	Title     template.HTML
	Sub       string
	ExtraHTML template.HTML
	AmtClass  string
	AmtStr    string
}

var activityFeedRowTmpl = template.Must(template.New("activity-feed-row").Parse(
	`<div class="row">` +
		`<div class="row-avatar" style="background:{{.Color}};color:#fff">{{.Initials}}</div>` +
		`<div class="row-body"><div class="row-title">{{.Title}}</div><div class="row-sub">{{.Sub}}</div></div>` +
		`<div class="row-right">{{.ExtraHTML}}<div class="row-amt {{.AmtClass}} mono">{{.AmtStr}}</div></div>` +
		`</div>`))

// payRequestBtnTmpl / payInstallmentBtnTmpl — onclick IDs are in a JS string
// literal context; html/template applies JS string escaping automatically.

type payRequestBtnData struct {
	RequestID string
}

var payRequestBtnTmpl = template.Must(template.New("pay-req-btn").Parse(
	`<button onclick="payRequest('{{.RequestID}}')" style="background:var(--indigo);color:#fff;border:none;border-radius:8px;padding:6px 12px;font-size:12px;font-weight:700;cursor:pointer;font-family:inherit;margin-bottom:4px;">Pay It</button>`))

type payInstallmentBtnData struct {
	InstallmentID string
	PaymentID     string
	AmountCents   int
}

var payInstallmentBtnTmpl = template.Must(template.New("pay-inst-btn").Parse(
	`<button onclick="payInstallment('{{.InstallmentID}}','{{.PaymentID}}',{{.AmountCents}})" style="background:var(--indigo);color:#fff;border:none;border-radius:8px;padding:6px 12px;font-size:12px;font-weight:700;cursor:pointer;font-family:inherit;margin-bottom:4px;">Pay Off</button>`))

// ---- Groups view rows --------------------------------------------------------

type groupInviteRowData struct {
	InviteID string
	SenderID string
	Date     string
}

var groupInviteRowTmpl = template.Must(template.New("group-invite-row").Parse(
	`<div style="display:flex;align-items:center;gap:12px;padding:12px 14px;border-bottom:1px solid var(--border-soft);" data-invite-id="{{.InviteID}}">` +
		`<div class="row-avatar" style="background:var(--surface-3);color:var(--text-dim)">` +
		`<svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>` +
		`</div>` +
		`<div style="flex:1;min-width:0;">` +
		`<div style="font-size:14px;font-weight:600;color:var(--text);">Group invitation</div>` +
		`<div style="font-size:12px;color:var(--text-faint);">from @{{.SenderID}} · {{.Date}}</div>` +
		`</div>` +
		`<div style="display:flex;gap:6px;">` +
		`<button onclick="acceptGroupInvitation('{{.InviteID}}')" style="background:var(--emerald);color:#fff;border:none;border-radius:8px;padding:6px 12px;font-size:12px;font-weight:700;cursor:pointer;font-family:inherit;">Join</button>` +
		`<button onclick="declineGroupInvitation('{{.InviteID}}')" style="background:var(--surface-3);color:var(--text-dim);border:1px solid var(--border);border-radius:8px;padding:6px 12px;font-size:12px;font-weight:700;cursor:pointer;font-family:inherit;">Decline</button>` +
		`</div>` +
		`</div>`))

// groupListRowTmpl — .GroupID and .Name in onclick attributes are in a JS
// string literal context and are JS-escaped by html/template automatically,
// replacing the explicit template.JSEscapeString calls from the old code.
type groupListRowData struct {
	GroupID    string
	Color      string
	Initials   string
	Name       string
	IsCreator  bool
	MemberText string
}

var groupListRowTmpl = template.Must(template.New("group-list-row").Parse(
	`<div class="group-row" data-group-id="{{.GroupID}}">` +
		`<div class="row-avatar" style="background:{{.Color}};color:#fff">{{.Initials}}</div>` +
		`<div class="row-body" style="flex:1;min-width:0;cursor:pointer;" onclick="goGroupDetail('{{.GroupID}}','{{.Name}}')">` +
		`<div class="row-title"><b>{{.Name}}</b>` +
		`{{if .IsCreator}} <span style="font-size:10px;background:var(--indigo-soft);color:var(--indigo-hi);padding:2px 6px;border-radius:4px;font-weight:700;">CREATOR</span>{{end}}` +
		`</div>` +
		`<div class="row-sub" id="gmembers-{{.GroupID}}">{{.MemberText}}</div>` +
		`</div>` +
		`<div class="friend-actions">` +
		`<button title="Invite member" onclick="openInviteSheet('{{.GroupID}}','{{.Name}}')">` +
		`<svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="8.5" cy="7" r="4"/><line x1="20" y1="8" x2="20" y2="14"/><line x1="23" y1="11" x2="17" y2="11"/></svg>` +
		`</button>` +
		`<button class="remove" title="Leave group" onclick="confirmLeaveGroup('{{.GroupID}}','{{.Name}}')">` +
		`<svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>` +
		`</button>` +
		`</div>` +
		`</div>`))

// ---- Wallet view rows --------------------------------------------------------

// All wallet row fields are server-determined constants or server-formatted
// numbers — no user-supplied text appears here.
type walletTxnRowData struct {
	Arrow    string // "↓" or "↑"
	Label    string // "Deposit" or "Withdrawal"
	DirClass string // "dir-in" or "dir-out"
	DirLabel string // "IN" or "OUT"
	Date     string
	AmtClass string // "pos" or "neg"
	Prefix   string // "+" or "−"
	Amount   string // "$X.XX"
}

var walletTxnRowTmpl = template.Must(template.New("wallet-txn-row").Parse(
	`<div class="row">` +
		`<div class="row-avatar" style="background:var(--surface-3);color:var(--text-dim);font-size:16px;">{{.Arrow}}</div>` +
		`<div class="row-body"><div class="row-title"><b>{{.Label}}</b><span class="dir-badge {{.DirClass}}">{{.DirLabel}}</span></div>` +
		`<div class="row-sub">{{.Date}}</div></div>` +
		`<div class="row-right"><div class="row-amt {{.AmtClass}} mono">{{.Prefix}}{{.Amount}}</div></div>` +
		`</div>`))

// ---- Analytics view rows -----------------------------------------------------

type analyticsRecipientRowData struct {
	Color     string
	Initials  string
	Name      string // user-supplied profile name
	Handle    string // validated ^[a-z0-9_]+$ but escaped for defence-in-depth
	TotalSent string // server-formatted "$X"
}

var analyticsRecipientRowTmpl = template.Must(template.New("analytics-recip-row").Parse(
	`<div class="row">` +
		`<div class="row-avatar" style="background:{{.Color}};color:#fff">{{.Initials}}</div>` +
		`<div class="row-body"><div class="row-title"><b>{{.Name}}</b></div><div class="row-sub">@{{.Handle}}</div></div>` +
		`<div class="row-right"><div class="row-amt neg mono">−{{.TotalSent}}</div><div class="row-time">sent</div></div>` +
		`</div>`))

// analyticsCreditRowTmpl uses {{if .IsPositive}} to branch on avatar colour
// rather than passing a server-computed style string as a CSS-typed field.
type analyticsCreditRowData struct {
	IsPositive bool
	AmtClass   string // "pos" or "neg"
	DeltaStr   string // "+5" or "-3" — numeric, no user input
	Score      int
	Date       string
}

var analyticsCreditRowTmpl = template.Must(template.New("analytics-credit-row").Parse(
	`<div class="row">` +
		`{{if .IsPositive}}<div class="row-avatar" style="background:var(--emerald-soft);color:var(--emerald-hi);font-size:14px;">▲</div>` +
		`{{else}}<div class="row-avatar" style="background:var(--rose-soft);color:var(--rose-hi);font-size:14px;">▼</div>{{end}}` +
		`<div class="row-body"><div class="row-title"><b>Score adjustment</b></div><div class="row-sub">New score: {{.Score}}</div></div>` +
		`<div class="row-right"><div class="row-amt {{.AmtClass}} mono">{{.DeltaStr}}</div><div class="row-time">{{.Date}}</div></div>` +
		`</div>`))

// ---- Settings view: colour swatches -----------------------------------------

type colorSwatchData struct {
	Hex      string
	Label    string
	Selected bool
}

// colorSwatchTmpl renders one palette button. .Hex appears in three contexts:
//   - data-color="{{.Hex}}"  → HTML attribute  (HTML-escaped)
//   - onclick="...('{{.Hex}}')" → JS string literal (JS-escaped)
//   - style="...background:{{.Hex}}..." → CSS value  (CSS-filtered)
//
// All three are handled automatically by html/template's context-aware escaping.
var colorSwatchTmpl = template.Must(template.New("color-swatch").Parse(
	`<button class="color-swatch" data-color="{{.Hex}}" onclick="submitUpdateColor('{{.Hex}}')" ` +
		`style="width:32px;height:32px;border-radius:50%;border:none;cursor:pointer;transition:transform .15s;` +
		`background:{{.Hex}};{{if .Selected}}box-shadow:0 0 0 3px var(--surface),0 0 0 5px {{.Hex}};{{end}}" ` +
		`title="{{.Label}}"></button>`))

// ---- Social (friends) view rows ---------------------------------------------

type friendRequestRowData struct {
	RequestID string
	Color     string
	Initials  string
	SenderID  string // user-supplied ID, validated ^[a-z0-9_]+$ but escaped for defence
}

var friendRequestRowTmpl = template.Must(template.New("friend-req-row").Parse(
	`<div style="display:flex;align-items:center;gap:12px;padding:12px 14px;border-bottom:1px solid var(--border-soft);" data-req-row="{{.RequestID}}">` +
		`<div class="row-avatar" style="background:{{.Color}};color:#fff">{{.Initials}}</div>` +
		`<div style="flex:1;min-width:0;">` +
		`<div style="font-size:14px;font-weight:600;color:var(--text);">@{{.SenderID}}</div>` +
		`<div style="font-size:12px;color:var(--text-faint);">Wants to be friends</div>` +
		`</div>` +
		`<div style="display:flex;gap:6px;">` +
		`<button onclick="acceptFriendRequest('{{.RequestID}}')" style="background:var(--emerald);color:#fff;border:none;border-radius:8px;padding:6px 12px;font-size:12px;font-weight:700;cursor:pointer;font-family:inherit;">Accept</button>` +
		`<button onclick="declineFriendRequest('{{.RequestID}}')" style="background:var(--surface-3);color:var(--text-dim);border:1px solid var(--border);border-radius:8px;padding:6px 12px;font-size:12px;font-weight:700;cursor:pointer;font-family:inherit;">Decline</button>` +
		`</div>` +
		`</div>`))

// friendListRowTmpl — .Name in onclick JS string literals is JS-escaped by
// html/template, replacing the explicit template.JSEscapeString calls from
// the previous implementation.
type friendListRowData struct {
	SearchKey string // lower-cased, used as data-name for client-side filter
	FriendID  string
	Color     string
	Initials  string
	Name      string
	Handle    string
}

var friendListRowTmpl = template.Must(template.New("friend-list-row").Parse(
	`<div class="friend-row" data-name="{{.SearchKey}}" data-friend-id="{{.FriendID}}">` +
		`<div class="row-avatar" style="background:{{.Color}};color:#fff">{{.Initials}}</div>` +
		`<div class="row-body" style="flex:1;min-width:0;"><b>{{.Name}}</b><div>@{{.Handle}}</div></div>` +
		`<div class="friend-actions">` +
		`<button title="History" onclick="openHistorySheet('{{.FriendID}}','{{.Name}}')">` +
		`<svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 .49-3.51"/></svg>` +
		`</button>` +
		`<button title="Pay" onclick="openPayToFriend('{{.FriendID}}')">` +
		`<svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>` +
		`</button>` +
		`<button class="remove" title="Remove" onclick="removeFriend(this,'{{.FriendID}}','{{.Name}}')">` +
		`<svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-2 14a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2L5 6"/></svg>` +
		`</button>` +
		`</div>` +
		`</div>`))
