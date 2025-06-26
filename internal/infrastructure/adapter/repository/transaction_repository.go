package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/model"
	"gorm.io/gorm"
)

// TransactionRepository implements TransactionRepository interface using GORM
type TransactionRepository struct {
	db              *gorm.DB
	logger          coreport.Logger
	errorClassifier *ErrorClassifier
}

// NewTransactionRepository creates a new TransactionRepository instance
func NewTransactionRepository(db *gorm.DB, logger coreport.Logger) *TransactionRepository {
	return &TransactionRepository{
		db:              db,
		logger:          logger,
		errorClassifier: NewErrorClassifier(),
	}
}

// entityToModel converts a transaction entity to a database model
func (r *TransactionRepository) entityToModel(transaction *entity.Transaction) model.Transaction {
	return model.Transaction{
		UserID:        transaction.UserID,
		TransactionID: transaction.TransactionID,
		SourceType:    string(transaction.SourceType),
		State:         string(transaction.State),
		Amount:        transaction.Amount,
		AmountInCents: transaction.AmountInCents,
		CreatedAt:     transaction.CreatedAt,
		ProcessedAt:   transaction.ProcessedAt,
		ResultBalance: transaction.ResultBalance,
		Status:        string(transaction.Status),
		ErrorMessage:  transaction.ErrorMessage,
	}
}

// Create saves a new transaction with optimized retry mechanism
func (r *TransactionRepository) Create(ctx context.Context, transaction *entity.Transaction) error {
	r.logger.Debug("Creating transaction", map[string]any{
		"transaction_id": transaction.TransactionID,
		"user_id":        transaction.UserID,
	})

	// Convert entity to model
	transactionModel := r.entityToModel(transaction)

	// Direct approach without excessive retries - optimize for speed
	result := r.db.WithContext(ctx).Create(&transactionModel)

	if result.Error != nil {
		// Check for duplicate key error
		if r.errorClassifier.IsDuplicateKeyError(result.Error) {
			// Specific handling for duplicate key errors
			return r.handleDuplicateTransactionError(transaction)
		}

		// For other errors
		r.logger.Error("Failed to create transaction", map[string]any{
			"transaction_id": transaction.TransactionID,
			"user_id":        transaction.UserID,
			"error":          result.Error.Error(),
		})
		return fmt.Errorf("%w: %s", errs.ErrDatabaseConnection, result.Error.Error())
	}

	r.logger.Info("Transaction created successfully", map[string]any{
		"transaction_id": transaction.TransactionID,
		"user_id":        transaction.UserID,
	})
	return nil
}

// handleDuplicateTransactionError handles duplicate transaction errors specifically
func (r *TransactionRepository) handleDuplicateTransactionError(transaction *entity.Transaction) error {
	r.logger.Warn("Duplicate transaction detected", map[string]any{
		"transaction_id": transaction.TransactionID,
		"user_id":        transaction.UserID,
	})
	return errs.ErrDuplicateTransaction
}

// Update updates an existing transaction with optimized approach
func (r *TransactionRepository) Update(ctx context.Context, transaction *entity.Transaction) error {
	r.logger.Debug("Updating transaction", map[string]any{
		"transaction_id": transaction.TransactionID,
		"status":         transaction.Status,
	})

	// Convert entity to model
	transactionModel := r.entityToModel(transaction)

	// Update only necessary fields with direct approach
	result := r.db.WithContext(ctx).Model(&model.Transaction{}).
		Where("transaction_id = ?", transaction.TransactionID).
		Updates(map[string]interface{}{
			"status":         transactionModel.Status,
			"processed_at":   transactionModel.ProcessedAt,
			"result_balance": transactionModel.ResultBalance,
			"error_message":  transactionModel.ErrorMessage,
		})

	if result.Error != nil {
		r.logger.Error("Failed to update transaction", map[string]any{
			"transaction_id": transaction.TransactionID,
			"error":          result.Error.Error(),
		})
		return fmt.Errorf("%w: %s", errs.ErrDatabaseConnection, result.Error.Error())
	}

	if result.RowsAffected == 0 {
		r.logger.Warn("Transaction not found during update", map[string]any{
			"transaction_id": transaction.TransactionID,
		})
		return errs.ErrTransactionNotFound
	}

	r.logger.Debug("Transaction updated successfully", map[string]any{
		"transaction_id": transaction.TransactionID,
		"status":         transaction.Status,
	})
	return nil
}

// TransactionExists checks if a transaction with the given ID already exists
func (r *TransactionRepository) TransactionExists(ctx context.Context, transactionID string) (bool, error) {
	r.logger.Debug("Checking if transaction exists", map[string]any{
		"transaction_id": transactionID,
	})

	var count int64
	result := r.db.WithContext(ctx).Model(&model.Transaction{}).
		Where("transaction_id = ?", transactionID).
		Count(&count)

	if result.Error != nil {
		r.logger.Error("Failed to check transaction existence", map[string]any{
			"transaction_id": transactionID,
			"error":          result.Error.Error(),
		})
		return false, fmt.Errorf("%w: %s", errs.ErrDatabaseConnection, result.Error.Error())
	}

	exists := count > 0
	r.logger.Debug("Transaction existence check completed", map[string]any{
		"transaction_id": transactionID,
		"exists":         exists,
	})
	return exists, nil
}

// modelToEntity converts a transaction model to an entity
func (r *TransactionRepository) modelToEntity(model *model.Transaction) *entity.Transaction {
	transaction := &entity.Transaction{
		ID:            model.ID,
		UserID:        model.UserID,
		TransactionID: model.TransactionID,
		SourceType:    entity.SourceType(model.SourceType),
		State:         entity.TransactionState(model.State),
		Amount:        model.Amount,
		AmountInCents: model.AmountInCents,
		CreatedAt:     model.CreatedAt,
		ProcessedAt:   model.ProcessedAt,
		ResultBalance: model.ResultBalance,
		Status:        entity.TransactionStatus(model.Status),
		ErrorMessage:  model.ErrorMessage,
	}

	return transaction
}

// GetByTransactionID retrieves a transaction by its external transaction ID
func (r *TransactionRepository) GetByTransactionID(ctx context.Context, transactionID string) (*entity.Transaction, error) {
	r.logger.Debug("Getting transaction by ID", map[string]any{
		"transaction_id": transactionID,
	})

	var transactionModel model.Transaction
	result := r.db.WithContext(ctx).
		Where("transaction_id = ?", transactionID).
		First(&transactionModel)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			r.logger.Warn("Transaction not found", map[string]any{
				"transaction_id": transactionID,
			})
			return nil, errs.ErrTransactionNotFound
		}
		r.logger.Error("Failed to get transaction", map[string]any{
			"transaction_id": transactionID,
			"error":          result.Error.Error(),
		})
		return nil, fmt.Errorf("%w: %s", errs.ErrDatabaseConnection, result.Error.Error())
	}

	// Convert model to entity
	transaction := r.modelToEntity(&transactionModel)

	r.logger.Debug("Transaction retrieved successfully", map[string]any{
		"transaction_id": transactionID,
		"user_id":        transaction.UserID,
		"status":         transaction.Status,
	})

	return transaction, nil
}
