package repository

import (
	"context"

	"github.com/google/uuid"
	customerpkg "github.com/mikios34/delivery-backend/customer"
	"github.com/mikios34/delivery-backend/entity"
	"gorm.io/gorm"
)

// GormCustomerRepo implements customer.CustomerRepository using GORM.
type GormCustomerRepo struct {
	db *gorm.DB
}

func NewGormCustomerRepo(db *gorm.DB) customerpkg.CustomerRepository {
	return &GormCustomerRepo{db: db}
}

func (r *GormCustomerRepo) StoreUser(ctx context.Context, u *entity.User) (*entity.User, error) {
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

func (r *GormCustomerRepo) StoreCustomer(ctx context.Context, c *entity.Customer) (*entity.Customer, error) {
	if err := r.db.WithContext(ctx).Create(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

func (r *GormCustomerRepo) GetCustomerByID(ctx context.Context, id uuid.UUID) (*entity.Customer, error) {
	var c entity.Customer
	if err := r.db.WithContext(ctx).First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *GormCustomerRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	var u entity.User
	if err := r.db.WithContext(ctx).First(&u, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *GormCustomerRepo) PhoneExists(ctx context.Context, phone string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&entity.User{}).Where("phone = ? AND role = ?", phone, "customer").Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
