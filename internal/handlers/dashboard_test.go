package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/connorpodea/splitit/internal/models"
)

// TestGetInitialView_RendersActivityFeed verifies that a payment between users is
// server-rendered into the activity feed HTML on the initial page load.
func TestGetInitialView_RendersActivityFeed(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 10000)
	seedUser(t, s, "bob", 0)

	if err := s.Pay(&models.Payment{
		SenderID: "alice", ReceiverID: "bob",
		AmountCents: 500, TotalAmountCents: 500, PaymentType: "direct",
	}); err != nil {
		t.Fatalf("Pay: %v", err)
	}

	rr := httptest.NewRecorder()
	h.GetInitialView(rr, sessionReq(t, http.MethodGet, "/", "", "alice"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "You paid") {
		t.Error("expected server-rendered activity feed to contain 'You paid'")
	}
	if !strings.Contains(body, "bob") {
		t.Error("expected server-rendered activity feed to contain peer name 'bob'")
	}
}

// TestGetInitialView_RendersGroupMemberCounts verifies that the group tab shows
// the member count server-rendered rather than "Loading…".
func TestGetInitialView_RendersGroupMemberCounts(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 0)

	g := &models.Group{Name: "Test Squad", CreatorID: "alice"}
	if err := s.CreateGroup(g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if err := s.SendGroupInvitation(g.ID, "alice", "bob"); err != nil {
		t.Fatalf("SendGroupInvitation: %v", err)
	}
	invs, _ := s.ListIncomingGroupInvitations("bob")
	if len(invs) > 0 {
		s.AcceptGroupInvitation(invs[0].ID, "bob")
	}

	rr := httptest.NewRecorder()
	h.GetInitialView(rr, sessionReq(t, http.MethodGet, "/", "", "alice"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "2 members") {
		t.Errorf("expected '2 members' in groups tab, body excerpt: %s",
			body[max(0, strings.Index(body, "Test Squad")-50):min(len(body), strings.Index(body, "Test Squad")+200)])
	}
}

// TestGetInitialView_RendersWalletStats verifies that wallet stats are
// server-rendered into the HTML on page load.
func TestGetInitialView_RendersWalletStats(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 5000)
	seedUser(t, s, "bob", 0)

	if err := s.Deposit("alice", 10000); err != nil {
		t.Fatalf("Deposit: %v", err)
	}

	rr := httptest.NewRecorder()
	h.GetInitialView(rr, sessionReq(t, http.MethodGet, "/", "", "alice"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "BNPL Utilization") {
		t.Error("expected server-rendered wallet stats to contain 'BNPL Utilization'")
	}
	if !strings.Contains(body, "Last 30d") {
		t.Error("expected server-rendered wallet stats to contain 'Last 30d'")
	}
}

// TestGetInitialView_RendersAnalytics verifies that analytics content is
// server-rendered into the HTML on page load.
func TestGetInitialView_RendersAnalytics(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 0)

	rr := httptest.NewRecorder()
	h.GetInitialView(rr, sessionReq(t, http.MethodGet, "/", "", "alice"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Top Recipients") {
		t.Error("expected server-rendered analytics to contain 'Top Recipients'")
	}
	if !strings.Contains(body, "This Month") {
		t.Error("expected server-rendered analytics to contain 'This Month'")
	}
	if !strings.Contains(body, "Credit Score History") {
		t.Error("expected server-rendered analytics to contain 'Credit Score History'")
	}
}

// TestGetInitialView_RendersSettingsToggles verifies that the settings toggles
// reflect actual DB state rather than hardcoded defaults.
func TestGetInitialView_RendersSettingsToggles(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 0)

	// Set discoverable=false so we can verify the unchecked state is rendered.
	if err := s.UpsertUserSettings(&models.UserSettings{
		UserID:             "alice",
		EmailNotifications: true,
		IsDiscoverable:     false,
	}); err != nil {
		t.Fatalf("UpsertUserSettings: %v", err)
	}

	rr := httptest.NewRecorder()
	h.GetInitialView(rr, sessionReq(t, http.MethodGet, "/", "", "alice"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	// When discoverable is false, the checkbox should be rendered without "checked"
	// and the adjacent email checkbox should still have "checked".
	if !strings.Contains(body, `id="toggle-email-notif" checked`) {
		t.Error("expected email notifications toggle to be checked")
	}
	if strings.Contains(body, `id="toggle-discoverable" checked`) {
		t.Error("expected discoverable toggle to be unchecked when IsDiscoverable=false")
	}
}

// TestGetInitialView_NoLazyLoadPlaceholders verifies that no lazy-loading
// "Loading…" placeholder text appears in the rendered HTML.
func TestGetInitialView_NoLazyLoadPlaceholders(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 0)

	rr := httptest.NewRecorder()
	h.GetInitialView(rr, sessionReq(t, http.MethodGet, "/", "", "alice"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	// None of the tab-level lazy-load placeholder messages should appear.
	// (viewGroupDetail's "Loading…" is intentional — it's populated on user-click.)
	for _, placeholder := range []string{
		"Loading activity…",
		"Loading stats…",
		"Loading analytics…",
	} {
		if strings.Contains(body, placeholder) {
			t.Errorf("found lazy-load placeholder in server-rendered HTML: %q", placeholder)
		}
	}
	// No data-onload attributes should be present.
	if strings.Contains(body, "data-onload") {
		t.Error("found data-onload attribute in server-rendered HTML — lazy loading not fully removed")
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
