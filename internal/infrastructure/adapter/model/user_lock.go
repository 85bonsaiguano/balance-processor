package model

import (
	"time"
)

// UserLock represents a lock on a user record for transaction processing
type UserLock struct {
	UserID    uint64    `gorm:"primaryKey;not null"`
	LockedAt  time.Time `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"` // Standard GORM timestamp
	UpdatedAt time.Time `gorm:"not null"` // Standard GORM timestamp
}

// TableName specifies the table name for UserLock
func (UserLock) TableName() string {
	return "user_locks"
}
