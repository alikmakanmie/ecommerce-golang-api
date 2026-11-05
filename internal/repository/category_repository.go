package repository

import (
	"ecommerce-api/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryRepository struct{ db *gorm.DB }

func NewCategoryRepository(db *gorm.DB) *CategoryRepository { return &CategoryRepository{db: db} }
func (r *CategoryRepository) WithTx(tx *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: tx}
}

func (r *CategoryRepository) Create(c *domain.Category) error { return r.db.Create(c).Error }

func (r *CategoryRepository) FindByID(id uuid.UUID) (*domain.Category, error) {
	var c domain.Category
	if err := r.db.First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepository) FindBySlug(slug string) (*domain.Category, error) {
	var c domain.Category
	if err := r.db.Where("slug = ?", slug).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepository) List() ([]domain.Category, error) {
	var list []domain.Category
	err := r.db.Order("name ASC").Find(&list).Error
	return list, err
}

func (r *CategoryRepository) Update(c *domain.Category) error { return r.db.Save(c).Error }

func (r *CategoryRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&domain.Category{}, "id = ?", id).Error
}
