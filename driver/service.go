package driver

import (
	"context"

	"github.com/mikios34/delivery-backend/models"
)

// package-level shim for driver subpackages. Real implementation is in
// driver/service/service.go

// DriverService defines business operations for drivers.
type DriverService interface {
	CreateDriver(ctx context.Context, d *models.Driver) (*models.Driver, error)
	ListDrivers(ctx context.Context) ([]models.Driver, error)
	GetDriver(ctx context.Context, id uint) (*models.Driver, error)
	DeleteDriver(ctx context.Context, id uint) error
}
