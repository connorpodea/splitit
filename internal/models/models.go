package models

type User struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Email       string  `json:"email"`
	PhoneNumber string  `json:"phone_number"`
	Balance     float64 `json:"balance"`
	CreditScore uint8   `json:"credit_score"`
	CreditLimit float64 `json:"credit_limit"`
	CreatedAt   string  `json:"created_at"`
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

type Installment struct {
	ID           string  `json:"id"`
	PaymentID    string  `json:"paymentID"`
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
	Accepted   bool   `json:"accepted"`
	CreatedAt  string `json:"created_at"`
}
