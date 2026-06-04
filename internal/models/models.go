package models

import "time"

type User struct {
	ID           string  `json:"id"`
	PasswordHash string  `json:"-"`
	Name         string  `json:"name"`
	Email        string  `json:"email"`
	PhoneNumber  string  `json:"phone_number"`
	Balance      float64 `json:"balance"`
	CreditScore  uint8   `json:"credit_score"`
	CreditLimit  float64 `json:"credit_limit"`
	CreatedAt    string  `json:"created_at"`
}

type Profile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	CreatedAt   string `json:"created_at"`
}

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

type PaymentWithInstallments struct {
	Payment      Payment       `json:"payment"`
	Installments []Installment `json:"installment"`
}

type PaymentRequest struct {
	ID          string  `json:"id"`
	RequesterID string  `json:"requester_id"`
	PayerID     string  `json:"payer_id"`
	Amount      float64 `json:"amount"`
	Note        string  `json:"note"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
}

type Installment struct {
	ID           string  `json:"id"`
	PaymentID    string  `json:"payment_id"`
	UserWithDebt string  `json:"user_id"`
	Amount       float64 `json:"amount"`
	DueDate      string  `json:"due_date"`
	IsPaid       bool    `json:"is_paid"`
	CreatedAt    string  `json:"created_at"`
}

type FriendRequest struct {
	ID         string `json:"id"`
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	CreatedAt  string `json:"created_at"`
}

type Group struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatorID string `json:"creator_id"`
	CreatedAt string `json:"created_at"`
}

type GroupMember struct {
	GroupID  string `json:"group_id"`
	MemberID string `json:"member_id"`
	JoinedAt string `json:"joined_at"`
}

type GroupInvitation struct {
	ID         string `json:"id"`
	GroupID    string `json:"group_id"`
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	CreatedAt  string `json:"created_at"`
}

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

type UserSettings struct {
	UserID             string    `json:"user_id"`
	Theme              string    `json:"theme"`                // "light" or "dark"
	DefaultLandingPage string    `json:"default_landing_page"` // "dashboard" or "activity" feed
	EmailNotifications bool      `json:"email_notifications"`  // true/false toggle for transfer receipts
	IsDiscoverable     bool      `json:"is_discoverable"`      // true/false: can other users find them in search?
	UpdatedAt          time.Time `json:"updated_at"`
}
