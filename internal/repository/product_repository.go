package repository

import (
	"errors"
	"strings"

	"ecommerce-api/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductRepository struct{ db *gorm.DB }

func NewProductRepository(db *gorm.DB) *ProductRepository { return &ProductRepository{db: db} }
func (r *ProductRepository) WithTx(tx *gorm.DB) *ProductRepository {
	return &ProductRepository{db: tx}
}

func (r *ProductRepository) Create(p *domain.Product) error { return r.db.Create(p).Error }

func (r *ProductRepository) FindByID(id uuid.UUID) (*domain.Product, error) {
	var p domain.Product
	if err := r.db.Preload("Category").First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepository) FindBySlug(slug string) (*domain.Product, error) {
	var p domain.Product
	if err := r.db.Preload("Category").Where("slug = ?", slug).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepository) FindBySKU(sku string) (*domain.Product, error) {
	var p domain.Product
	if err := r.db.Where("sku = ?", sku).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepository) List(search, categoryID string, minPrice, maxPrice int64, page, limit int) ([]domain.Product, int64, error) {
	var products []domain.Product
	var total int64
	q := r.db.Model(&domain.Product{}).Preload("Category").Where("is_active = ?", true)
	if strings.TrimSpace(search) != "" {
		like := "%" + strings.TrimSpace(search) + "%"
		q = q.Where("LOWER(name) LIKE LOWER(?) OR LOWER(description) LIKE LOWER(?)", like, like)
	}
	if categoryID != "" {
		if id, err := uuid.Parse(categoryID); err == nil {
			q = q.Where("category_id = ?", id)
		}
	}
	if minPrice > 0 {
		q = q.Where("price >= ?", minPrice)
	}
	if maxPrice > 0 {
		q = q.Where("price <= ?", maxPrice)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := 0
	if page > 1 {
		offset = (page - 1) * limit
	}
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&products).Error; err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

func (r *ProductRepository) Update(p *domain.Product) error { return r.db.Save(p).Error }

func (r *ProductRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&domain.Product{}, "id = ?", id).Error
}

func (r *ProductRepository) DecreaseStock(tx *gorm.DB, productID uuid.UUID, qty int) error {
	res := tx.Model(&domain.Product{}).Where("id = ? AND stock >= ?", productID, qty).UpdateColumn("stock", gorm.Expr("stock - ?", qty))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("stok tidak mencukupi")
	}
	return nil
}

func (r *ProductRepository) IncreaseStock(tx *gorm.DB, productID uuid.UUID, qty int) error {
	return tx.Model(&domain.Product{}).Where("id = ?", productID).UpdateColumn("stock", gorm.Expr("stock + ?", qty)).Error
}
