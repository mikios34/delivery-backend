package order

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/mikios34/delivery-backend/entity"
)

// Repository defines DB operations for orders and order types.
type Repository interface {
	CreateOrder(ctx context.Context, o *entity.Order) (*entity.Order, error)
	GetOrderByID(ctx context.Context, id uuid.UUID) (*entity.Order, error)
	UpdateOrderStatus(ctx context.Context, id uuid.UUID, status entity.OrderStatus) error
	AssignCourier(ctx context.Context, id uuid.UUID, courierID uuid.UUID) error
	ClearAssignment(ctx context.Context, id uuid.UUID) error
	ListAssignedOlderThan(ctx context.Context, cutoff time.Time) ([]entity.Order, error)

	ListOrderTypes(ctx context.Context) ([]entity.OrderType, error)
	CreateOrderType(ctx context.Context, t *entity.OrderType) (*entity.OrderType, error)
}
