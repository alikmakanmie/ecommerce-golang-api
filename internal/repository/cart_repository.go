package repository

import (
	"time"

	"ecommerce-api/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	// Race-safe create: bila dua goroutine membuat keranjang bersamaan,
	// salah satunya menang, yang lain mengabaikan konflik lalu membaca ulang.
	cart = domain.Cart{UserID: userID}
	if err := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoNothing: true,
	}).Create(&cart).Error; err != nil {
		return nil, err
	}
	if err := r.db.Preload("Items.Product").Where("user_id = ?", userID).First(&cart).Error; err != nil {
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
	return r.UpsertItemAtomic(cartID, productID, qty, price)
}

// UpsertItemAtomic menambah kuantitas secara atomik dalam satu statement SQL:
// INSERT ... ON CONFLICT (cart_id, product_id) DO UPDATE SET quantity = quantity + ?
// Dua transaksi bersamaan tidak bisa saling menimpa (lost-update) karena baris
// yang berkonflik dikunci database sampai transaksi pertama commit; increment
// dievaluasi dari nilai terbaru di dalam database, bukan dari nilai yang dibaca
// aplikasi sebelumnya. Mengembalikan item sesudah upsert.
func (r *CartRepository) UpsertItemAtomic(cartID, productID uuid.UUID, qty int, price int64) (*domain.CartItem, error) {
	item := &domain.CartItem{CartID: cartID, ProductID: productID, Quantity: qty, Price: price}
	err := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "cart_id"}, {Name: "product_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"quantity":   gorm.Expr("quantity + ?", qty),
			"price":      price,
			"updated_at": time.Now(),
			// Baris lama bisa berstatus soft-deleted (ClearItems memakai
			// soft delete GORM); upsert harus membangunkannya kembali,
			// kalau tidak FindItem (yang memfilter deleted_at) tak menemukannya.
			"deleted_at": nil,
		}),
	}).Create(item).Error
	if err != nil {
		return nil, err
	}
	return r.FindItem(cartID, productID)
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
