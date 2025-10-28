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
	UserID string `json:"user_id"`
	Role   string `json:"-"` // used for JWT only; not returned in response
	// Optional: attach specific profile IDs based on role
	CourierID  string `json:"courier_id,omitempty"`
	CustomerID string `json:"customer_id,omitempty"`
	AdminID    string `json:"admin_id,omitempty"`
	Token      string `json:"token"`
	// User profile details included in login response for convenience
	FirstName      string  `json:"first_name"`
	LastName       string  `json:"last_name"`
	Phone          string  `json:"phone"`
	ProfilePicture *string `json:"profile_picture,omitempty"`
}

// Service provides login/auth operations (no password; trusts Firebase UID or phone verified externally).
type Service interface {
	Login(ctx context.Context, req LoginRequest) (*Principal, error)
}
