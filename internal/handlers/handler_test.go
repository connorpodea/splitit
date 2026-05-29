package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/connorpodea/splitit/internal/handlers"
	"github.com/connorpodea/splitit/internal/models"
	"github.com/connorpodea/splitit/internal/store"
)

// =========================================================================
// TEST HELPERS
// =========================================================================

// newTestHandler spins up a fresh in-memory store and returns a wired handler
func newTestHandler(t *testing.T) *handlers.Handler {
	t.Helper()
	s, err := store.NewFromPath(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize test store: %v", err)
	}

	// Seed the treasury account required by BNPL operations
	treasury := &models.User{
		ID: "app_treasury", Name: "SplitIt Treasury Pool",
		Email: "treasury@splitit.internal", PhoneNumber: "000-000-0000",
		Balance: 10000.00, CreditScore: 100, CreditLimit: 0,
	}
	if err := s.CreateUser(treasury); err != nil {
		t.Fatalf("Failed to seed treasury: %v", err)
	}

	return handlers.New(s)
}

// postJSON fires a POST request with a JSON body against the given handler func
func postJSON(t *testing.T, handler http.HandlerFunc, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}

// getWithQuery fires a GET request with URL query params against the given handler func
func getWithQuery(handler http.HandlerFunc, params map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}

// =========================================================================
// CreateUser handler tests
// =========================================================================

func TestCreateUserHandler_Success(t *testing.T) {
	h := newTestHandler(t)

	user := models.User{
		ID: "cpodea", Name: "Connor", Email: "cpodea@gmail.com",
		PhoneNumber: "123-456-7890", Balance: 500.00,
		CreditScore: 50, CreditLimit: 1000.00,
	}
	w := postJSON(t, h.CreateUser, user)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d — body: %s", w.Code, w.Body.String())
	}

	// Verify the response body echoes back the created user
	var resp models.User
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.ID != "cpodea" {
		t.Errorf("Expected ID 'cpodea', got '%s'", resp.ID)
	}
	if resp.Name != "Connor" {
		t.Errorf("Expected name 'Connor', got '%s'", resp.Name)
	}
}

func TestCreateUserHandler_InvalidJSON(t *testing.T) {
	h := newTestHandler(t)

	// Send raw malformed JSON
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for malformed JSON, got %d", w.Code)
	}
}

func TestCreateUserHandler_WrongMethod(t *testing.T) {
	h := newTestHandler(t)

	// Send a GET request to a POST-only endpoint
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.CreateUser(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for wrong method, got %d", w.Code)
	}
}

// =========================================================================
// GetUser handler tests
// =========================================================================

func TestGetUserHandler_Success(t *testing.T) {
	h := newTestHandler(t)

	// First create the user via the handler
	user := models.User{
		ID: "jpodea", Name: "Jason", Email: "jpodea@asu.edu",
		PhoneNumber: "987-654-3210", Balance: 100.00,
		CreditScore: 80, CreditLimit: 1000.00,
	}
	postJSON(t, h.CreateUser, user)

	// Now fetch via GET with ?id=jpodea
	w := getWithQuery(h.GetUser, map[string]string{"id": "jpodea"})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp models.User
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Name != "Jason" {
		t.Errorf("Expected name 'Jason', got '%s'", resp.Name)
	}
	if resp.Balance != 100.00 {
		t.Errorf("Expected balance 100.00, got %.2f", resp.Balance)
	}
}

func TestGetUserHandler_MissingID(t *testing.T) {
	h := newTestHandler(t)

	// Fire GET with no id parameter
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.GetUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing ID, got %d", w.Code)
	}
}

func TestGetUserHandler_NotFound(t *testing.T) {
	h := newTestHandler(t)

	// Request a user ID that was never created
	w := getWithQuery(h.GetUser, map[string]string{"id": "nonexistent_user"})

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for unknown user, got %d", w.Code)
	}
}

func TestGetUserHandler_WrongMethod(t *testing.T) {
	h := newTestHandler(t)

	// Send a POST to a GET-only endpoint
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()
	h.GetUser(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for wrong method, got %d", w.Code)
	}
}
