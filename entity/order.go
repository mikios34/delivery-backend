package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrderType is a configurable type for orders (e.g., document, electronics, goods).
type OrderType struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name      string         `json:"name" gorm:"type:text;uniqueIndex;not null"`
	Active    bool           `json:"active" gorm:"default:true;index"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// OrderStatus enumerates the lifecycle of an order.
type OrderStatus string

const (
	OrderPending   OrderStatus = "pending"   // created, awaiting dispatch
	OrderAssigned  OrderStatus = "assigned"  // assigned to a courier, awaiting accept/decline
	OrderAccepted  OrderStatus = "accepted"  // courier accepted
	OrderDeclined  OrderStatus = "declined"  // courier declined -> will be redispatched
	OrderArrived   OrderStatus = "arrived"   // courier arrived at pickup
	OrderPickedUp  OrderStatus = "picked_up" // package picked up
	OrderDelivered OrderStatus = "delivered" // delivered
	// No nearby driver found after timeout-based reassignment attempts
	OrderNoNearbyDriver OrderStatus = "no_nearby_driver"
	// Cancellation states
	OrderCanceledByCustomer OrderStatus = "canceled_by_customer" // customer canceled before completion
	OrderCanceledByCourier  OrderStatus = "canceled_by_courier"  // courier canceled (e.g. after accept but before completion)
)

// Order captures a delivery request by a customer.
type Order struct {
	ID              uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	CustomerID      uuid.UUID  `json:"customer_id" gorm:"type:uuid;index;not null"`
	AssignedCourier *uuid.UUID `json:"assigned_courier,omitempty" gorm:"type:uuid;index;default:null"`
	TypeID          uuid.UUID  `json:"type_id" gorm:"type:uuid;index;not null"`
	// VehicleTypeID is the selected pricing vehicle type (e.g., bike, motorbike, car). Nullable for legacy rows.
	VehicleTypeID  uuid.UUID `json:"vehicle_type_id" gorm:"type:uuid;index"`
	ReceiverPhone  string    `json:"receiver_phone" gorm:"type:text;not null"`
	PickupAddress  string    `json:"pickup_address" gorm:"type:text;not null"`
	PickupLat      *float64  `json:"pickup_lat,omitempty" gorm:"type:double precision"`
	PickupLng      *float64  `json:"pickup_lng,omitempty" gorm:"type:double precision"`
	DropoffAddress string    `json:"dropoff_address" gorm:"type:text;not null"`
	DropoffLat     *float64  `json:"dropoff_lat,omitempty" gorm:"type:double precision"`
	DropoffLng     *float64  `json:"dropoff_lng,omitempty" gorm:"type:double precision"`
	// EstimatedPriceCents stores the pre-quote price used at creation (minor units)
	EstimatedPriceCents int64          `json:"estimated_price_cents" gorm:"type:bigint;not null;default:0"`
	Status              OrderStatus    `json:"status" gorm:"type:text;index;not null;default:'pending'"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index"`
}

// OrderAssignmentAttempt records a courier that has been tried for an order.
type OrderAssignmentAttempt struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	OrderID   uuid.UUID      `json:"order_id" gorm:"type:uuid;index;not null"`
	CourierID uuid.UUID      `json:"courier_id" gorm:"type:uuid;index;not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
