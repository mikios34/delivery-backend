package admin

import (
	"context"

	"github.com/mikios34/delivery-backend/entity"
)

// RegisterAdminRequest carries the data required to register an admin.
type RegisterAdminRequest struct {
	FirstName   string
	LastName    string
	Phone       string
	FirebaseUID string
}

// AdminService exposes admin-related business operations.
type AdminService interface {
	RegisterAdmin(ctx context.Context, req RegisterAdminRequest) (*entity.Admin, error)
}
