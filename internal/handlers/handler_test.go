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

// seedStandardUsers creates connor, jason, and alice via the handler
func seedStandardUsers(t *testing.T, h *handlers.Handler) {
	t.Helper()
	users := []models.User{
		{
			ID: "cpodea", Name: "Connor", Email: "cpodea@gmail.com",
			PhoneNumber: "123-456-7890", Balance: 500.00,
			CreditScore: 50, CreditLimit: 1000.00,
		},
		{
			ID: "jpodea", Name: "Jason", Email: "jpodea@asu.edu",
			PhoneNumber: "987-654-3210", Balance: 100.00,
			CreditScore: 80, CreditLimit: 1000.00,
		},
		{
			ID: "alice_w", Name: "Alice", Email: "alice@gmail.com",
			PhoneNumber: "555-555-5555", Balance: 150.00,
			CreditScore: 95, CreditLimit: 1500.00,
		},
	}
	for _, u := range users {
		w := postJSON(t, h.CreateUser, u)
		if w.Code != http.StatusCreated {
			t.Fatalf("Failed to seed user '%s': status %d body %s", u.ID, w.Code, w.Body.String())
		}
	}
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
	seedStandardUsers(t, h)

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

// =========================================================================
// ListUsers handler tests
// =========================================================================

func TestListUsersHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	w := getWithQuery(h.ListUsers, map[string]string{})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp []models.User
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	// Treasury + 3 seeded users = 4 total
	if len(resp) < 4 {
		t.Errorf("Expected at least 4 users, got %d", len(resp))
	}
}

func TestListUsersHandler_WrongMethod(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()
	h.ListUsers(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for wrong method, got %d", w.Code)
	}
}

// =========================================================================
// GetProfile handler tests
// =========================================================================

func TestGetProfileHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	w := getWithQuery(h.GetProfile, map[string]string{"user_id": "cpodea"})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp models.Profile
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Name != "Connor" {
		t.Errorf("Expected name 'Connor', got '%s'", resp.Name)
	}
	if resp.Email != "cpodea@gmail.com" {
		t.Errorf("Expected email 'cpodea@gmail.com', got '%s'", resp.Email)
	}
}

func TestGetProfileHandler_MissingID(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.GetProfile(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing user_id, got %d", w.Code)
	}
}

func TestGetProfileHandler_NotFound(t *testing.T) {
	h := newTestHandler(t)

	w := getWithQuery(h.GetProfile, map[string]string{"user_id": "ghost_user"})

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for unknown profile, got %d", w.Code)
	}
}

// =========================================================================
// ListProfiles handler tests
// =========================================================================

func TestListProfilesHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	w := getWithQuery(h.ListProfiles, map[string]string{})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp []models.Profile
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(resp) < 4 {
		t.Errorf("Expected at least 4 profiles, got %d", len(resp))
	}
}

// =========================================================================
// Pay handler tests
// =========================================================================

func TestPayHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	tx := models.Payment{
		ID: "p2p_tx_001", SenderID: "cpodea", ReceiverID: "jpodea",
		Amount: 25.00, TotalAmount: 25.00, Note: "Dinner",
		PaymentType: "peer_to_peer", TotalInstallments: 1, Status: "completed",
	}
	w := postJSON(t, h.Pay, tx)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestPayHandler_InsufficientFunds(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	// Connor only has $500.00 — attempt $9999.00
	tx := models.Payment{
		ID: "overdraft_001", SenderID: "cpodea", ReceiverID: "jpodea",
		Amount: 9999.00, TotalAmount: 9999.00, Note: "Overdraft attempt",
		PaymentType: "peer_to_peer", TotalInstallments: 1, Status: "completed",
	}
	w := postJSON(t, h.Pay, tx)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for insufficient funds, got %d", w.Code)
	}
}

func TestPayHandler_InvalidJSON(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Pay(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for malformed JSON, got %d", w.Code)
	}
}

func TestPayHandler_WrongMethod(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.Pay(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for wrong method, got %d", w.Code)
	}
}

// =========================================================================
// GetPayment handler tests
// =========================================================================

func TestGetPaymentHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	// Create a payment first
	tx := models.Payment{
		ID: "p2p_tx_001", SenderID: "cpodea", ReceiverID: "jpodea",
		Amount: 25.00, TotalAmount: 25.00, Note: "Dinner",
		PaymentType: "peer_to_peer", TotalInstallments: 1, Status: "completed",
	}
	postJSON(t, h.Pay, tx)

	w := getWithQuery(h.GetPayment, map[string]string{"payment_id": "p2p_tx_001"})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp models.Payment
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.SenderID != "cpodea" {
		t.Errorf("Expected sender 'cpodea', got '%s'", resp.SenderID)
	}
	if resp.Amount != 25.00 {
		t.Errorf("Expected amount 25.00, got %.2f", resp.Amount)
	}
}

func TestGetPaymentHandler_MissingID(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.GetPayment(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing payment_id, got %d", w.Code)
	}
}

func TestGetPaymentHandler_NotFound(t *testing.T) {
	h := newTestHandler(t)

	w := getWithQuery(h.GetPayment, map[string]string{"payment_id": "nonexistent_payment"})

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for unknown payment, got %d", w.Code)
	}
}

// =========================================================================
// CreateBNPLLoan handler tests
// =========================================================================

func TestCreateBNPLLoanHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	loan := models.Payment{
		ID: "bnpl_desk_001", SenderID: "cpodea", ReceiverID: "jpodea",
		TotalAmount: 200.00, Note: "Developer Desk",
		TotalInstallments: 4, Status: "pending",
	}
	w := postJSON(t, h.CreateBNPLLoan, loan)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestCreateBNPLLoanHandler_ZeroInstallmentsRejected(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	loan := models.Payment{
		ID: "bnpl_bad_001", SenderID: "cpodea", ReceiverID: "jpodea",
		TotalAmount: 100.00, TotalInstallments: 0, Status: "pending",
	}
	w := postJSON(t, h.CreateBNPLLoan, loan)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for zero installments, got %d", w.Code)
	}
}

func TestCreateBNPLLoanHandler_OverCreditLimitRejected(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	// Connor's limit is $1000.00 — attempt $1100.00
	loan := models.Payment{
		ID: "bnpl_over_001", SenderID: "cpodea", ReceiverID: "jpodea",
		TotalAmount: 1100.00, TotalInstallments: 4, Status: "pending",
	}
	w := postJSON(t, h.CreateBNPLLoan, loan)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for over credit limit, got %d", w.Code)
	}
}

func TestCreateBNPLLoanHandler_InvalidJSON(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateBNPLLoan(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for malformed JSON, got %d", w.Code)
	}
}

func TestCreateBNPLLoanHandler_WrongMethod(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.CreateBNPLLoan(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for wrong method, got %d", w.Code)
	}
}

// =========================================================================
// PayInstallment handler tests
// =========================================================================

func TestPayInstallmentHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	// Create a BNPL loan first so installments exist
	loan := models.Payment{
		ID: "bnpl_desk_001", SenderID: "cpodea", ReceiverID: "jpodea",
		TotalAmount: 200.00, Note: "Developer Desk",
		TotalInstallments: 4, Status: "pending",
	}
	postJSON(t, h.CreateBNPLLoan, loan)

	// Pay installment 2
	w := postJSON(t, h.PayInstallment, map[string]any{
		"installment_id": "inst_bnpl_desk_001_2",
		"payment_id":     "bnpl_desk_001",
		"user_id":        "cpodea",
		"amount":         51.50,
	})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestPayInstallmentHandler_InsufficientFunds(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	// Create a BNPL loan
	loan := models.Payment{
		ID: "bnpl_desk_001", SenderID: "cpodea", ReceiverID: "jpodea",
		TotalAmount: 200.00, TotalInstallments: 4, Status: "pending",
	}
	postJSON(t, h.CreateBNPLLoan, loan)

	// Drain Connor's balance so he can't afford the installment
	drain := models.Payment{
		ID: "drain_001", SenderID: "cpodea", ReceiverID: "jpodea",
		Amount: 448.00, TotalAmount: 448.00,
		PaymentType: "peer_to_peer", TotalInstallments: 1, Status: "completed",
	}
	postJSON(t, h.Pay, drain)

	w := postJSON(t, h.PayInstallment, map[string]any{
		"installment_id": "inst_bnpl_desk_001_2",
		"payment_id":     "bnpl_desk_001",
		"user_id":        "cpodea",
		"amount":         51.50,
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for insufficient funds, got %d", w.Code)
	}
}

func TestPayInstallmentHandler_InvalidJSON(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.PayInstallment(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for malformed JSON, got %d", w.Code)
	}
}

func TestPayInstallmentHandler_WrongMethod(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.PayInstallment(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for wrong method, got %d", w.Code)
	}
}

// =========================================================================
// ListInstallments handler tests
// =========================================================================

func TestListInstallmentsHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	// Create a BNPL loan so there are installments to list
	loan := models.Payment{
		ID: "bnpl_desk_001", SenderID: "cpodea", ReceiverID: "jpodea",
		TotalAmount: 200.00, TotalInstallments: 4, Status: "pending",
	}
	postJSON(t, h.CreateBNPLLoan, loan)

	w := getWithQuery(h.ListInstallments, map[string]string{"user_id": "cpodea"})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp []models.Installment
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(resp) != 4 {
		t.Errorf("Expected 4 installments for Pay-in-4 loan, got %d", len(resp))
	}
}

func TestListInstallmentsHandler_MissingUserID(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ListInstallments(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing user_id, got %d", w.Code)
	}
}

func TestListInstallmentsHandler_WrongMethod(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()
	h.ListInstallments(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for wrong method, got %d", w.Code)
	}
}

// =========================================================================
// ListOverdueInstallments handler tests
// =========================================================================

func TestListOverdueInstallmentsHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	// Create a loan — installments are future-dated so none should be overdue
	loan := models.Payment{
		ID: "bnpl_desk_001", SenderID: "cpodea", ReceiverID: "jpodea",
		TotalAmount: 200.00, TotalInstallments: 4, Status: "pending",
	}
	postJSON(t, h.CreateBNPLLoan, loan)

	w := getWithQuery(h.ListOverdueInstallments, map[string]string{"user_id": "cpodea"})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp []models.Installment
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		// An empty array from the server can decode as null — that is acceptable here
		resp = []models.Installment{}
	}
	if len(resp) != 0 {
		t.Errorf("Expected 0 overdue installments for future-dated loan, got %d", len(resp))
	}
}

func TestListOverdueInstallmentsHandler_MissingUserID(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ListOverdueInstallments(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing user_id, got %d", w.Code)
	}
}

// =========================================================================
// Friend request handler tests
// =========================================================================

func TestSendFriendRequestHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	req := models.FriendRequest{
		ID: "freq_001", SenderID: "cpodea", ReceiverID: "jpodea",
	}
	w := postJSON(t, h.SendFriendRequest, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestSendFriendRequestHandler_InvalidJSON(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.SendFriendRequest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for malformed JSON, got %d", w.Code)
	}
}

func TestSendFriendRequestHandler_WrongMethod(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.SendFriendRequest(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for wrong method, got %d", w.Code)
	}
}

func TestListIncomingFriendRequestsHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	// Send two requests to Jason
	postJSON(t, h.SendFriendRequest, models.FriendRequest{ID: "freq_001", SenderID: "cpodea", ReceiverID: "jpodea"})
	postJSON(t, h.SendFriendRequest, models.FriendRequest{ID: "freq_002", SenderID: "alice_w", ReceiverID: "jpodea"})

	w := getWithQuery(h.ListIncomingFriendRequests, map[string]string{"user_id": "jpodea"})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp []models.FriendRequest
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("Expected 2 incoming requests for Jason, got %d", len(resp))
	}
}

func TestListIncomingFriendRequestsHandler_MissingUserID(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ListIncomingFriendRequests(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing user_id, got %d", w.Code)
	}
}

func TestListOutgoingFriendRequestsHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	// Alice sends two outgoing requests
	postJSON(t, h.SendFriendRequest, models.FriendRequest{ID: "freq_003", SenderID: "alice_w", ReceiverID: "cpodea"})
	postJSON(t, h.SendFriendRequest, models.FriendRequest{ID: "freq_004", SenderID: "alice_w", ReceiverID: "jpodea"})

	w := getWithQuery(h.ListOutgoingFriendRequests, map[string]string{"user_id": "alice_w"})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp []models.FriendRequest
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("Expected 2 outgoing requests for Alice, got %d", len(resp))
	}
}

func TestAcceptFriendRequestHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	postJSON(t, h.SendFriendRequest, models.FriendRequest{ID: "freq_001", SenderID: "cpodea", ReceiverID: "jpodea"})

	w := postJSON(t, h.AcceptFriendRequest, map[string]string{
		"request_id":  "freq_001",
		"sender_id":   "cpodea",
		"receiver_id": "jpodea",
	})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d — body: %s", w.Code, w.Body.String())
	}

	// Verify bidirectional friendship was created
	jFriends := getWithQuery(h.ListFriends, map[string]string{"user_id": "jpodea"})
	var friends []models.Profile
	json.NewDecoder(jFriends.Body).Decode(&friends)
	if len(friends) != 1 || friends[0].ID != "cpodea" {
		t.Errorf("Expected Jason's friends list to contain Connor after accept")
	}
}

func TestAcceptFriendRequestHandler_InvalidJSON(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.AcceptFriendRequest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for malformed JSON, got %d", w.Code)
	}
}

func TestDeclineFriendRequestHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	postJSON(t, h.SendFriendRequest, models.FriendRequest{ID: "freq_001", SenderID: "alice_w", ReceiverID: "jpodea"})

	w := postJSON(t, h.DeclineFriendRequest, map[string]string{"request_id": "freq_001"})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d — body: %s", w.Code, w.Body.String())
	}

	// Verify the request no longer appears in Jason's incoming queue
	incoming := getWithQuery(h.ListIncomingFriendRequests, map[string]string{"user_id": "jpodea"})
	var reqs []models.FriendRequest
	json.NewDecoder(incoming.Body).Decode(&reqs)
	if len(reqs) != 0 {
		t.Errorf("Expected 0 incoming requests after decline, got %d", len(reqs))
	}
}

func TestRemoveFriendMutualHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	// Create and accept a friend request first
	postJSON(t, h.SendFriendRequest, models.FriendRequest{ID: "freq_001", SenderID: "cpodea", ReceiverID: "alice_w"})
	postJSON(t, h.AcceptFriendRequest, map[string]string{
		"request_id":  "freq_001",
		"sender_id":   "cpodea",
		"receiver_id": "alice_w",
	})

	w := postJSON(t, h.RemoveFriendMutual, map[string]string{
		"user_id":   "cpodea",
		"friend_id": "alice_w",
	})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d — body: %s", w.Code, w.Body.String())
	}

	// Verify Connor no longer appears in Alice's list (bidirectional removal)
	aFriends := getWithQuery(h.ListFriends, map[string]string{"user_id": "alice_w"})
	var friends []models.Profile
	json.NewDecoder(aFriends.Body).Decode(&friends)
	for _, f := range friends {
		if f.ID == "cpodea" {
			t.Error("Connor still appears in Alice's friends list after mutual removal")
		}
	}
}

func TestListFriendsHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	postJSON(t, h.SendFriendRequest, models.FriendRequest{ID: "freq_001", SenderID: "cpodea", ReceiverID: "jpodea"})
	postJSON(t, h.AcceptFriendRequest, map[string]string{
		"request_id":  "freq_001",
		"sender_id":   "cpodea",
		"receiver_id": "jpodea",
	})

	w := getWithQuery(h.ListFriends, map[string]string{"user_id": "jpodea"})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp []models.Profile
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(resp) != 1 || resp[0].ID != "cpodea" {
		t.Errorf("Expected Jason's friends list to contain only Connor")
	}
}

func TestListFriendsHandler_MissingUserID(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ListFriends(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing user_id, got %d", w.Code)
	}
}

// =========================================================================
// Payment request handler tests
// =========================================================================

func TestCreatePaymentRequestHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	req := models.PaymentRequest{
		ID: "bill_001", RequesterID: "jpodea", PayerID: "cpodea",
		Amount: 22.75, Note: "Uber split", Status: "pending",
	}
	w := postJSON(t, h.CreatePaymentRequest, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestCreatePaymentRequestHandler_InvalidJSON(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreatePaymentRequest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for malformed JSON, got %d", w.Code)
	}
}

func TestCreatePaymentRequestHandler_WrongMethod(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.CreatePaymentRequest(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for wrong method, got %d", w.Code)
	}
}

func TestListIncomingPaymentRequestsHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	postJSON(t, h.CreatePaymentRequest, models.PaymentRequest{
		ID: "bill_001", RequesterID: "jpodea", PayerID: "cpodea",
		Amount: 22.75, Note: "Uber split", Status: "pending",
	})

	w := getWithQuery(h.ListIncomingPaymentRequests, map[string]string{"user_id": "cpodea"})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp []models.PaymentRequest
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("Expected 1 incoming payment request for Connor, got %d", len(resp))
	}
}

func TestListIncomingPaymentRequestsHandler_MissingUserID(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ListIncomingPaymentRequests(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing user_id, got %d", w.Code)
	}
}

func TestListOutgoingPaymentRequestsHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	// Jason sends two outgoing payment requests
	postJSON(t, h.CreatePaymentRequest, models.PaymentRequest{
		ID: "bill_001", RequesterID: "jpodea", PayerID: "cpodea",
		Amount: 22.75, Note: "Uber split", Status: "pending",
	})
	postJSON(t, h.CreatePaymentRequest, models.PaymentRequest{
		ID: "bill_002", RequesterID: "jpodea", PayerID: "alice_w",
		Amount: 30.00, Note: "Grocery run", Status: "pending",
	})

	w := getWithQuery(h.ListOutgoingPaymentRequests, map[string]string{"user_id": "jpodea"})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp []models.PaymentRequest
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("Expected 2 outgoing payment requests for Jason, got %d", len(resp))
	}
}

func TestUpdatePaymentRequestStatusHandler_Success(t *testing.T) {
	h := newTestHandler(t)
	seedStandardUsers(t, h)

	postJSON(t, h.CreatePaymentRequest, models.PaymentRequest{
		ID: "bill_001", RequesterID: "jpodea", PayerID: "cpodea",
		Amount: 22.75, Note: "Uber split", Status: "pending",
	})

	w := postJSON(t, h.UpdatePaymentRequestStatus, map[string]string{
		"payment_id": "bill_001",
		"new_status": "completed",
	})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d — body: %s", w.Code, w.Body.String())
	}

	// Verify the bill no longer appears in the pending incoming list for Connor
	incoming := getWithQuery(h.ListIncomingPaymentRequests, map[string]string{"user_id": "cpodea"})
	var reqs []models.PaymentRequest
	json.NewDecoder(incoming.Body).Decode(&reqs)
	if len(reqs) != 0 {
		t.Errorf("Expected 0 pending requests after status update to completed, got %d", len(reqs))
	}
}

func TestUpdatePaymentRequestStatusHandler_InvalidJSON(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.UpdatePaymentRequestStatus(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for malformed JSON, got %d", w.Code)
	}
}

func TestUpdatePaymentRequestStatusHandler_WrongMethod(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.UpdatePaymentRequestStatus(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for wrong method, got %d", w.Code)
	}
}