package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/mikios34/delivery-backend/courier"
	"github.com/mikios34/delivery-backend/entity"
	"gorm.io/gorm"
)

// GormCourierRepo implements courier.CourierRepository using GORM (v2).
type GormCourierRepo struct {
	db *gorm.DB
}

func NewGormCourierRepo(db *gorm.DB) courier.CourierRepository {
	return &GormCourierRepo{db: db}
}

func (r *GormCourierRepo) StoreUser(ctx context.Context, u *entity.User) (*entity.User, error) {
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

func (r *GormCourierRepo) StoreCourier(ctx context.Context, c *entity.Courier) (*entity.Courier, error) {
	if err := r.db.WithContext(ctx).Create(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

func (r *GormCourierRepo) GetCourierByID(ctx context.Context, id uuid.UUID) (*entity.Courier, error) {
	var c entity.Courier
	if err := r.db.WithContext(ctx).Preload("GuarantyPayments").First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *GormCourierRepo) GetCourierByUserID(ctx context.Context, userID uuid.UUID) (*entity.Courier, error) {
	var c entity.Courier
	if err := r.db.WithContext(ctx).First(&c, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *GormCourierRepo) PhoneExists(ctx context.Context, phone string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&entity.User{}).Where("phone = ? AND role = ?", phone, "courier").Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *GormCourierRepo) ListGuarantyOptions(ctx context.Context) ([]entity.GuarantyOption, error) {
	var opts []entity.GuarantyOption
	if err := r.db.WithContext(ctx).Where("active = ?", true).Find(&opts).Error; err != nil {
		return nil, err
	}
	return opts, nil
}

func (r *GormCourierRepo) CreateGuarantyPayment(ctx context.Context, gp *entity.GuarantyPayment) (*entity.GuarantyPayment, error) {
	if err := r.db.WithContext(ctx).Create(gp).Error; err != nil {
		return nil, err
	}
	return gp, nil
}

func (r *GormCourierRepo) UpdateAvailability(ctx context.Context, courierID uuid.UUID, available bool) error {
	return r.db.WithContext(ctx).Model(&entity.Courier{}).Where("id = ?", courierID).Update("available", available).Error
}

func (r *GormCourierRepo) UpdateLocation(ctx context.Context, courierID uuid.UUID, lat, lng *float64) error {
	updates := map[string]interface{}{
		"latitude":  lat,
		"longitude": lng,
	}
	return r.db.WithContext(ctx).Model(&entity.Courier{}).Where("id = ?", courierID).Updates(updates).Error
}
