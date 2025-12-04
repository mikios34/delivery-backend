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
	accessTTL := 15 * time.Minute
	refreshTTL := 30 * 24 * time.Hour
	token, err := authpkg.SignJWT(secret, p, accessTTL, "access")
	if err != nil {
		return nil, err
	}
	p.Token = token
	refresh, err := authpkg.SignJWT(secret, p, refreshTTL, "refresh")
	if err != nil {
		return nil, err
	}
	p.RefreshToken = refresh
	return p, nil
}

func (s *authService) Refresh(ctx context.Context, refreshToken string) (*authpkg.Principal, error) {
	if refreshToken == "" {
		return nil, errors.New("missing refresh token")
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-insecure-secret-change-me"
	}
	claims, err := authpkg.ParseAndValidate(secret, refreshToken)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != "refresh" {
		return nil, errors.New("invalid token type")
	}
	// Build principal from claims
	p := &authpkg.Principal{
		UserID:     claims.UserID,
		Role:       claims.Role,
		CourierID:  claims.CourierID,
		CustomerID: claims.CustomerID,
		AdminID:    claims.AdminID,
	}
	accessTTL := 15 * time.Minute
	token, err := authpkg.SignJWT(secret, p, accessTTL, "access")
	if err != nil {
		return nil, err
	}
	p.Token = token
	// Return same refresh token or issue a new one (rotation optional)
	// For simplicity, return the same refresh token
	p.RefreshToken = refreshToken
	return p, nil
}
