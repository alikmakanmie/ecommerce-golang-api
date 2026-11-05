package repository

import (
	"ecommerce-api/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderRepository struct{ db *gorm.DB }

func NewOrderRepository(db *gorm.DB) *OrderRepository { return &OrderRepository{db: db} }
func (r *OrderRepository) WithTx(tx *gorm.DB) *OrderRepository { return &OrderRepository{db: tx} }

func (r *OrderRepository) Create(tx *gorm.DB, order *domain.Order) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.Create(order).Error
}

func (r *OrderRepository) FindByID(id uuid.UUID) (*domain.Order, error) {
	var o domain.Order
	if err := r.db.Preload("Items").Preload("Payment").Preload("Address").First(&o, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrderRepository) FindByOrderNumber(number string) (*domain.Order, error) {
	var o domain.Order
	if err := r.db.Preload("Items").Preload("Payment").Where("order_number = ?", number).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrderRepository) ListByUser(userID uuid.UUID, page, limit int) ([]domain.Order, int64, error) {
	var list []domain.Order
	var total int64
	q := r.db.Model(&domain.Order{}).Where("user_id = ?", userID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := 0
	if page > 1 {
		offset = (page - 1) * limit
	}
	if err := q.Preload("Items").Preload("Payment").Order("created_at DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *OrderRepository) ListAll(page, limit int) ([]domain.Order, int64, error) {
	var list []domain.Order
	var total int64
	q := r.db.Model(&domain.Order{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := 0
	if page > 1 {
		offset = (page - 1) * limit
	}
	if err := q.Preload("Items").Preload("Payment").Order("created_at DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *OrderRepository) UpdateStatus(tx *gorm.DB, orderID uuid.UUID, status domain.OrderStatus) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.Model(&domain.Order{}).Where("id = ?", orderID).Update("status", status).Error
}

func (r *OrderRepository) Update(order *domain.Order) error {
	return r.db.Save(order).Error
}
