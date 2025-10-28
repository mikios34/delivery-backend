package service

import (
	"context"
	"errors"
	"os"
	"time"

	authpkg "github.com/mikios34/delivery-backend/auth"
	"github.com/mikios34/delivery-backend/entity"
)

type authService struct {
	repo authpkg.Repository
}

func NewAuthService(repo authpkg.Repository) authpkg.Service {
	return &authService{repo: repo}
}

func (s *authService) Login(ctx context.Context, req authpkg.LoginRequest) (*authpkg.Principal, error) {
	if req.FirebaseUID == "" && req.Phone == "" {
		return nil, errors.New("either firebase_uid or phone is required")
	}

	var user *entity.User
	var err error
	if req.FirebaseUID != "" {
		user, err = s.repo.GetUserByFirebaseUID(ctx, req.FirebaseUID)
	} else {
		user, err = s.repo.GetUserByPhone(ctx, req.Phone)
	}
	if err != nil {
		return nil, err
	}

	p := &authpkg.Principal{
		UserID:         user.ID.String(),
		Role:           user.Role,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		Phone:          user.Phone,
		ProfilePicture: user.ProfilePicture,
	}
	switch user.Role {
	case "courier":
		c, err := s.repo.GetCourierByUserID(ctx, user.ID)
		if err == nil {
			p.CourierID = c.ID.String()
		}
	case "customer":
		c, err := s.repo.GetCustomerByUserID(ctx, user.ID)
		if err == nil {
			p.CustomerID = c.ID.String()
		}
	case "admin":
		a, err := s.repo.GetAdminByUserID(ctx, user.ID)
		if err == nil {
			p.AdminID = a.ID.String()
		}
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-insecure-secret-change-me"
	}
	token, err := authpkg.SignJWT(secret, p, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	p.Token = token
	return p, nil
}
