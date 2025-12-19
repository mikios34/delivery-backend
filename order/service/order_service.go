package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mikios34/delivery-backend/entity"
	orderpkg "github.com/mikios34/delivery-backend/order"
)

type orderService struct {
	repo orderpkg.Repository
}

func NewOrderService(repo orderpkg.Repository) orderpkg.Service { return &orderService{repo: repo} }

func (s *orderService) CreateOrder(ctx context.Context, req orderpkg.CreateOrderRequest) (*entity.Order, error) {
	o := &entity.Order{
		CustomerID:          req.CustomerID,
		TypeID:              req.TypeID,
		VehicleTypeID:       req.VehicleTypeID,
		ReceiverPhone:       req.ReceiverPhone,
		PickupAddress:       req.PickupAddress,
		PickupLat:           req.PickupLat,
		PickupLng:           req.PickupLng,
		DropoffAddress:      req.DropoffAddress,
		DropoffLat:          req.DropoffLat,
		DropoffLng:          req.DropoffLng,
		EstimatedPriceCents: req.EstimatedPriceCents,
		Status:              entity.OrderPending,
	}
	return s.repo.CreateOrder(ctx, o)
}

func (s *orderService) ListOrderTypes(ctx context.Context) ([]entity.OrderType, error) {
	return s.repo.ListOrderTypes(ctx)
}

func (s *orderService) UpdateStatus(ctx context.Context, orderID uuid.UUID, newStatus entity.OrderStatus, byCourierID *uuid.UUID) (*entity.Order, error) {
	ord, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if byCourierID != nil && ord.AssignedCourier != nil && *byCourierID != *ord.AssignedCourier {
		return nil, fmt.Errorf("forbidden: not assigned courier")
	}
	if newStatus == entity.OrderDeclined {
		if err := s.repo.UpdateOrderStatus(ctx, orderID, newStatus); err != nil {
			return nil, err
		}
		if err := s.repo.ClearAssignment(ctx, orderID); err != nil {
			return nil, err
		}
		return s.repo.GetOrderByID(ctx, orderID)
	}
	if err := s.repo.UpdateOrderStatus(ctx, orderID, newStatus); err != nil {
		return nil, err
	}
	return s.repo.GetOrderByID(ctx, orderID)
}

// CancelByCustomer sets status to canceled_by_customer if order is not already delivered.
func (s *orderService) CancelByCustomer(ctx context.Context, orderID uuid.UUID) (*entity.Order, error) {
	ord, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if ord.Status == entity.OrderPickedUp || ord.Status == entity.OrderDelivered {
		return nil, fmt.Errorf("cannot cancel order after pickup or delivery")
	}
	if ord.Status == entity.OrderCanceledByCustomer || ord.Status == entity.OrderCanceledByCourier {
		return ord, nil
	}
	if err := s.repo.UpdateOrderStatus(ctx, orderID, entity.OrderCanceledByCustomer); err != nil {
		return nil, err
	}
	return s.repo.GetOrderByID(ctx, orderID)
}

// CancelByCourier sets status to canceled_by_courier if courier matches and not delivered.
func (s *orderService) CancelByCourier(ctx context.Context, orderID uuid.UUID, courierID uuid.UUID) (*entity.Order, error) {
	ord, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if ord.Status == entity.OrderPickedUp || ord.Status == entity.OrderDelivered {
		return nil, fmt.Errorf("cannot cancel order after pickup or delivery")
	}
	if ord.AssignedCourier == nil || *ord.AssignedCourier != courierID {
		return nil, fmt.Errorf("forbidden: not assigned courier")
	}
	if ord.Status == entity.OrderCanceledByCustomer || ord.Status == entity.OrderCanceledByCourier {
		return ord, nil
	}
	if err := s.repo.UpdateOrderStatus(ctx, orderID, entity.OrderCanceledByCourier); err != nil {
		return nil, err
	}
	return s.repo.GetOrderByID(ctx, orderID)
}
