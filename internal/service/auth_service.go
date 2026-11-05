package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"ecommerce-api/internal/config"
	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/repository"
	"ecommerce-api/pkg/utils"

	"gorm.io/gorm"
)

type AuthService struct {
	userRepo *repository.UserRepository
	cfg      *config.Config
}

func NewAuthService(userRepo *repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{userRepo: userRepo, cfg: cfg}
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newRefreshTokenRaw() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *AuthService) Register(name, email, password, phone string) (*domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := s.userRepo.FindByEmail(email); err == nil {
		return nil, errors.New("email sudah terdaftar")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	hash, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}
	user := &domain.User{
		Name:         strings.TrimSpace(name),
		Email:        email,
		PasswordHash: hash,
		Phone:        phone,
		Role:         domain.RoleCustomer,
		IsActive:     true,
	}
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) issueTokens(user *domain.User) (access, refresh string, err error) {
	access, err = utils.GenerateToken(user.ID, user.Email, user.Role, s.cfg.JWTAccessSecret, s.cfg.JWTAccessTTL)
	if err != nil {
		return "", "", err
	}
	refresh, err = newRefreshTokenRaw()
	if err != nil {
		return "", "", err
	}
	rt := &domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: hashRefreshToken(refresh),
		ExpiresAt: time.Now().Add(s.cfg.JWTRefreshTTL),
	}
	if err := s.userRepo.CreateRefreshToken(rt); err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (s *AuthService) Login(email, password string) (string, string, *domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return "", "", nil, errors.New("email atau password salah")
	}
	if !user.IsActive {
		return "", "", nil, errors.New("akun tidak aktif")
	}
	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return "", "", nil, errors.New("email atau password salah")
	}
	access, refresh, err := s.issueTokens(user)
	if err != nil {
		return "", "", nil, err
	}
	return access, refresh, user, nil
}

func (s *AuthService) Refresh(refreshToken string) (string, string, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return "", "", errors.New("refresh token wajib diisi")
	}
	stored, err := s.userRepo.FindRefreshTokenByHash(hashRefreshToken(refreshToken))
	if err != nil {
		return "", "", errors.New("refresh token tidak valid")
	}
	if !stored.IsValid() {
		return "", "", errors.New("refresh token sudah kedaluwarsa atau dicabut")
	}
	user, err := s.userRepo.FindByID(stored.UserID)
	if err != nil {
		return "", "", fmt.Errorf("pengguna tidak ditemukan: %w", err)
	}
	if err := s.userRepo.RevokeRefreshToken(stored.ID); err != nil {
		return "", "", err
	}
	return s.issueTokens(user)
}

func (s *AuthService) Logout(refreshToken string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return errors.New("refresh token wajib diisi")
	}
	stored, err := s.userRepo.FindRefreshTokenByHash(hashRefreshToken(refreshToken))
	if err != nil {
		return errors.New("refresh token tidak valid")
	}
	return s.userRepo.RevokeRefreshToken(stored.ID)
}
