package handlers

import "strings"

// friendlyError maps internal store/DB error messages to clean user-facing text.
// Raw SQL or internal detail is never exposed to the client.
func friendlyError(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "insufficient"):
		return "Insufficient balance to complete this transaction."
	case strings.Contains(msg, "sql: no rows"), strings.Contains(msg, "not found"):
		return "The requested item could not be found."
	case strings.Contains(msg, "UNIQUE constraint"):
		return "This record already exists."
	case strings.Contains(msg, "FOREIGN KEY constraint"):
		return "One or more referenced items could not be found."
	default:
		return "Something went wrong. Please try again."
	}
}
