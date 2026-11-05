package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"ecommerce-api/internal/config"
	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/payment"
	"ecommerce-api/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderService struct {
	db           *gorm.DB
	orderRepo    *repository.OrderRepository
	cartRepo     *repository.CartRepository
	productRepo  *repository.ProductRepository
	paymentRepo  *repository.PaymentRepository
	addressRepo  *repository.AddressRepository
	gateway      *payment.MockGateway
	cfg          *config.Config
}

func NewOrderService(db *gorm.DB, orderRepo *repository.OrderRepository, cartRepo *repository.CartRepository, productRepo *repository.ProductRepository, paymentRepo *repository.PaymentRepository, addressRepo *repository.AddressRepository, gateway *payment.MockGateway, cfg *config.Config) *OrderService {
	return &OrderService{db: db, orderRepo: orderRepo, cartRepo: cartRepo, productRepo: productRepo, paymentRepo: paymentRepo, addressRepo: addressRepo, gateway: gateway, cfg: cfg}
}

func generateOrderNumber() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("ORD-%s-%06d", time.Now().Format("20060102"), time.Now().UnixNano()%1000000)
	}
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return fmt.Sprintf("ORD-%s-%s", time.Now().Format("20060102"), string(b))
}

func (s *OrderService) Checkout(userID, addressID uuid.UUID, paymentMethod, notes string) (*domain.Order, error) {
	if strings.TrimSpace(paymentMethod) == "" {
		paymentMethod = "qris"
	}
	paymentMethod = strings.ToLower(strings.TrimSpace(paymentMethod))
	var result *domain.Order
	err := s.db.Transaction(func(tx *gorm.DB) error {
		cartRepo := s.cartRepo.WithTx(tx)
		productRepo := s.productRepo.WithTx(tx)
		orderRepo := s.orderRepo.WithTx(tx)
		paymentRepo := s.paymentRepo.WithTx(tx)
		addressRepo := s.addressRepo.WithTx(tx)

		cart, err := cartRepo.FindByUserID(userID)
		if err != nil {
			return errors.New("keranjang tidak ditemukan atau kosong")
		}
		// Reload items with product details inside tx.
		var items []domain.CartItem
		if err := tx.Preload("Product").Where("cart_id = ?", cart.ID).Find(&items).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return errors.New("keranjang kosong")
		}
		addr, err := addressRepo.FindByID(addressID)
		if err != nil {
			return errors.New("alamat tidak ditemukan")
		}
		if addr.UserID != userID {
			return errors.New("alamat bukan milik pengguna")
		}
		var subtotal int64
		orderItems := make([]domain.OrderItem, 0, len(items))
		for _, ci := range items {
			if !ci.Product.IsActive {
				return fmt.Errorf("produk %s tidak aktif", ci.Product.Name)
			}
			if ci.Product.Stock < ci.Quantity {
				return fmt.Errorf("stok produk %s tidak mencukupi", ci.Product.Name)
			}
			sub := ci.Price * int64(ci.Quantity)
			subtotal += sub
			orderItems = append(orderItems, domain.OrderItem{
				ProductID:   ci.ProductID,
				ProductName: ci.Product.Name,
				Price:       ci.Price,
				Quantity:    ci.Quantity,
				Subtotal:    sub,
			})
		}
		var shippingCost int64 = 10000
		if subtotal >= 500000 {
			shippingCost = 0
		}
		total := subtotal + shippingCost
		orderNumber := generateOrderNumber()
		order := &domain.Order{
			UserID:          userID,
			OrderNumber:     orderNumber,
			AddressID:       &addressID,
			ShippingName:    addr.RecipientName,
			ShippingPhone:   addr.Phone,
			ShippingAddress: addr.FullAddress(),
			Status:          domain.OrderPending,
			Subtotal:        subtotal,
			ShippingCost:    shippingCost,
			Total:           total,
			PaymentMethod:   paymentMethod,
			Notes:           notes,
			Items:           orderItems,
		}
		if err := orderRepo.Create(tx, order); err != nil {
			return err
		}
		for _, oi := range orderItems {
			if err := productRepo.DecreaseStock(tx, oi.ProductID, oi.Quantity); err != nil {
				return err
			}
		}
		txID, payURL, qr, expiresAt := s.gateway.CreateCharge(orderNumber, total, paymentMethod)
		pay := &domain.Payment{
			OrderID:       order.ID,
			TransactionID: txID,
			Gateway:       "mock",
			Method:        paymentMethod,
			Amount:        total,
			Status:        domain.PaymentPending,
			PaymentURL:    payURL,
			QRString:      qr,
			ExpiresAt:     &expiresAt,
		}
		if err := paymentRepo.Create(tx, pay); err != nil {
			return err
		}
		if err := cartRepo.ClearItems(cart.ID); err != nil {
			return err
		}
		order.Payment = pay
		result = order
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *OrderService) GetOrder(userID, orderID uuid.UUID, isAdmin bool) (*domain.Order, error) {
	order, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		return nil, errors.New("order tidak ditemukan")
	}
	if !isAdmin && order.UserID != userID {
		return nil, errors.New("order bukan milik pengguna")
	}
	return order, nil
}

func (s *OrderService) ListOrders(userID uuid.UUID, isAdmin bool, page, limit int) ([]domain.Order, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if isAdmin {
		return s.orderRepo.ListAll(page, limit)
	}
	return s.orderRepo.ListByUser(userID, page, limit)
}

func (s *OrderService) UpdateOrderStatus(orderID uuid.UUID, target domain.OrderStatus) (*domain.Order, error) {
	order, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		return nil, errors.New("order tidak ditemukan")
	}
	if !order.CanTransitionTo(target) {
		return nil, fmt.Errorf("transisi status dari %s ke %s tidak diizinkan", order.Status, target)
	}
	if err := s.orderRepo.UpdateStatus(nil, orderID, target); err != nil {
		return nil, err
	}
	order.Status = target
	return order, nil
}

func (s *OrderService) CancelOrder(userID, orderID uuid.UUID) (*domain.Order, error) {
	order, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		return nil, errors.New("order tidak ditemukan")
	}
	if order.UserID != userID {
		return nil, errors.New("order bukan milik pengguna")
	}
	if !order.CanCancel() {
		return nil, fmt.Errorf("order dengan status %s tidak dapat dibatalkan", order.Status)
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.orderRepo.WithTx(tx).UpdateStatus(tx, orderID, domain.OrderCancelled); err != nil {
			return err
		}
		for _, item := range order.Items {
			if err := s.productRepo.IncreaseStock(tx, item.ProductID, item.Quantity); err != nil {
				return err
			}
		}
		if order.Payment != nil && order.Payment.Status == domain.PaymentPending {
			now := time.Now()
			_ = now
			if err := s.paymentRepo.WithTx(tx).UpdateStatus(tx, order.Payment.ID, domain.PaymentCancel, nil); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	order.Status = domain.OrderCancelled
	return order, nil
}
