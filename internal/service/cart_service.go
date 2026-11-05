package service

import (
	"errors"

	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CartService struct {
	cartRepo    *repository.CartRepository
	productRepo *repository.ProductRepository
}

func NewCartService(cartRepo *repository.CartRepository, productRepo *repository.ProductRepository) *CartService {
	return &CartService{cartRepo: cartRepo, productRepo: productRepo}
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
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, errors.New("produk tidak ditemukan")
	}
	if !product.IsActive {
		return nil, errors.New("produk tidak aktif")
	}
	cart, err := s.cartRepo.GetOrCreateByUserID(userID)
	if err != nil {
		return nil, err
	}
	existing, err := s.cartRepo.FindItem(cart.ID, productID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	existingQty := 0
	if err == nil {
		existingQty = existing.Quantity
	}
	if product.Stock < existingQty+qty {
		return nil, errors.New("stok tidak mencukupi")
	}
	return s.cartRepo.AddOrUpdateItem(cart.ID, productID, qty, product.Price)
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
