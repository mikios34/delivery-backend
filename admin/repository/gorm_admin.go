package repository

import (
	"context"

	"github.com/google/uuid"
	adminpkg "github.com/mikios34/delivery-backend/admin"
	"github.com/mikios34/delivery-backend/entity"
	"gorm.io/gorm"
)

// GormAdminRepo implements admin.AdminRepository using GORM.
type GormAdminRepo struct {
	db *gorm.DB
}

func NewGormAdminRepo(db *gorm.DB) adminpkg.AdminRepository {
	return &GormAdminRepo{db: db}
}

func (r *GormAdminRepo) StoreUser(ctx context.Context, u *entity.User) (*entity.User, error) {
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

func (r *GormAdminRepo) StoreAdmin(ctx context.Context, a *entity.Admin) (*entity.Admin, error) {
	if err := r.db.WithContext(ctx).Create(a).Error; err != nil {
		return nil, err
	}
	return a, nil
}

func (r *GormAdminRepo) GetAdminByID(ctx context.Context, id uuid.UUID) (*entity.Admin, error) {
	var a entity.Admin
	if err := r.db.WithContext(ctx).First(&a, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *GormAdminRepo) PhoneExists(ctx context.Context, phone string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&entity.User{}).Where("phone = ? AND role = ?", phone, "admin").Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
