package service

import (
	"errors"

	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/repository"
	"ecommerce-api/pkg/utils"

	"gorm.io/gorm"
)

type LoyaltyService struct {
	db   *gorm.DB
	repo *repository.LoyaltyRepository
}

func NewLoyaltyService(db *gorm.DB, repo *repository.LoyaltyRepository) *LoyaltyService {
	return &LoyaltyService{db: db, repo: repo}
}

// LoyaltyStatus adalah respons GET /loyalty/:phone.
type LoyaltyStatus struct {
	Phone          string `json:"phone"`
	Points         int    `json:"points"`
	Stamps         int    `json:"stamps"`
	RewardEligible bool   `json:"reward_eligible"`
	NextRewardAt   int    `json:"next_reward_at"`
}

// GetByPhone mengembalikan status stamp pelanggan (0 bila belum pernah
// pesan). Nomor dinormalisasi dulu agar 08xx/628xx jadi satu pelanggan.
func (s *LoyaltyService) GetByPhone(rawPhone string) (*LoyaltyStatus, error) {
	phone := utils.NormalizePhone(rawPhone)
	if phone == "" {
		return nil, errors.New("nomor HP tidak valid")
	}
	l, err := s.repo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &LoyaltyStatus{Phone: phone, Points: 0, Stamps: 0, RewardEligible: false, NextRewardAt: domain.RewardThreshold}, nil
		}
		return nil, err
	}
	return &LoyaltyStatus{
		Phone: phone, Points: l.Points, Stamps: l.Points,
		RewardEligible: l.Points >= domain.RewardThreshold,
		NextRewardAt:   domain.RewardThreshold,
	}, nil
}

// AwardStampTx memberi +1 stamp untuk order yang pertama kali lunas,
// di dalam transaksi pemanggil. Idempotent: tidak melakukan apa-apa bila
// order.LoyaltyAwarded sudah true atau nomor HP kosong. Juga menandai
// order agar pemanggilan berikutnya tidak double-count.
func (s *LoyaltyService) AwardStampTx(tx *gorm.DB, order *domain.Order) error {
	if order == nil || order.LoyaltyAwarded {
		return nil
	}
	phone := utils.NormalizePhone(order.CustomerPhone)
	if phone == "" {
		return nil
	}
	repo := s.repo.WithTx(tx)
	if _, err := repo.AddPointsTx(tx, phone, 1); err != nil {
		return err
	}
	return tx.Model(&domain.Order{}).Where("id = ?", order.ID).
		Update("loyalty_awarded", true).Error
}
