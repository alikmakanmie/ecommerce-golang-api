package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentStatus string

const (
	PaymentPending    PaymentStatus = "pending"
	PaymentSettlement PaymentStatus = "settlement"
	PaymentExpire     PaymentStatus = "expire"
	PaymentCancel     PaymentStatus = "cancel"
	PaymentFailed     PaymentStatus = "failed"
)

type Payment struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	OrderID       uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null" json:"order_id"`
	TransactionID string         `gorm:"size:100;uniqueIndex;not null" json:"transaction_id"`
	Gateway       string         `gorm:"size:30;not null;default:mock" json:"gateway"`
	Method        string         `gorm:"size:30;not null" json:"method"`
	Amount        int64          `gorm:"not null" json:"amount"`
	Status        PaymentStatus  `gorm:"size:20;not null;default:pending;index" json:"status"`
	PaymentURL    string         `gorm:"size:500" json:"payment_url,omitempty"`
	QRString      string         `gorm:"size:500" json:"qr_string,omitempty"`
	RawPayload    string         `gorm:"type:text" json:"-"`
	ExpiresAt     *time.Time     `json:"expires_at,omitempty"`
	PaidAt        *time.Time     `json:"paid_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

func (p *Payment) IsPaid() bool {
	return p.Status == PaymentSettlement
}
