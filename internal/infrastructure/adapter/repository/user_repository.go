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

// getOperationType returns "credit" for positive or zero changes and "debit" for negative changes
func getOperationType(balanceChange int64) string {
	if balanceChange >= 0 {
		return "credit"
	}
	return "debit"
}

// UserRepository implements UserRepository interface using GORM
type UserRepository struct {
	db              *gorm.DB
	timeProvider    coreport.TimeProvider
	logger          coreport.Logger
	errorClassifier *ErrorClassifier
}

// NewUserRepository creates a new UserRepository instance
func NewUserRepository(db *gorm.DB, timeProvider coreport.TimeProvider, logger coreport.Logger) *UserRepository {
	return &UserRepository{
		db:              db,
		timeProvider:    timeProvider,
		logger:          logger,
		errorClassifier: NewErrorClassifier(),
	}
}

// modelToEntity converts a user model to an entity
func (r *UserRepository) modelToEntity(userModel *model.User) (*entity.User, error) {
	user, err := entity.NewUser(userModel.ID, entity.AmountInCentsToString(userModel.Balance), r.timeProvider)
	if err != nil {
		r.logger.Error("Failed to create user entity", map[string]any{
			"user_id": userModel.ID,
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("%w: failed to create user entity: %s", errs.ErrInternalServer, err.Error())
	}

	// Set additional properties
	user.CreatedAt = userModel.CreatedAt
	user.UpdatedAt = userModel.UpdatedAt
	user.TransactionCount = userModel.TransactionCount

	return user, nil
}

// handleDatabaseError standardizes database error handling
func (r *UserRepository) handleDatabaseError(operation string, err error, userID uint64) error {
	r.logger.Error(fmt.Sprintf("Database error when %s", operation), map[string]any{
		"user_id": userID,
		"error":   err.Error(),
	})

	if errors.Is(err, gorm.ErrRecordNotFound) {
		r.logger.Warn("User not found", map[string]any{
			"user_id": userID,
		})
		return errs.ErrUserNotFound
	}

	if r.errorClassifier.IsDuplicateKeyError(err) {
		r.logger.Warn("Duplicate user operation", map[string]any{
			"user_id": userID,
		})
		return errs.ErrDuplicateUser
	}

	if r.errorClassifier.IsLockError(err) {
		r.logger.Warn("User is locked by another transaction", map[string]any{
			"user_id": userID,
			"error":   err.Error(),
		})
		return errs.ErrUserLocked
	}

	return fmt.Errorf("%w: %s", errs.ErrDatabaseConnection, err.Error())
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uint64) (*entity.User, error) {
	r.logger.Debug("Getting user by ID", map[string]any{
		"user_id": id,
	})

	var userModel model.User
	result := r.db.WithContext(ctx).First(&userModel, id)

	if result.Error != nil {
		return nil, r.handleDatabaseError("getting user", result.Error, id)
	}

	// Convert model to entity
	user, err := r.modelToEntity(&userModel)
	if err != nil {
		return nil, err
	}

	r.logger.Debug("User retrieved successfully", map[string]any{
		"user_id":      id,
		"balance":      user.GetBalance(),
		"tx_count":     user.TransactionCount,
		"last_updated": user.UpdatedAt,
	})

	return user, nil
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *entity.User) error {
	r.logger.Debug("Creating new user", map[string]any{
		"user_id": user.ID,
		"balance": user.GetBalance(),
	})

	// Get balance in cents from user entity
	balanceCents := user.Balance()

	userModel := model.User{
		ID:               user.ID,
		Balance:          balanceCents,
		CreatedAt:        user.CreatedAt,
		UpdatedAt:        user.UpdatedAt,
		TransactionCount: user.TransactionCount,
	}

	result := r.db.WithContext(ctx).Create(&userModel)

	if result.Error != nil {
		return r.handleDatabaseError("creating user", result.Error, user.ID)
	}

	r.logger.Info("User created successfully", map[string]any{
		"user_id": user.ID,
		"balance": user.GetBalance(),
	})
	return nil
}

// Update updates user information
func (r *UserRepository) Update(ctx context.Context, user *entity.User) error {
	r.logger.Debug("Updating user", map[string]any{
		"user_id":  user.ID,
		"balance":  user.GetBalance(),
		"tx_count": user.TransactionCount,
	})

	// Get balance in cents from user entity
	balanceCents := user.Balance()

	result := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", user.ID).
		Updates(map[string]interface{}{
			"balance":           balanceCents,
			"updated_at":        user.UpdatedAt,
			"transaction_count": user.TransactionCount,
		})

	if result.Error != nil {
		return r.handleDatabaseError("updating user", result.Error, user.ID)
	}

	if result.RowsAffected == 0 {
		r.logger.Warn("User not found during update", map[string]any{
			"user_id": user.ID,
		})
		return errs.ErrUserNotFound
	}

	r.logger.Info("User updated successfully", map[string]any{
		"user_id":  user.ID,
		"balance":  user.GetBalance(),
		"tx_count": user.TransactionCount,
	})
	return nil
}

// ProcessTransaction updates user balance atomically within a transaction
func (r *UserRepository) ProcessTransaction(ctx context.Context, userID uint64, balanceChange int64) (*entity.User, error) {
	r.logger.Debug("Processing transaction", map[string]any{
		"user_id":        userID,
		"balance_change": balanceChange,
		"operation_type": getOperationType(balanceChange),
		"change_amount":  entity.AmountInCentsToString(balanceChange),
	})

	var user *entity.User

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get and lock user record with FOR UPDATE clause
		// This ensures we get a strong exclusive row lock
		var userModel model.User
		result := tx.Set("gorm:query_option", "FOR UPDATE").First(&userModel, userID)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				r.logger.Warn("User not found during transaction processing", map[string]any{
					"user_id": userID,
				})
				return errs.ErrUserNotFound
			}
			r.logger.Error("Database error when locking user", map[string]any{
				"user_id": userID,
				"error":   result.Error.Error(),
			})
			return result.Error
		}

		// Calculate new balance
		newBalance := userModel.Balance + balanceChange

		// Check for negative balance
		if newBalance < 0 {
			r.logger.Warn("Insufficient balance for transaction", map[string]any{
				"user_id":          userID,
				"current_balance":  entity.AmountInCentsToString(userModel.Balance),
				"requested_change": entity.AmountInCentsToString(balanceChange),
				"operation_type":   "debit",
			})
			return errs.ErrInsufficientBalance
		}

		// Increment transaction count and update balance
		userModel.TransactionCount++
		userModel.Balance = newBalance
		userModel.UpdatedAt = r.timeProvider.Now()

		// Update user
		result = tx.Model(&userModel).Updates(map[string]interface{}{
			"balance":           userModel.Balance,
			"updated_at":        userModel.UpdatedAt,
			"transaction_count": userModel.TransactionCount,
		})

		if result.Error != nil {
			r.logger.Error("Failed to update user in transaction", map[string]any{
				"user_id": userID,
				"error":   result.Error.Error(),
			})
			return result.Error
		}

		// Convert model to entity
		var err error
		user, err = r.modelToEntity(&userModel)
		if err != nil {
			return err
		}

		r.logger.Debug("Transaction processing completed in DB transaction", map[string]any{
			"user_id":     userID,
			"new_balance": user.GetBalance(),
			"tx_count":    user.TransactionCount,
		})

		return nil
	})

	if err != nil {
		if errors.Is(err, errs.ErrUserNotFound) || errors.Is(err, errs.ErrInsufficientBalance) {
			// These errors are already logged above
			return nil, err
		}
		if r.errorClassifier.IsLockError(err) {
			r.logger.Warn("User is locked by another transaction", map[string]any{
				"user_id": userID,
				"error":   err.Error(),
			})
			return nil, errs.ErrUserLocked
		}
		r.logger.Error("Database error during transaction processing", map[string]any{
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("%w: %s", errs.ErrDatabaseConnection, err.Error())
	}

	r.logger.Info("Transaction processed successfully", map[string]any{
		"user_id":        userID,
		"balance_change": entity.AmountInCentsToString(balanceChange),
		"new_balance":    user.GetBalance(),
		"operation_type": getOperationType(balanceChange),
		"tx_count":       user.TransactionCount,
	})

	return user, nil
}
