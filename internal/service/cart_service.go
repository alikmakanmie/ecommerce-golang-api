package service

import (
	"errors"

	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CartService struct {
	db          *gorm.DB
	cartRepo    *repository.CartRepository
	productRepo *repository.ProductRepository
}

func NewCartService(db *gorm.DB, cartRepo *repository.CartRepository, productRepo *repository.ProductRepository) *CartService {
	return &CartService{db: db, cartRepo: cartRepo, productRepo: productRepo}
}

func (s *CartService) GetCart(userID uuid.UUID) (*domain.Cart, error) {
	cart, err := s.cartRepo.GetOrCreateByUserID(userID)
	if err != nil {
		return nil, err
	}
	return cart, nil
}

func (s *CartService) AddItem(userID, productID uuid.UUID, qty int) (*domain.CartItem, error) {
	if qty <= 0 {
		return nil, errors.New("kuantitas harus lebih dari 0")
	}
	var result *domain.CartItem
	err := s.db.Transaction(func(tx *gorm.DB) error {
		productRepo := s.productRepo.WithTx(tx)
		cartRepo := s.cartRepo.WithTx(tx)

		product, err := productRepo.FindByID(productID)
		if err != nil {
			return errors.New("produk tidak ditemukan")
		}
		if !product.IsActive {
			return errors.New("produk tidak aktif")
		}
		cart, err := cartRepo.GetOrCreateByUserID(userID)
		if err != nil {
			return err
		}
		// Cek stok cepat sebelum upsert (memberi error lebih awal pada
		// kasus umum); penjaga final yang benar-benar aman ada di bawah,
		// sesudah upsert, karena cek ini sendiri masih bisa balapan.
		existing, err := cartRepo.FindItem(cart.ID, productID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		existingQty := 0
		if err == nil {
			existingQty = existing.Quantity
		}
		if product.Stock < existingQty+qty {
			return errors.New("stok tidak mencukupi")
		}
		// Increment atomik di database (ON CONFLICT DO UPDATE).
		item, err := cartRepo.UpsertItemAtomic(cart.ID, productID, qty, product.Price)
		if err != nil {
			return err
		}
		// Penjaga stok atomik: upsert sudah commit ke dalam transaksi ini
		// dan baris yang berkonflik dikunci sampai transaksi selesai, jadi
		// kuantitas yang dibaca di sini adalah nilai final yang benar.
		// Bila melebihi stok, kembalikan error agar transaksi di-rollback.
		if item.Quantity > product.Stock {
			return errors.New("stok tidak mencukupi")
		}
		result = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *CartService) UpdateItemQty(userID, itemID uuid.UUID, qty int) error {
	if qty <= 0 {
		return errors.New("kuantitas harus lebih dari 0")
	}
	cart, err := s.cartRepo.FindByUserID(userID)
	if err != nil {
		return errors.New("keranjang tidak ditemukan")
	}
	item, err := s.cartRepo.FindItemByID(itemID)
	if err != nil {
		return errors.New("item keranjang tidak ditemukan")
	}
	if item.CartID != cart.ID {
		return errors.New("item bukan milik pengguna")
	}
	if item.Product.Stock < qty {
		return errors.New("stok tidak mencukupi")
	}
	return s.cartRepo.UpdateItemQty(itemID, qty)
}

func (s *CartService) RemoveItem(userID, itemID uuid.UUID) error {
	cart, err := s.cartRepo.FindByUserID(userID)
	if err != nil {
		return errors.New("keranjang tidak ditemukan")
	}
	item, err := s.cartRepo.FindItemByID(itemID)
	if err != nil {
		return errors.New("item keranjang tidak ditemukan")
	}
	if item.CartID != cart.ID {
		return errors.New("item bukan milik pengguna")
	}
	return s.cartRepo.RemoveItem(itemID)
}

func (s *CartService) ClearCart(userID uuid.UUID) error {
	cart, err := s.cartRepo.FindByUserID(userID)
	if err != nil {
		return errors.New("keranjang tidak ditemukan")
	}
	return s.cartRepo.ClearItems(cart.ID)
}
