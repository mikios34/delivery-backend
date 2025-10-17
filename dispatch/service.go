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

	// Use pickup coordinates if present; otherwise use a large radius from (0,0)
	centerLat, centerLng := 0.0, 0.0
	radiusKm := 10.0
	if ord.PickupLat != nil && ord.PickupLng != nil {
		centerLat = *ord.PickupLat
		centerLng = *ord.PickupLng
		radiusKm = 10.0
	} else {
		// No pickup coordinates provided; search globally with a large radius
		radiusKm = 20000.0
	}

	list, err := s.courier.ListAvailableCouriersNear(ctx, centerLat, centerLng, radiusKm, 50)
	if err != nil {
		return nil, nil, err
	}
	if len(list) == 0 {
		// No available couriers right now -> mark as no_nearby_driver.
		if err := s.orders.MarkNoNearbyDriver(ctx, ord.ID); err != nil {
			return ord, nil, err
		}
		updated, err := s.orders.GetOrderByID(ctx, ord.ID)
		if err != nil {
			return nil, nil, err
		}
		// Notify customer via hub if available
		if s.hub != nil {
			payload := realtime.OrderStatusPayload{OrderID: updated.ID.String(), Status: string(updated.Status)}
			_ = s.hub.NotifyCustomer(updated.CustomerID.String(), "order.status", payload)
			_ = s.hub.NotifyCustomer(updated.CustomerID.String(), "order.no_nearby_driver", payload)
		}
		return updated, nil, nil
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
		// Also notify the customer that the order is assigned
		payload := realtime.OrderStatusPayload{OrderID: updated.ID.String(), Status: string(entity.OrderAssigned)}
		_ = s.hub.NotifyCustomer(updated.CustomerID.String(), "order.status", payload)
		_ = s.hub.NotifyCustomer(updated.CustomerID.String(), "order.assigned", payload)
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
		// remember the previous assigned courier to avoid reassigning to the same one
		var prev *uuid.UUID
		if o.AssignedCourier != nil {
			cid := *o.AssignedCourier
			prev = &cid
		}
		// clear current assignment
		if err := s.orders.ClearAssignment(ctx, o.ID); err != nil {
			continue
		}
		// try to find a new courier excluding the previous one if any
		reassigned, courier, err := s.findAndAssignExcluding(ctx, o.ID, prev)
		if err == nil && courier != nil {
			_ = reassigned
			count++
			continue
		}
		// No courier available -> atomically clear assignment and mark as no_nearby_driver
		if err := s.orders.MarkNoNearbyDriver(ctx, o.ID); err == nil && s.hub != nil {
			payload := realtime.OrderStatusPayload{OrderID: o.ID.String(), Status: string(entity.OrderNoNearbyDriver)}
			_ = s.hub.NotifyCustomer(o.CustomerID.String(), "order.status", payload)
			_ = s.hub.NotifyCustomer(o.CustomerID.String(), "order.no_nearby_driver", payload)
		}
	}
	return count, nil
}

// findAndAssignExcluding is like FindAndAssign but avoids selecting the excluded courier when provided.
func (s *service) findAndAssignExcluding(ctx context.Context, orderID uuid.UUID, exclude *uuid.UUID) (*entity.Order, *entity.Courier, error) {
	ord, err := s.orders.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, nil, err
	}

	centerLat, centerLng := 0.0, 0.0
	radiusKm := 10.0
	if ord.PickupLat != nil && ord.PickupLng != nil {
		centerLat = *ord.PickupLat
		centerLng = *ord.PickupLng
		radiusKm = 10.0
	} else {
		// No pickup coordinates provided; search globally with a large radius
		radiusKm = 20000.0
	}

	list, err := s.courier.ListAvailableCouriersNear(ctx, centerLat, centerLng, radiusKm, 50)
	if err != nil {
		return nil, nil, err
	}
	if len(list) == 0 {
		return ord, nil, nil
	}
	// pick the first courier not equal to exclude
	var chosen *entity.Courier
	for i := range list {
		c := list[i]
		if exclude != nil && c.ID == *exclude {
			continue
		}
		chosen = &c
		break
	}
	if chosen == nil {
		// only available candidate was the excluded one
		return ord, nil, nil
	}

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
	return updated, chosen, nil
}
