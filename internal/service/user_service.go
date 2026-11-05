package service

import (
	"errors"
	"strings"

	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/repository"

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
