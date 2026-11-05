package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleCustomer Role = "customer"
)

type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name         string         `gorm:"size:100;not null" json:"name"`
	Email        string         `gorm:"size:150;uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	Phone        string         `gorm:"size:20" json:"phone"`
	Role         Role           `gorm:"size:20;not null;default:customer;index" json:"role"`
	IsActive     bool           `gorm:"not null;default:true" json:"is_active"`
	Addresses    []Address      `gorm:"foreignKey:UserID" json:"addresses,omitempty"`
	Cart         *Cart          `gorm:"foreignKey:UserID" json:"-"`
	Orders       []Order        `gorm:"foreignKey:UserID" json:"-"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

type Address struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Label         string         `gorm:"size:50;not null" json:"label"`
	RecipientName string         `gorm:"size:100;not null" json:"recipient_name"`
	Phone         string         `gorm:"size:20;not null" json:"phone"`
	Street        string         `gorm:"size:255;not null" json:"street"`
	City          string         `gorm:"size:100;not null" json:"city"`
	Province      string         `gorm:"size:100;not null" json:"province"`
	PostalCode    string         `gorm:"size:10;not null" json:"postal_code"`
	IsDefault     bool           `gorm:"not null;default:false" json:"is_default"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (a *Address) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

func (a *Address) FullAddress() string {
	return a.Street + ", " + a.City + ", " + a.Province + " " + a.PostalCode
}
