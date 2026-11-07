package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderStatus string

const (
	OrderPending    OrderStatus = "pending"
	OrderPaid       OrderStatus = "paid"
	OrderProcessing OrderStatus = "processing"
	OrderShipped    OrderStatus = "shipped"
	OrderCompleted  OrderStatus = "completed"
	OrderCancelled  OrderStatus = "cancelled"
)

type Order struct {
	ID              uuid.UUID   `gorm:"type:uuid;primaryKey" json:"id"`
	UserID          *uuid.UUID  `gorm:"type:uuid;index" json:"user_id,omitempty"`
	OrderNumber     string      `gorm:"size:30;uniqueIndex;not null" json:"order_number"`
	CustomerName    string      `gorm:"size:100;not null" json:"customer_name"`
	CustomerPhone   string      `gorm:"size:20" json:"customer_phone"`
	OrderType       string      `gorm:"size:20;not null;default:delivery" json:"order_type"`
	TableNumber     string      `gorm:"size:20" json:"table_number,omitempty"`
	AddressID       *uuid.UUID  `gorm:"type:uuid;index" json:"address_id,omitempty"`
	Address         *Address    `gorm:"foreignKey:AddressID" json:"shipping_address,omitempty"`
	ShippingName    string      `gorm:"size:100;not null" json:"shipping_name"`
	ShippingPhone   string      `gorm:"size:20;not null" json:"shipping_phone"`
	ShippingAddress string      `gorm:"size:500;not null" json:"shipping_address_text"`
	Status          OrderStatus `gorm:"size:20;not null;default:pending;index" json:"status"`
	Subtotal        int64       `gorm:"not null" json:"subtotal"`
	// DiscountAmount adalah potongan voucher (rupiah); VoucherCode kode yang dipakai.
	DiscountAmount int64  `gorm:"not null;default:0" json:"discount_amount"`
	VoucherCode    string `gorm:"size:50" json:"voucher_code,omitempty"`
	ShippingCost   int64  `gorm:"not null;default:0" json:"shipping_cost"`
	Total          int64  `gorm:"not null" json:"total"`
	// LoyaltyAwarded menandai stamp loyalty sudah diberikan untuk order ini
	// (anti double-count bila status paid/completed disentuh berkali-kali).
	LoyaltyAwarded bool        `gorm:"not null;default:false" json:"-"`
	PaymentMethod  string      `gorm:"size:30;not null" json:"payment_method"`
	Notes           string      `gorm:"size:500" json:"notes"`
	// Audit trail single-store: staf (admin/superadmin) terakhir yang mengubah status.
	ProcessedByUserID *uuid.UUID     `gorm:"type:uuid;index" json:"processed_by_user_id,omitempty"`
	ProcessedByName   string         `gorm:"size:100" json:"processed_by_name,omitempty"`
	Items             []OrderItem    `gorm:"foreignKey:OrderID" json:"items"`
	Payment           *Payment       `gorm:"foreignKey:OrderID" json:"payment,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

func (o *Order) CanCancel() bool {
	return o.Status == OrderPending || o.Status == OrderPaid
}

func (o *Order) CanTransitionTo(target OrderStatus) bool {
	allowed := map[OrderStatus][]OrderStatus{
		OrderPending:    {OrderPaid, OrderCancelled},
		OrderPaid:       {OrderProcessing, OrderCancelled},
		OrderProcessing: {OrderShipped, OrderCancelled},
		OrderShipped:    {OrderCompleted},
		OrderCompleted:  {},
		OrderCancelled:  {},
	}
	for _, s := range allowed[o.Status] {
		if s == target {
			return true
		}
	}
	return false
}

type OrderItem struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	OrderID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"order_id"`
	ProductID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"product_id"`
	ProductName string         `gorm:"size:150;not null" json:"product_name"`
	Price       int64          `gorm:"not null" json:"price"`
	CostPrice   int64          `gorm:"not null;default:0" json:"cost_price"`
	// SelectedModifiers: varian yang dipilih pelanggan, array datar JSON
	// [{"name":"Iced","price_delta":0,"cost_delta":0}, ...].
	// UnitPrice = harga dasar + jumlah price_delta; UnitCost = HPP dasar +
	// jumlah cost_delta. Price/CostPrice selalu disinkronkan dengan
	// UnitPrice/UnitCost agar analytics lama tetap akurat.
	SelectedModifiers RawJSON `gorm:"type:text;column:selected_modifiers" json:"selected_modifiers"`
	UnitPrice         int64   `gorm:"not null;default:0" json:"unit_price"`
	UnitCost          int64   `gorm:"not null;default:0" json:"unit_cost"`
	Quantity          int     `gorm:"not null;check:quantity>0" json:"quantity"`
	Notes             string  `gorm:"size:200" json:"notes,omitempty"`
	Subtotal          int64   `gorm:"not null" json:"subtotal"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (i *OrderItem) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}
