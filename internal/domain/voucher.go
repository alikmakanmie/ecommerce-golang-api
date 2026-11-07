package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Voucher adalah kode promo: percentage (discount_value = persen, dicap
// max_discount) atau fixed (discount_value = nominal rupiah, dicap
// max_discount bila > 0). Quota 0 = tanpa batas pemakaian.
type Voucher struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Code           string     `gorm:"size:50;uniqueIndex;not null" json:"code"`
	DiscountType   string     `gorm:"size:20;not null" json:"discount_type"`
	DiscountValue  int64      `gorm:"not null;check:discount_value>=0" json:"discount_value"`
	MinOrderAmount int64      `gorm:"not null;default:0;check:min_order_amount>=0" json:"min_order_amount"`
	MaxDiscount    int64      `gorm:"not null;default:0;check:max_discount>=0" json:"max_discount"`
	Quota          int        `gorm:"not null;default:0;check:quota>=0" json:"quota"`
	UsedCount      int        `gorm:"not null;default:0;check:used_count>=0" json:"used_count"`
	IsActive       bool       `gorm:"not null;default:true;index" json:"is_active"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (v *Voucher) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}

// CustomerLoyalty melacak stamp pelanggan guest berdasarkan nomor WhatsApp
// yang dinormalisasi (format kanonis 62xxx, lihat utils.NormalizePhone).
// Aturan: +1 stamp tiap pesanan lunas pertama kali (paid/completed);
// 10 stamp = reward_eligible (reward diberikan manual/di luar sistem).
type CustomerLoyalty struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	CustomerPhone string         `gorm:"size:20;uniqueIndex;not null" json:"customer_phone"`
	Points        int            `gorm:"not null;default:0;check:points>=0" json:"points"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (l *CustomerLoyalty) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}

// RewardThreshold adalah jumlah stamp untuk satu reward (kopi gratis).
const RewardThreshold = 10
