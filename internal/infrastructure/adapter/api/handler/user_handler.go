package handler

import (
	"errors"
	"net/http"
	"strconv"

	domainerr "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	userUseCase "github.com/amirhossein-jamali/balance-processor/internal/domain/usecase/user"
	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/api/dto"
	"github.com/gin-gonic/gin"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userService *userUseCase.UserUseCase
	logger      coreport.Logger
}

// NewUserHandler creates a new user handler instance
func NewUserHandler(
	userService *userUseCase.UserUseCase,
	logger coreport.Logger,
) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// GetBalance handles the GET /user/{userId}/balance endpoint
func (h *UserHandler) GetBalance(c *gin.Context) {
	// Extract user ID from path
	userIDParam := c.Param("userId")
	userID, err := strconv.ParseUint(userIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    domainerr.ErrorCode(domainerr.ErrInvalidUserID),
			Message: "Invalid user ID format",
		})
		return
	}

	// Get user balance
	balanceResponse, err := h.userService.GetBalance(c.Request.Context(), userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMessage := "Internal server error"

		// Map domain errors to HTTP status codes
		if errors.Is(err, domainerr.ErrUserNotFound) {
			statusCode = http.StatusNotFound
			errorMessage = "User not found"
		}

		h.logger.Error("Error getting user balance", map[string]any{
			"userId": userID,
			"error":  err.Error(),
		})

		c.JSON(statusCode, dto.ErrorResponse{
			Code:    domainerr.ErrorCode(err),
			Message: errorMessage,
		})
		return
	}

	// Return success response - direct mapping from usecase response to DTO
	c.JSON(http.StatusOK, dto.BalanceResponse{
		UserID:  balanceResponse.UserID,
		Balance: balanceResponse.Balance,
	})
}
