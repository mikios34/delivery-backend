package auth

import (
	"context"
)

// LoginRequest supports two modes: phone or firebase_uid. One must be provided.
type LoginRequest struct {
	Phone       string
	FirebaseUID string
}

type Principal struct {
	UserID string
	Role   string
	// Optional: attach specific profile IDs based on role
	CourierID  string
	CustomerID string
	AdminID    string
	Token      string
}

// Service provides login/auth operations (no password; trusts Firebase UID or phone verified externally).
type Service interface {
	Login(ctx context.Context, req LoginRequest) (*Principal, error)
}
