package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Cart struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	Items     []CartItem     `gorm:"foreignKey:CartID" json:"items"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (c *Cart) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

func (c *Cart) Subtotal() int64 {
	var total int64
	for _, item := range c.Items {
		total += item.Subtotal()
	}
	return total
}

func (c *Cart) TotalItems() int {
	total := 0
	for _, item := range c.Items {
		total += item.Quantity
	}
	return total
}

type CartItem struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	CartID    uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:idx_cart_product,priority:1" json:"cart_id"`
	ProductID uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:idx_cart_product,priority:2" json:"product_id"`
	Product   Product        `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Quantity  int            `gorm:"not null;check:quantity>0" json:"quantity"`
	Price     int64          `gorm:"not null;check:price>=0" json:"price"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (i *CartItem) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

func (i *CartItem) Subtotal() int64 {
	return i.Price * int64(i.Quantity)
}
