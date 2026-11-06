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

// ListUsers mengembalikan daftar user dengan filter + paginasi (khusus superadmin).
func (r *UserRepository) ListUsers(page, perPage int, search string, role domain.Role, isActive *bool) ([]domain.User, int64, error) {
	q := r.db.Model(&domain.User{})
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("name LIKE ? OR email LIKE ?", like, like)
	}
	if role != "" {
		q = q.Where("role = ?", string(role))
	}
	if isActive != nil {
		q = q.Where("is_active = ?", *isActive)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []domain.User
	if err := q.Order("created_at DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// UserStats menghitung total user per role + jumlah akun nonaktif.
func (r *UserRepository) UserStats() (total, admins, superadmins, customers, inactive int64, err error) {
	if err = r.db.Model(&domain.User{}).Count(&total).Error; err != nil {
		return
	}
	if err = r.db.Model(&domain.User{}).Where("role = ?", string(domain.RoleAdmin)).Count(&admins).Error; err != nil {
		return
	}
	if err = r.db.Model(&domain.User{}).Where("role = ?", string(domain.RoleSuperAdmin)).Count(&superadmins).Error; err != nil {
		return
	}
	if err = r.db.Model(&domain.User{}).Where("role = ?", string(domain.RoleCustomer)).Count(&customers).Error; err != nil {
		return
	}
	err = r.db.Model(&domain.User{}).Where("is_active = ?", false).Count(&inactive).Error
	return
}

// RevokeAllRefreshTokens mencabut seluruh refresh token milik user (dipakai
// saat akun dinonaktifkan agar sesi yang sedang berjalan ikut mati).
func (r *UserRepository) RevokeAllRefreshTokens(userID uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&domain.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

// Delete menghapus user (soft delete via gorm.DeletedAt).
func (r *UserRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&domain.User{}, "id = ?", id).Error
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
