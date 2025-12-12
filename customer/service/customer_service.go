package service

import (
	"context"
	"errors"

	customerpkg "github.com/mikios34/delivery-backend/customer"
	"github.com/mikios34/delivery-backend/entity"
)

// customerService implements CustomerService.
type customerService struct {
	repo customerpkg.CustomerRepository
}

// NewCustomerService constructs a CustomerService backed by the provided repository.
func NewCustomerService(repo customerpkg.CustomerRepository) customerpkg.CustomerService {
	return &customerService{repo: repo}
}

// RegisterCustomer creates a base User with role "customer" and a Customer profile.
func (s *customerService) RegisterCustomer(ctx context.Context, req customerpkg.RegisterCustomerRequest) (*entity.Customer, error) {
	// check phone uniqueness among customers
	exists, err := s.repo.PhoneExists(ctx, req.Phone)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("customer with this phone already exists")
	}

	// create user
	u := &entity.User{
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Phone:         req.Phone,
		Role:          "customer",
		PhoneVerified: true,
	}
	if req.ProfilePicture != nil {
		u.ProfilePicture = req.ProfilePicture
	}
	if req.FirebaseUID != "" {
		uid := req.FirebaseUID
		u.FirebaseUID = &uid
	}
	createdUser, err := s.repo.StoreUser(ctx, u)
	if err != nil {
		return nil, err
	}

	// create customer profile
	c := &entity.Customer{UserID: createdUser.ID, Active: true}
	createdCustomer, err := s.repo.StoreCustomer(ctx, c)
	if err != nil {
		return nil, err
	}
	return createdCustomer, nil
}
