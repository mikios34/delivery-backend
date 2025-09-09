package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Admin represents an admin profile linked to a base User.
type Admin struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID    uuid.UUID      `json:"user_id" gorm:"type:uuid;index;not null"`
	Active    bool           `json:"active" gorm:"default:true;index"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
