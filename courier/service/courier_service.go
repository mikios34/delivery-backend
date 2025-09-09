package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/mikios34/delivery-backend/courier"
	"github.com/mikios34/delivery-backend/entity"
)

// courierService implements CourierService.
type courierService struct {
	repo courier.CourierRepository
}

// NewCourierService constructs a CourierService backed by the provided repository.
func NewCourierService(repo courier.CourierRepository) courier.CourierService {
	return &courierService{repo: repo}
}

func (s *courierService) ListGuarantyOptions(ctx context.Context) ([]entity.GuarantyOption, error) {
	return s.repo.ListGuarantyOptions(ctx)
}

// RegisterCourier performs validations and persists User, Courier and GuarantyPayment.
// This implementation is sequential (calls repository methods). If you need atomic DB transactions,
// we can extend the repository to expose transaction support and update this method accordingly.
func (s *courierService) RegisterCourier(ctx context.Context, req courier.RegisterCourierRequest) (*entity.Courier, error) {
	// check phone uniqueness
	exists, err := s.repo.PhoneExists(ctx, req.Phone)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("courier with this phone already exists")
	}

	// create user
	u := &entity.User{
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Phone:         req.Phone,
		Role:          "courier",
		PhoneVerified: true,
	}
	if req.FirebaseUID != "" {
		uid := req.FirebaseUID
		u.FirebaseUID = &uid
	}
	createdUser, err := s.repo.StoreUser(ctx, u)
	if err != nil {
		return nil, err
	}

	// confirm guaranty option exists & active
	opts, err := s.repo.ListGuarantyOptions(ctx)
	if err != nil {
		return nil, err
	}
	var selected *entity.GuarantyOption
	for _, o := range opts {
		if o.ID == req.GuarantyOptionID {
			tmp := o
			selected = &tmp
			break
		}
	}
	if selected == nil {
		return nil, errors.New("selected guaranty option not found or inactive")
	}

	// create courier
	c := &entity.Courier{
		UserID:           createdUser.ID,
		HasVehicle:       req.HasVehicle,
		PrimaryVehicle:   req.PrimaryVehicle,
		VehicleDetails:   req.VehicleDetails,
		GuarantyOptionID: &selected.ID,
		GuarantyPaid:     false,
		Active:           true,
	}
	createdCourier, err := s.repo.StoreCourier(ctx, c)
	if err != nil {
		return nil, err
	}

	// create guaranty payment placeholder
	gp := &entity.GuarantyPayment{
		CourierID:        createdCourier.ID,
		GuarantyOptionID: selected.ID,
		AmountCents:      selected.AmountCents,
		// Currency:         selected.Currency,
		Paid: false,
	}
	if _, err := s.repo.CreateGuarantyPayment(ctx, gp); err != nil {
		return nil, err
	}

	return createdCourier, nil
}

func (s *courierService) SetAvailability(ctx context.Context, courierID uuid.UUID, available bool) error {
	return s.repo.UpdateAvailability(ctx, courierID, available)
}

func (s *courierService) UpdateLocation(ctx context.Context, courierID uuid.UUID, lat, lng *float64) error {
	return s.repo.UpdateLocation(ctx, courierID, lat, lng)
}
