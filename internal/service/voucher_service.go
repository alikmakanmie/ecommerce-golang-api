package service

import (
	"errors"
	"strings"
	"time"

	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VoucherService struct {
	db   *gorm.DB
	repo *repository.VoucherRepository
}

func NewVoucherService(db *gorm.DB, repo *repository.VoucherRepository) *VoucherService {
	return &VoucherService{db: db, repo: repo}
}

type VoucherInput struct {
	Code           string
	DiscountType   string
	DiscountValue  int64
	MinOrderAmount int64
	MaxDiscount    int64
	Quota          int
	IsActive       bool
	ExpiresAt      *time.Time
}

// NormalizeVoucherCode menyeragamkan kode: trim + uppercase.
func NormalizeVoucherCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// CalcDiscount menghitung potongan untuk voucher yang sudah lolos validasi
// (tanpa cek aktif/expired/kuota/min — lihat Validate).
// percentage: subtotal * value / 100, dicap max_discount bila > 0.
// fixed: value, dicap max_discount bila > 0. Hasil tidak pernah melebihi subtotal.
func CalcDiscount(v *domain.Voucher, subtotal int64) int64 {
	var d int64
	switch v.DiscountType {
	case "percentage":
		d = subtotal * v.DiscountValue / 100
	case "fixed":
		d = v.DiscountValue
	default:
		return 0
	}
	if v.MaxDiscount > 0 && d > v.MaxDiscount {
		d = v.MaxDiscount
	}
	if d > subtotal {
		d = subtotal
	}
	if d < 0 {
		d = 0
	}
	return d
}

// Validate memeriksa apakah voucher bisa dipakai untuk subtotal tertentu.
// Mengembalikan voucher + nominal diskon, atau error dengan pesan yang aman
// ditampilkan ke pelanggan (dipakai endpoint publik & checkout).
func (s *VoucherService) Validate(code string, subtotal int64) (*domain.Voucher, int64, error) {
	code = NormalizeVoucherCode(code)
	if code == "" {
		return nil, 0, errors.New("kode voucher wajib diisi")
	}
	if subtotal < 0 {
		return nil, 0, errors.New("subtotal tidak valid")
	}
	v, err := s.repo.FindByCode(code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, errors.New("kode voucher tidak ditemukan")
		}
		return nil, 0, err
	}
	if !v.IsActive {
		return nil, 0, errors.New("voucher tidak aktif")
	}
	if v.ExpiresAt != nil && time.Now().After(*v.ExpiresAt) {
		return nil, 0, errors.New("voucher sudah kedaluwarsa")
	}
	if v.Quota > 0 && v.UsedCount >= v.Quota {
		return nil, 0, errors.New("kuota voucher sudah habis")
	}
	if subtotal < v.MinOrderAmount {
		return nil, 0, errors.New("belanja minimum untuk voucher ini belum tercapai")
	}
	return v, CalcDiscount(v, subtotal), nil
}

// ApplyInTx memvalidasi voucher di dalam transaksi checkout lalu menaikkan
// used_count secara atomik (guard kuota di repository). Mengembalikan
// voucher + diskon; bila kode kosong, diskon 0 tanpa error.
func (s *VoucherService) ApplyInTx(tx *gorm.DB, code string, subtotal int64) (*domain.Voucher, int64, error) {
	code = NormalizeVoucherCode(code)
	if code == "" {
		return nil, 0, nil
	}
	repo := s.repo.WithTx(tx)
	v, err := repo.FindByCode(code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, errors.New("kode voucher tidak ditemukan")
		}
		return nil, 0, err
	}
	if !v.IsActive {
		return nil, 0, errors.New("voucher tidak aktif")
	}
	if v.ExpiresAt != nil && time.Now().After(*v.ExpiresAt) {
		return nil, 0, errors.New("voucher sudah kedaluwarsa")
	}
	if v.Quota > 0 && v.UsedCount >= v.Quota {
		return nil, 0, errors.New("kuota voucher sudah habis")
	}
	if subtotal < v.MinOrderAmount {
		return nil, 0, errors.New("belanja minimum untuk voucher ini belum tercapai")
	}
	if err := repo.IncrementUsedCount(tx, v.ID); err != nil {
		return nil, 0, err
	}
	return v, CalcDiscount(v, subtotal), nil
}

func (s *VoucherService) List() ([]domain.Voucher, error) {
	return s.repo.List()
}

func (s *VoucherService) Create(in VoucherInput) (*domain.Voucher, error) {
	in.Code = NormalizeVoucherCode(in.Code)
	if err := validateVoucherInput(in); err != nil {
		return nil, err
	}
	if _, err := s.repo.FindByCode(in.Code); err == nil {
		return nil, errors.New("kode voucher sudah digunakan")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	v := &domain.Voucher{
		Code: in.Code, DiscountType: in.DiscountType, DiscountValue: in.DiscountValue,
		MinOrderAmount: in.MinOrderAmount, MaxDiscount: in.MaxDiscount,
		Quota: in.Quota, IsActive: in.IsActive, ExpiresAt: in.ExpiresAt,
	}
	if err := s.repo.Create(v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *VoucherService) Update(id uuid.UUID, in VoucherInput) (*domain.Voucher, error) {
	in.Code = NormalizeVoucherCode(in.Code)
	if err := validateVoucherInput(in); err != nil {
		return nil, err
	}
	v, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if in.Code != v.Code {
		if _, err := s.repo.FindByCode(in.Code); err == nil {
			return nil, errors.New("kode voucher sudah digunakan")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	v.Code = in.Code
	v.DiscountType = in.DiscountType
	v.DiscountValue = in.DiscountValue
	v.MinOrderAmount = in.MinOrderAmount
	v.MaxDiscount = in.MaxDiscount
	v.Quota = in.Quota
	v.IsActive = in.IsActive
	v.ExpiresAt = in.ExpiresAt
	if err := s.repo.Update(v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *VoucherService) Delete(id uuid.UUID) error {
	if _, err := s.repo.FindByID(id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

func validateVoucherInput(in VoucherInput) error {
	in.Code = NormalizeVoucherCode(in.Code)
	if in.Code == "" {
		return errors.New("code voucher wajib diisi")
	}
	switch in.DiscountType {
	case "percentage":
		if in.DiscountValue <= 0 || in.DiscountValue > 100 {
			return errors.New("discount_value percentage harus 1-100")
		}
	case "fixed":
		if in.DiscountValue <= 0 {
			return errors.New("discount_value fixed harus lebih dari 0")
		}
	default:
		return errors.New("discount_type harus percentage atau fixed")
	}
	if in.MinOrderAmount < 0 || in.MaxDiscount < 0 || in.Quota < 0 {
		return errors.New("min_order_amount, max_discount, dan quota tidak boleh negatif")
	}
	return nil
}
