package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/mikios34/delivery-backend/entity"
	orderpkg "github.com/mikios34/delivery-backend/order"
	"gorm.io/gorm"
)

type GormOrderRepo struct{ db *gorm.DB }

func NewGormOrderRepo(db *gorm.DB) orderpkg.Repository { return &GormOrderRepo{db: db} }

func (r *GormOrderRepo) CreateOrder(ctx context.Context, o *entity.Order) (*entity.Order, error) {
	if err := r.db.WithContext(ctx).Create(o).Error; err != nil {
		return nil, err
	}
	return o, nil
}

func (r *GormOrderRepo) GetOrderByID(ctx context.Context, id uuid.UUID) (*entity.Order, error) {
	var o entity.Order
	if err := r.db.WithContext(ctx).First(&o, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *GormOrderRepo) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status entity.OrderStatus) error {
	return r.db.WithContext(ctx).Model(&entity.Order{}).Where("id = ?", id).Update("status", status).Error
}

func (r *GormOrderRepo) AssignCourier(ctx context.Context, id uuid.UUID, courierID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.Order{}).Where("id = ?", id).Update("assigned_courier", courierID).Error
}

func (r *GormOrderRepo) ClearAssignment(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.Order{}).Where("id = ?", id).Update("assigned_courier", nil).Error
}

func (r *GormOrderRepo) ListOrderTypes(ctx context.Context) ([]entity.OrderType, error) {
	var types []entity.OrderType
	if err := r.db.WithContext(ctx).Where("active = ?", true).Find(&types).Error; err != nil {
		return nil, err
	}
	return types, nil
}

func (r *GormOrderRepo) CreateOrderType(ctx context.Context, t *entity.OrderType) (*entity.OrderType, error) {
	if err := r.db.WithContext(ctx).Create(t).Error; err != nil {
		return nil, err
	}
	return t, nil
}

func (r *GormOrderRepo) ListAssignedOlderThan(ctx context.Context, cutoff time.Time) ([]entity.Order, error) {
	var list []entity.Order
	if err := r.db.WithContext(ctx).
		Where("status = ? AND updated_at < ?", entity.OrderAssigned, cutoff).
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *GormOrderRepo) CountAssignedOrders(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Order{}).Where("status = ?", entity.OrderAssigned).Count(&count).Error
	return count, err
}

func (r *GormOrderRepo) MarkNoNearbyDriver(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.Order{}).Where("id = ?", id).Updates(map[string]interface{}{
		"assigned_courier": nil,
		"status":           entity.OrderNoNearbyDriver,
	}).Error
}
