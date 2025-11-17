package kakaopay

import "errors"

var (
	// ErrInvalidRequest is returned when the request parameters are invalid
	ErrInvalidRequest = errors.New("invalid request parameters")

	// ErrPaymentFailed is returned when the payment process fails
	ErrPaymentFailed = errors.New("payment failed")

	// ErrPaymentCancelled is returned when the payment is cancelled
	ErrPaymentCancelled = errors.New("payment cancelled")

	// ErrInvalidTransaction is returned when the transaction ID is invalid
	ErrInvalidTransaction = errors.New("invalid transaction ID")

	// ErrNetworkError is returned when there's a network communication error
	ErrNetworkError = errors.New("network error")

	// ErrUnauthorized is returned when the API key is invalid
	ErrUnauthorized = errors.New("unauthorized: invalid API key")

	// ErrInsufficientAmount is returned when cancel amount exceeds approved amount
	ErrInsufficientAmount = errors.New("cancel amount exceeds approved amount")

	// ErrAlreadyProcessed is returned when the transaction is already processed
	ErrAlreadyProcessed = errors.New("transaction already processed")
)
