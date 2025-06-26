package model

import (
	"time"

	"gorm.io/gorm"
)

// MigrationVersion represents a database migration version
type MigrationVersion struct {
	ID        uint           `gorm:"primaryKey;autoIncrement"`
	Version   string         `gorm:"type:varchar(20);not null;index"`
	AppliedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	Details   string         `gorm:"type:text;null"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for the migration version model
func (MigrationVersion) TableName() string {
	return "migration_versions"
}
