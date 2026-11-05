package repository

import (
	"ecommerce-api/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CartRepository struct{ db *gorm.DB }

func NewCartRepository(db *gorm.DB) *CartRepository { return &CartRepository{db: db} }
func (r *CartRepository) WithTx(tx *gorm.DB) *CartRepository { return &CartRepository{db: tx} }

func (r *CartRepository) GetOrCreateByUserID(userID uuid.UUID) (*domain.Cart, error) {
	var cart domain.Cart
	err := r.db.Preload("Items.Product").Where("user_id = ?", userID).First(&cart).Error
	if err == nil {
		return &cart, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	cart = domain.Cart{UserID: userID}
	if err := r.db.Create(&cart).Error; err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *CartRepository) FindByUserID(userID uuid.UUID) (*domain.Cart, error) {
	var cart domain.Cart
	if err := r.db.Preload("Items.Product").Where("user_id = ?", userID).First(&cart).Error; err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *CartRepository) FindItem(cartID, productID uuid.UUID) (*domain.CartItem, error) {
	var item domain.CartItem
	if err := r.db.Where("cart_id = ? AND product_id = ?", cartID, productID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CartRepository) AddOrUpdateItem(cartID, productID uuid.UUID, qty int, price int64) (*domain.CartItem, error) {
	existing, err := r.FindItem(cartID, productID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == nil {
		existing.Quantity += qty
		existing.Price = price
		if err := r.db.Save(existing).Error; err != nil {
			return nil, err
		}
		return existing, nil
	}
	item := &domain.CartItem{CartID: cartID, ProductID: productID, Quantity: qty, Price: price}
	if err := r.db.Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (r *CartRepository) UpdateItemQty(itemID uuid.UUID, qty int) error {
	return r.db.Model(&domain.CartItem{}).Where("id = ?", itemID).Update("quantity", qty).Error
}

func (r *CartRepository) RemoveItem(itemID uuid.UUID) error {
	return r.db.Delete(&domain.CartItem{}, "id = ?", itemID).Error
}

func (r *CartRepository) ClearItems(cartID uuid.UUID) error {
	return r.db.Delete(&domain.CartItem{}, "cart_id = ?", cartID).Error
}

func (r *CartRepository) FindItemByID(itemID uuid.UUID) (*domain.CartItem, error) {
	var item domain.CartItem
	if err := r.db.Preload("Product").First(&item, "id = ?", itemID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
