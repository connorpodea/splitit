package handlers

import (
	"bytes"
	"html/template"
	"log"
)

// renderTmpl executes t with data and returns the resulting HTML string.
// Execution errors are logged and return an empty string rather than
// propagating, so a template bug degrades gracefully instead of panicking.
func renderTmpl(t *template.Template, data any) string {
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		log.Printf("[WARN] template %q failed: %v", t.Name(), err)
		return ""
	}
	return buf.String()
}

// ---- Friend picker (fp-item) --------------------------------------------------
// All user-supplied fields (ID, Name) are rendered through html/template, which
// applies context-sensitive escaping automatically:
//   - HTML text context  → HTML entity escaping
//   - HTML attribute     → HTML attribute escaping
//   - JS string context  → JS string escaping  (onclick '...' arguments)
//   - CSS value context  → CSS value filtering  (style=background:...)

type fpItemData struct {
	PickerID string // server-controlled constant, e.g. "pay-recv"
	ID       string // user-supplied: database primary key
	Name     string // user-supplied: display name
	Initials string // derived from Name
	Color    string // validated hex colour
}

var friendPickerItemTmpl = template.Must(template.New("fp-item").Parse(
	`<div class="fp-item" data-id="{{.ID}}" onclick="pickFriend('{{.PickerID}}','{{.ID}}','{{.Name}}')" title="{{.Name}}">` +
		`<div class="fp-avatar-sm" style="background:{{.Color}};color:#fff">{{.Initials}}</div>` +
		`<div class="fp-name">{{.Name}}</div>` +
		`</div>`,
))

// ---- Group friend picker (fp-item variant) ------------------------------------

type gpItemData struct {
	PickerID string
	ID       string
	Name     string
	Initials string
	Color    string
}

var groupFriendPickerItemTmpl = template.Must(template.New("gp-item").Parse(
	`<div class="fp-item" data-id="{{.ID}}" onclick="pickFriend('{{.PickerID}}','{{.ID}}','{{.Name}}')" title="{{.Name}}">` +
		`<div class="fp-avatar-sm" style="background:{{.Color}};color:#fff">{{.Initials}}</div>` +
		`<div class="fp-name">{{.Name}}</div>` +
		`</div>`,
))

// ---- Group picker collapsible row --------------------------------------------

type gpRowData struct {
	PickerID string
	ID       string
	Name     string
	Initials string
	Color    string
}

var groupPickerRowTmpl = template.Must(template.New("gp-row").Parse(
	`<div class="gp-group-row" onclick="toggleGroupMembers('{{.PickerID}}','{{.ID}}','{{.Name}}',this)">` +
		`<div class="fp-avatar-sm" style="background:{{.Color}};color:#fff">{{.Initials}}</div>` +
		`<div class="fp-name">{{.Name}}</div>` +
		`<svg class="gp-chev" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M9 18l6-6-6-6"/></svg>` +
		`</div>` +
		`<div class="gp-members-wrap" id="gpm-{{.PickerID}}-{{.ID}}" style="display:none;padding-left:12px;"></div>`,
))

// ---- Notification row --------------------------------------------------------

type notifRowData struct {
	ID       string
	LinkView string
	IconSVG  template.HTML // server-controlled static SVG markup — safe to pass unescaped
	Color    string
	Title    string // server-generated, but escaped for defence-in-depth
	Body     string // contains user-supplied strings (sender IDs, amounts)
	Date     string
}

var notifRowTmpl = template.Must(template.New("notif-row").Parse(
	`<div class="row" style="cursor:pointer;" onclick="dismissNotif('{{.ID}}','{{.LinkView}}',this)">` +
		`<div class="row-avatar" style="background:{{.Color}};color:#fff">{{.IconSVG}}</div>` +
		`<div class="row-body">` +
		`<div class="row-title"><b>{{.Title}}</b></div>` +
		`<div class="row-sub">{{.Body}}</div>` +
		`</div>` +
		`<div class="row-right"><div class="row-time">{{.Date}}</div></div>` +
		`</div>`,
))
