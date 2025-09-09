package admin

import (
	"context"

	"github.com/google/uuid"
	"github.com/mikios34/delivery-backend/entity"
)

// AdminRepository specifies admin related database operations.
type AdminRepository interface {
	StoreUser(ctx context.Context, u *entity.User) (*entity.User, error)
	StoreAdmin(ctx context.Context, a *entity.Admin) (*entity.Admin, error)
	GetAdminByID(ctx context.Context, id uuid.UUID) (*entity.Admin, error)
	PhoneExists(ctx context.Context, phone string) (bool, error)
}
