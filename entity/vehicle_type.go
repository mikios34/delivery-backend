package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VehicleTypeConfig defines pricing configuration per vehicle type for fare estimation.
// Table name intentionally set to "vehicle_types" to match requested DB table.
type VehicleTypeConfig struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Code        string         `json:"code" gorm:"type:text;uniqueIndex;not null"` // e.g., "bike", "motorbike", "car"
	Name        string         `json:"name" gorm:"type:text;not null"`             // Display label (e.g., "Bike")
	Active      bool           `json:"active" gorm:"default:true;index"`
	BaseFare    float64        `json:"base_fare" gorm:"type:double precision;default:0"`
	PerKm       float64        `json:"per_km" gorm:"type:double precision;default:0"`
	PerMinute   float64        `json:"per_minute" gorm:"type:double precision;default:0"`
	AvgSpeedKmh float64        `json:"avg_speed_kmh" gorm:"type:double precision;default:30"`
	MinimumFare float64        `json:"minimum_fare" gorm:"type:double precision;default:0"`
	BookingFee  float64        `json:"booking_fee" gorm:"type:double precision;default:0"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (VehicleTypeConfig) TableName() string { return "vehicle_types" }
