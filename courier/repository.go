package courier

import (
	"context"

	"github.com/google/uuid"
	"github.com/mikios34/delivery-backend/entity"
)

// CourierRepository specifies courier related database operations.
type CourierRepository interface {
	StoreUser(ctx context.Context, u *entity.User) (*entity.User, error)
	StoreCourier(ctx context.Context, c *entity.Courier) (*entity.Courier, error)
	GetCourierByID(ctx context.Context, id uuid.UUID) (*entity.Courier, error)
	GetCourierByUserID(ctx context.Context, userID uuid.UUID) (*entity.Courier, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	PhoneExists(ctx context.Context, phone string) (bool, error)
	ListGuarantyOptions(ctx context.Context) ([]entity.GuarantyOption, error)
	CreateGuarantyPayment(ctx context.Context, gp *entity.GuarantyPayment) (*entity.GuarantyPayment, error)
	UpdateAvailability(ctx context.Context, courierID uuid.UUID, available bool) error
	UpdateLocation(ctx context.Context, courierID uuid.UUID, lat, lng *float64) error
	ListAvailableCouriersNear(ctx context.Context, centerLat, centerLng, radiusKm float64, limit int) ([]entity.Courier, error)
}
