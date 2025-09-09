package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Customer represents a customer profile linked to a base User.
// Minimal for now; extend with addresses, preferences, etc. later.
type Customer struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID    uuid.UUID      `json:"user_id" gorm:"type:uuid;index;not null"`
	Active    bool           `json:"active" gorm:"default:true;index"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
