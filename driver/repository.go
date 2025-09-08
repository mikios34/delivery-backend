package driver

import (
	"context"

	"github.com/mikios34/delivery-backend/models"
)

// package-level shim for driver subpackages. Concrete implementations live
// in driver/repository and driver/service subpackages.

// DriverRepository defines storage operations for drivers.
type DriverRepository interface {
	StoreDriver(ctx context.Context, driver *models.Driver) (*models.Driver, error)
	Drivers(ctx context.Context) ([]models.Driver, error)
	DeleteDriver(ctx context.Context, id uint) error
	DriverByName(ctx context.Context, name string) ([]models.Driver, error)
}
