package dispatch

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/mikios34/delivery-backend/courier"
	"github.com/mikios34/delivery-backend/entity"
	"github.com/mikios34/delivery-backend/order"
	"github.com/mikios34/delivery-backend/realtime"
)

// Service defines dispatching operations.
type Service interface {
	// FindAndAssign picks an available courier and assigns the order (sets status to assigned).
	FindAndAssign(ctx context.Context, orderID uuid.UUID) (*entity.Order, *entity.Courier, error)
	// Dispatch is an alias to FindAndAssign for compatibility with handlers.
	Dispatch(ctx context.Context, orderID uuid.UUID) (*entity.Order, *entity.Courier, error)
	// ReassignTimedOut looks for orders stuck in assigned beyond cutoff and reassigns.
	// If no alternative courier is found, the order is marked as no_nearby_driver.
	ReassignTimedOut(ctx context.Context, cutoff time.Time) (int, error)
}

type service struct {
	orders  order.Repository
	courier courier.CourierRepository
	hub     *realtime.Hub
}

func New(orders order.Repository, courier courier.CourierRepository, hub *realtime.Hub) Service {
	return &service{orders: orders, courier: courier, hub: hub}
}

// naive selection: take first available courier with a location (placeholder).
func (s *service) FindAndAssign(ctx context.Context, orderID uuid.UUID) (*entity.Order, *entity.Courier, error) {
	ord, err := s.orders.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, nil, err
	}

	// Use pickup coordinates if present; otherwise centerless lookup
	centerLat, centerLng := 0.0, 0.0
	if ord.PickupLat != nil {
		centerLat = *ord.PickupLat
	}
	if ord.PickupLng != nil {
		centerLng = *ord.PickupLng
	}

	list, err := s.courier.ListAvailableCouriersNear(ctx, centerLat, centerLng, 10.0, 50)
	if err != nil {
		return nil, nil, err
	}
	if len(list) == 0 {
		return ord, nil, nil
	}

	chosen := list[0]
	if err := s.orders.AssignCourier(ctx, ord.ID, chosen.ID); err != nil {
		return nil, nil, err
	}
	if err := s.orders.UpdateOrderStatus(ctx, ord.ID, entity.OrderAssigned); err != nil {
		return nil, nil, err
	}

	updated, err := s.orders.GetOrderByID(ctx, ord.ID)
	if err != nil {
		return nil, nil, err
	}

	if s.hub != nil {
		_ = s.hub.Notify(chosen.ID.String(), "order.assigned", realtime.AssignmentPayload{OrderID: updated.ID.String(), CustomerID: updated.CustomerID.String()})
	}
	return updated, &chosen, nil
}

func (s *service) Dispatch(ctx context.Context, orderID uuid.UUID) (*entity.Order, *entity.Courier, error) {
	return s.FindAndAssign(ctx, orderID)
}

func (s *service) ReassignTimedOut(ctx context.Context, cutoff time.Time) (int, error) {
	list, err := s.orders.ListAssignedOlderThan(ctx, cutoff)
	if err != nil {
		return 0, err
	}
	count := 0
	for i := range list {
		o := &list[i]
		// clear current assignment
		if err := s.orders.ClearAssignment(ctx, o.ID); err != nil {
			continue
		}
		if updated, courier, err := s.FindAndAssign(ctx, o.ID); err == nil && courier != nil {
			count++
			_ = updated
			continue
		}
		// No courier available -> mark as no_nearby_driver
		_ = s.orders.UpdateOrderStatus(ctx, o.ID, entity.OrderNoNearbyDriver)
	}
	return count, nil
}
