package repository

import (
	"context"
	"time"

	"github.com/mikios34/delivery-backend/models"
	"gorm.io/gorm"
)

// Repository handles DB operations for drivers.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// NewRepositoryImpl is an explicit implementation constructor (alias).
func NewRepositoryImpl(db *gorm.DB) *Repository {
	return NewRepository(db)
}

func (r *Repository) ListDrivers(ctx context.Context) ([]models.Driver, error) {
	var drivers []models.Driver
	if err := r.db.WithContext(ctx).Find(&drivers).Error; err != nil {
		return nil, err
	}
	return drivers, nil
}

// Drivers is an alias that satisfies the DriverRepository interface.
func (r *Repository) Drivers(ctx context.Context) ([]models.Driver, error) {
	return r.ListDrivers(ctx)
}

func (r *Repository) GetDriverByID(ctx context.Context, id uint) (*models.Driver, error) {
	var d models.Driver
	if err := r.db.WithContext(ctx).First(&d, id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repository) CreateDriver(ctx context.Context, d *models.Driver) error {
	return r.db.WithContext(ctx).Create(d).Error
}

// StoreDriver creates the driver and returns the created record (with ID).
func (r *Repository) StoreDriver(ctx context.Context, d *models.Driver) (*models.Driver, error) {
	if err := r.db.WithContext(ctx).Create(d).Error; err != nil {
		return nil, err
	}
	return d, nil
}

// DeleteDriver deletes a driver by ID.
func (r *Repository) DeleteDriver(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Driver{}, id).Error
}

// DriverByName finds drivers matching the provided name.
func (r *Repository) DriverByName(ctx context.Context, name string) ([]models.Driver, error) {
	var drivers []models.Driver
	if err := r.db.WithContext(ctx).Where("name = ?", name).Find(&drivers).Error; err != nil {
		return nil, err
	}
	return drivers, nil
}

// CreateOTP stores an OTP for the given phone number.
func (r *Repository) CreateOTP(ctx context.Context, phone, otp string, expiresAt time.Time) error {
	record := &models.DriverOTP{
		Phone:     phone,
		OTP:       otp,
		ExpiresAt: expiresAt,
		Used:      false,
	}
	return r.db.WithContext(ctx).Create(record).Error
}

// VerifyOTP checks the otp for the phone, marks it used and returns true if valid.
func (r *Repository) VerifyOTP(ctx context.Context, phone, otp string) (bool, error) {
	var rec models.DriverOTP
	if err := r.db.WithContext(ctx).Where("phone = ? AND otp = ? AND used = false", phone, otp).First(&rec).Error; err != nil {
		return false, err
	}
	if time.Now().After(rec.ExpiresAt) {
		return false, nil
	}
	// mark used
	rec.Used = true
	if err := r.db.WithContext(ctx).Save(&rec).Error; err != nil {
		return false, err
	}
	// mark or create driver as verified
	var d models.Driver
	if err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&d).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			d = models.Driver{Phone: phone, PhoneVerified: true, VerifiedAt: ptrTime(time.Now())}
			if err := r.db.WithContext(ctx).Create(&d).Error; err != nil {
				return false, err
			}
			return true, nil
		}
		return false, err
	}
	d.PhoneVerified = true
	d.VerifiedAt = ptrTime(time.Now())
	if err := r.db.WithContext(ctx).Save(&d).Error; err != nil {
		return false, err
	}
	return true, nil
}

// helper to get *time.Time
func ptrTime(t time.Time) *time.Time { return &t }
