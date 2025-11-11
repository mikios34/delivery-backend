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

	// GetActiveOrderForCustomer returns the most recently updated active order for a customer
	// Active means status NOT IN (no_nearby_driver, delivered)
	GetActiveOrderForCustomer(ctx context.Context, customerID uuid.UUID) (*entity.Order, error)

	// ListActiveOrdersForCustomer returns all active orders for a customer ordered by updated_at DESC.
	// Active means status NOT IN (no_nearby_driver, delivered)
	ListActiveOrdersForCustomer(ctx context.Context, customerID uuid.UUID) ([]entity.Order, error)

	// GetActiveOrderForCourier returns the most recently updated active order assigned to a courier
	// Active means status NOT IN (no_nearby_driver, delivered)
	GetActiveOrderForCourier(ctx context.Context, courierID uuid.UUID) (*entity.Order, error)

	// ListDeliveredOrdersForCourier returns orders with status=delivered for the given courier
	// Results should be ordered by updated_at desc so newest deliveries appear first.
	// Supports pagination via limit/offset. A caller may pass 0 for limit to use DB defaults.
	ListDeliveredOrdersForCourier(ctx context.Context, courierID uuid.UUID, limit, offset int) ([]entity.Order, error)

	// CountDeliveredOrdersForCourier returns total delivered orders for the given courier (for pagination metadata)
	CountDeliveredOrdersForCourier(ctx context.Context, courierID uuid.UUID) (int64, error)

	// Pricing configs (vehicle types with pricing)
	ListActiveVehicleTypes(ctx context.Context) ([]entity.VehicleTypeConfig, error)
}
