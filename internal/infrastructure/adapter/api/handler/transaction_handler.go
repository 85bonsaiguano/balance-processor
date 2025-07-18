package handler

import (
	"net/http"
	"strconv"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
	domainerr "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	transactionUseCase "github.com/amirhossein-jamali/balance-processor/internal/domain/usecase/transaction"
	userUseCase "github.com/amirhossein-jamali/balance-processor/internal/domain/usecase/user"
	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/api/dto"
	"github.com/gin-gonic/gin"
)

// TransactionHandler handles transaction-related HTTP requests
type TransactionHandler struct {
	transactionService *transactionUseCase.Service
	userService        *userUseCase.UserUseCase
	logger             coreport.Logger
}

// NewTransactionHandler creates a new transaction handler instance
func NewTransactionHandler(
	transactionService *transactionUseCase.Service,
	userService *userUseCase.UserUseCase,
	logger coreport.Logger,
) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
		userService:        userService,
		logger:             logger,
	}
}

// ProcessTransaction handles the POST /user/{userId}/transaction endpoint
func (h *TransactionHandler) ProcessTransaction(c *gin.Context) {
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

	// Get Source-Type from header
	sourceType := c.GetHeader("Source-Type")
	if sourceType == "" {
		h.logger.Error("Missing Source-Type header", nil)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    domainerr.ErrorCode(domainerr.ErrInvalidRequest),
			Message: "Missing required header: Source-Type",
		})
		return
	}

	// Validate Source-Type
	if sourceType != "game" && sourceType != "server" && sourceType != "payment" {
		h.logger.Error("Invalid Source-Type header", map[string]any{
			"sourceType": sourceType,
		})
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    domainerr.ErrorCode(domainerr.ErrInvalidRequest),
			Message: "Invalid Source-Type. Must be one of: game, server, payment",
		})
		return
	}

	// Parse request body
	var req dto.TransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid transaction request format", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    domainerr.ErrorCode(domainerr.ErrInvalidRequest),
			Message: "Invalid request format: " + err.Error(),
		})
		return
	}

	// Check if user exists
	exists, err := h.userService.UserExists(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Error checking user existence", map[string]any{
			"userId": userID,
			"error":  err.Error(),
		})
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    domainerr.ErrorCode(domainerr.ErrInternalServer),
			Message: "Internal server error",
		})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Code:    domainerr.ErrorCode(domainerr.ErrUserNotFound),
			Message: "User not found",
		})
		return
	}

	// Map to domain request
	transactionReq := transactionUseCase.TransactionRequest{
		State:         req.State,
		Amount:        req.Amount,
		TransactionID: req.TransactionID,
		SourceType:    entity.SourceType(sourceType),
	}

	// Process the transaction
	result, err := h.transactionService.ProcessTransaction(c.Request.Context(), userID, transactionReq)

	// Return appropriate response based on result
	if err != nil {
		// The result already contains the right status code and error message
		// If we've reached here, the usecase returned a result with an error
		c.JSON(result.StatusCode, dto.ErrorResponse{
			Code:    domainerr.ErrorCode(err),
			Message: result.ErrorMessage,
		})
		return
	}

	// Success response
	c.JSON(http.StatusOK, dto.TransactionResponse{
		TransactionID: req.TransactionID,
		UserID:        userID,
		Success:       result.Success,
		ResultBalance: result.ResultBalance,
		ErrorMessage:  result.ErrorMessage,
	})
}
