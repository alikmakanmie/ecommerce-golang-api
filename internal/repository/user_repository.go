package repository

import (
	"time"

	"ecommerce-api/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) *UserRepository { return &UserRepository{db: db} }
func (r *UserRepository) WithTx(tx *gorm.DB) *UserRepository {
	return &UserRepository{db: tx}
}

func (r *UserRepository) Create(user *domain.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) FindByEmail(email string) (*domain.User, error) {
	var u domain.User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByID(id uuid.UUID) (*domain.User, error) {
	var u domain.User
	if err := r.db.Preload("Addresses").First(&u, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Update(user *domain.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) CreateRefreshToken(t *domain.RefreshToken) error {
	return r.db.Create(t).Error
}

func (r *UserRepository) FindRefreshTokenByHash(hash string) (*domain.RefreshToken, error) {
	var t domain.RefreshToken
	if err := r.db.Where("token_hash = ?", hash).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *UserRepository) RevokeRefreshToken(id uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&domain.RefreshToken{}).Where("id = ?", id).Update("revoked_at", now).Error
}
