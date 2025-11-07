package order

import (
	"context"

	"github.com/google/uuid"
	"github.com/mikios34/delivery-backend/entity"
)

type CreateOrderRequest struct {
	CustomerID          uuid.UUID
	TypeID              uuid.UUID
	VehicleTypeID       uuid.UUID
	ReceiverPhone       string
	PickupAddress       string
	PickupLat           *float64
	PickupLng           *float64
	DropoffAddress      string
	DropoffLat          *float64
	DropoffLng          *float64
	EstimatedPriceCents int64
}

type Service interface {
	CreateOrder(ctx context.Context, req CreateOrderRequest) (*entity.Order, error)
	ListOrderTypes(ctx context.Context) ([]entity.OrderType, error)
	UpdateStatus(ctx context.Context, orderID uuid.UUID, newStatus entity.OrderStatus, byCourierID *uuid.UUID) (*entity.Order, error)
}
