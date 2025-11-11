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

func (r *GormOrderRepo) RecordAssignmentAttempt(ctx context.Context, orderID, courierID uuid.UUID) error {
	rec := &entity.OrderAssignmentAttempt{OrderID: orderID, CourierID: courierID}
	return r.db.WithContext(ctx).Create(rec).Error
}

func (r *GormOrderRepo) ListTriedCouriers(ctx context.Context, orderID uuid.UUID) (map[uuid.UUID]struct{}, error) {
	var recs []entity.OrderAssignmentAttempt
	if err := r.db.WithContext(ctx).Where("order_id = ?", orderID).Find(&recs).Error; err != nil {
		return nil, err
	}
	m := make(map[uuid.UUID]struct{}, len(recs))
	for i := range recs {
		m[recs[i].CourierID] = struct{}{}
	}
	return m, nil
}

func (r *GormOrderRepo) GetActiveOrderForCustomer(ctx context.Context, customerID uuid.UUID) (*entity.Order, error) {
	var o entity.Order
	err := r.db.WithContext(ctx).
		Where("customer_id = ? AND status NOT IN (?, ?)", customerID, entity.OrderNoNearbyDriver, entity.OrderDelivered).
		Order("updated_at DESC").
		First(&o).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (r *GormOrderRepo) ListActiveOrdersForCustomer(ctx context.Context, customerID uuid.UUID) ([]entity.Order, error) {
	var list []entity.Order
	if err := r.db.WithContext(ctx).
		Where("customer_id = ? AND status NOT IN (?, ?)", customerID, entity.OrderNoNearbyDriver, entity.OrderDelivered).
		Order("updated_at DESC").
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *GormOrderRepo) GetActiveOrderForCourier(ctx context.Context, courierID uuid.UUID) (*entity.Order, error) {
	var o entity.Order
	err := r.db.WithContext(ctx).
		Where("assigned_courier = ? AND status NOT IN (?, ?)", courierID, entity.OrderNoNearbyDriver, entity.OrderDelivered).
		Order("updated_at DESC").
		First(&o).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

// ListDeliveredOrdersForCourier returns delivered orders assigned to the given courier ordered
// by updated_at DESC (most recent first).
func (r *GormOrderRepo) ListDeliveredOrdersForCourier(ctx context.Context, courierID uuid.UUID, limit, offset int) ([]entity.Order, error) {
	var list []entity.Order
	q := r.db.WithContext(ctx).
		Where("assigned_courier = ? AND status = ?", courierID, entity.OrderDelivered).
		Order("updated_at DESC")

	// Apply pagination if provided
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}

	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// CountDeliveredOrdersForCourier returns the total number of delivered orders for the given courier.
func (r *GormOrderRepo) CountDeliveredOrdersForCourier(ctx context.Context, courierID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.Order{}).
		Where("assigned_courier = ? AND status = ?", courierID, entity.OrderDelivered).
		Count(&count).Error
	return count, err
}

// ListActiveVehicleTypes returns active vehicle types with pricing info for fare estimations.
func (r *GormOrderRepo) ListActiveVehicleTypes(ctx context.Context) ([]entity.VehicleTypeConfig, error) {
	var list []entity.VehicleTypeConfig
	if err := r.db.WithContext(ctx).Where("active = ?", true).Order("name ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
