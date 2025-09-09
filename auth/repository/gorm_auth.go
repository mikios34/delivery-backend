package repository

import (
	"context"

	"github.com/google/uuid"
	authpkg "github.com/mikios34/delivery-backend/auth"
	"github.com/mikios34/delivery-backend/entity"
	"gorm.io/gorm"
)

type GormAuthRepo struct {
	db *gorm.DB
}

func NewGormAuthRepo(db *gorm.DB) authpkg.Repository {
	return &GormAuthRepo{db: db}
}

func (r *GormAuthRepo) GetUserByPhone(ctx context.Context, phone string) (*entity.User, error) {
	var u entity.User
	if err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *GormAuthRepo) GetUserByFirebaseUID(ctx context.Context, uid string) (*entity.User, error) {
	var u entity.User
	if err := r.db.WithContext(ctx).Where("firebase_uid = ?", uid).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *GormAuthRepo) GetCourierByUserID(ctx context.Context, userID uuid.UUID) (*entity.Courier, error) {
	var c entity.Courier
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *GormAuthRepo) GetCustomerByUserID(ctx context.Context, userID uuid.UUID) (*entity.Customer, error) {
	var c entity.Customer
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *GormAuthRepo) GetAdminByUserID(ctx context.Context, userID uuid.UUID) (*entity.Admin, error) {
	var a entity.Admin
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}
