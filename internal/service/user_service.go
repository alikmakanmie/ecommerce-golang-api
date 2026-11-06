package service

import (
	"errors"
	"strings"

	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/repository"
	"ecommerce-api/pkg/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo    *repository.UserRepository
	addressRepo *repository.AddressRepository
}

func NewUserService(userRepo *repository.UserRepository, addressRepo *repository.AddressRepository) *UserService {
	return &UserService{userRepo: userRepo, addressRepo: addressRepo}
}

func (s *UserService) GetProfile(userID uuid.UUID) (*domain.User, error) {
	return s.userRepo.FindByID(userID)
}

func (s *UserService) UpdateProfile(userID uuid.UUID, name, phone string) (*domain.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) != "" {
		user.Name = strings.TrimSpace(name)
	}
	user.Phone = phone
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) ListAddresses(userID uuid.UUID) ([]domain.Address, error) {
	return s.addressRepo.FindByUserID(userID)
}

func (s *UserService) CreateAddress(userID uuid.UUID, addr *domain.Address) (*domain.Address, error) {
	addr.UserID = userID
	if addr.IsDefault {
		if err := s.addressRepo.ClearDefault(userID); err != nil {
			return nil, err
		}
	} else {
		// First address becomes default automatically.
		existing, err := s.addressRepo.FindByUserID(userID)
		if err != nil {
			return nil, err
		}
		if len(existing) == 0 {
			addr.IsDefault = true
		}
	}
	if err := s.addressRepo.Create(addr); err != nil {
		return nil, err
	}
	return addr, nil
}

func (s *UserService) DeleteAddress(userID, addressID uuid.UUID) error {
	addr, err := s.addressRepo.FindByID(addressID)
	if err != nil {
		return err
	}
	if addr.UserID != userID {
		return errors.New("alamat bukan milik pengguna")
	}
	return s.addressRepo.Delete(addressID)
}

func (s *UserService) SetDefaultAddress(userID, addressID uuid.UUID) (*domain.Address, error) {
	addr, err := s.addressRepo.FindByID(addressID)
	if err != nil {
		return nil, err
	}
	if addr.UserID != userID {
		return nil, errors.New("alamat bukan milik pengguna")
	}
	if err := s.addressRepo.ClearDefault(userID); err != nil {
		return nil, err
	}
	addr.IsDefault = true
	if err := s.addressRepo.Update(addr); err != nil {
		return nil, err
	}
	return addr, nil
}

var _ = gorm.ErrRecordNotFound

// ---- Manajemen user (khusus superadmin) ----

var (
	ErrUserNotFound = errors.New("pengguna tidak ditemukan")
	ErrSelfAction   = errors.New("tidak bisa mengubah akun sendiri")
	ErrEmailTaken   = errors.New("email sudah terdaftar")
)

// ListUsers mengembalikan daftar user + total (paginasi dijaga di sini).
func (s *UserService) ListUsers(page, perPage int, search string, role domain.Role, isActive *bool) ([]domain.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return s.userRepo.ListUsers(page, perPage, search, role, isActive)
}

// UserStats mengembalikan ringkasan jumlah user untuk kartu dashboard superadmin.
func (s *UserService) UserStats() (map[string]int64, error) {
	total, admins, superadmins, customers, inactive, err := s.userRepo.UserStats()
	if err != nil {
		return nil, err
	}
	return map[string]int64{
		"total":      total,
		"admin":      admins,
		"superadmin": superadmins,
		"customer":   customers,
		"inactive":   inactive,
		"active":     total - inactive,
	}, nil
}

// CreateUser membuat akun baru langsung (admin toko / superadmin / customer).
func (s *UserService) CreateUser(name, email, password, phone string, role domain.Role) (*domain.User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	if name == "" {
		return nil, errors.New("nama wajib diisi")
	}
	if !strings.Contains(email, "@") {
		return nil, errors.New("email tidak valid")
	}
	if len(password) < 8 {
		return nil, errors.New("password minimal 8 karakter")
	}
	switch role {
	case domain.RoleCustomer, domain.RoleAdmin, domain.RoleSuperAdmin:
	case "":
		role = domain.RoleCustomer
	default:
		return nil, errors.New("role tidak valid")
	}
	if _, err := s.userRepo.FindByEmail(email); err == nil {
		return nil, ErrEmailTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	hash, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}
	user := &domain.User{Name: name, Email: email, PasswordHash: hash, Phone: phone, Role: role, IsActive: true}
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUserRole mengubah role user; superadmin tidak bisa mengubah role dirinya sendiri.
func (s *UserService) UpdateUserRole(actorID, targetID uuid.UUID, role domain.Role) (*domain.User, error) {
	switch role {
	case domain.RoleCustomer, domain.RoleAdmin, domain.RoleSuperAdmin:
	default:
		return nil, errors.New("role tidak valid")
	}
	if actorID == targetID {
		return nil, ErrSelfAction
	}
	user, err := s.userRepo.FindByID(targetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	user.Role = role
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUserStatus mengaktifkan/menonaktifkan akun; akun nonaktif ditolak login
// (AuthService.Login) dan seluruh refresh token-nya dicabut di sini.
func (s *UserService) UpdateUserStatus(actorID, targetID uuid.UUID, active bool) (*domain.User, error) {
	if actorID == targetID && !active {
		return nil, ErrSelfAction
	}
	user, err := s.userRepo.FindByID(targetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	user.IsActive = active
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}
	if !active {
		if err := s.userRepo.RevokeAllRefreshTokens(targetID); err != nil {
			return nil, err
		}
	}
	return user, nil
}

// DeleteUser menghapus akun; superadmin tidak bisa menghapus akunnya sendiri.
func (s *UserService) DeleteUser(actorID, targetID uuid.UUID) error {
	if actorID == targetID {
		return ErrSelfAction
	}
	if _, err := s.userRepo.FindByID(targetID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	return s.userRepo.Delete(targetID)
}
