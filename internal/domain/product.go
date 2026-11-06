package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Category struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Slug        string         `gorm:"size:120;uniqueIndex;not null" json:"slug"`
	Description string         `gorm:"size:500" json:"description"`
	ImageURL    string         `gorm:"size:255" json:"image_url"`
	Products    []Product      `gorm:"foreignKey:CategoryID" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (c *Category) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type Product struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	CategoryID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"category_id"`
	Category    Category       `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Name        string         `gorm:"size:150;not null;index" json:"name"`
	Slug        string         `gorm:"size:170;uniqueIndex;not null" json:"slug"`
	Description string         `gorm:"type:text" json:"description"`
	Price       int64          `gorm:"not null;check:price>=0" json:"price"`
	CostPrice   int64          `gorm:"not null;default:0;check:cost_price>=0" json:"cost_price"`
	Stock       int            `gorm:"not null;default:0;check:stock>=0" json:"stock"`
	SKU         string         `gorm:"size:50;uniqueIndex;not null" json:"sku"`
	ImageURL    string         `gorm:"size:255" json:"image_url"`
	WeightGram  int            `gorm:"not null;default:0" json:"weight_gram"`
	IsActive    bool           `gorm:"not null;default:true;index" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
