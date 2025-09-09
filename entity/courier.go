package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VehicleType enumerates courier vehicle capabilities.
type VehicleType string

const (
	VehicleBike    VehicleType = "bike"
	VehicleMotor   VehicleType = "motorbike"
	VehicleCar     VehicleType = "car"
	VehicleBicycle VehicleType = "bicycle"
	VehicleTaxi    VehicleType = "taxi"
	VehicleBus     VehicleType = "bus"
	VehicleTrain   VehicleType = "train"
	VehicleWalker  VehicleType = "walker"
	VehicleOther   VehicleType = "other"
)

// User is a minimal auth profile used for couriers (keeps parity with existing entity.User if present).
// If your repo already has an entity.User, you can remove/reuse it — this is intentionally minimal.
type User struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	FirstName string    `json:"first_name" gorm:"type:text;not null"`
	LastName  string    `json:"last_name" gorm:"type:text;not null"`
	// Email         *string        `json:"email,omitempty" gorm:"type:text;uniqueIndex;default:null"`
	Phone         string         `json:"phone" gorm:"type:text;index;not null"`
	FirebaseUID   *string        `json:"firebase_uid,omitempty" gorm:"type:text;uniqueIndex;default:null"`
	PhoneVerified bool           `json:"phone_verified" gorm:"default:false;index"`
	Role          string         `json:"role" gorm:"type:text;index;not null"` // e.g., "courier"
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
	// Relation: Courier (one-to-one)
	Courier Courier `json:"courier,omitempty" gorm:"constraint:OnDelete:CASCADE"`
}

// GuarantyOption represents an admin-configurable guaranty amount option shown in the courier signup dropdown.
type GuarantyOption struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Label       string    `json:"label" gorm:"type:text;not null"` // e.g., "Small: $50"
	AmountCents int64     `json:"amount_cents" gorm:"type:bigint;not null"`
	// Currency    string         `json:"currency" gorm:"type:text;default:'USD'"`
	Active    bool           `json:"active" gorm:"default:true;index"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// GuarantyPayment records the payment/commitment for the guaranty.
type GuarantyPayment struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	CourierID        uuid.UUID      `json:"courier_id" gorm:"type:uuid;index;not null"`
	GuarantyOptionID uuid.UUID      `json:"guaranty_option_id" gorm:"type:uuid;index;not null"`
	AmountCents      int64          `json:"amount_cents" gorm:"type:bigint;not null"`
	Currency         string         `json:"currency" gorm:"type:text;default:'USD'"`
	Provider         string         `json:"provider,omitempty" gorm:"type:text"`     // e.g., "stripe"
	ProviderRef      string         `json:"provider_ref,omitempty" gorm:"type:text"` // payment id / link
	Paid             bool           `json:"paid" gorm:"default:false;index"`
	PaidAt           *time.Time     `json:"paid_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

// Courier stores courier-specific data collected at registration and afterwards.
type Courier struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID            uuid.UUID      `json:"user_id" gorm:"type:uuid;index;not null"`
	HasVehicle        bool           `json:"has_vehicle" gorm:"default:false;index"`
	PrimaryVehicle    VehicleType    `json:"primary_vehicle" gorm:"type:text;index"`
	VehicleDetails    string         `json:"vehicle_details,omitempty" gorm:"type:text"`
	GuarantyOptionID  *uuid.UUID     `json:"guaranty_option_id,omitempty" gorm:"type:uuid;index;default:null"`
	GuarantyPaid      bool           `json:"guaranty_paid" gorm:"default:false;index"`
	Active            bool           `json:"active" gorm:"default:true;index"`
	Available         bool           `json:"available" gorm:"default:false;index"`
	Latitude          *float64       `json:"latitude,omitempty" gorm:"type:double precision"`
	Longitude         *float64       `json:"longitude,omitempty" gorm:"type:double precision"`
	LocationUpdatedAt *time.Time     `json:"location_updated_at,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
	// Relations
	// GuarantyPayments []GuarantyPayment `json:"guaranty_payments,omitempty" gorm:"foreignKey:CourierID;constraint:OnDelete:CASCADE"`
}
