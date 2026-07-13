package handlers

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/connorpodea/splitit/internal/models"
)

// ---- view: Home ---------------------------------------------------------------

func viewHome(name string, user *models.User, overdueInstallments []models.InstallmentDetail, installments []models.InstallmentDetail, incomingRequests []models.PaymentRequest) string {
	firstName := strings.Fields(name)
	greet := "there"
	if len(firstName) > 0 {
		greet = firstName[0]
	}
	balanceWhole := user.BalanceCents / 100
	balanceCentsFrac := user.BalanceCents % 100
	var outstandingBNPLCents int
	splitIDs := make(map[string]struct{})
	for _, inst := range installments {
		if !inst.IsPaid {
			outstandingBNPLCents += inst.AmountCents
			splitIDs[inst.PaymentID] = struct{}{}
		}
	}
	activeSplits := len(splitIDs)

	var rowsBuf strings.Builder
	count := 0
	for _, req := range incomingRequests {
		if count >= 5 {
			break
		}
		ini := strings.ToUpper(req.RequesterID)
		if len([]rune(ini)) > 2 {
			ini = string([]rune(ini)[:2])
		}
		date := req.CreatedAt
		if len(date) >= 10 {
			date = date[:10]
		}
		rowsBuf.WriteString(renderTmpl(homeRequestRowTmpl, homeRequestRowData{
			Color:       "#4ade80",
			Initials:    ini,
			RequesterID: req.RequesterID,
			Note:        req.Note,
			Amount:      fmt.Sprintf("$%d.%02d", req.AmountCents/100, req.AmountCents%100),
			Date:        date,
		}))
		count++
	}
	for _, inst := range installments {
		if count >= 5 {
			break
		}
		peerColor := inst.PeerColor
		if peerColor == "" {
			peerColor = "#4ade80"
		}
		peerLabel := inst.PeerName
		if peerLabel == "" {
			peerLabel = inst.PeerID
		}
		statusLabel := "Due " + inst.DueDate
		amtClass := "bnpl"
		if inst.IsPaid {
			statusLabel = "Paid"
			amtClass = "pos"
		}
		sub := statusLabel
		if inst.Note != "" {
			sub = inst.Note + " · " + statusLabel
		}
		rowsBuf.WriteString(renderTmpl(homeBNPLRowTmpl, homeBNPLRowData{
			Color:    peerColor,
			PeerName: peerLabel,
			Sub:      sub,
			AmtClass: amtClass,
			Amount:   fmt.Sprintf("$%d.%02d", inst.AmountCents/100, inst.AmountCents%100),
			Date:     inst.DueDate,
		}))
		count++
	}
	recentRows := rowsBuf.String()
	if recentRows == "" {
		recentRows = `<div style="text-align:center;padding:28px 16px;color:var(--text-mute);font-size:13px;">No recent activity yet.</div>`
	}
	return `
<section class="view active" data-view="home">
  <div class="greeting" style="font-size:14px;color:var(--text-mute);margin-bottom:14px;">Welcome back, <b style="color:var(--text);">` + template.HTMLEscapeString(greet) + `</b></div>
  <div class="hero">
    <div class="hero-label">Available balance</div>
    <div class="hero-amount mono">$` + fmt.Sprintf("%d", balanceWhole) + `<span class="cents">.` + fmt.Sprintf("%02d", balanceCentsFrac) + `</span></div>
    <div class="hero-meta">
      <span><b>$` + fmt.Sprintf("%d.%02d", outstandingBNPLCents/100, outstandingBNPLCents%100) + `</b> outstanding</span>
      <span style="opacity:.4;">·</span>
      <span><b>` + fmt.Sprintf("%d", activeSplits) + `</b> active BNPL</span>
    </div>
  </div>
  <div class="actions">
    <button class="action req" onclick="openSheet('request')">
      <span class="ico"><svg width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M12 5v14"/><path d="M5 12h14"/></svg></span>
      <span class="lbl-stack"><b>Request</b><small>Request from a friend or group</small></span>
    </button>
    <button class="action pay" onclick="openSheet('pay')">
      <span class="ico"><svg width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg></span>
      <span class="lbl-stack"><b>Pay</b><small>Pay a friend or group, now or later</small></span>
    </button>
  </div>
  <div class="section-row"><h2>Recent activity</h2><button class="linklike" onclick="goView('activity')">See all</button></div>
  <div class="card">` + recentRows + `</div>
</section>`
}

// ---- view: Activity -----------------------------------------------------------

func viewActivity(installments []models.InstallmentDetail, overdueInstallments []models.InstallmentDetail, _ []models.PaymentRequest, activityFeed []models.ActivityItem) string {
	overdueIDs := make(map[string]bool)
	for _, inst := range overdueInstallments {
		overdueIDs[inst.ID] = true
	}

	var activeBuf strings.Builder
	for _, inst := range installments {
		if inst.IsPaid || overdueIDs[inst.ID] {
			continue
		}
		peerColor := inst.PeerColor
		if peerColor == "" {
			peerColor = "#4ade80"
		}
		peerLabel := inst.PeerName
		if peerLabel == "" {
			peerLabel = inst.PeerID
		}
		sub := "Due " + inst.DueDate
		if inst.Note != "" {
			sub = inst.Note + " · Due " + inst.DueDate
		}
		activeBuf.WriteString(renderTmpl(activityBNPLActiveRowTmpl, activityBNPLActiveRowData{
			Color:    peerColor,
			PeerName: peerLabel,
			Sub:      sub,
			Amount:   fmt.Sprintf("$%d.%02d", inst.AmountCents/100, inst.AmountCents%100),
		}))
	}
	activeRows := activeBuf.String()
	if activeRows == "" {
		activeRows = `<div style="text-align:center;padding:28px 16px;color:var(--text-mute);font-size:13px;">No active BNPL plans.</div>`
	}

	overdueSubtabBadge := ""
	if len(overdueInstallments) > 0 {
		overdueSubtabBadge = ` <span class="badge mono">` + fmt.Sprintf("%d", len(overdueInstallments)) + `</span>`
	}
	overdueContent := ""
	if len(overdueInstallments) == 0 {
		overdueContent = `<div class="empty"><div class="empty-icon success"><svg width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg></div><div class="empty-title">You're all caught up</div><div class="empty-sub">No overdue bills. Your Splitit Score is safe.</div></div>`
	} else {
		var overdueTotalCents int
		for _, inst := range overdueInstallments {
			overdueTotalCents += inst.AmountCents
		}
		var overdueBuf strings.Builder
		for _, inst := range overdueInstallments {
			peerColor := inst.PeerColor
			if peerColor == "" {
				peerColor = "#4ade80"
			}
			peerLabel := inst.PeerName
			if peerLabel == "" {
				peerLabel = inst.PeerID
			}
			sub := "Due " + inst.DueDate
			if inst.Note != "" {
				sub = inst.Note + " · Due " + inst.DueDate
			}
			overdueBuf.WriteString(renderTmpl(activityBNPLOverdueRowTmpl, activityBNPLOverdueRowData{
				Color:         peerColor,
				PeerName:      peerLabel,
				Sub:           sub,
				InstallmentID: inst.ID,
				PaymentID:     inst.PaymentID,
				AmountCents:   inst.AmountCents,
				Amount:        fmt.Sprintf("$%d.%02d", inst.AmountCents/100, inst.AmountCents%100),
			}))
		}
		overdueContent = `<div class="overdue-banner"><svg width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg><div class="ob-text"><strong>$` + fmt.Sprintf("%d.%02d", overdueTotalCents/100, overdueTotalCents%100) + ` overdue across ` + fmt.Sprintf("%d", len(overdueInstallments)) + ` bills</strong><div>Late fees may apply. Pay now to keep your Splitit Score intact.</div></div><button onclick="payAllOverdue()">Pay all</button></div><div class="card">` + overdueBuf.String() + `</div>`
	}

	var feedBuf strings.Builder
	for _, item := range activityFeed {
		amt := fmt.Sprintf("%d.%02d", item.AmountCents/100, item.AmountCents%100)
		peerName := item.PeerName
		if peerName == "" {
			peerName = item.PeerID
		}
		if peerName == "" {
			peerName = "?"
		}
		ini := initials(peerName)
		if ini == "" {
			ini = "?"
		}
		avColor := item.PeerColor
		if avColor == "" {
			avColor = "#4ade80"
		}
		date := item.CreatedAt
		if len(date) > 16 {
			date = date[:16]
		}
		date = strings.ReplaceAll(date, "T", " ")
		escapedName := template.HTMLEscapeString(peerName)
		var amtCls, prefix string
		var title template.HTML
		var extraHTML template.HTML
		switch item.Kind {
		case "payment_sent":
			amtCls, prefix = "neg", "−"
			title = template.HTML("You paid <b>" + escapedName + "</b>")
		case "payment_received":
			amtCls, prefix = "pos", "+"
			title = template.HTML("<b>" + escapedName + "</b> paid you")
		case "request_sent":
			amtCls, prefix = "req", ""
			title = template.HTML("You requested from <b>" + escapedName + "</b>")
		case "request_received":
			amtCls, prefix = "req", ""
			title = template.HTML("<b>" + escapedName + "</b> requested")
			if item.Status == "pending" {
				extraHTML = template.HTML(renderTmpl(payRequestBtnTmpl, payRequestBtnData{RequestID: item.ID}))
			}
		case "installment_due":
			amtCls, prefix = "bnpl", ""
			title = template.HTML("BNPL to <b>" + escapedName + "</b>")
			extraHTML = template.HTML(renderTmpl(payInstallmentBtnTmpl, payInstallmentBtnData{
				InstallmentID: item.ID,
				PaymentID:     item.PaymentID,
				AmountCents:   item.AmountCents,
			}))
		default:
			amtCls, prefix = "", ""
			title = template.HTML(template.HTMLEscapeString(item.Kind))
		}
		amtStr := "$" + amt
		if prefix != "" {
			amtStr = prefix + amtStr
		}
		statusSuffix := ""
		if item.Status != "" && item.Status != "completed" {
			statusSuffix = " · " + item.Status
		}
		var sub string
		if item.Note == "" {
			sub = item.Status + " · " + date
		} else {
			sub = item.Note + statusSuffix + " · " + date
		}
		feedBuf.WriteString(renderTmpl(activityFeedRowTmpl, activityFeedRowData{
			Color:     avColor,
			Initials:  ini,
			Title:     title,
			Sub:       sub,
			ExtraHTML: extraHTML,
			AmtClass:  amtCls,
			AmtStr:    amtStr,
		}))
	}
	var feedContent string
	if feedBuf.Len() == 0 {
		feedContent = `<div style="text-align:center;padding:28px 16px;color:var(--text-mute);font-size:13px;">No activity yet.</div>`
	} else {
		feedContent = `<div class="card">` + feedBuf.String() + `</div>`
	}

	return `
<section class="view" data-view="activity">
  <div class="section-row" style="margin-top:0;"><h2 style="font-size:22px;font-weight:800;letter-spacing:-0.02em;">Activity</h2></div>
  <div class="subtabs" role="tablist">
    <button class="subtab active" data-pane="all" onclick="goPane(this)">Feed</button>
    <button class="subtab" data-pane="active" onclick="goPane(this)">Active Installments</button>
    <button class="subtab" data-pane="overdue" onclick="goPane(this)">Overdue Installments` + overdueSubtabBadge + `</button>
  </div>
  <div data-pane-content="all" class="pane-content"><div id="unified-feed">` + feedContent + `</div></div>
  <div data-pane-content="active" class="pane-content" style="display:none;"><div class="card">` + activeRows + `</div></div>
  <div data-pane-content="overdue" class="pane-content" style="display:none;">` + overdueContent + `</div>
</section>`
}

// ---- view: Groups -------------------------------------------------------------

func viewGroups(groups []models.Group, groupInvitations []models.GroupInvitation, userID string, memberCounts map[string]int) string {
	invitesSection := ""
	if len(groupInvitations) > 0 {
		var invBuf strings.Builder
		for _, inv := range groupInvitations {
			date := inv.CreatedAt
			if len(date) >= 10 {
				date = date[:10]
			}
			invBuf.WriteString(renderTmpl(groupInviteRowTmpl, groupInviteRowData{
				InviteID: inv.ID,
				SenderID: inv.SenderID,
				Date:     date,
			}))
		}
		invitesSection = `<div style="margin-bottom:16px;"><div style="font-size:11px;font-weight:700;color:var(--text-mute);text-transform:uppercase;letter-spacing:0.06em;margin-bottom:8px;">Pending invitations</div><div class="card">` + invBuf.String() + `</div></div>`
	}

	groupsContent := ""
	if len(groups) == 0 {
		groupsContent = `<div class="empty"><div class="empty-icon"><svg width="24" height="24" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg></div><div class="empty-title">No groups yet</div><div class="empty-sub">Create a group to split bills and manage shared expenses with multiple people.</div><button onclick="openSheet('create-group')">Create your first group</button></div>`
	} else {
		hexPalette := []string{"#c084fc", "#22d3ee", "#4ade80", "#facc15", "#fb7185", "#db2777"}
		var groupsBuf strings.Builder
		for i, g := range groups {
			hex := hexPalette[i%len(hexPalette)]
			ini := ""
			runes := []rune(g.Name)
			if len(runes) >= 2 {
				ini = strings.ToUpper(string(runes[0])) + strings.ToUpper(string(runes[1]))
			} else if len(runes) == 1 {
				ini = strings.ToUpper(string(runes[0]))
			}
			count := memberCounts[g.ID]
			memberText := fmt.Sprintf("%d member", count)
			if count != 1 {
				memberText += "s"
			}
			groupsBuf.WriteString(renderTmpl(groupListRowTmpl, groupListRowData{
				GroupID:    g.ID,
				Color:      hex,
				Initials:   ini,
				Name:       g.Name,
				IsCreator:  g.CreatorID == userID,
				MemberText: memberText,
			}))
		}
		groupsContent = `<div class="card" id="groups-list">` + groupsBuf.String() + `</div>`
	}

	return `
<section class="view" data-view="groups">
  <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:16px;">
    <h2 style="font-size:22px;font-weight:800;letter-spacing:-0.02em;margin:0;">Groups</h2>
    <button class="friend-add" onclick="openSheet('create-group')">
      <svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path d="M12 5v14"/><path d="M5 12h14"/></svg>
      New group
    </button>
  </div>
  ` + invitesSection + `
  ` + groupsContent + `
</section>`
}

// ---- view: Wallet -------------------------------------------------------------

func viewWallet(user *models.User, w *walletDashboard) string {
	balanceWhole := user.BalanceCents / 100
	balanceFrac := user.BalanceCents % 100

	utilPct := int(w.Utilization * 100)
	if utilPct > 100 {
		utilPct = 100
	}
	var utilColor string
	if utilPct >= 80 {
		utilColor = "var(--rose-hi)"
	} else if utilPct >= 50 {
		utilColor = "var(--amber-hi)"
	} else {
		utilColor = "var(--emerald-hi)"
	}
	netColor := "var(--emerald-hi)"
	netPfx := "+"
	netAbs := w.NetCents
	if netAbs < 0 {
		netColor = "var(--rose-hi)"
		netPfx = "−"
		netAbs = -netAbs
	}

	statsHTML := `<div class="stat-grid" style="margin:0 0 16px;">` +
		`<div class="stat-tile"><div class="stat-lbl">Sent (Last 30d)</div><div class="stat-val mono" style="color:var(--rose-hi);">$` + fmt.Sprintf("%d", w.SentCents/100) + `</div></div>` +
		`<div class="stat-tile"><div class="stat-lbl">Received (Last 30d)</div><div class="stat-val green mono">$` + fmt.Sprintf("%d", w.RecvCents/100) + `</div></div>` +
		`<div class="stat-tile"><div class="stat-lbl">Net (Last 30d)</div><div class="stat-val mono" style="color:` + netColor + `;">` + netPfx + `$` + fmt.Sprintf("%d", netAbs/100) + `</div></div>` +
		`</div>` +
		`<div class="card" style="padding:16px;margin-bottom:16px;">` +
		`<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:10px;">` +
		`<div style="font-size:13px;font-weight:700;color:var(--text-mute);text-transform:uppercase;letter-spacing:.05em;">BNPL Utilization</div>` +
		`<div class="mono" style="font-size:15px;font-weight:700;color:` + utilColor + `;">` + fmt.Sprintf("%d", utilPct) + `%</div>` +
		`</div>` +
		`<div class="progress"><span style="width:` + fmt.Sprintf("%d", utilPct) + `%;background:` + utilColor + `;"></span></div>` +
		`<div style="display:flex;justify-content:space-between;margin-top:8px;font-size:11px;color:var(--text-faint);">` +
		`<span>$` + fmt.Sprintf("%d", w.OutstandingCents/100) + ` outstanding</span>` +
		`<span>$` + fmt.Sprintf("%d", w.OverdueCents/100) + ` overdue</span>` +
		`<span>$` + fmt.Sprintf("%d", w.SettledCents/100) + ` settled</span>` +
		`</div></div>`

	var txnsHTML string
	if len(w.Transactions) == 0 {
		txnsHTML = `<div style="text-align:center;padding:24px;color:var(--text-mute);font-size:13px;">No wallet transactions yet.</div>`
	} else {
		var txnBuf strings.Builder
		for i, t := range w.Transactions {
			if i >= 30 {
				break
			}
			isDeposit := t.TransactionType == "deposit"
			var amtCls, dc, dl, pfx, lbl, arrow string
			if isDeposit {
				amtCls, dc, dl, pfx, lbl, arrow = "pos", "dir-in", "IN", "+", "Deposit", "↓"
			} else {
				amtCls, dc, dl, pfx, lbl, arrow = "neg", "dir-out", "OUT", "−", "Withdrawal", "↑"
			}
			date := t.CreatedAt
			if len(date) > 10 {
				date = date[:10]
			}
			txnBuf.WriteString(renderTmpl(walletTxnRowTmpl, walletTxnRowData{
				Arrow:    arrow,
				Label:    lbl,
				DirClass: dc,
				DirLabel: dl,
				Date:     date,
				AmtClass: amtCls,
				Prefix:   pfx,
				Amount:   fmt.Sprintf("$%d.%02d", t.AmountCents/100, t.AmountCents%100),
			}))
		}
		txnsHTML = `<div class="card">` + txnBuf.String() + `</div>`
	}

	return `
<section class="view" data-view="wallet">
  <div style="display:flex;align-items:center;gap:12px;margin-bottom:20px;">
    <button class="gd-back-btn" onclick="goView('profile')" aria-label="Back to profile">
      <svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M19 12H5"/><path d="M12 19l-7-7 7-7"/></svg>
    </button>
    <div style="font-size:22px;font-weight:800;letter-spacing:-0.02em;">Wallet</div>
  </div>
  <div class="hero" style="margin-bottom:16px;">
    <div class="hero-label">Cash balance</div>
    <div class="hero-amount mono">$` + fmt.Sprintf("%d", balanceWhole) + `<span class="cents">.` + fmt.Sprintf("%02d", balanceFrac) + `</span></div>
    <div style="display:flex;gap:10px;margin-top:16px;">
      <button onclick="openSheet('deposit')" style="flex:1;background:var(--emerald);color:#fff;border:none;border-radius:10px;padding:11px;font-size:14px;font-weight:700;cursor:pointer;font-family:inherit;display:flex;align-items:center;justify-content:center;gap:6px;">
        <svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path d="M12 5v14"/><path d="M5 12h14"/></svg>
        Deposit
      </button>
      <button onclick="openSheet('withdraw')" style="flex:1;background:var(--rose-hi);color:var(--text);border:1px solid var(--border);border-radius:10px;padding:11px;font-size:14px;font-weight:700;cursor:pointer;font-family:inherit;display:flex;align-items:center;justify-content:center;gap:6px;">
        <svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path d="M5 12h14"/></svg>
        Withdraw
      </button>
    </div>
  </div>
  <div id="wallet-stats">` + statsHTML + `</div>
  <div class="section-row"><h2>Transaction history</h2></div>
  <div id="wallet-txns">` + txnsHTML + `</div>
</section>`
}

// ---- view: Analytics ----------------------------------------------------------

func viewAnalytics(a *analyticsDashboard) string {
	hexPalette := []string{"#4ade80", "#22d3ee", "#fb7185", "#facc15", "#c084fc", "#f97316", "#e11d48", "#db2777"}

	rHTML := `<div class="section-row" style="margin-top:0;"><h2>Top Recipients (Last 90d)</h2></div>`
	if len(a.Recipients) == 0 {
		rHTML += `<div style="color:var(--text-mute);font-size:13px;padding:12px 0;">No payment data yet.</div>`
	} else {
		var recipBuf strings.Builder
		for i, r := range a.Recipients {
			dn := r.Profile.Name
			if dn == "" {
				dn = r.Profile.ID
			}
			if dn == "" {
				dn = "?"
			}
			ini := initials(dn)
			avColor := r.Profile.ProfileColor
			if avColor == "" {
				avColor = hexPalette[i%len(hexPalette)]
			}
			recipBuf.WriteString(renderTmpl(analyticsRecipientRowTmpl, analyticsRecipientRowData{
				Color:     avColor,
				Initials:  ini,
				Name:      dn,
				Handle:    r.Profile.ID,
				TotalSent: fmt.Sprintf("$%d", r.TotalSentCents/100),
			}))
		}
		rHTML += `<div class="card">` + recipBuf.String() + `</div>`
	}

	sentCents, recvCents, bnplCents := 0, 0, 0
	if a.Monthly != nil {
		sentCents = a.Monthly.TotalOutCents
		recvCents = a.Monthly.TotalInCents
		bnplCents = a.Monthly.ActiveBNPLChargesCents
	}
	mHTML := `<div class="section-row"><h2>This Month</h2></div>` +
		`<div class="stat-grid" style="margin-bottom:16px;">` +
		`<div class="stat-tile"><div class="stat-lbl">Sent</div><div class="stat-val mono" style="color:var(--rose-hi);">$` + fmt.Sprintf("%d", sentCents/100) + `</div></div>` +
		`<div class="stat-tile"><div class="stat-lbl">Received</div><div class="stat-val green mono">$` + fmt.Sprintf("%d", recvCents/100) + `</div></div>` +
		`<div class="stat-tile"><div class="stat-lbl">BNPL</div><div class="stat-val indigo mono">$` + fmt.Sprintf("%d", bnplCents/100) + `</div></div>` +
		`</div>`

	cHTML := `<div class="section-row"><h2>Credit Score History</h2></div>`
	if len(a.CreditHistory) == 0 {
		cHTML += `<div style="color:var(--text-mute);font-size:13px;padding:12px 0;">No score history yet.</div>`
	} else {
		var creditBuf strings.Builder
		for i, e := range a.CreditHistory {
			if i >= 12 {
				break
			}
			isPos := e.Delta >= 0
			date := e.CreatedAt.Format("2006-01-02")
			amtCls := "neg"
			deltaStr := fmt.Sprintf("%d", e.Delta)
			if isPos {
				amtCls = "pos"
				deltaStr = "+" + fmt.Sprintf("%d", e.Delta)
			}
			creditBuf.WriteString(renderTmpl(analyticsCreditRowTmpl, analyticsCreditRowData{
				IsPositive: isPos,
				AmtClass:   amtCls,
				DeltaStr:   deltaStr,
				Score:      e.Score,
				Date:       date,
			}))
		}
		cHTML += `<div class="card">` + creditBuf.String() + `</div>`
	}

	return `
<section class="view" data-view="analytics">
  <div style="display:flex;align-items:center;gap:12px;margin-bottom:20px;">
    <button class="gd-back-btn" onclick="goView('profile')" aria-label="Back to profile">
      <svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M19 12H5"/><path d="M12 19l-7-7 7-7"/></svg>
    </button>
    <div style="font-size:22px;font-weight:800;letter-spacing:-0.02em;">Analytics</div>
  </div>
  <div id="analytics-content">` + rHTML + mHTML + cHTML + `</div>
</section>`
}

// ---- view: Settings -----------------------------------------------------------

func viewSettings(user *models.User, settings *models.UserSettings) string {
	name := displayName(user)
	email := user.Email
	phone := user.PhoneNumber

	profileColor := user.ProfileColor
	if profileColor == "" {
		profileColor = "#4ade80"
	}
	type colorEntry struct{ hex, label string }
	palette := []colorEntry{
		{"#4ade80", "Mint Green"}, {"#22d3ee", "Electric Cyan"}, {"#fb7185", "Soft Coral"},
		{"#facc15", "Mustard Gold"}, {"#c084fc", "Bright Lavender"}, {"#f97316", "Tangerine"},
		{"#e11d48", "Crimson"}, {"#db2777", "Magenta"},
	}
	var swatchBuf strings.Builder
	for _, c := range palette {
		swatchBuf.WriteString(renderTmpl(colorSwatchTmpl, colorSwatchData{
			Hex:      c.hex,
			Label:    c.label,
			Selected: c.hex == profileColor,
		}))
	}
	swatches := swatchBuf.String()

	emailChecked := "checked"
	discChecked := "checked"
	if settings != nil {
		if !settings.EmailNotifications {
			emailChecked = ""
		}
		if !settings.IsDiscoverable {
			discChecked = ""
		}
	}

	return `
<section class="view" data-view="settings">
  <div style="display:flex;align-items:center;gap:12px;margin-bottom:20px;">
    <button class="gd-back-btn" onclick="goView('profile')" aria-label="Back to profile">
      <svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M19 12H5"/><path d="M12 19l-7-7 7-7"/></svg>
    </button>
    <div style="font-size:22px;font-weight:800;letter-spacing:-0.02em;">Settings</div>
  </div>

  <div class="section-row"><h2>Profile</h2></div>
  <div class="card" style="margin-bottom:16px;">
    <div style="padding:16px;display:flex;flex-direction:column;gap:14px;">
      <div>
        <label class="modal-lbl">Display name</label>
        <div style="display:flex;gap:8px;">
          <input class="modal-inp" id="settings-name" value="` + template.HTMLEscapeString(name) + `" placeholder="Your name" />
          <button onclick="submitUpdateField('name',document.getElementById('settings-name').value,'name')" style="background:var(--indigo);color:#fff;border:none;border-radius:10px;padding:0 16px;font-size:13px;font-weight:700;cursor:pointer;font-family:inherit;white-space:nowrap;">Save</button>
        </div>
      </div>
      <div>
        <label class="modal-lbl">Email</label>
        <div style="display:flex;gap:8px;">
          <input class="modal-inp" id="settings-email" type="email" value="` + template.HTMLEscapeString(email) + `" placeholder="email@example.com" />
          <button onclick="submitUpdateField('email',document.getElementById('settings-email').value,'email')" style="background:var(--indigo);color:#fff;border:none;border-radius:10px;padding:0 16px;font-size:13px;font-weight:700;cursor:pointer;font-family:inherit;white-space:nowrap;">Save</button>
        </div>
      </div>
      <div>
        <label class="modal-lbl">Phone</label>
        <div style="display:flex;gap:8px;">
          <input class="modal-inp" id="settings-phone" type="tel" value="` + template.HTMLEscapeString(phone) + `" placeholder="123-456-7890" />
          <button onclick="submitUpdateField('phone',document.getElementById('settings-phone').value,'phone')" style="background:var(--indigo);color:#fff;border:none;border-radius:10px;padding:0 16px;font-size:13px;font-weight:700;cursor:pointer;font-family:inherit;white-space:nowrap;">Save</button>
        </div>
      </div>
      <div>
        <label class="modal-lbl">Profile color</label>
        <div style="display:flex;gap:10px;flex-wrap:wrap;margin-top:6px;" id="color-picker">` + swatches + `</div>
      </div>
    </div>
  </div>

  <div class="section-row"><h2>Security</h2></div>
  <div class="card" style="margin-bottom:16px;">
    <div style="padding:16px;display:flex;flex-direction:column;gap:12px;">
      <div><label class="modal-lbl">Current password</label><input class="modal-inp" id="settings-cur-pw" type="password" placeholder="••••••••" /></div>
      <div><label class="modal-lbl">New password</label><input class="modal-inp" id="settings-new-pw" type="password" placeholder="Min 8 characters" /></div>
      <div><label class="modal-lbl">Confirm new password</label><input class="modal-inp" id="settings-conf-pw" type="password" placeholder="Re-enter new password" /></div>
      <button class="submit-btn" onclick="submitUpdatePassword()">Update password</button>
    </div>
  </div>

  <div class="section-row"><h2>Privacy</h2></div>
  <div class="card" id="privacy-card" style="margin-bottom:16px;">
    <div class="toggle-row">
      <div><div class="toggle-label">Email notifications</div><div class="toggle-sub">Receive payment and activity alerts by email</div></div>
      <label class="toggle-switch"><input type="checkbox" id="toggle-email-notif" ` + emailChecked + ` onchange="submitUpdateSettings()"><span class="toggle-slider"></span></label>
    </div>
    <div class="toggle-row">
      <div><div class="toggle-label">Discoverable</div><div class="toggle-sub">Allow others to find your account by name or ID</div></div>
      <label class="toggle-switch"><input type="checkbox" id="toggle-discoverable" ` + discChecked + ` onchange="submitUpdateSettings()"><span class="toggle-slider"></span></label>
    </div>
  </div>

  <div class="section-row" style="margin-top:28px;"><h2 style="color:var(--rose-hi);">Danger zone</h2></div>
  <div class="card" style="border-color:rgba(239,68,68,0.2);">
    <div style="padding:16px;">
      <div style="font-size:14px;color:var(--text);font-weight:600;margin-bottom:4px;">Deactivate account</div>
      <div style="font-size:13px;color:var(--text-mute);margin-bottom:14px;">Permanently deactivates your account. This action cannot be undone.</div>
      <button onclick="deactivateAccount()" style="background:none;border:1px solid rgba(239,68,68,0.4);color:var(--rose-hi);border-radius:10px;padding:10px 18px;font-size:13px;font-weight:700;cursor:pointer;font-family:inherit;transition:background .2s;" onmouseover="this.style.background='rgba(239,68,68,0.08)'" onmouseout="this.style.background='none'">Deactivate my account</button>
    </div>
  </div>
</section>`
}

// ---- view: Group Detail -------------------------------------------------------

func viewGroupDetail() string {
	return `
<section class="view" data-view="group-detail">
  <div style="display:flex;align-items:center;gap:12px;margin-bottom:20px;">
    <button class="gd-back-btn" onclick="goView('groups')" aria-label="Back to groups">
      <svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M19 12H5"/><path d="M12 19l-7-7 7-7"/></svg>
    </button>
    <div id="group-detail-name" style="font-size:22px;font-weight:800;letter-spacing:-0.02em;">Group</div>
  </div>
  <div class="subtabs" role="tablist">
    <button class="subtab active" data-pane="gd-members" onclick="goPane(this)">Members</button>
    <button class="subtab" data-pane="gd-activity" onclick="goPane(this)">Activity</button>
  </div>
  <div data-pane-content="gd-members" class="pane-content">
    <div id="group-detail-members"><div style="color:var(--text-mute);font-size:13px;padding:12px 0;text-align:center;">Loading…</div></div>
  </div>
  <div data-pane-content="gd-activity" class="pane-content" style="display:none;">
    <div id="group-detail-activity"><div style="color:var(--text-mute);font-size:13px;padding:12px 0;text-align:center;">Loading…</div></div>
  </div>
</section>`
}

// ---- Sheets ------------------------------------------------------------------

func createGroupSheetHTML() string {
	return `
<div class="sheet-backdrop" id="create-group-sheet" onclick="if(event.target===this) closeSheet('create-group')">
  <div class="sheet">
    <div class="sheet-handle"></div>
    <div class="sheet-title">Create group</div>
    <div class="sheet-pane active" style="display:flex;">
      <div><label class="modal-lbl">Group name</label><input class="modal-inp" id="new-group-name" placeholder="e.g. Apartment, Trip to NYC" maxlength="60" /></div>
      <button class="submit-btn emerald" onclick="submitCreateGroup()">Create group</button>
    </div>
  </div>
</div>`
}

func inviteToGroupSheetHTML(friends []models.Profile) string {
	invitePicker := friendPickerHTML("invite-recv", friends)
	return `
<div class="sheet-backdrop" id="invite-group-sheet" onclick="if(event.target===this) closeSheet('invite-group')">
  <div class="sheet">
    <div class="sheet-handle"></div>
    <div class="sheet-title" id="invite-sheet-title">Invite to group</div>
    <input type="hidden" id="invite-group-id" value="" />
    <div class="sheet-pane active" style="display:flex;">
      <div>
        <label class="modal-lbl">Select a friend</label>
        <div class="friend-picker" id="invite-recv-picker">` + invitePicker + `</div>
        <input type="hidden" id="invite-recv-val" value="" />
      </div>
      <button class="submit-btn" onclick="submitGroupInvite()">Send invitation</button>
    </div>
  </div>
</div>`
}

func depositSheetHTML() string {
	return `
<div class="sheet-backdrop" id="deposit-sheet" onclick="if(event.target===this) closeSheet('deposit')">
  <div class="sheet">
    <div class="sheet-handle"></div>
    <div class="sheet-title">Deposit funds</div>
    <div class="sheet-pane active" style="display:flex;">
      <div class="amount-big mono atm-wrap" data-atm="deposit" id="deposit-atm" tabindex="0" onclick="this.focus()">
        <span class="dollar">$</span><span id="deposit-display">0.00</span><span class="atm-cursor"></span>
      </div>
      <div class="chip-row" style="justify-content:center;">
        <button class="chip" onclick="setAtmCents('deposit',2500)">$25</button>
        <button class="chip" onclick="setAtmCents('deposit',5000)">$50</button>
        <button class="chip" onclick="setAtmCents('deposit',10000)">$100</button>
        <button class="chip" onclick="setAtmCents('deposit',25000)">$250</button>
      </div>
      <button class="submit-btn emerald" onclick="submitDeposit()">Deposit</button>
    </div>
  </div>
</div>`
}

func withdrawSheetHTML() string {
	return `
<div class="sheet-backdrop" id="withdraw-sheet" onclick="if(event.target===this) closeSheet('withdraw')">
  <div class="sheet">
    <div class="sheet-handle"></div>
    <div class="sheet-title">Withdraw funds</div>
    <div class="sheet-pane active" style="display:flex;">
      <div class="amount-big mono atm-wrap" data-atm="withdraw" id="withdraw-atm" tabindex="0" onclick="this.focus()">
        <span class="dollar">$</span><span id="withdraw-display">0.00</span><span class="atm-cursor"></span>
      </div>
      <div class="chip-row" style="justify-content:center;">
        <button class="chip" onclick="setAtmCents('withdraw',2500)">$25</button>
        <button class="chip" onclick="setAtmCents('withdraw',5000)">$50</button>
        <button class="chip" onclick="setAtmCents('withdraw',10000)">$100</button>
      </div>
      <button class="submit-btn amber" onclick="submitWithdraw()">Withdraw</button>
    </div>
  </div>
</div>`
}

func historySheetHTML() string {
	return `
<div class="sheet-backdrop" id="history-sheet" onclick="if(event.target===this) closeSheet('history')">
  <div class="sheet">
    <div class="sheet-handle"></div>
    <div class="sheet-title" id="history-sheet-title">Payment history</div>
    <div class="sheet-pane active" style="display:flex;padding-top:8px;">
      <div id="history-list" style="display:flex;flex-direction:column;max-height:480px;overflow-y:auto;"></div>
    </div>
  </div>
</div>`
}

// ---- Styles ------------------------------------------------------------------

func dashboardStyles() string {
	return `
<style>
  @import url('https://fonts.googleapis.com/css2?family=Onest:wght@400;500;600;700;800&family=JetBrains+Mono:wght@500;600;700&display=swap');

  html { scroll-padding-top: 62px; }

  :root {
    --bg: #020617; --surface: #0b1220; --surface-2: #0f172a; --surface-3: #1e293b;
    --border: #1e293b; --border-soft: #131c30;
    --text: #f1f5f9; --text-dim: #94a3b8; --text-mute: #64748b; --text-faint: #475569;
    --indigo: #6366f1; --indigo-soft: #1e1b4b; --indigo-hi: #818cf8;
    --emerald: #10b981; --emerald-soft: #052e16; --emerald-hi: #34d399;
    --amber: #f59e0b; --amber-hi: #fbbf24;
    --rose: #ef4444; --rose-soft: #1c0a0a; --rose-hi: #f87171;
  }

  .desktop-only { display: flex !important; }
  .mobile-only  { display: none  !important; }
  @media (max-width: 1023px) {
    .desktop-only { display: none  !important; }
    .mobile-only  { display: flex  !important; }
  }

  /* Inline back button used in Settings, Wallet, Analytics, and Group Detail views */
  .gd-back-btn { width: 34px; height: 34px; border-radius: 50%; background: var(--surface-2); border: 1px solid var(--border); color: var(--text-dim); cursor: pointer; display: flex; align-items: center; justify-content: center; flex-shrink: 0; transition: background .15s, color .15s; }
  .gd-back-btn:hover { background: var(--surface-3); color: var(--text); }

  #main-application-viewport { max-width: 100% !important; padding: 0 !important; }
  body { background: var(--bg) !important; padding: 0 !important; display: block !important; min-height: 100vh; align-items: initial !important; justify-content: initial !important; }

  .app-shell { font-family: 'Onest', system-ui, -apple-system, sans-serif; color: var(--text); background: var(--bg); min-height: 100vh; min-height: 100dvh; -webkit-font-smoothing: antialiased; }
  .mono { font-family: 'JetBrains Mono', monospace; font-feature-settings: 'tnum' 1; }

  /* ---- Topbar ---- */
  .topbar { position: sticky; top: 0; z-index: 40; background: rgba(2,6,23,0.85); backdrop-filter: blur(14px); -webkit-backdrop-filter: blur(14px); border-bottom: 1px solid var(--border-soft); }
  .topbar-inner { max-width: 1280px; margin: 0 auto; display: flex; align-items: center; justify-content: space-between; padding: 14px 20px; }
  .brand-btn { background: none; border: none; cursor: pointer; padding: 0; }
  .brand { font-size: 38px; font-weight: 800; letter-spacing: -0.04em; color: var(--text); }
  .brand span { color: var(--indigo); }
  .topbar-actions { display: flex; align-items: center; gap: 10px; }
  .icon-btn { position: relative; width: 38px; height: 38px; border-radius: 50%; background: var(--surface-2); border: 1px solid var(--border); color: var(--text-dim); cursor: pointer; display: flex; align-items: center; justify-content: center; transition: background .15s, color .15s; }
  .icon-btn:hover { background: var(--surface-3); color: var(--text); }
  .icon-btn .dot { position: absolute; top: 9px; right: 10px; width: 7px; height: 7px; border-radius: 50%; background: var(--emerald-hi); border: 2px solid var(--surface-2); }
  .avatar-btn { width: 38px; height: 38px; border-radius: 50%; border: 2px solid rgba(255,255,255,0.15); cursor: pointer; display: flex; align-items: center; justify-content: center; font-size: 13px; font-weight: 700; color: #fff; font-family: inherit; transition: transform .15s; }
  .avatar-btn:hover { transform: scale(1.06); }

  /* ---- Layout ---- */
  .app-body { max-width: 1280px; margin: 0 auto; display: grid; grid-template-columns: 1fr; min-height: calc(100vh - 60px); }
  .sidebar { display: none; }
  .content { padding: 18px 16px 100px; width: 100%; max-width: 720px; margin: 0 auto; box-sizing: border-box; }
  .view { display: none; animation: fadeUp .35s cubic-bezier(0.16,1,0.3,1) both; }
  .view.active { display: block; }
  @keyframes fadeUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }
  @keyframes fadeIn  { from { opacity: 0; } to { opacity: 1; } }

  /* ---- Hero ---- */
  .hero { position: relative; background: linear-gradient(135deg,#1e1b4b 0%,#0f172a 60%,#0b1220 100%); border: 1px solid #312e81; border-radius: 20px; padding: 22px 22px 24px; overflow: hidden; }
  .hero::before { content:""; position:absolute; inset:-40% -10% auto auto; width:280px; height:280px; background:radial-gradient(circle,rgba(99,102,241,.35),transparent 60%); pointer-events:none; }
  .hero-label { position:relative; font-size:11px; color:var(--indigo-hi); font-weight:700; letter-spacing:.08em; text-transform:uppercase; }
  .hero-amount { position:relative; font-size:clamp(34px,7vw,44px); font-weight:800; color:#f8fafc; letter-spacing:-.035em; margin-top:6px; line-height:1; }
  .hero-amount .cents { font-size:.55em; color:#94a3b8; font-weight:700; }
  .hero-meta { position:relative; display:flex; gap:18px; margin-top:16px; font-size:13px; color:var(--text-dim); }
  .hero-meta b { color:var(--text); font-weight:600; }

  /* ---- Actions ---- */
  .actions { display:grid; grid-template-columns:1fr 1fr; gap:12px; margin:18px 0; }
  .action { display:flex; align-items:center; gap:12px; background:var(--surface-2); border:1px solid var(--border); border-radius:16px; padding:16px 18px; cursor:pointer; text-align:left; color:var(--text); font-family:inherit; font-size:15px; font-weight:600; transition:border-color .2s,transform .1s,background .2s; }
  .action:hover { border-color:var(--indigo); background:#131c30; }
  .action:active { transform:scale(.98); }
  .action .ico { width:38px; height:38px; border-radius:12px; display:flex; align-items:center; justify-content:center; flex-shrink:0; }
  .action.pay .ico { background:var(--indigo-soft); color:var(--indigo-hi); }
  .action.req .ico { background:#1c0a00; color:var(--amber-hi); }
  .action .lbl-stack { display:flex; flex-direction:column; }
  .action .lbl-stack b { font-size:15px; font-weight:700; }
  .action .lbl-stack small { font-size:12px; color:var(--text-mute); font-weight:500; margin-top:2px; }

  /* ---- Quick strip ---- */
  .quick-strip { display:grid; grid-template-columns:1fr 1fr; gap:10px; margin-bottom:18px; }
  .qstat { background:var(--surface-2); border:1px solid var(--border); border-radius:14px; padding:12px 14px; display:flex; align-items:center; justify-content:space-between; gap:8px; cursor:pointer; transition:border-color .2s,background .2s; }
  .qstat:hover { border-color:var(--surface-3); }
  .qstat-label { font-size:12px; color:var(--text-mute); font-weight:600; }
  .qstat-val { font-size:16px; color:var(--text); font-weight:700; }
  .qstat.warn { border-color:rgba(239,68,68,.35); background:rgba(239,68,68,.06); }
  .qstat.warn .qstat-val { color:var(--rose-hi); }
  .qstat.warn .qstat-label { color:#fca5a5; }

  /* ---- Section row ---- */
  .section-row { display:flex; align-items:center; justify-content:space-between; margin:22px 0 10px; }
  .section-row h2 { font-size:15px; font-weight:700; color:var(--text); margin:0; letter-spacing:-.01em; }
  .section-row a, .section-row button.linklike { font-size:13px; color:var(--indigo-hi); text-decoration:none; font-weight:600; background:none; border:none; cursor:pointer; font-family:inherit; }
  .section-row a:hover, .section-row button.linklike:hover { color:var(--indigo); }

  /* ---- Card / Row ---- */
  .card { background:var(--surface-2); border:1px solid var(--border); border-radius:16px; overflow:hidden; }
  .row { display:flex; align-items:center; gap:12px; padding:14px 16px; border-bottom:1px solid var(--border-soft); transition:background .15s; }
  .row:last-child { border-bottom:none; }
  .row:hover { background:var(--surface-3); }
  .row-avatar { width:40px; height:40px; border-radius:50%; display:flex; align-items:center; justify-content:center; font-size:13px; font-weight:700; flex-shrink:0; font-family:inherit; }
  .row-body { flex:1; min-width:0; }
  .row-title { font-size:14px; color:var(--text); font-weight:500; line-height:1.35; }
  .row-title b { font-weight:700; }
  .row-sub { font-size:12px; color:var(--text-faint); margin-top:2px; }
  .row-right { text-align:right; flex-shrink:0; }
  .row-amt { font-size:15px; font-weight:700; letter-spacing:-.01em; }
  .row-amt.pos  { color:var(--emerald-hi); }
  .row-amt.neg  { color:var(--rose-hi); }
  .row-amt.req  { color:var(--amber-hi); }
  .row-amt.bnpl { color:var(--indigo-hi); }
  .row-time { color:var(--text-faint); font-size:11px; margin-top:2px; }

  /* ---- Subtabs ---- */
  .subtabs { display:flex; gap:4px; background:var(--surface-2); border:1px solid var(--border); border-radius:12px; padding:4px; margin-bottom:16px; }
  .subtab { flex:1; background:none; border:none; cursor:pointer; padding:9px 8px; border-radius:8px; font-size:13px; font-weight:600; color:var(--text-mute); font-family:inherit; transition:background .15s,color .15s; display:flex; align-items:center; justify-content:center; gap:6px; }
  .subtab:hover { color:var(--text-dim); }
  .subtab.active { background:var(--surface-3); color:var(--text); }
  .subtab .badge { display:inline-flex; align-items:center; justify-content:center; min-width:18px; height:18px; padding:0 5px; border-radius:9px; background:var(--rose); color:#fff; font-size:10px; font-weight:700; font-family:'JetBrains Mono',monospace; }

  /* ---- Overdue ---- */
  .overdue-row { background:rgba(239,68,68,.06); }
  .overdue-row:hover { background:rgba(239,68,68,.12); }
  .overdue-row .row-title b { color:var(--rose-hi); }
  .overdue-banner { background:linear-gradient(135deg,rgba(239,68,68,.12),rgba(239,68,68,.04)); border:1px solid rgba(239,68,68,.35); border-radius:14px; padding:14px 16px; margin-bottom:14px; display:flex; align-items:center; gap:12px; }
  .overdue-banner svg { flex-shrink:0; color:var(--rose-hi); }
  .overdue-banner .ob-text { flex:1; }
  .overdue-banner .ob-text strong { color:var(--rose-hi); font-weight:700; font-size:14px; }
  .overdue-banner .ob-text div { color:#fca5a5; font-size:12px; margin-top:2px; }
  .overdue-banner button { background:var(--rose); color:#fff; border:none; border-radius:10px; padding:9px 14px; font-size:13px; font-weight:700; cursor:pointer; font-family:inherit; transition:background .2s; }
  .overdue-banner button:hover { background:#dc2626; }

  /* ---- Pills / progress ---- */
  .pill { display:inline-flex; align-items:center; padding:2px 8px; border-radius:99px; font-size:11px; font-weight:600; background:var(--indigo-soft); color:var(--indigo-hi); margin-left:8px; }
  .pill.warn { background:rgba(239,68,68,.15); color:var(--rose-hi); }
  .pill.ok   { background:var(--emerald-soft); color:var(--emerald-hi); }
  .progress { height:4px; border-radius:2px; background:var(--surface-3); overflow:hidden; margin-top:8px; }
  .progress > span { display:block; height:100%; background:var(--indigo); border-radius:2px; }
  .progress.warn > span { background:var(--rose); }

  /* ---- Friends / Social ---- */
  .friends-header { display:flex; align-items:center; justify-content:space-between; margin-bottom:14px; }
  .friends-count { font-size:24px; font-weight:800; letter-spacing:-.02em; }
  .friends-count small { font-size:14px; color:var(--text-mute); font-weight:600; margin-left:6px; }
  .friend-add { background:var(--indigo); color:#fff; border:none; border-radius:10px; padding:9px 14px; font-size:13px; font-weight:700; cursor:pointer; font-family:inherit; display:flex; align-items:center; gap:6px; transition:background .2s; }
  .friend-add:hover { background:#4f46e5; }
  .search-inp { width:100%; box-sizing:border-box; background:var(--surface-2); border:1px solid var(--border); border-radius:12px; padding:11px 14px 11px 38px; font-size:14px; color:var(--text); outline:none; font-family:inherit; transition:border-color .2s; }
  .search-inp:focus { border-color:var(--indigo); }
  .search-inp::placeholder { color:var(--text-faint); }
  .search-wrap { position:relative; margin-bottom:14px; }
  .search-wrap svg { position:absolute; left:12px; top:50%; transform:translateY(-50%); color:var(--text-faint); pointer-events:none; }
  .friend-row { display:flex; align-items:center; gap:12px; padding:12px 14px; border-bottom:1px solid var(--border-soft); }
  .friend-row:last-child { border-bottom:none; }
  .friend-row .row-body b { font-size:14px; color:var(--text); font-weight:600; }
  .friend-row .row-body div { font-size:12px; color:var(--text-faint); margin-top:2px; }
  .friend-actions { display:flex; gap:6px; }
  .friend-actions button { background:var(--surface-3); border:1px solid var(--border); color:var(--text-dim); cursor:pointer; width:32px; height:32px; border-radius:8px; display:flex; align-items:center; justify-content:center; transition:background .15s,color .15s,border-color .15s; }
  .friend-actions button:hover { background:var(--surface-2); color:var(--text); border-color:var(--indigo); }
  .friend-actions button.remove:hover { color:var(--rose-hi); border-color:var(--rose); }

  /* ---- Groups ---- */
  .group-row { display:flex; align-items:center; gap:12px; padding:14px 16px; border-bottom:1px solid var(--border-soft); transition:background .15s; }
  .group-row:last-child { border-bottom:none; }
  .group-row:hover { background:var(--surface-3); }
  .badge { display:inline-flex; align-items:center; justify-content:center; min-width:18px; height:18px; padding:0 5px; border-radius:9px; background:var(--rose); color:#fff; font-size:10px; font-weight:700; }

  /* ---- Profile ---- */
  .profile-hero { background:var(--surface-2); border:1px solid var(--border); border-radius:20px; padding:24px; margin-bottom:16px; text-align:center; }
  .avatar-xl { width:80px; height:80px; border-radius:50%; border:2px solid rgba(255,255,255,0.15); margin:0 auto 14px; display:flex; align-items:center; justify-content:center; font-size:28px; font-weight:700; color:#fff; font-family:inherit; }
  .profile-name { font-size:22px; font-weight:800; color:var(--text); letter-spacing:-.02em; }
  .profile-handle, .profile-email, .profile-phone_number { font-size:13px; color:var(--text-faint); margin-top:2px; }
  .profile-handle { color:var(--text-mute); margin-top:4px; }

  /* ---- Score card ---- */
  .score-card { background:linear-gradient(135deg,#1e1b4b 0%,#0f172a 100%); border:1px solid #312e81; border-radius:20px; padding:22px; margin-bottom:16px; }
  .score-header { display:flex; align-items:center; justify-content:space-between; margin-bottom:14px; }
  .score-title { font-size:13px; color:var(--indigo-hi); font-weight:700; letter-spacing:.06em; text-transform:uppercase; }
  .score-pill { background:rgba(52,211,153,.12); color:var(--emerald-hi); padding:3px 10px; border-radius:99px; font-size:11px; font-weight:700; letter-spacing:.03em; }
  .score-val { font-size:56px; font-weight:800; line-height:1; color:var(--text); letter-spacing:-.04em; }
  .score-val small { font-size:18px; color:var(--text-mute); font-weight:600; margin-left:4px; }
  .score-track { height:8px; border-radius:4px; background:rgba(255,255,255,.08); overflow:hidden; margin:16px 0 8px; }
  .score-track > span { display:block; height:100%; background:linear-gradient(90deg,var(--rose),var(--amber),var(--emerald-hi)); border-radius:4px; }
  .score-track-labels { display:flex; justify-content:space-between; font-size:10px; color:var(--text-faint); font-weight:600; letter-spacing:.05em; text-transform:uppercase; }

  /* ---- Stat grid ---- */
  .stat-grid { display:grid; grid-template-columns:repeat(3,1fr); gap:10px; margin-bottom:16px; }
  .stat-tile { background:var(--surface-2); border:1px solid var(--border); border-radius:14px; padding:14px; text-align:left; }
  .stat-tile .stat-lbl { font-size:11px; color:var(--text-mute); font-weight:600; letter-spacing:.05em; text-transform:uppercase; margin-bottom:6px; }
  .stat-tile .stat-val { font-size:18px; font-weight:700; color:var(--text); letter-spacing:-.02em; }
  .stat-tile .stat-val.green { color:var(--emerald-hi); }
  .stat-tile .stat-val.indigo { color:var(--indigo-hi); }

  /* ---- Profile menu ---- */
  .menu-list { background:var(--surface-2); border:1px solid var(--border); border-radius:14px; overflow:hidden; }
  .menu-item { width:100%; background:none; border:none; display:flex; align-items:center; gap:12px; padding:14px 16px; cursor:pointer; color:var(--text); font-size:14px; font-weight:500; font-family:inherit; text-align:left; border-bottom:1px solid var(--border-soft); transition:background .15s; }
  .menu-item:last-child { border-bottom:none; }
  .menu-item:hover { background:var(--surface-3); }
  .menu-item svg { color:var(--text-mute); flex-shrink:0; }
  .menu-item.danger { color:var(--rose-hi); }
  .menu-item.danger svg { color:var(--rose-hi); }
  .menu-item .chev { margin-left:auto; color:var(--text-faint); }

  /* ---- Empty state ---- */
  .empty { text-align:center; padding:36px 20px; background:var(--surface-2); border:1px dashed var(--border); border-radius:16px; }
  .empty-icon { width:56px; height:56px; border-radius:50%; background:var(--surface-3); color:var(--text-dim); margin:0 auto 12px; display:flex; align-items:center; justify-content:center; }
  .empty-icon.success { background:var(--emerald-soft); color:var(--emerald-hi); }
  .empty-title { font-size:15px; font-weight:700; color:var(--text); margin-bottom:4px; }
  .empty-sub { font-size:13px; color:var(--text-mute); max-width:280px; margin:0 auto; }
  .empty button { margin-top:14px; background:var(--indigo); color:#fff; border:none; border-radius:10px; padding:9px 16px; font-size:13px; font-weight:700; cursor:pointer; font-family:inherit; }
  .empty button:hover { background:#4f46e5; }

  /* ---- Settings: toggles ---- */
  .toggle-row { display:flex; align-items:center; justify-content:space-between; padding:14px 16px; border-bottom:1px solid var(--border-soft); }
  .toggle-row:last-child { border-bottom:none; }
  .toggle-label { font-size:14px; font-weight:600; color:var(--text); }
  .toggle-sub { font-size:12px; color:var(--text-mute); margin-top:2px; }
  .toggle-switch { position:relative; width:44px; height:24px; flex-shrink:0; }
  .toggle-switch input { opacity:0; width:0; height:0; position:absolute; }
  .toggle-slider { position:absolute; cursor:pointer; inset:0; background:var(--surface-3); border-radius:12px; transition:.2s; }
  .toggle-slider:before { content:""; position:absolute; height:18px; width:18px; left:3px; bottom:3px; background:var(--text-dim); border-radius:50%; transition:.2s; }
  .toggle-switch input:checked + .toggle-slider { background:var(--indigo); }
  .toggle-switch input:checked + .toggle-slider:before { transform:translateX(20px); background:#fff; }

  /* ---- Wallet: direction badges ---- */
  .dir-badge { font-size:10px; font-weight:700; letter-spacing:.05em; padding:2px 7px; border-radius:6px; margin-left:6px; }
  .dir-in  { background:var(--emerald-soft); color:var(--emerald-hi); }
  .dir-out { background:var(--rose-soft);    color:var(--rose-hi); }

  /* ---- Mobile tabbar ---- */
  .tabbar { position:fixed; bottom:0; left:0; right:0; z-index:30; background:rgba(2,6,23,.92); backdrop-filter:blur(14px); -webkit-backdrop-filter:blur(14px); border-top:1px solid var(--border-soft); display:grid; grid-template-columns:repeat(5,1fr); padding:6px 6px max(6px,env(safe-area-inset-bottom)); }
  .tab { background:none; border:none; cursor:pointer; display:flex; flex-direction:column; align-items:center; gap:2px; padding:8px 0; color:var(--text-faint); font-family:inherit; font-size:10px; font-weight:600; transition:color .15s; border-radius:10px; }
  .tab span { letter-spacing:.01em; }
  .tab.active { color:var(--indigo-hi); }
  .tab:active { background:var(--surface-2); }

  /* ---- Sheets ---- */
  .sheet-backdrop { display:none; position:fixed; inset:0; z-index:60; background:rgba(0,0,0,.7); align-items:flex-end; justify-content:center; }
  .sheet-backdrop.open { display:flex; animation:fadeIn .2s both; }
  .sheet { background:var(--surface-2); border:1px solid var(--border); border-radius:24px 24px 0 0; width:100%; max-width:540px; padding:0 0 max(28px,env(safe-area-inset-bottom)); animation:slideUp .35s cubic-bezier(0.34,1.36,0.64,1) both; max-height:92vh; max-height:92dvh; overflow-y:auto; }
  @keyframes slideUp { from { transform:translateY(100%); } to { transform:translateY(0); } }
  .sheet-handle { width:40px; height:4px; background:var(--surface-3); border-radius:2px; margin:12px auto 0; }
  .sheet-title { font-size:18px; font-weight:700; color:var(--text); padding:14px 22px 4px; }
  .sheet-tabs { display:flex; padding:0 22px; border-bottom:1px solid var(--border-soft); margin-top:8px; }
  .sheet-tab { flex:1; background:none; border:none; cursor:pointer; padding:12px 4px; font-size:14px; font-weight:600; color:var(--text-mute); font-family:inherit; border-bottom:2px solid transparent; transition:color .2s,border-color .2s; }
  .sheet-tab.active { color:var(--indigo); border-bottom-color:var(--indigo); }
  .sheet-pane { padding:20px 22px; display:none; flex-direction:column; gap:16px; }
  .sheet-pane.active { display:flex; }

  /* ---- Modal inputs ---- */
  .modal-inp, .modal-sel { width:100%; box-sizing:border-box; background:var(--bg); border:1px solid var(--border); border-radius:12px; padding:12px 14px; font-size:14px; color:var(--text); outline:none; font-family:inherit; transition:border-color .2s; }
  .modal-inp:focus, .modal-sel:focus { border-color:var(--indigo); }
  .modal-inp::placeholder { color:var(--text-faint); }
  .modal-lbl { display:block; font-size:11px; color:var(--text-mute); margin-bottom:6px; font-weight:700; letter-spacing:.06em; text-transform:uppercase; }
  .chip-row { display:flex; gap:6px; flex-wrap:wrap; }
  .chip { background:var(--bg); border:1px solid var(--border); border-radius:20px; padding:6px 14px; color:var(--text-dim); font-size:13px; font-weight:600; cursor:pointer; font-family:inherit; transition:border-color .15s,color .15s,background .15s; }
  .chip:hover { color:var(--text); border-color:var(--surface-3); }
  .chip.active { background:var(--indigo-soft); border-color:var(--indigo); color:var(--indigo-hi); }
  .amount-big { font-family:'JetBrains Mono',monospace; font-size:44px; font-weight:700; color:var(--text); letter-spacing:-.03em; text-align:center; padding:8px 0 4px; }
  .amount-big .dollar { color:var(--text-faint); margin-right:4px; }

  /* ---- Submit button ---- */
  .submit-btn { width:100%; background:var(--indigo); color:#fff; border:none; border-radius:12px; padding:14px; font-size:15px; font-weight:700; cursor:pointer; font-family:inherit; transition:background .2s,transform .1s; margin-top:4px; }
  .submit-btn:hover { background:#4f46e5; }
  .submit-btn:active { transform:scale(.99); }
  .submit-btn.amber   { background:var(--amber); }
  .submit-btn.amber:hover   { background:#d97706; }
  .submit-btn.emerald { background:var(--emerald); }
  .submit-btn.emerald:hover { background:#059669; }

  /* ---- Info card ---- */
  .info-card { background:var(--bg); border:1px solid var(--border-soft); border-radius:12px; padding:12px 14px; display:flex; justify-content:space-between; align-items:center; font-size:13px; }
  .info-card .label { color:var(--text-mute); font-weight:600; }
  .info-card .val { color:var(--emerald-hi); font-weight:700; font-family:'JetBrains Mono',monospace; }

  /* ---- Toast ---- */
  #toast { display:none; position:fixed; top:72px; left:50%; transform:translateX(-50%); background:var(--emerald-soft); border:1px solid #166534; color:#4ade80; border-radius:12px; padding:11px 20px; font-size:13px; font-weight:600; font-family:inherit; z-index:200; white-space:nowrap; box-shadow:0 8px 24px rgba(0,0,0,.4); }
  #toast.show { display:block; animation:fadeUp .2s both; }
  #toast.warn { background:var(--rose-soft); border-color:#7f1d1d; color:var(--rose-hi); }

  /* ---- Responsive ---- */
  @media (min-width: 640px) {
    .content { padding:28px 24px 100px; max-width:760px; }
    .actions { gap:14px; }
    .stat-grid { gap:12px; }
    .hero { padding:26px 28px 28px; border-radius:24px; }
  }
  @media (min-width: 1024px) {
    .app-body { grid-template-columns:240px 1fr; gap:28px; padding:0 20px; }
    .sidebar { display:block; padding:28px 0; position:sticky; top:60px; align-self:start; }
    .side-nav { display:flex; flex-direction:column; gap:4px; }
    .side-link { display:flex; align-items:center; gap:12px; width:100%; padding:10px 14px; background:none; border:none; border-radius:10px; color:var(--text-dim); font-family:inherit; font-size:14px; font-weight:600; cursor:pointer; text-align:left; transition:background .15s,color .15s; }
    .side-link:hover { background:var(--surface-2); color:var(--text); }
    .side-link.active { background:var(--indigo-soft); color:var(--indigo-hi); }
    .side-link svg { flex-shrink:0; }
    .side-promo { display:block; margin-top:24px; background:linear-gradient(135deg,#1e1b4b 0%,#0f172a 100%); border:1px solid #312e81; border-radius:16px; padding:18px; }
    .promo-title { font-size:11px; color:var(--indigo-hi); font-weight:700; letter-spacing:.06em; text-transform:uppercase; }
    .promo-score { font-size:32px; font-weight:800; color:var(--text); margin-top:6px; letter-spacing:-.03em; }
    .promo-score .promo-small { font-size:14px; color:var(--text-mute); font-weight:600; margin-left:4px; }
    .promo-sub { font-size:12px; color:var(--text-dim); margin-top:4px; }
    .content { padding:28px 0 60px; max-width:none; margin:0; }
    .tabbar { display:none; }
    .topbar-inner { padding:14px 28px; }
  }
  .side-promo { display:none; }

  /* ---- Friend picker ---- */
  .friend-picker { display:flex; gap:8px; overflow-x:auto; padding:4px 2px; scrollbar-width:none; -ms-overflow-style:none; }
  .friend-picker::-webkit-scrollbar { display:none; }
  .fp-item { display:flex; flex-direction:column; align-items:center; gap:6px; cursor:pointer; padding:10px 12px; border-radius:14px; border:1.5px solid var(--border); background:var(--bg); min-width:64px; flex-shrink:0; transition:border-color .15s,background .15s; }
  .fp-item:hover { border-color:var(--indigo); }
  .fp-item.selected { border-color:var(--indigo); background:var(--indigo-soft); }
  .fp-avatar-sm { width:40px; height:40px; border-radius:50%; display:flex; align-items:center; justify-content:center; font-size:13px; font-weight:700; }
  .fp-name { font-size:11px; font-weight:600; color:var(--text-dim); text-align:center; max-width:60px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
  .fp-item.selected .fp-name { color:var(--indigo-hi); }
  .fp-empty { font-size:13px; color:var(--text-mute); padding:8px 0; }

  /* ---- Group picker section (Pay / Request sheets) ---- */
  .gp-section { margin-top:14px; }
  .gp-label { font-size:11px; font-weight:700; color:var(--text-mute); text-transform:uppercase; letter-spacing:0.06em; margin-bottom:6px; }
  .gp-group-row { display:flex; align-items:center; gap:8px; padding:8px 10px; border-radius:10px; cursor:pointer; transition:background .15s; }
  .gp-group-row:hover { background:var(--surface-3); }
  .gp-group-row .fp-name { flex:1; font-size:13px; font-weight:600; color:var(--text); }
  .gp-chev { color:var(--text-mute); flex-shrink:0; transition:transform .2s; }
  .gp-members-wrap { margin-bottom:4px; }

  /* ---- ATM input ---- */
  .atm-wrap { cursor:text; user-select:none; }
  .atm-cursor { display:none; width:2px; height:.8em; background:var(--indigo); margin-left:2px; vertical-align:middle; border-radius:1px; animation:atmBlink .8s step-end infinite; }
  .atm-wrap:focus { outline:none; }
  .atm-wrap:focus .atm-cursor { display:inline-block; }
  @keyframes atmBlink { 50% { opacity:0; } }
</style>
`
}

// ---- Script ------------------------------------------------------------------

func dashboardScript() string {
	return `
<script>
  if (history.scrollRestoration) history.scrollRestoration = 'manual';
  window.scrollTo(0, 0);

  // ---- ATM Input System --------------------------------------------------------
  var _atmState = {};
  function initAtm(id) { _atmState[id] = 0; _updateAtmDisplay(id); }
  function atmKey(id, key) {
    var c = _atmState[id] || 0;
    if (key >= '0' && key <= '9') { c = Math.min(c * 10 + parseInt(key, 10), 9999999); }
    else if (key === 'Backspace')  { c = Math.floor(c / 10); }
    _atmState[id] = c; _updateAtmDisplay(id);
  }
  function setAtmCents(id, cents) { _atmState[id] = cents; _updateAtmDisplay(id); }
  function getAtmCents(id) { return _atmState[id] || 0; }
  function _updateAtmDisplay(id) {
    var el = document.getElementById(id + '-display');
    if (el) el.textContent = ((_atmState[id] || 0) / 100).toFixed(2);
    // Fire a per-ATM update hook if registered (e.g. BNPL breakdown)
    var hook = window['_onAtmUpdate_' + id.replace(/-/g, '_')];
    if (typeof hook === 'function') hook();
  }

  // ---- Friend/Group picker ----------------------------------------------------
  function pickFriend(pickerID, friendID, name) {
    document.querySelectorAll('#' + pickerID + '-picker .fp-item').forEach(function(el) { el.classList.remove('selected'); });
    var gp = document.getElementById(pickerID + '-group-picker');
    if (gp) gp.querySelectorAll('.fp-item').forEach(function(el) { el.classList.remove('selected'); });
    var item = document.querySelector('#' + pickerID + '-picker [data-id="' + friendID + '"]');
    if (!item && gp) item = gp.querySelector('[data-id="' + friendID + '"]');
    if (item) item.classList.add('selected');
    var valEl = document.getElementById(pickerID + '-val');
    if (valEl) valEl.value = friendID;
  }

  // ---- Initials helper -------------------------------------------------------
  function computeInitials(name) {
    if (!name) return '?';
    var parts = (name || '').trim().split(/\s+/);
    if (parts.length >= 2) return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
    return (Array.from(parts[0])[0] || '?').toUpperCase();
  }

  // ---- View router with lazy-load support ------------------------------------
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

  // ---- Sub-pane router -------------------------------------------------------
  // Scoped to the nearest [data-view] ancestor so switching panes in group-detail
  // does not affect the activity view's pane state, and vice versa.
  function goPane(btn) {
    var target  = btn.getAttribute('data-pane');
    var section = btn.closest('[data-view]') || document;
    section.querySelectorAll('.subtab').forEach(function(b) { b.classList.remove('active'); });
    btn.classList.add('active');
    section.querySelectorAll('[data-pane-content]').forEach(function(c) {
      c.style.display = c.getAttribute('data-pane-content') === target ? 'block' : 'none';
    });
  }

  // ---- Sheets ----------------------------------------------------------------
  function openSheet(which) {
    var sheet = document.getElementById(which + '-sheet');
    if (!sheet) return;
    sheet.classList.add('open');
    sheet.querySelectorAll('.atm-wrap').forEach(function(el) {
      var id = el.getAttribute('data-atm');
      if (id) initAtm(id);
    });
    sheet.querySelectorAll('.fp-item').forEach(function(el) { el.classList.remove('selected'); });
    sheet.querySelectorAll('[id$="-val"]').forEach(function(el) { el.value = ''; });
    requestAnimationFrame(function() {
      var first = sheet.querySelector('.atm-wrap');
      if (first) first.focus();
    });
  }
  function closeSheet(which) {
    document.getElementById(which + '-sheet').classList.remove('open');
    // Clear addfriend state on close
    if (which === 'addfriend') {
      var inp = document.getElementById('addfriend-search-inp');
      var res = document.getElementById('addfriend-results');
      if (inp) inp.value = '';
      if (res) res.innerHTML = '';
    }
  }
  function payTab(btn) {
    var target = btn.getAttribute('data-paytab');
    document.querySelectorAll('.sheet-tab').forEach(function(b) { b.classList.remove('active'); });
    btn.classList.add('active');
    document.querySelectorAll('[data-pay-pane]').forEach(function(p) {
      p.classList.toggle('active', p.getAttribute('data-pay-pane') === target);
    });
  }

  // ---- Toast -----------------------------------------------------------------
  var _toastTimer = null;
  function showToast(msg, kind) {
    var t = document.getElementById('toast');
    t.textContent = msg;
    t.className = kind === 'warn' ? 'show warn' : 'show';
    clearTimeout(_toastTimer);
    _toastTimer = setTimeout(function() { t.className = ''; }, 2600);
  }

  // ---- API helpers -----------------------------------------------------------
  function getCsrfToken() {
    var m = document.cookie.match('(?:^|;)\\s*csrf_token=([^;]+)');
    return m ? m[1] : '';
  }
  function apiPost(url, body, onOk, onErr) {
    fetch(url, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCsrfToken() }, credentials: 'same-origin', body: JSON.stringify(body) })
      .then(function(res) { return res.json().then(function(d) { return { ok: res.ok, data: d }; }); })
      .then(function(r) { if (r.ok) { onOk(r.data); } else { onErr(r.data.error || 'Request failed'); } })
      .catch(function() { onErr('Network error'); });
  }

  function apiGet(url, onOk, onErr) {
    fetch(url, { credentials: 'same-origin' })
      .then(function(r) { return r.json(); })
      .then(onOk)
      .catch(function() { if (onErr) onErr('Network error'); });
  }

  // ---- Pay / BNPL / Request --------------------------------------------------
  function openPayToFriend(friendID) {
    openSheet('pay');
    requestAnimationFrame(function() {
      var item = document.querySelector('#pay-recv-picker [data-id="' + friendID + '"]');
      if (item) item.click();
    });
  }

  function submitSendPayment() {
    var receiverID  = document.getElementById('pay-recv-val').value;
    var amountCents = getAtmCents('pay-send');
    var note        = document.getElementById('pay-note-inp').value;
    if (!receiverID)      { showToast('Please select a recipient', 'warn'); return; }
    if (amountCents <= 0) { showToast('Please enter a valid amount', 'warn'); return; }
    apiPost('/payments/pay', { receiver_id: receiverID, amount_cents: amountCents, note: note, payment_type: 'direct' },
      function() { closeSheet('pay'); showToast('Payment sent'); setTimeout(function() { location.reload(); }, 1200); },
      function(e) { showToast(e, 'warn'); });
  }

  function submitBNPL() {
    var receiverID  = document.getElementById('bnpl-recv-val').value;
    var amountCents = getAtmCents('bnpl');
    var note        = document.getElementById('bnpl-note-inp').value;
    if (!receiverID)      { showToast('Please select a recipient', 'warn'); return; }
    if (amountCents <= 0) { showToast('Please enter a valid amount', 'warn'); return; }
    // Always Pay-in-4. Send the item price as total_amount_cents so the backend fee engine can apply the credit-score-based rate.
    apiPost('/bnpl/loan/create', { receiver_id: receiverID, total_amount_cents: amountCents, note: note, payment_type: 'bnpl', total_installments: 4 },
      function() { closeSheet('pay'); showToast('BNPL plan approved'); setTimeout(function() { location.reload(); }, 1200); },
      function(e) { showToast(e, 'warn'); });
  }

  // ---- BNPL breakdown (mirrors CalculateFeeRate in store/installments.go) ------
  function _bnplFeeRate(score) {
    if (score >= 90) return 0.01;
    if (score >= 75) return 0.02;
    if (score >= 50) return 0.03;
    return 0.07;
  }
  window._onAtmUpdate_bnpl = function() {
    var cents  = getAtmCents('bnpl');
    var score  = parseInt((document.getElementById('app-root') || {}).dataset.creditScore || '50', 10);
    var rate   = _bnplFeeRate(score);
    var fee    = Math.round(cents * rate);
    var total  = cents + fee;
    var base   = Math.floor(total / 4);
    var rem    = total - (base * 4);
    var upfront    = base + rem;
    var recurring  = base;

    var breakdownEl = document.getElementById('bnpl-breakdown');
    if (!breakdownEl) return;
    if (cents <= 0) { breakdownEl.style.display = 'none'; return; }
    breakdownEl.style.display = 'block';
    document.getElementById('bnpl-due-now').textContent    = '$' + (upfront   / 100).toFixed(2);
    document.getElementById('bnpl-recurring').textContent  = '$' + (recurring / 100).toFixed(2);
    document.getElementById('bnpl-rate-display').textContent = (rate * 100).toFixed(2) + '%';
    var tc = document.getElementById('bnpl-total-cents');
    if (tc) tc.value = total;
  };

  // ---- Group picker (Pay / Request sheets) -----------------------------------
  function toggleGroupMembers(pickerID, groupID, groupName, rowEl) {
    var wrapID = 'gpm-' + pickerID + '-' + groupID;
    var wrap   = document.getElementById(wrapID);
    if (!wrap) return;
    var isOpen = wrap.style.display !== 'none';
    if (isOpen) {
      wrap.style.display = 'none';
      if (rowEl) rowEl.querySelector('.gp-chev') && (rowEl.querySelector('.gp-chev').style.transform = '');
      return;
    }
    // Already loaded — just show
    if (wrap.dataset.loaded === '1') {
      wrap.style.display = 'block';
      if (rowEl) rowEl.querySelector('.gp-chev') && (rowEl.querySelector('.gp-chev').style.transform = 'rotate(90deg)');
      return;
    }
    // Fetch members and render
    wrap.innerHTML = '<div style="padding:6px 0;font-size:12px;color:var(--text-mute);">Loading…</div>';
    wrap.style.display = 'block';
    var hexPalette = ['#4ade80','#22d3ee','#fb7185','#facc15','#c084fc','#f97316','#e11d48','#db2777'];
    fetch('/groups/members?group_id=' + encodeURIComponent(groupID), { credentials: 'same-origin' })
      .then(function(r) { return r.json(); })
      .then(function(members) {
        var arr = Array.isArray(members) ? members : [];
        if (!arr.length) { wrap.innerHTML = '<div style="padding:6px 0;font-size:12px;color:var(--text-mute);">No members.</div>'; return; }
        wrap.innerHTML = arr.map(function(m, i) {
          var dn  = m.name || ('@' + m.id);
          var ini = computeInitials(dn);
          var avStyle = 'background:' + (m.profile_color || hexPalette[i % hexPalette.length]) + ';color:#fff';
          var sid = (m.id || '').replace(/'/g, '');
          var sdn = dn.replace(/'/g, '');
          return '<div class="fp-item" data-id="' + m.id + '" onclick="pickFriend(\'' + pickerID + '\',\'' + sid + '\',\'' + sdn + '\')">' +
            '<div class="fp-avatar-sm" style="' + avStyle + '">' + ini + '</div>' +
            '<div class="fp-name">' + sdn + '</div>' +
            '</div>';
        }).join('');
        wrap.dataset.loaded = '1';
        if (rowEl) rowEl.querySelector('.gp-chev') && (rowEl.querySelector('.gp-chev').style.transform = 'rotate(90deg)');
      })
      .catch(function() { wrap.innerHTML = '<div style="padding:6px 0;font-size:12px;color:var(--rose-hi);">Could not load members.</div>'; });
  }

  function submitRequest() {
    var payerID     = document.getElementById('req-from-val').value;
    var amountCents = getAtmCents('req');
    var note        = document.getElementById('req-note-inp').value;
    if (!payerID)         { showToast('Please select a friend', 'warn'); return; }
    if (amountCents <= 0) { showToast('Please enter a valid amount', 'warn'); return; }
    apiPost('/payments/request/create', { payer_id: payerID, amount_cents: amountCents, note: note },
      function() { closeSheet('request'); showToast('Request sent'); },
      function(e) { showToast(e, 'warn'); });
  }

  // ---- Friend search (add-friend sheet) --------------------------------------
  var _allProfiles = null;
  var _profilesLoading = false;

  function initProfileSearch() {
    if (_allProfiles !== null || _profilesLoading) return;
    _profilesLoading = true;
    document.getElementById('addfriend-results').innerHTML = '<div style="color:var(--text-mute);font-size:13px;padding:8px 0;">Loading users…</div>';
    fetch('/profiles/list', { credentials: 'same-origin' })
      .then(function(r) { return r.json(); })
      .then(function(data) {
        _allProfiles = Array.isArray(data) ? data : [];
        _profilesLoading = false;
        searchUsersToAdd(document.getElementById('addfriend-search-inp').value);
      })
      .catch(function() {
        _profilesLoading = false;
        document.getElementById('addfriend-results').innerHTML = '<div style="color:var(--rose-hi);font-size:13px;padding:8px 0;">Could not load users.</div>';
      });
  }

  function searchUsersToAdd(q) {
    var container = document.getElementById('addfriend-results');
    if (!_allProfiles) { initProfileSearch(); return; }
    q = (q || '').toLowerCase().trim();
    var hexPalette = ['#4ade80','#22d3ee','#fb7185','#facc15','#c084fc','#f97316','#e11d48','#db2777'];
    var matches = _allProfiles.filter(function(p) {
      if (p.id === window._selfID) return false;
      if (window._friendIDs && window._friendIDs.has(p.id)) return false;
      if (!q) return true;
      var qd = q.replace(/\D/g,'');
      var pd = (p.phone_number||'').replace(/\D/g,'');
      return (p.id||'').toLowerCase().indexOf(q) !== -1 || (p.name||'').toLowerCase().indexOf(q) !== -1 || (qd.length >= 3 && pd.indexOf(qd) !== -1);
    }).slice(0, 10);
    if (!matches.length) { container.innerHTML = '<div style="color:var(--text-mute);font-size:13px;padding:8px 0;">No users found.</div>'; return; }
    container.innerHTML = matches.map(function(p, i) {
      var dn = p.name || p.id;
      var ini = computeInitials(dn);
      var avStyle = 'background:' + (p.profile_color || hexPalette[i % hexPalette.length]) + ';color:#fff';
      var safe = dn.replace(/'/g,'');
      return '<div style="display:flex;align-items:center;gap:12px;padding:10px 12px;background:var(--bg);border:1px solid var(--border);border-radius:12px;">' +
        '<div class="row-avatar" style="' + avStyle + ';width:36px;height:36px;font-size:12px;flex-shrink:0;">' + ini + '</div>' +
        '<div style="flex:1;min-width:0;"><div style="font-size:14px;font-weight:600;color:var(--text);">' + dn + '</div><div style="font-size:12px;color:var(--text-faint);">@' + p.id + '</div></div>' +
        '<button onclick="sendFriendRequestTo(\'' + p.id + '\',\'' + safe + '\')" style="background:var(--indigo);color:#fff;border:none;border-radius:8px;padding:6px 12px;font-size:12px;font-weight:700;cursor:pointer;font-family:inherit;flex-shrink:0;">Add</button>' +
        '</div>';
    }).join('');
  }

  function sendFriendRequestTo(id, name) {
    apiPost('/friends/request/send', { receiver_id: id },
      function() { closeSheet('addfriend'); showToast('Friend request sent to ' + name); },
      function(e) { showToast(e, 'warn'); });
  }

  // ---- Notifications ---------------------------------------------------------
  function dismissNotif(notifID, linkView, row) {
    apiPost('/notifications/seen', { notif_id: notifID },
      function() {
        row.style.transition = 'opacity .2s,transform .2s';
        row.style.opacity = '0';
        row.style.transform = 'translateX(12px)';
        setTimeout(function() {
          row.remove();
          var list = document.getElementById('notif-list');
          if (list && list.querySelectorAll('.row').length === 0) {
            list.outerHTML = '<div class="empty"><div class="empty-icon success"><svg width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg></div><div class="empty-title">All caught up</div><div class="empty-sub">No new notifications.</div></div>';
          }
          var badge = document.querySelector('.icon-btn span[style*="ef4444"]');
          if (badge) { var cur = parseInt(badge.textContent||'1',10)-1; if (cur <= 0) badge.remove(); else badge.textContent = cur; }
        }, 200);
        if (linkView) goView(linkView);
      },
      function(e) { showToast(e, 'warn'); });
  }

  function clearAllNotifs() {
    apiPost('/notifications/clear', {},
      function() {
        var list = document.getElementById('notif-list');
        if (list) list.outerHTML = '<div class="empty"><div class="empty-icon success"><svg width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg></div><div class="empty-title">All caught up</div><div class="empty-sub">No new notifications.</div></div>';
        var badge = document.querySelector('.icon-btn span[style*="ef4444"]');
        if (badge) badge.remove();
        showToast('All notifications cleared');
      },
      function(e) { showToast(e, 'warn'); });
  }

  // ---- Friend requests -------------------------------------------------------
  function acceptFriendRequest(requestID) {
    apiPost('/friends/request/accept', { request_id: requestID },
      function() { showToast('Friend request accepted'); setTimeout(function() { location.reload(); }, 800); },
      function(e) { showToast(e, 'warn'); });
  }
  function declineFriendRequest(requestID) {
    apiPost('/friends/request/decline', { request_id: requestID },
      function() {
        var row = document.querySelector('[data-req-row="' + requestID + '"]');
        if (row) { row.style.transition='opacity .2s'; row.style.opacity='0'; setTimeout(function(){row.remove();},200); }
        showToast('Request declined');
      },
      function(e) { showToast(e, 'warn'); });
  }
  function removeFriend(btn, friendID, name) {
    apiPost('/friends/remove', { friend_id: friendID },
      function() {
        var row = btn.closest('.friend-row');
        row.style.transition = 'opacity .2s,transform .2s';
        row.style.opacity = '0'; row.style.transform = 'translateX(-20px)';
        setTimeout(function() {
          row.remove();
          showToast(name + ' removed');
          var remaining = document.querySelectorAll('#friend-list .friend-row').length;
          if (!remaining) { document.getElementById('friend-list').style.display='none'; document.getElementById('friends-empty').style.display='block'; }
        }, 200);
      },
      function(e) { showToast(e, 'warn'); });
  }

  // ---- Installments ----------------------------------------------------------
  function payInstallment(installmentID, paymentID, amountCents) {
    apiPost('/bnpl/installment/pay', { installment_id: installmentID, payment_id: paymentID, amount_cents: amountCents },
      function() { showToast('Installment paid'); setTimeout(function() { location.reload(); }, 1200); },
      function(e) { showToast(e, 'warn'); });
  }
  function payAllOverdue() {
    var rows = document.querySelectorAll('[data-installment-id]');
    if (!rows.length) { showToast('No overdue installments'); return; }
    var pending = rows.length, failed = 0;
    rows.forEach(function(row) {
      fetch('/bnpl/installment/pay', { method:'POST', headers:{'Content-Type':'application/json','X-CSRF-Token':getCsrfToken()}, credentials:'same-origin',
        body: JSON.stringify({ installment_id: row.getAttribute('data-installment-id'), payment_id: row.getAttribute('data-payment-id'), amount_cents: parseInt(row.getAttribute('data-amount-cents')||'0',10) })
      }).then(function(res){if(!res.ok)failed++;}).catch(function(){failed++;})
        .finally(function() { pending--; if (!pending) { showToast(failed?failed+' payment(s) failed':'All overdue installments paid', failed?'warn':''); setTimeout(function(){location.reload();},1200); } });
    });
  }

  // ---- Friend filter ---------------------------------------------------------
  function filterFriends(q) {
    q = (q||'').toLowerCase().trim();
    var rows = document.querySelectorAll('#friend-list .friend-row');
    var visible = 0;
    rows.forEach(function(r) { var match = !q || r.getAttribute('data-name').indexOf(q) !== -1; r.style.display = match?'':'none'; if(match)visible++; });
    document.getElementById('friend-list').style.opacity = (!visible && q) ? '0.4' : '1';
  }

  // ---- P2P history sheet -----------------------------------------------------
  function openHistorySheet(friendID, friendName) {
    var titleEl = document.getElementById('history-sheet-title');
    var listEl  = document.getElementById('history-list');
    if (titleEl) titleEl.textContent = 'Payments with ' + friendName;
    if (listEl)  listEl.innerHTML = '<div style="color:var(--text-mute);font-size:13px;padding:20px;text-align:center;">Loading…</div>';
    openSheet('history');
    fetch('/payments/between?other_id=' + encodeURIComponent(friendID), { credentials: 'same-origin' })
      .then(function(r) { return r.json(); })
      .then(function(payments) {
        if (!Array.isArray(payments) || !payments.length) {
          listEl.innerHTML = '<div style="text-align:center;padding:28px;color:var(--text-mute);font-size:13px;">No payment history with this person.</div>';
          return;
        }
        listEl.innerHTML = payments.map(function(p) {
          var isSent = p.sender_id === window._selfID;
          var date = (p.created_at||'').slice(0,10);
          var dirCls = isSent ? 'dir-out' : 'dir-in';
          var dirLbl = isSent ? 'SENT' : 'IN';
          var amtCls = isSent ? 'neg' : 'pos';
          var prefix = isSent ? '−' : '+';
          var note   = p.note || (isSent ? 'Payment sent' : 'Payment received');
          return '<div class="row">' +
            '<div class="row-body"><div class="row-title"><b>' + note + '</b><span class="dir-badge ' + dirCls + '">' + dirLbl + '</span></div>' +
            '<div class="row-sub">' + (p.payment_type||'direct') + ' · ' + date + '</div></div>' +
            '<div class="row-right"><div class="row-amt ' + amtCls + ' mono">' + prefix + '$' + ((p.amount_cents||0)/100).toFixed(2) + '</div></div></div>';
        }).join('');
      })
      .catch(function() {
        if (listEl) listEl.innerHTML = '<div style="color:var(--rose-hi);font-size:13px;padding:20px;text-align:center;">Could not load history.</div>';
      });
  }

  function openInviteSheet(groupID, groupName) {
    var idEl    = document.getElementById('invite-group-id');
    var titleEl = document.getElementById('invite-sheet-title');
    if (idEl)    idEl.value = groupID;
    if (titleEl) titleEl.textContent = 'Invite to ' + groupName;
    openSheet('invite-group');
  }

  function submitGroupInvite() {
    var groupID    = (document.getElementById('invite-group-id') || {}).value || '';
    var receiverID = (document.getElementById('invite-recv-val') || {}).value || '';
    if (!groupID)    { showToast('Group ID missing', 'warn'); return; }
    if (!receiverID) { showToast('Select a friend to invite', 'warn'); return; }
    apiPost('/groups/invite/send', { group_id: groupID, receiver_id: receiverID },
      function() { closeSheet('invite-group'); showToast('Invitation sent'); },
      function(e) { showToast(e, 'warn'); });
  }

  function submitCreateGroup() {
    var name = (document.getElementById('new-group-name').value || '').trim();
    if (!name) { showToast('Please enter a group name', 'warn'); return; }
    apiPost('/groups/create', { name: name },
      function() { closeSheet('create-group'); showToast('Group created'); setTimeout(function(){location.reload();},1000); },
      function(e) { showToast(e, 'warn'); });
  }

  function acceptGroupInvitation(id) {
    apiPost('/groups/invite/accept', { invitation_id: id },
      function() { showToast('Joined group'); setTimeout(function(){location.reload();},800); },
      function(e) { showToast(e, 'warn'); });
  }

  function declineGroupInvitation(id) {
    apiPost('/groups/invite/decline', { invitation_id: id },
      function() {
        var row = document.querySelector('[data-invite-id="' + id + '"]');
        if (row) { row.style.transition='opacity .2s'; row.style.opacity='0'; setTimeout(function(){row.remove();},200); }
        showToast('Invitation declined');
      },
      function(e) { showToast(e, 'warn'); });
  }

  function confirmLeaveGroup(groupID, name) {
    if (!confirm('Leave group "' + name + '"? You will need a new invitation to rejoin.')) return;
    apiPost('/groups/leave', { group_id: groupID },
      function() { showToast('Left group'); setTimeout(function(){location.reload();},800); },
      function(e) { showToast(e, 'warn'); });
  }

  // ---- Group Detail ----------------------------------------------------------
  function goGroupDetail(groupID, groupName) {
    var nameEl = document.getElementById('group-detail-name');
    if (nameEl) nameEl.textContent = groupName;

    // Reset panes to initial state before loading so stale content is cleared
    var membersPane  = document.querySelector('[data-pane-content="gd-members"]');
    var activityPane = document.querySelector('[data-pane-content="gd-activity"]');
    if (membersPane)  { membersPane.style.display  = 'block'; membersPane.innerHTML  = '<div id="group-detail-members"><div style="color:var(--text-mute);font-size:13px;padding:12px 0;text-align:center;">Loading…</div></div>'; }
    if (activityPane) { activityPane.style.display = 'none';  activityPane.innerHTML = '<div id="group-detail-activity"><div style="color:var(--text-mute);font-size:13px;padding:12px 0;text-align:center;">Loading…</div></div>'; }

    // Reset subtabs so "Members" is active
    var section = document.querySelector('[data-view="group-detail"]');
    if (section) {
      section.querySelectorAll('.subtab').forEach(function(t, i) { t.classList.toggle('active', i === 0); });
    }

    goView('group-detail');
    loadGroupDetailData(groupID);
  }

  function loadGroupDetailData(groupID) {
    var hexPalette = ['#4ade80','#22d3ee','#fb7185','#facc15','#c084fc','#f97316','#e11d48','#db2777'];

    // Members
    fetch('/groups/members?group_id=' + encodeURIComponent(groupID), { credentials: 'same-origin' })
      .then(function(r) { return r.json(); })
      .then(function(members) {
        var arr = Array.isArray(members) ? members : [];
        var el  = document.getElementById('group-detail-members');
        if (!el) return;
        if (!arr.length) {
          el.innerHTML = '<div style="text-align:center;padding:28px 16px;color:var(--text-mute);font-size:13px;">No members found.</div>';
          return;
        }
        var rows = arr.map(function(m, i) {
          var ini     = computeInitials(m.name || m.id || '?');
          var avStyle = 'background:' + (m.profile_color || hexPalette[i % hexPalette.length]) + ';color:#fff';
          var date    = m.created_at ? m.created_at.slice(0, 10) : '';
          var dn      = m.name || ('@' + m.id);
          return '<div class="row">' +
            '<div class="row-avatar" style="' + avStyle + '">' + ini + '</div>' +
            '<div class="row-body"><div class="row-title"><b>' + dn + '</b></div><div class="row-sub">@' + (m.id || '') + '</div></div>' +
            '<div class="row-right"><div class="row-time">' + date + '</div></div>' +
            '</div>';
        }).join('');
        el.innerHTML = '<div class="card">' + rows + '</div>';
      })
      .catch(function() {
        var el = document.getElementById('group-detail-members');
        if (el) el.innerHTML = '<div style="color:var(--rose-hi);font-size:13px;padding:20px;text-align:center;">Could not load members.</div>';
      });

    // Activity log
    fetch('/groups/activity?group_id=' + encodeURIComponent(groupID), { credentials: 'same-origin' })
      .then(function(r) { return r.json(); })
      .then(function(payments) {
        var arr = Array.isArray(payments) ? payments : [];
        var el  = document.getElementById('group-detail-activity');
        if (!el) return;
        if (!arr.length) {
          el.innerHTML = '<div style="text-align:center;padding:28px 16px;color:var(--text-mute);font-size:13px;">No group activity yet.</div>';
          return;
        }
        var rows = arr.map(function(p, i) {
          var avStyle = 'background:' + hexPalette[i % hexPalette.length] + ';color:#fff';
          var sini    = p.sender_id ? p.sender_id.slice(0, 2).toUpperCase() : 'PA';
          var date    = p.created_at ? p.created_at.slice(0, 16).replace('T',' ') : '';
          var type    = p.payment_type === 'bnpl' ? ' <span class="pill">BNPL</span>' : '';
          var note    = p.note || p.payment_type || '';
          var cents   = p.amount_cents || 0;
          var isSelf  = p.sender_id === window._selfID;
          var amtCls  = isSelf ? 'neg' : 'pos';
          var prefix  = isSelf ? '−' : '+';
          return '<div class="row">' +
            '<div class="row-avatar" style="' + avStyle + '">' + sini + '</div>' +
            '<div class="row-body">' +
              '<div class="row-title"><b>@' + (p.sender_id || '') + '</b> → <b>@' + (p.receiver_id || '') + '</b>' + type + '</div>' +
              '<div class="row-sub">' + (note ? note + ' · ' : '') + date + '</div>' +
            '</div>' +
            '<div class="row-right">' +
              '<div class="row-amt ' + amtCls + ' mono">' + prefix + '$' + (cents / 100).toFixed(2) + '</div>' +
            '</div>' +
            '</div>';
        }).join('');
        el.innerHTML = '<div class="card">' + rows + '</div>';
      })
      .catch(function() {
        var el = document.getElementById('group-detail-activity');
        if (el) el.innerHTML = '<div style="color:var(--rose-hi);font-size:13px;padding:20px;text-align:center;">Could not load activity.</div>';
      });
  }

  function submitDeposit() {
    var amtCents = getAtmCents('deposit');
    if (amtCents <= 0) { showToast('Please enter an amount', 'warn'); return; }
    apiPost('/wallet/deposit', { amount_cents: amtCents },
      function() { closeSheet('deposit'); showToast('Deposit successful'); setTimeout(function(){location.reload();},1000); },
      function(e) { showToast(e, 'warn'); });
  }

  function submitWithdraw() {
    var amtCents = getAtmCents('withdraw');
    if (amtCents <= 0) { showToast('Please enter an amount', 'warn'); return; }
    apiPost('/wallet/withdraw', { amount_cents: amtCents },
      function() { closeSheet('withdraw'); showToast('Withdrawal successful'); setTimeout(function(){location.reload();},1000); },
      function(e) { showToast(e, 'warn'); });
  }

  function submitUpdateField(field, value, label) {
    value = (value || '').trim();
    if (!value) { showToast('Please enter a ' + label, 'warn'); return; }
    var urls = { name: '/users/update/name', email: '/users/update/email', phone: '/users/update/phone' };
    var url = urls[field];
    if (!url) { showToast('Unknown field', 'warn'); return; }
    apiPost(url, { value: value },
      function() { showToast(label.charAt(0).toUpperCase() + label.slice(1) + ' updated'); },
      function(e) { showToast(e, 'warn'); });
  }

  function submitUpdatePassword() {
    var cur  = document.getElementById('settings-cur-pw').value;
    var nw   = document.getElementById('settings-new-pw').value;
    var conf = document.getElementById('settings-conf-pw').value;
    if (!cur || !nw)  { showToast('Fill in all password fields', 'warn'); return; }
    if (nw !== conf)  { showToast('New passwords do not match', 'warn'); return; }
    if (nw.length < 8){ showToast('Password must be at least 8 characters', 'warn'); return; }
    apiPost('/users/update/password', { current_password: cur, new_password: nw },
      function() {
        showToast('Password updated');
        ['settings-cur-pw','settings-new-pw','settings-conf-pw'].forEach(function(id){ document.getElementById(id).value=''; });
      },
      function(e) { showToast(e, 'warn'); });
  }

  function submitUpdateSettings() {
    var emailNotif   = document.getElementById('toggle-email-notif').checked;
    var discoverable = document.getElementById('toggle-discoverable').checked;
    apiPost('/settings/update', { email_notifications: emailNotif, is_discoverable: discoverable },
      function() { showToast('Privacy settings saved'); },
      function(e) { showToast(e, 'warn'); });
  }

  function deactivateAccount() {
    if (!confirm('Permanently deactivate your account? This cannot be undone.')) return;
    apiPost('/users/deactivate', {},
      function() { showToast('Account deactivated'); setTimeout(function(){location.reload();},1000); },
      function(e) { showToast(e, 'warn'); });
  }

  function payRequest(requestID) {
    apiPost('/payments/request/fulfill', { request_id: requestID },
      function() {
        showToast('Payment sent');
        setTimeout(function() { location.reload(); }, 1200);
      },
      function(e) { showToast(e, 'warn'); }
    );
  }

  // ---- Profile color ---------------------------------------------------------
  function submitUpdateColor(color) {
    apiPost('/users/update/color', { color: color },
      function() {
        showToast('Color updated');
        document.querySelectorAll('.avatar-btn, .avatar-xl').forEach(function(el) {
          el.style.background = color;
          el.style.color = '#fff';
        });
        document.querySelectorAll('.color-swatch').forEach(function(sw) {
          sw.style.boxShadow = sw.dataset.color === color
            ? '0 0 0 3px var(--surface),0 0 0 5px ' + color : '';
        });
      },
      function(e) { showToast(e, 'warn'); }
    );
  }

  // ---- Sign out --------------------------------------------------------------
  function signOut() {
    fetch('/users/logout', { method:'POST', credentials:'same-origin', headers:{'X-CSRF-Token':getCsrfToken()} })
      .then(function() { location.reload(); })
      .catch(function() { document.cookie='session_user_id=; Max-Age=0; path=/'; location.reload(); });
  }

  // ---- Global keyboard: ATM routing + Escape ---------------------------------
  document.addEventListener('keydown', function(e) {
    var active = document.activeElement;
    if (active && active.classList.contains('atm-wrap')) {
      var atmId = active.getAttribute('data-atm');
      if (atmId && ((e.key >= '0' && e.key <= '9') || e.key === 'Backspace')) {
        atmKey(atmId, e.key); e.preventDefault(); return;
      }
    }
    if (e.key !== 'Escape') return;
    ['pay','request','addfriend','create-group','invite-group','deposit','withdraw','history'].forEach(function(id) { closeSheet(id); });
  });
</script>
`
}

// ---- Main shell --------------------------------------------------------------

func dashboardHTML(user *models.User, friends []models.Profile, installments []models.InstallmentDetail, overdueInstallments []models.InstallmentDetail, incomingRequests []models.PaymentRequest, friendRequests []models.FriendRequest, notifications []models.Notification, groups []models.Group, groupInvitations []models.GroupInvitation, activityFeed []models.ActivityItem, memberCounts map[string]int, wDash *walletDashboard, aDash *analyticsDashboard, userSettings *models.UserSettings) string {
	name := displayName(user)
	avatar := initials(name)
	handle := user.ID
	email := user.Email
	phone := user.PhoneNumber
	if email == "" {
		email = handle + "@splitit.app"
	}
	profileColor := user.ProfileColor
	if profileColor == "" {
		profileColor = "#4ade80"
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
	notifCount := len(notifications)

	notifBadge := `<span class="dot"></span>`
	if notifCount > 0 {
		notifBadge = `<span style="position:absolute;top:6px;right:6px;width:16px;height:16px;border-radius:50%;background:#ef4444;border:2px solid #0f172a;display:flex;align-items:center;justify-content:center;font-size:9px;font-weight:700;color:#fff;font-family:'JetBrains Mono',monospace;">` + fmt.Sprintf("%d", notifCount) + `</span>`
	}

	return dashboardStyles() + `
<div id="app-root" class="app-shell" data-uid="` + handle + `" data-credit-score="` + fmt.Sprintf("%d", user.CreditScore) + `">
  <div id="toast"></div>

  <header class="topbar">
    <div class="topbar-inner">
      <div style="display:flex;align-items:center;gap:2px;">
        <button class="brand-btn" onclick="goView('home')" aria-label="Home">
          <span class="brand">split<span>it</span></span>
        </button>
      </div>
      <div class="topbar-actions">
        <button class="icon-btn" aria-label="Notifications" onclick="goView('notifications')">
          <svg width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>
          ` + notifBadge + `
        </button>
        <button class="avatar-btn" style="background:` + profileColor + `;color:#fff" onclick="goView('profile')" aria-label="Profile">` + avatar + `</button>
      </div>
    </div>
  </header>

  <div class="app-body">
    <aside class="sidebar">
      <nav class="side-nav">
        <button class="side-link active" data-tab="home" onclick="goView('home')">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M3 12l9-9 9 9"/><path d="M5 10v10h14V10"/></svg>Home
        </button>
        <button class="side-link" data-tab="activity" onclick="goView('activity')">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M3 12h4l3-9 4 18 3-9h4"/></svg>Activity
        </button>
        <button class="side-link" data-tab="social" onclick="goView('social')">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>Social
        </button>
        <button class="side-link" data-tab="groups" onclick="goView('groups')">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/><line x1="19" y1="8" x2="19" y2="14"/><line x1="22" y1="11" x2="16" y2="11"/></svg>Groups
        </button>
        <button class="side-link" data-tab="analytics" onclick="goView('analytics')">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 0 1 3 19.875v-6.75Z"/><path stroke-linecap="round" stroke-linejoin="round" d="M9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V8.625Z"/><path stroke-linecap="round" stroke-linejoin="round" d="M16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V4.125Z"/></svg>Analytics
        </button>
        <button class="side-link" data-tab="wallet" onclick="goView('wallet')">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M21 12a2.25 2.25 0 0 0-2.25-2.25H15a3 3 0 1 1-6 0H5.25A2.25 2.25 0 0 0 3 12m18 0v6A2.25 2.25 0 0 1 18.75 20H5.25A2.25 2.25 0 0 1 3 18v-6m18 0V9A2.25 2.25 0 0 0 18.75 6.75H5.25A2.25 2.25 0 0 0 3 9v3M16.5 10.5a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5Z"/></svg>Wallet
        </button>
        <button class="side-link" data-tab="profile" onclick="goView('profile')">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>Profile
        </button>
        <button class="side-link" data-tab="settings" onclick="goView('settings')">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>Settings
        </button>
        <button class="side-link" onclick="signOut()" style="color:#fca5a5;">
          <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>Sign out
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
` + viewActivity(installments, overdueInstallments, incomingRequests, activityFeed) + `
` + viewFriends(friends, friendRequests) + `
` + viewGroups(groups, groupInvitations, handle, memberCounts) + `
` + viewGroupDetail() + `
` + viewAnalytics(aDash) + `
` + viewWallet(user, wDash) + `
` + viewNotifications(notifications) + `
` + viewProfile(user, avatar, name, handle, email, phone, friends, installments, groups) + `
` + viewSettings(user, userSettings) + `
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
    <button class="tab" data-tab="groups" onclick="goView('groups')" style="position:relative;">
      <svg width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
      <span>Groups</span>
    </button>
    <button class="tab" data-tab="profile" onclick="goView('profile')">
      <svg width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
      <span>Profile</span>
    </button>
  </nav>

` + paySheetHTML(friends, groups, availableCreditCents) + `
` + requestSheetHTML(friends, groups) + `
` + addFriendSheetHTML() + `
` + createGroupSheetHTML() + `
` + inviteToGroupSheetHTML(friends) + `
` + depositSheetHTML() + `
` + withdrawSheetHTML() + `
` + historySheetHTML() + `
` + sessionContextScript(handle, friends) + `
` + dashboardScript() + `
</div>`
}
