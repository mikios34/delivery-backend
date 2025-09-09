package customer

import (
	"context"

	"github.com/mikios34/delivery-backend/entity"
)

// RegisterCustomerRequest carries the data required to register a customer.
type RegisterCustomerRequest struct {
	FirstName   string
	LastName    string
	Phone       string
	FirebaseUID string
}

// CustomerService exposes customer-related business operations.
type CustomerService interface {
	RegisterCustomer(ctx context.Context, req RegisterCustomerRequest) (*entity.Customer, error)
}
