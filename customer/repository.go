package customer

import (
	"context"

	"github.com/google/uuid"
	"github.com/mikios34/delivery-backend/entity"
)

// CustomerRepository specifies customer related database operations.
type CustomerRepository interface {
	StoreUser(ctx context.Context, u *entity.User) (*entity.User, error)
	StoreCustomer(ctx context.Context, c *entity.Customer) (*entity.Customer, error)
	GetCustomerByID(ctx context.Context, id uuid.UUID) (*entity.Customer, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	PhoneExists(ctx context.Context, phone string) (bool, error)
}
