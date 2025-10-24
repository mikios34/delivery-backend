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
	if err := r.db.WithContext(ctx).First(&c, "id = ?", id).Error; err != nil {
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

func (r *GormCourierRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	var u entity.User
	if err := r.db.WithContext(ctx).First(&u, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *GormCourierRepo) GetUserByCourierID(ctx context.Context, courierID uuid.UUID) (*entity.User, error) {
	var u entity.User
	// Join couriers->users by user_id
	if err := r.db.WithContext(ctx).
		Model(&entity.User{}).
		Joins("JOIN couriers c ON c.user_id = users.id").
		Where("c.id = ?", courierID).
		First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
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

func (r *GormCourierRepo) ListAvailableCouriersNear(ctx context.Context, centerLat, centerLng, radiusKm float64, limit int) ([]entity.Courier, error) {
	// Haversine expression; Postgres syntax with RADIANS
	const haversineExpr = `
		(2 * 6371 * ASIN(SQRT(
			POWER(SIN(RADIANS($1 - latitude) / 2), 2) +
			COS(RADIANS($1)) * COS(RADIANS(latitude)) * POWER(SIN(RADIANS($2 - longitude) / 2), 2)
		)))
	`

	// Exclude couriers with an active order (assigned/accepted/arrived/picked_up)
	sql := `
		SELECT id, user_id, has_vehicle, primary_vehicle, vehicle_details,
		       guaranty_option_id, guaranty_paid, active, available,
		       latitude, longitude, created_at, updated_at, deleted_at
		FROM couriers c
		WHERE c.available = TRUE AND c.active = TRUE AND c.latitude IS NOT NULL AND c.longitude IS NOT NULL
		  AND NOT EXISTS (
		    SELECT 1 FROM orders o
		    WHERE o.assigned_courier = c.id
		      AND o.status IN ('assigned','accepted','arrived','picked_up')
		  )
		  AND ` + haversineExpr + ` <= $3
		ORDER BY ` + haversineExpr + ` ASC
		LIMIT $4
	`

	var list []entity.Courier
	if err := r.db.WithContext(ctx).Raw(sql, centerLat, centerLng, radiusKm, limit).Scan(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
