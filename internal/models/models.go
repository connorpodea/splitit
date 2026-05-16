package models

import "time"

// User is anyone on the platform — sender or receiver.
// There is no "lender" role. The platform itself is the lender.
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`

	// TODO: add balance (platform wallet), linked bank account, credit limit
	// TODO: add KYC status before allowing BNPL
}

// Payment is the core object — one person paying another.
// It starts as a simple request and then either goes through immediately
// or gets split into a BNPL plan.
type Payment struct {
	ID         string        `json:"id"`
	SenderID   string        `json:"sender_id"`
	ReceiverID string        `json:"receiver_id"`
	Amount     float64       `json:"amount"`
	Note       string        `json:"note"`       // e.g. "dinner", "rent", "concert tickets"
	Method     PaymentMethod `json:"method"`     // how the sender is paying
	Status     PaymentStatus `json:"status"`
	CreatedAt  time.Time     `json:"created_at"`

	// Only set when method = MethodBNPL
	PlanID string `json:"plan_id,omitempty"`

	// TODO: add currency, attachments, reactions (Venmo-style)
}

type PaymentMethod string

const (
	// Sender pays right now — straight through, like Venmo.
	MethodInstant PaymentMethod = "instant"

	// Sender pays over time. Receiver still gets the money now.
	// Platform fronts the money and collects installments from sender.
	MethodBNPL PaymentMethod = "bnpl"
)

type PaymentStatus string

const (
	StatusPending   PaymentStatus = "pending"    // created, not yet settled
	StatusSettled   PaymentStatus = "settled"    // receiver got the money
	StatusFailed    PaymentStatus = "failed"     // something went wrong
)

// BNPLPlan is created when the sender chooses "pay over time".
// The platform pays the receiver upfront, then collects from the sender.
type BNPLPlan struct {
	ID           string     `json:"id"`
	PaymentID    string     `json:"payment_id"`
	SenderID     string     `json:"sender_id"`
	Total        float64    `json:"total"`        // what sender owes (principal + fee)
	Principal    float64    `json:"principal"`    // original payment amount
	Fee          float64    `json:"fee"`          // platform's cut
	Installments int        `json:"installments"` // how many payments
	Status       PlanStatus `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`

	// TODO: add fee_rate, grace_period, late_fee_amount
	// TODO: add approval status (platform may decline based on credit)
}

type PlanStatus string

const (
	PlanActive    PlanStatus = "active"
	PlanCompleted PlanStatus = "completed"
	PlanDefaulted PlanStatus = "defaulted"
)

// Installment is one chunk of a BNPLPlan that the sender owes the platform.
type Installment struct {
	ID        string            `json:"id"`
	PlanID    string            `json:"plan_id"`
	Number    int               `json:"number"`   // 1, 2, 3...
	Amount    float64           `json:"amount"`
	DueDate   time.Time         `json:"due_date"`
	PaidAt    *time.Time        `json:"paid_at,omitempty"`
	Status    InstallmentStatus `json:"status"`

	// TODO: add late_fee_applied, payment_method (card, bank, wallet)
}

type InstallmentStatus string

const (
	InstallmentPending InstallmentStatus = "pending"
	InstallmentPaid    InstallmentStatus = "paid"
	InstallmentOverdue InstallmentStatus = "overdue"
)
