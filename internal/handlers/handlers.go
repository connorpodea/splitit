package handlers

// import (
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"strings"

// 	"github.com/connorpodea/splitit/internal/bnpl"
// 	"github.com/connorpodea/splitit/internal/models"
// 	"github.com/connorpodea/splitit/internal/store"
// )

// type Handler struct {
// 	store *store.Store
// }

// func New(s *store.Store) *Handler {
// 	return &Handler{store: s}
// }

// func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
// 	mux.HandleFunc("/api/users", h.handleUsers)
// 	mux.HandleFunc("/api/users/", h.handleUserByID)

// 	mux.HandleFunc("/api/payments", h.handlePayments)
// 	mux.HandleFunc("/api/payments/", h.handlePaymentByID)

// 	mux.HandleFunc("/api/plans", h.handlePlans)
// 	mux.HandleFunc("/api/plans/", h.handlePlanRoutes)

// 	mux.HandleFunc("/api/pay/", h.handlePayInstallment)
// }

// // --- Users ---

// func (h *Handler) handleUsers(w http.ResponseWriter, r *http.Request) {
// 	switch r.Method {
// 	case http.MethodGet:
// 		respond(w, h.store.ListUsers())
// 	case http.MethodPost:
// 		var u models.User
// 		if err := decode(r, &u); err != nil {
// 			httpErr(w, err.Error(), 400)
// 			return
// 		}
// 		if u.ID == "" || u.Name == "" || u.Email == "" {
// 			httpErr(w, "id, name, email required", 400)
// 			return
// 		}
// 		h.store.CreateUser(&u)
// 		respond(w, u)
// 	default:
// 		httpErr(w, "method not allowed", 405)
// 	}
// }

// func (h *Handler) handleUserByID(w http.ResponseWriter, r *http.Request) {
// 	id := strings.TrimPrefix(r.URL.Path, "/api/users/")
// 	u, err := h.store.GetUser(id)
// 	if err != nil {
// 		httpErr(w, err.Error(), 404)
// 		return
// 	}
// 	respond(w, u)
// }

// // --- Payments ---
// //
// // POST /api/payments  — send money to someone
// //
// // Body:
// //   {
// //     "id":          "p1",
// //     "sender_id":   "alice",
// //     "receiver_id": "bob",
// //     "amount":      120.00,
// //     "note":        "dinner",
// //     "method":      "instant" | "bnpl",
// //     "installments": 4        // only needed if method = bnpl
// //   }
// //
// // When method = "instant":
// //   - payment is marked settled immediately
// //   - TODO: actually move money (Stripe, Plaid, etc.)
// //
// // When method = "bnpl":
// //   - platform pays receiver now (TODO: real payout)
// //   - a BNPLPlan is created for the sender
// //   - payment is settled, plan is active

// func (h *Handler) handlePayments(w http.ResponseWriter, r *http.Request) {
// 	switch r.Method {
// 	case http.MethodGet:
// 		respond(w, h.store.ListPayments())
// 	case http.MethodPost:
// 		var body struct {
// 			models.Payment
// 			Installments int `json:"installments"`
// 		}
// 		if err := decode(r, &body); err != nil {
// 			httpErr(w, err.Error(), 400)
// 			return
// 		}
// 		p := &body.Payment
// 		if p.ID == "" || p.SenderID == "" || p.ReceiverID == "" || p.Amount <= 0 {
// 			httpErr(w, "id, sender_id, receiver_id, amount required", 400)
// 			return
// 		}
// 		if p.SenderID == p.ReceiverID {
// 			httpErr(w, "sender and receiver must be different users", 400)
// 			return
// 		}

// 		// Make sure both users exist
// 		if _, err := h.store.GetUser(p.SenderID); err != nil {
// 			httpErr(w, "sender not found", 404)
// 			return
// 		}
// 		if _, err := h.store.GetUser(p.ReceiverID); err != nil {
// 			httpErr(w, "receiver not found", 404)
// 			return
// 		}

// 		if err := h.store.CreatePayment(p); err != nil {
// 			httpErr(w, err.Error(), 500)
// 			return
// 		}

// 		if p.Method == models.MethodBNPL {
// 			// Platform fronts the money to the receiver.
// 			// TODO: trigger actual payout to receiver here.
// 			plan, schedule, err := bnpl.CreatePlan(p, body.Installments)
// 			if err != nil {
// 				httpErr(w, err.Error(), 400)
// 				return
// 			}

// 			h.store.CreatePlan(plan)
// 			h.store.SaveInstallments(plan.ID, schedule)
// 			h.store.SettlePayment(p.ID, plan.ID)

// 			respond(w, map[string]any{
// 				"payment":  p,
// 				"plan":     plan,
// 				"schedule": schedule,
// 			})
// 		} else {
// 			// Instant: just settle it.
// 			// TODO: real money movement.
// 			h.store.SettlePayment(p.ID, "")
// 			p.Status = models.StatusSettled
// 			respond(w, p)
// 		}
// 	default:
// 		httpErr(w, "method not allowed", 405)
// 	}
// }

// func (h *Handler) handlePaymentByID(w http.ResponseWriter, r *http.Request) {
// 	id := strings.TrimPrefix(r.URL.Path, "/api/payments/")
// 	p, err := h.store.GetPayment(id)
// 	if err != nil {
// 		httpErr(w, err.Error(), 404)
// 		return
// 	}
// 	respond(w, p)
// }

// // --- Plans ---

// func (h *Handler) handlePlans(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodGet {
// 		httpErr(w, "method not allowed", 405)
// 		return
// 	}
// 	userID := r.URL.Query().Get("user_id")
// 	if userID != "" {
// 		respond(w, h.store.ListPlansByUser(userID))
// 	} else {
// 		respond(w, h.store.ListPlans())
// 	}
// }

// // GET  /api/plans/:id                → get plan
// // GET  /api/plans/:id/installments   → get schedule
// func (h *Handler) handlePlanRoutes(w http.ResponseWriter, r *http.Request) {
// 	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/plans/"), "/")
// 	planID := parts[0]

// 	if len(parts) == 2 && parts[1] == "installments" {
// 		items, err := h.store.GetInstallments(planID)
// 		if err != nil {
// 			httpErr(w, err.Error(), 404)
// 			return
// 		}
// 		respond(w, items)
// 		return
// 	}

// 	plan, err := h.store.GetPlan(planID)
// 	if err != nil {
// 		httpErr(w, err.Error(), 404)
// 		return
// 	}
// 	respond(w, plan)
// }

// // --- Pay an installment ---
// //
// // POST /api/pay/:planID/:number
// // Marks one installment as paid. If all are paid, closes the plan.
// // TODO: take actual payment (card charge, wallet debit, etc.)

// func (h *Handler) handlePayInstallment(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodPost {
// 		httpErr(w, "method not allowed", 405)
// 		return
// 	}
// 	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/pay/"), "/")
// 	if len(parts) != 2 {
// 		httpErr(w, "path: /api/pay/{planID}/{number}", 400)
// 		return
// 	}
// 	planID := parts[0]
// 	n, err := parseInt(parts[1])
// 	if err != nil {
// 		httpErr(w, "installment number must be an integer", 400)
// 		return
// 	}

// 	if err := h.store.MarkInstallmentPaid(planID, n); err != nil {
// 		httpErr(w, err.Error(), 400)
// 		return
// 	}

// 	// Check if every installment is now paid → close the plan
// 	done, _ := h.store.CheckPlanComplete(planID)
// 	if done {
// 		h.store.CompletePlan(planID)
// 	}

// 	respond(w, map[string]any{"status": "paid", "plan_completed": done})
// }

// // --- helpers ---

// func decode(r *http.Request, v any) error {
// 	return json.NewDecoder(r.Body).Decode(v)
// }

// func respond(w http.ResponseWriter, v any) {
// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(v)
// }

// func httpErr(w http.ResponseWriter, msg string, code int) {
// 	http.Error(w, msg, code)
// }

// func parseInt(s string) (int, error) {
// 	n := 0
// 	if len(s) == 0 {
// 		return 0, fmt.Errorf("empty")
// 	}
// 	for _, c := range s {
// 		if c < '0' || c > '9' {
// 			return 0, fmt.Errorf("not a number")
// 		}
// 		n = n*10 + int(c-'0')
// 	}
// 	return n, nil
// }
