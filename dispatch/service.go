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

	// ReassignAfterDecline attempts to reassign an order immediately after a decline by a courier.
	// It avoids offering to the declining courier and, if no alternative is available, marks
	// the order as no_nearby_driver and notifies the customer.
	ReassignAfterDecline(ctx context.Context, orderID uuid.UUID, declinedBy uuid.UUID) (*entity.Order, *entity.Courier, error)
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

	// Do not assign couriers for canceled/delivered orders.
	if ord.Status == entity.OrderCanceledByCustomer || ord.Status == entity.OrderCanceledByCourier || ord.Status == entity.OrderDelivered {
		return ord, nil, nil
	}

	// If already assigned and not canceled, keep the assignment (avoid re-assigning).
	if ord.AssignedCourier != nil {
		c, err := s.courier.GetCourierByID(ctx, *ord.AssignedCourier)
		if err != nil {
			return ord, nil, nil
		}
		return ord, c, nil
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
	_ = s.orders.RecordAssignmentAttempt(ctx, ord.ID, chosen.ID)

	updated, err := s.orders.GetOrderByID(ctx, ord.ID)
	if err != nil {
		return nil, nil, err
	}

	if s.hub != nil {
		// Notify courier with full order details
		cap := realtime.OrderAssignedPayload{
			OrderID:        updated.ID.String(),
			CustomerID:     updated.CustomerID.String(),
			PickupAddress:  updated.PickupAddress,
			PickupLat:      updated.PickupLat,
			PickupLng:      updated.PickupLng,
			DropoffAddress: updated.DropoffAddress,
			DropoffLat:     updated.DropoffLat,
			DropoffLng:     updated.DropoffLng,
			ReceiverPhone:  updated.ReceiverPhone,
		}
		_ = s.hub.Notify(chosen.ID.String(), "order.assigned", cap)

		// Also notify the customer that the order is assigned with order details
		pickupAddr := updated.PickupAddress
		dropoffAddr := updated.DropoffAddress
		receiverPhone := updated.ReceiverPhone
		payload := realtime.OrderStatusPayload{
			OrderID:        updated.ID.String(),
			Status:         string(entity.OrderAssigned),
			PickupAddress:  &pickupAddr,
			PickupLat:      updated.PickupLat,
			PickupLng:      updated.PickupLng,
			DropoffAddress: &dropoffAddr,
			DropoffLat:     updated.DropoffLat,
			DropoffLng:     updated.DropoffLng,
			ReceiverPhone:  &receiverPhone,
		}
		_ = s.hub.NotifyCustomer(updated.CustomerID.String(), "order.status", payload)
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
		// Proactively notify the currently assigned courier that their assignment timed out
		if s.hub != nil && prev != nil {
			_ = s.hub.Notify(prev.String(), "order.assignment_timed_out", realtime.AssignmentPayload{OrderID: o.ID.String(), CustomerID: o.CustomerID.String()})
		}
		// clear current assignment
		if err := s.orders.ClearAssignment(ctx, o.ID); err != nil {
			continue
		}
		// try to find a new courier excluding the previous one if any
		reassigned, courier, err := s.findAndAssignExcluding(ctx, o.ID, prev)
		if err == nil && courier != nil {
			// Notify the previously assigned courier that the order was reassigned away
			if s.hub != nil && prev != nil {
				_ = s.hub.Notify(prev.String(), "order.reassigned_away", realtime.AssignmentPayload{OrderID: reassigned.ID.String(), CustomerID: reassigned.CustomerID.String()})
			}
			count++
			continue
		}
		// No courier available -> atomically clear assignment and mark as no_nearby_driver
		if err := s.orders.MarkNoNearbyDriver(ctx, o.ID); err == nil {
			if s.hub != nil {
				// Notify customer
				payload := realtime.OrderStatusPayload{OrderID: o.ID.String(), Status: string(entity.OrderNoNearbyDriver)}
				_ = s.hub.NotifyCustomer(o.CustomerID.String(), "order.status", payload)
				// Notify previously assigned courier that the job is no longer active
				if prev != nil {
					_ = s.hub.Notify(prev.String(), "order.no_nearby_driver", realtime.AssignmentPayload{OrderID: o.ID.String(), CustomerID: o.CustomerID.String()})
				}
			}
		}
	}
	return count, nil
}

// ReassignAfterDecline attempts to reassign an order after a courier declines it.
// If no available alternative courier is found, mark as no_nearby_driver and notify the customer.
func (s *service) ReassignAfterDecline(ctx context.Context, orderID uuid.UUID, declinedBy uuid.UUID) (*entity.Order, *entity.Courier, error) {
	// Try to find a new courier excluding the declining one
	reassigned, chosen, err := s.findAndAssignExcluding(ctx, orderID, &declinedBy)
	if err != nil {
		return nil, nil, err
	}
	if chosen != nil {
		// Notifications are handled inside findAndAssignExcluding
		return reassigned, chosen, nil
	}
	// No alternative courier -> mark no_nearby_driver and notify
	if err := s.orders.MarkNoNearbyDriver(ctx, orderID); err != nil {
		return nil, nil, err
	}
	updated, err := s.orders.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, nil, err
	}
	if s.hub != nil {
		payload := realtime.OrderStatusPayload{OrderID: updated.ID.String(), Status: string(entity.OrderNoNearbyDriver)}
		_ = s.hub.NotifyCustomer(updated.CustomerID.String(), "order.status", payload)
	}
	return updated, nil, nil
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
	tried, _ := s.orders.ListTriedCouriers(ctx, ord.ID)
	var chosen *entity.Courier
	for i := range list {
		c := list[i]
		if exclude != nil && c.ID == *exclude {
			continue
		}
		if _, seen := tried[c.ID]; seen {
			// Skip couriers already tried for this order
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
	_ = s.orders.RecordAssignmentAttempt(ctx, ord.ID, chosen.ID)

	updated, err := s.orders.GetOrderByID(ctx, ord.ID)
	if err != nil {
		return nil, nil, err
	}

	if s.hub != nil {
		// Notify courier with full order details
		cap := realtime.OrderAssignedPayload{
			OrderID:        updated.ID.String(),
			CustomerID:     updated.CustomerID.String(),
			PickupAddress:  updated.PickupAddress,
			PickupLat:      updated.PickupLat,
			PickupLng:      updated.PickupLng,
			DropoffAddress: updated.DropoffAddress,
			DropoffLat:     updated.DropoffLat,
			DropoffLng:     updated.DropoffLng,
			ReceiverPhone:  updated.ReceiverPhone,
		}
		_ = s.hub.Notify(chosen.ID.String(), "order.assigned", cap)

		// Also notify the customer that the order is (re)assigned with order details
		pickupAddr := updated.PickupAddress
		dropoffAddr := updated.DropoffAddress
		receiverPhone := updated.ReceiverPhone
		payload := realtime.OrderStatusPayload{
			OrderID:        updated.ID.String(),
			Status:         string(entity.OrderAssigned),
			PickupAddress:  &pickupAddr,
			PickupLat:      updated.PickupLat,
			PickupLng:      updated.PickupLng,
			DropoffAddress: &dropoffAddr,
			DropoffLat:     updated.DropoffLat,
			DropoffLng:     updated.DropoffLng,
			ReceiverPhone:  &receiverPhone,
		}
		_ = s.hub.NotifyCustomer(updated.CustomerID.String(), "order.status", payload)
	}
	return updated, chosen, nil
}
