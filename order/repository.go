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
	CountAssignedOrders(ctx context.Context) (int64, error)
	// MarkNoNearbyDriver clears assignment and sets status to no_nearby_driver atomically
	MarkNoNearbyDriver(ctx context.Context, id uuid.UUID) error

	// Assignment attempts tracking to avoid reassigning the same courier
	RecordAssignmentAttempt(ctx context.Context, orderID, courierID uuid.UUID) error
	ListTriedCouriers(ctx context.Context, orderID uuid.UUID) (map[uuid.UUID]struct{}, error)

	ListOrderTypes(ctx context.Context) ([]entity.OrderType, error)
	CreateOrderType(ctx context.Context, t *entity.OrderType) (*entity.OrderType, error)
}
