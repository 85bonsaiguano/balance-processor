package dto

import (
	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
)

// BalanceResponse represents the API response for a user's balance
type BalanceResponse struct {
	UserID  uint64 `json:"userId"`
	Balance string `json:"balance"`
}

// UserToBalanceResponse converts a domain User entity to a BalanceResponse DTO
func UserToBalanceResponse(user *entity.User) BalanceResponse {
	return BalanceResponse{
		UserID:  user.ID,
		Balance: user.GetBalance(),
	}
}
