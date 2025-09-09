package service

import (
	"context"
	"errors"

	adminpkg "github.com/mikios34/delivery-backend/admin"
	"github.com/mikios34/delivery-backend/entity"
)

// adminService implements AdminService.
type adminService struct {
	repo adminpkg.AdminRepository
}

// NewAdminService constructs an AdminService backed by the provided repository.
func NewAdminService(repo adminpkg.AdminRepository) adminpkg.AdminService {
	return &adminService{repo: repo}
}

// RegisterAdmin creates a base User with role "admin" and an Admin profile.
func (s *adminService) RegisterAdmin(ctx context.Context, req adminpkg.RegisterAdminRequest) (*entity.Admin, error) {
	// check phone uniqueness among admins
	exists, err := s.repo.PhoneExists(ctx, req.Phone)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("admin with this phone already exists")
	}

	// create user
	u := &entity.User{
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Phone:         req.Phone,
		Role:          "admin",
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

	// create admin profile
	a := &entity.Admin{UserID: createdUser.ID, Active: true}
	createdAdmin, err := s.repo.StoreAdmin(ctx, a)
	if err != nil {
		return nil, err
	}
	return createdAdmin, nil
}
