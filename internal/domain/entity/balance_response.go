package entity

// BalanceResponse represents the response for the balance endpoint
type BalanceResponse struct {
	UserID  uint64 `json:"userId"`
	Balance string `json:"balance"`
}

// UserToBalanceResponse converts a User entity to a BalanceResponse DTO
// This is a separate function rather than a method on User to keep domain models clean
func UserToBalanceResponse(user *User) BalanceResponse {
	return BalanceResponse{
		UserID:  user.ID,
		Balance: user.GetBalance(),
	}
}
