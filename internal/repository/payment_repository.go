package repository

import (
	"time"

	"ecommerce-api/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentRepository struct{ db *gorm.DB }

func NewPaymentRepository(db *gorm.DB) *PaymentRepository { return &PaymentRepository{db: db} }
func (r *PaymentRepository) WithTx(tx *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: tx}
}

func (r *PaymentRepository) Create(tx *gorm.DB, p *domain.Payment) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.Create(p).Error
}

func (r *PaymentRepository) FindByTransactionID(txID string) (*domain.Payment, error) {
	var p domain.Payment
	if err := r.db.Where("transaction_id = ?", txID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PaymentRepository) FindByOrderID(orderID uuid.UUID) (*domain.Payment, error) {
	var p domain.Payment
	if err := r.db.Where("order_id = ?", orderID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PaymentRepository) UpdateStatus(tx *gorm.DB, paymentID uuid.UUID, status domain.PaymentStatus, paidAt *time.Time) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	updates := map[string]interface{}{"status": status}
	if paidAt != nil {
		updates["paid_at"] = paidAt
	}
	return db.Model(&domain.Payment{}).Where("id = ?", paymentID).Updates(updates).Error
}

func (r *PaymentRepository) Update(p *domain.Payment) error {
	return r.db.Save(p).Error
}
