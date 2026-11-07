package repository

import (
	"errors"

	"ecommerce-api/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VoucherRepository struct{ db *gorm.DB }

func NewVoucherRepository(db *gorm.DB) *VoucherRepository { return &VoucherRepository{db: db} }
func (r *VoucherRepository) WithTx(tx *gorm.DB) *VoucherRepository {
	return &VoucherRepository{db: tx}
}

func (r *VoucherRepository) Create(v *domain.Voucher) error {
	// Tangkap dulu: GORM mengisi nilai default tag (default:true) kembali
	// ke struct saat Create, jadi cek is_active harus sebelum insert.
	wantInactive := !v.IsActive
	if err := r.db.Create(v).Error; err != nil {
		return err
	}
	if wantInactive {
		if err := r.db.Model(v).UpdateColumn("is_active", false).Error; err != nil {
			return err
		}
		v.IsActive = false
	}
	return nil
}

func (r *VoucherRepository) FindByID(id uuid.UUID) (*domain.Voucher, error) {
	var v domain.Voucher
	if err := r.db.First(&v, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VoucherRepository) FindByCode(code string) (*domain.Voucher, error) {
	var v domain.Voucher
	if err := r.db.First(&v, "code = ?", code).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VoucherRepository) List() ([]domain.Voucher, error) {
	var list []domain.Voucher
	if err := r.db.Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *VoucherRepository) Update(v *domain.Voucher) error {
	return r.db.Save(v).Error
}

func (r *VoucherRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&domain.Voucher{}, "id = ?", id).Error
}

// IncrementUsedCount menaikkan used_count secara atomik dengan guard kuota:
// hanya berhasil (RowsAffected == 1) bila kuota masih tersedia (atau 0 =
// tanpa batas). Dipakai di dalam transaksi checkout agar dua order serentak
// tidak bisa memakai voucher yang sama melewati kuota.
func (r *VoucherRepository) IncrementUsedCount(tx *gorm.DB, id uuid.UUID) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	res := db.Model(&domain.Voucher{}).
		Where("id = ? AND (quota <= 0 OR used_count < quota)", id).
		UpdateColumn("used_count", gorm.Expr("used_count + 1"))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("kuota voucher sudah habis")
	}
	return nil
}

type LoyaltyRepository struct{ db *gorm.DB }

func NewLoyaltyRepository(db *gorm.DB) *LoyaltyRepository { return &LoyaltyRepository{db: db} }
func (r *LoyaltyRepository) WithTx(tx *gorm.DB) *LoyaltyRepository {
	return &LoyaltyRepository{db: tx}
}

func (r *LoyaltyRepository) FindByPhone(phone string) (*domain.CustomerLoyalty, error) {
	var l domain.CustomerLoyalty
	if err := r.db.First(&l, "customer_phone = ?", phone).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// AddPointsTx menambah stamp pada baris loyalty (dibuat bila belum ada),
// di dalam transaksi pemanggil. Dipakai saat order pertama kali lunas.
func (r *LoyaltyRepository) AddPointsTx(tx *gorm.DB, phone string, points int) (*domain.CustomerLoyalty, error) {
	db := r.db
	if tx != nil {
		db = tx
	}
	var l domain.CustomerLoyalty
	if err := db.First(&l, "customer_phone = ?", phone).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		l = domain.CustomerLoyalty{CustomerPhone: phone, Points: 0}
		if err := db.Create(&l).Error; err != nil {
			return nil, err
		}
	}
	if err := db.Model(&domain.CustomerLoyalty{}).Where("id = ?", l.ID).
		UpdateColumn("points", gorm.Expr("points + ?", points)).Error; err != nil {
		return nil, err
	}
	l.Points += points
	return &l, nil
}
