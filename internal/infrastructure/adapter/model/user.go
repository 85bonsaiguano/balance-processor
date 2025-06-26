package model

import (
	"time"
)

// User represents the database model for users
type User struct {
	ID               uint64    `gorm:"primaryKey"`
	Balance          int64     `gorm:"not null"` // Balance in cents
	CreatedAt        time.Time `gorm:"not null"`
	UpdatedAt        time.Time `gorm:"not null"`
	TransactionCount uint64    `gorm:"default:0"`
}

// TableName specifies the table name for User
func (User) TableName() string {
	return "users"
}
