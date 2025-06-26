package dto

// TransactionRequest represents the API request for processing a transaction
type TransactionRequest struct {
	State         string `json:"state" binding:"required,oneof=win lose"`
	Amount        string `json:"amount" binding:"required"`
	TransactionID string `json:"transactionId" binding:"required"`
}

// TransactionResponse represents the API response for a processed transaction
type TransactionResponse struct {
	TransactionID string `json:"transactionId"`
	UserID        uint64 `json:"userId"`
	Success       bool   `json:"success"`
	ResultBalance string `json:"resultBalance,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
}
