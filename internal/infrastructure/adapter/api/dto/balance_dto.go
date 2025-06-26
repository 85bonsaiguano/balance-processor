package dto

// BalanceResponse represents the API response for a user's balance
type BalanceResponse struct {
	UserID  uint64 `json:"userId"`
	Balance string `json:"balance"`
}
