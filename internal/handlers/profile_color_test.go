package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/connorpodea/splitit/internal/models"
)

// seedUserWithColor inserts a user then immediately sets their profile color,
// mirroring the store-level helper in the store tests.
func seedHandlerUserWithColor(t *testing.T, h interface{}, s interface {
	CreateUser(*models.User) error
	UpdateProfileColor(string, string) error
}, id, color string, balanceCents int) {
	t.Helper()
	err := s.CreateUser(&models.User{
		ID:               id,
		PasswordHash:     "hash",
		Name:             id,
		Email:            id + "@test.com",
		PhoneNumber:      "0000000000",
		BalanceCents:     balanceCents,
		CreditScore:      50,
		CreditLimitCents: 100000,
	})
	if err != nil {
		t.Fatalf("CreateUser(%s): %v", id, err)
	}
	if err = s.UpdateProfileColor(id, color); err != nil {
		t.Fatalf("UpdateProfileColor(%s, %s): %v", id, color, err)
	}
}

// --- GetUnifiedActivity endpoint exposes peer_color ---

// TestGetUnifiedActivity_ResponseContainsPeerColor verifies that the /payments/activity
// endpoint returns JSON objects that include the peer_color field populated with the
// counterparty's current profile color.
func TestGetUnifiedActivity_ResponseContainsPeerColor(t *testing.T) {
	h, s := newTestHandler(t)
	seedHandlerUserWithColor(t, h, s, "alice", "#fb7185", 10000)
	seedHandlerUserWithColor(t, h, s, "bob", "#22d3ee", 0)

	if err := s.Pay(&models.Payment{
		SenderID: "alice", ReceiverID: "bob",
		AmountCents: 500, TotalAmountCents: 500, PaymentType: "direct",
	}); err != nil {
		t.Fatalf("Pay: %v", err)
	}

	rr := httptest.NewRecorder()
	h.GetUnifiedActivity(rr, sessionReq(t, h, http.MethodGet, "/payments/activity", "", "alice"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var items []map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&items); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one activity item")
	}

	var found bool
	for _, item := range items {
		if item["kind"] == "payment_sent" {
			color, _ := item["peer_color"].(string)
			if color != "#22d3ee" {
				t.Errorf("peer_color: want '#22d3ee' (bob's color), got %q", color)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("no payment_sent item found in response")
	}
}

// TestGetUnifiedActivity_PeerColorUpdatesAfterProfileChange verifies that changing a
// user's profile color is immediately reflected in the other user's activity feed on
// the next request — no cache flush needed.
func TestGetUnifiedActivity_PeerColorUpdatesAfterProfileChange(t *testing.T) {
	h, s := newTestHandler(t)
	seedHandlerUserWithColor(t, h, s, "alice", "#fb7185", 10000)
	seedHandlerUserWithColor(t, h, s, "bob", "#22d3ee", 0)

	if err := s.Pay(&models.Payment{
		SenderID: "alice", ReceiverID: "bob",
		AmountCents: 800, TotalAmountCents: 800, PaymentType: "direct",
	}); err != nil {
		t.Fatalf("Pay: %v", err)
	}

	// bob changes his profile color
	rr := httptest.NewRecorder()
	h.UpdateProfileColor(rr, sessionReq(t, h, http.MethodPost, "/users/update/color",
		`{"color":"#c084fc"}`, "bob"))
	if rr.Code != http.StatusOK {
		t.Fatalf("UpdateProfileColor: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// alice's feed should now reflect bob's new color
	rr2 := httptest.NewRecorder()
	h.GetUnifiedActivity(rr2, sessionReq(t, h, http.MethodGet, "/payments/activity", "", "alice"))

	if rr2.Code != http.StatusOK {
		t.Fatalf("GetUnifiedActivity: expected 200, got %d", rr2.Code)
	}

	var items []map[string]interface{}
	if err := json.NewDecoder(rr2.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}

	var found bool
	for _, item := range items {
		if item["kind"] == "payment_sent" {
			color, _ := item["peer_color"].(string)
			if color != "#c084fc" {
				t.Errorf("retroactive peer_color: want '#c084fc', got %q", color)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("no payment_sent item found in response")
	}
}

// TestGetUnifiedActivity_PeerColorNeverEmpty verifies that the peer_color field is
// always present and non-empty, even for users without an explicitly set color.
func TestGetUnifiedActivity_PeerColorNeverEmpty(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 5000)
	seedUser(t, s, "bob", 0)

	if err := s.Pay(&models.Payment{
		SenderID: "alice", ReceiverID: "bob",
		AmountCents: 100, TotalAmountCents: 100, PaymentType: "direct",
	}); err != nil {
		t.Fatalf("Pay: %v", err)
	}

	rr := httptest.NewRecorder()
	h.GetUnifiedActivity(rr, sessionReq(t, h, http.MethodGet, "/payments/activity", "", "alice"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var items []map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one item")
	}
	for _, item := range items {
		color, _ := item["peer_color"].(string)
		if color == "" {
			t.Errorf("item kind=%q has empty peer_color — must always have a fallback", item["kind"])
		}
	}
}
