package models

import "time"

// User represents a registered account holder with financial and identity attributes.
type User struct {
	ID           string  `json:"id"`
	PasswordHash string  `json:"-"`
	Name         string  `json:"name"`
	Email        string  `json:"email"`
	PhoneNumber  string  `json:"phone_number"`
	Balance      float64 `json:"balance"`
	CreditScore  uint8   `json:"credit_score"`
	CreditLimit  float64 `json:"credit_limit"`
	IsActive     bool    `json:"is_active"`
	CreatedAt    string  `json:"created_at"`
}

// Profile is the public-facing subset of a User, safe to expose to other users.
type Profile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	CreatedAt   string `json:"created_at"`
}

// Payment records a single monetary transfer between two accounts, including BNPL master loan records.
type Payment struct {
	ID                string  `json:"id"`
	SenderID          string  `json:"sender_id"`
	ReceiverID        string  `json:"receiver_id"`
	Amount            float64 `json:"amount"`
	TotalAmount       float64 `json:"total_amount"`
	Note              string  `json:"note"`
	PaymentType       string  `json:"payment_type"`
	TotalInstallments uint8   `json:"total_installments"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"created_at"`
}

// PaymentWithInstallments bundles a master payment record with its full installment schedule.
type PaymentWithInstallments struct {
	Payment      Payment       `json:"payment"`
	Installments []Installment `json:"installment"`
}

// PaymentRequest represents a pending demand from one user asking another to send money.
type PaymentRequest struct {
	ID          string  `json:"id"`
	RequesterID string  `json:"requester_id"`
	PayerID     string  `json:"payer_id"`
	Amount      float64 `json:"amount"`
	Note        string  `json:"note"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
}

// Installment represents a single scheduled repayment slice within a BNPL loan plan.
type Installment struct {
	ID           string  `json:"id"`
	PaymentID    string  `json:"payment_id"`
	UserWithDebt string  `json:"user_id"`
	Amount       float64 `json:"amount"`
	DueDate      string  `json:"due_date"`
	IsPaid       bool    `json:"is_paid"`
	CreatedAt    string  `json:"created_at"`
}

// FriendRequest represents a pending social connection request between two users.
type FriendRequest struct {
	ID         string `json:"id"`
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	CreatedAt  string `json:"created_at"`
}

// Group represents a named collection of users for shared billing and payment splitting.
type Group struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatorID string `json:"creator_id"`
	CreatedAt string `json:"created_at"`
}

// GroupMember records the membership relationship between a user and a group.
type GroupMember struct {
	GroupID  string `json:"group_id"`
	MemberID string `json:"member_id"`
	JoinedAt string `json:"joined_at"`
}

// GroupInvitation represents a pending invite sent by a group member to an outside user.
type GroupInvitation struct {
	ID         string `json:"id"`
	GroupID    string `json:"group_id"`
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	CreatedAt  string `json:"created_at"`
}

// Notification represents an in-app alert delivered to a user for events like payments or friend requests.
type Notification struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	LinkView  string `json:"link_view"`
	IsSeen    bool   `json:"is_seen"`
	CreatedAt string `json:"created_at"`
}

// UserSettings holds per-user preference and privacy configuration stored in the user_settings table.
type UserSettings struct {
	UserID             string    `json:"user_id"`
	Theme              string    `json:"theme"`
	EmailNotifications bool      `json:"email_notifications"`
	IsDiscoverable     bool      `json:"is_discoverable"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// SpendingTotals aggregates directional cash flow metrics across a bounded time window for a single account.
type SpendingTotals struct {
	TotalSent     float64 `json:"total_sent"`
	TotalReceived float64 `json:"total_received"`
	Net           float64 `json:"net"`
}

// MonthlySummary captures aggregated financial activity for a single calendar billing month.
type MonthlySummary struct {
	Year              int        `json:"year"`
	Month             time.Month `json:"month"`
	TotalOut          float64    `json:"total_out"`
	TotalIn           float64    `json:"total_in"`
	ActiveBNPLCharges float64    `json:"active_bnpl_charges"`
}

// TopRecipient pairs a peer identity with the total volume of funds directed toward them.
type TopRecipient struct {
	Profile   Profile `json:"profile"`
	TotalSent float64 `json:"total_sent"`
}

// CreditScoreLog records a single audit checkpoint in a user's credit score history.
// Backed by the credit_score_log table.
type CreditScoreLog struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Score     int       `json:"score"`
	Delta     int       `json:"delta"`
	CreatedAt time.Time `json:"created_at"`
}

// InstallmentSummary provides a consolidated snapshot of a user's installment plan portfolio.
type InstallmentSummary struct {
	TotalSettled     float64 `json:"total_settled"`
	TotalOutstanding float64 `json:"total_outstanding"`
	TotalOverdue     float64 `json:"total_overdue"`
}

// LinkedCard represents a tokenized external funding instrument registered to a user account.
// Raw card numbers are never stored — only opaque provider token references.
type LinkedCard struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	TokenRef  string `json:"token_ref"`
	Last4     string `json:"last4"`
	Brand     string `json:"brand"`
	IsDefault bool   `json:"is_default"`
	CreatedAt string `json:"created_at"`
}

// WalletTx records an absolute balance mutation event (deposit or withdrawal) on a user account.
// Distinct from peer-to-peer Payment records in the payments ledger.
type WalletTx struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	Type      string  `json:"type"` // "deposit" | "withdrawal"
	Amount    float64 `json:"amount"`
	CreatedAt string  `json:"created_at"`
}
