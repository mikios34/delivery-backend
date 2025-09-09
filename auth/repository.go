package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/mikios34/delivery-backend/entity"
)

// Repository exposes read operations used for authentication.
type Repository interface {
	GetUserByPhone(ctx context.Context, phone string) (*entity.User, error)
	GetUserByFirebaseUID(ctx context.Context, uid string) (*entity.User, error)

	GetCourierByUserID(ctx context.Context, userID uuid.UUID) (*entity.Courier, error)
	GetCustomerByUserID(ctx context.Context, userID uuid.UUID) (*entity.Customer, error)
	GetAdminByUserID(ctx context.Context, userID uuid.UUID) (*entity.Admin, error)
}
