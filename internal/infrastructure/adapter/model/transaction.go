package model

import (
	"time"
)

// Transaction represents the database model for transactions
type Transaction struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement"`
	UserID        uint64    `gorm:"not null;index"`
	TransactionID string    `gorm:"uniqueIndex;not null;size:255"`
	SourceType    string    `gorm:"not null;size:50"`
	State         string    `gorm:"not null;size:50"`
	Amount        string    `gorm:"not null;size:50"`
	AmountInCents int64     `gorm:"not null"`
	CreatedAt     time.Time `gorm:"not null"`
	ProcessedAt   *time.Time
	ResultBalance string `gorm:"size:50"`
	Status        string `gorm:"not null;size:50"`
	ErrorMessage  string `gorm:"type:text"`

	// Define relationships
	User User `gorm:"foreignKey:UserID;references:ID"`
}

// TableName specifies the table name for Transaction
func (Transaction) TableName() string {
	return "transactions"
}
