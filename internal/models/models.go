package models

type User struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Email   string  `json:"email"`
	Balance float64 `json:"balance"`
}

type Payment struct {
	ID         string  `json:"id"`
	SenderID   string  `json:"sender_id"`
	ReceiverID string  `json:"receiver_id"`
	Amount     float64 `json:"amount"`
	Note       string  `json:"note"`
}
