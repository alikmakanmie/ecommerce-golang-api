package repository

import (
	"ecommerce-api/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AddressRepository struct{ db *gorm.DB }

func NewAddressRepository(db *gorm.DB) *AddressRepository { return &AddressRepository{db: db} }
func (r *AddressRepository) WithTx(tx *gorm.DB) *AddressRepository {
	return &AddressRepository{db: tx}
}

func (r *AddressRepository) Create(a *domain.Address) error {
	return r.db.Create(a).Error
}

func (r *AddressRepository) FindByUserID(userID uuid.UUID) ([]domain.Address, error) {
	var list []domain.Address
	err := r.db.Where("user_id = ?", userID).Order("is_default DESC, created_at ASC").Find(&list).Error
	return list, err
}

func (r *AddressRepository) FindByID(id uuid.UUID) (*domain.Address, error) {
	var a domain.Address
	if err := r.db.First(&a, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AddressRepository) FindDefault(userID uuid.UUID) (*domain.Address, error) {
	var a domain.Address
	if err := r.db.Where("user_id = ? AND is_default = ?", userID, true).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AddressRepository) ClearDefault(userID uuid.UUID) error {
	return r.db.Model(&domain.Address{}).Where("user_id = ?", userID).Update("is_default", false).Error
}

func (r *AddressRepository) ClearDefaultTx(tx *gorm.DB, userID uuid.UUID) error {
	return tx.Model(&domain.Address{}).Where("user_id = ?", userID).Update("is_default", false).Error
}

func (r *AddressRepository) Update(a *domain.Address) error {
	return r.db.Save(a).Error
}

func (r *AddressRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&domain.Address{}, "id = ?", id).Error
}
