package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- Email validation ---

func TestUpdateEmail_RejectsInvalidFormat(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 0)

	cases := []string{"notanemail", "missing@tld", "@no-local.com", "spaces in@email.com"}
	for _, bad := range cases {
		rr := httptest.NewRecorder()
		h.UpdateEmail(rr, sessionReq(t, http.MethodPost, "/users/update/email",
			`{"value":"`+bad+`"}`, "alice"))
		if rr.Code != http.StatusBadRequest {
			t.Errorf("email %q: expected 400, got %d", bad, rr.Code)
		}
	}
}

func TestUpdateEmail_AcceptsValidFormat(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 0)

	rr := httptest.NewRecorder()
	h.UpdateEmail(rr, sessionReq(t, http.MethodPost, "/users/update/email",
		`{"value":"alice@example.com"}`, "alice"))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// --- Phone validation ---

func TestUpdatePhoneNumber_RejectsBadLength(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 0)

	cases := []string{"123456789", "12345678901", "abcdefghij", ""}
	for _, bad := range cases {
		rr := httptest.NewRecorder()
		h.UpdatePhoneNumber(rr, sessionReq(t, http.MethodPost, "/users/update/phone",
			`{"value":"`+bad+`"}`, "alice"))
		if rr.Code != http.StatusBadRequest {
			t.Errorf("phone %q: expected 400, got %d", bad, rr.Code)
		}
	}
}

func TestUpdatePhoneNumber_AcceptsExactlyTenDigits(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 0)

	rr := httptest.NewRecorder()
	h.UpdatePhoneNumber(rr, sessionReq(t, http.MethodPost, "/users/update/phone",
		`{"value":"(555) 867-5309"}`, "alice"))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// --- Profile color validation ---

func TestUpdateProfileColor_RejectsDisallowedHex(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 0)

	cases := []string{"av-rose", "#ffffff", "red", "#000000", ""}
	for _, bad := range cases {
		rr := httptest.NewRecorder()
		h.UpdateProfileColor(rr, sessionReq(t, http.MethodPost, "/users/update/color",
			`{"color":"`+bad+`"}`, "alice"))
		if rr.Code != http.StatusBadRequest {
			t.Errorf("color %q: expected 400, got %d", bad, rr.Code)
		}
	}
}

// --- Registration name combination ---

func TestCreateUser_CombinesFirstLastName(t *testing.T) {
	h, s := newTestHandler(t)

	rr := httptest.NewRecorder()
	h.CreateUser(rr, httptest.NewRequest(http.MethodPost, "/users/create",
		strings.NewReader(`{"id":"jtest","password":"pass","first_name":"Jason","last_name":"Podea","email":"j@test.com","phone_number":"5558675309"}`)))
	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("expected 2xx, got %d: %s", rr.Code, rr.Body.String())
	}

	user, err := s.GetUser("jtest")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if user.Name != "Jason Podea" {
		t.Errorf("name: want 'Jason Podea', got %q", user.Name)
	}
}

func TestCreateUser_RejectsMissingLastName(t *testing.T) {
	h, _ := newTestHandler(t)

	rr := httptest.NewRecorder()
	h.CreateUser(rr, httptest.NewRequest(http.MethodPost, "/users/create",
		strings.NewReader(`{"id":"jtest2","password":"pass","first_name":"Jason","last_name":"","email":"j2@test.com","phone_number":"5558675309"}`)))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}
