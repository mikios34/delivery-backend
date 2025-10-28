package courier

import (
	"context"

	"github.com/google/uuid"
	"github.com/mikios34/delivery-backend/entity"
)

// RegisterCourierRequest carries the data required to register a courier.
// The handler is expected to verify Firebase phone auth and provide the FirebaseUID before calling the service.
type RegisterCourierRequest struct {
	FirstName        string
	LastName         string
	Phone            string
	FirebaseUID      string
	HasVehicle       bool
	PrimaryVehicle   entity.VehicleType
	VehicleDetails   string
	GuarantyOptionID uuid.UUID
	ProfilePicture   string // optional URL; empty means none
}

// CourierService exposes courier-related business operations.
type CourierService interface {
	RegisterCourier(ctx context.Context, req RegisterCourierRequest) (*entity.Courier, error)
	ListGuarantyOptions(ctx context.Context) ([]entity.GuarantyOption, error)
	SetAvailability(ctx context.Context, courierID uuid.UUID, available bool) error
	UpdateLocation(ctx context.Context, courierID uuid.UUID, lat, lng *float64) error
}
