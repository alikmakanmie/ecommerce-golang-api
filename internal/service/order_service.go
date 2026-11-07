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
	db          *gorm.DB
	orderRepo   *repository.OrderRepository
	cartRepo    *repository.CartRepository
	productRepo *repository.ProductRepository
	paymentRepo *repository.PaymentRepository
	addressRepo *repository.AddressRepository
	gateway     *payment.MockGateway
	cfg         *config.Config
	voucherSvc  *VoucherService
	loyaltySvc  *LoyaltyService
}

func NewOrderService(db *gorm.DB, orderRepo *repository.OrderRepository, cartRepo *repository.CartRepository, productRepo *repository.ProductRepository, paymentRepo *repository.PaymentRepository, addressRepo *repository.AddressRepository, gateway *payment.MockGateway, cfg *config.Config, voucherSvc *VoucherService, loyaltySvc *LoyaltyService) *OrderService {
	return &OrderService{db: db, orderRepo: orderRepo, cartRepo: cartRepo, productRepo: productRepo, paymentRepo: paymentRepo, addressRepo: addressRepo, gateway: gateway, cfg: cfg, voucherSvc: voucherSvc, loyaltySvc: loyaltySvc}
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

func (s *OrderService) Checkout(userID, addressID uuid.UUID, paymentMethod, notes, voucherCode string) (*domain.Order, error) {
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
				CostPrice:   ci.Product.CostPrice,
				UnitPrice:   ci.Price,
				UnitCost:    ci.Product.CostPrice,
				Quantity:    ci.Quantity,
				Subtotal:    sub,
			})
		}
		var shippingCost int64 = 10000
		if subtotal >= 500000 {
			shippingCost = 0
		}
		// Voucher (opsional): validasi + potongan + increment used_count
		// atomik di dalam transaksi yang sama.
		var discount int64
		var voucher *domain.Voucher
		if strings.TrimSpace(voucherCode) != "" {
			v, d, err := s.voucherSvc.ApplyInTx(tx, voucherCode, subtotal)
			if err != nil {
				return err
			}
			voucher = v
			discount = d
		}
		total := subtotal - discount + shippingCost
		orderNumber := generateOrderNumber()
		order := &domain.Order{
			UserID:          &userID,
			OrderNumber:     orderNumber,
			CustomerName:    addr.RecipientName,
			CustomerPhone:   addr.Phone,
			OrderType:       "delivery",
			AddressID:       &addressID,
			ShippingName:    addr.RecipientName,
			ShippingPhone:   addr.Phone,
			ShippingAddress: addr.FullAddress(),
			Status:          domain.OrderPending,
			Subtotal:        subtotal,
			DiscountAmount:  discount,
			ShippingCost:    shippingCost,
			Total:           total,
			PaymentMethod:   paymentMethod,
			Notes:           notes,
			Items:           orderItems,
		}
		if voucher != nil {
			order.VoucherCode = voucher.Code
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

// Guest checkout ala coffee shop: tanpa login, tanpa keranjang server.
// Pelanggan kirim items langsung; stok divalidasi + dikurangi atomik di
// dalam satu transaksi (cek stok lalu DecreaseStock dengan guard stock>=qty).
type GuestCheckoutItem struct {
	ProductID         uuid.UUID                 `json:"product_id"`
	Quantity          int                       `json:"quantity"`
	Notes             string                    `json:"notes"`
	SelectedModifiers []domain.SelectedModifier `json:"selected_modifiers"`
}

type GuestDeliveryAddress struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	Province   string `json:"province"`
	PostalCode string `json:"postal_code"`
}

type GuestCheckoutInput struct {
	Items           []GuestCheckoutItem   `json:"items"`
	CustomerName    string                `json:"customer_name"`
	CustomerPhone   string                `json:"customer_phone"`
	OrderType       string                `json:"order_type"`
	TableNumber     string                `json:"table_number"`
	PaymentMethod   string                `json:"payment_method"`
	Notes           string                `json:"notes"`
	VoucherCode     string                `json:"voucher_code"`
	DeliveryAddress *GuestDeliveryAddress `json:"delivery_address"`
}

func (s *OrderService) GuestCheckout(in GuestCheckoutInput) (*domain.Order, error) {
	if len(in.Items) == 0 {
		return nil, errors.New("items tidak boleh kosong")
	}
	in.CustomerName = strings.TrimSpace(in.CustomerName)
	if in.CustomerName == "" {
		return nil, errors.New("customer_name wajib diisi")
	}
	in.OrderType = strings.ToLower(strings.TrimSpace(in.OrderType))
	switch in.OrderType {
	case "dine_in", "takeaway", "delivery":
	default:
		return nil, errors.New("order_type harus dine_in, takeaway, atau delivery")
	}
	if in.OrderType == "dine_in" && strings.TrimSpace(in.TableNumber) == "" {
		return nil, errors.New("table_number wajib diisi untuk dine_in")
	}
	if (in.OrderType == "takeaway" || in.OrderType == "delivery") && strings.TrimSpace(in.CustomerPhone) == "" {
		return nil, errors.New("customer_phone wajib diisi untuk takeaway dan delivery")
	}
	if in.OrderType == "delivery" {
		if in.DeliveryAddress == nil ||
			strings.TrimSpace(in.DeliveryAddress.Street) == "" ||
			strings.TrimSpace(in.DeliveryAddress.City) == "" ||
			strings.TrimSpace(in.DeliveryAddress.Province) == "" ||
			strings.TrimSpace(in.DeliveryAddress.PostalCode) == "" {
			return nil, errors.New("delivery_address (street, city, province, postal_code) wajib diisi untuk delivery")
		}
	}
	paymentMethod := strings.ToLower(strings.TrimSpace(in.PaymentMethod))
	if paymentMethod == "" {
		paymentMethod = "qris"
	}
	if paymentMethod != "qris" && paymentMethod != "va" {
		return nil, errors.New("payment_method harus qris atau va")
	}

	var result *domain.Order
	err := s.db.Transaction(func(tx *gorm.DB) error {
		productRepo := s.productRepo.WithTx(tx)
		orderRepo := s.orderRepo.WithTx(tx)
		paymentRepo := s.paymentRepo.WithTx(tx)

		var subtotal int64
		orderItems := make([]domain.OrderItem, 0, len(in.Items))
		for _, it := range in.Items {
			if it.Quantity <= 0 {
				return errors.New("quantity harus lebih dari 0")
			}
			product, err := productRepo.FindByID(it.ProductID)
			if err != nil {
				return errors.New("produk tidak ditemukan")
			}
			if !product.IsActive {
				return fmt.Errorf("produk %s tidak aktif", product.Name)
			}
			if !product.IsAvailable {
				return fmt.Errorf("produk %s sedang habis hari ini", product.Name)
			}
			if product.Stock < it.Quantity {
				return fmt.Errorf("stok produk %s tidak mencukupi", product.Name)
			}
			// Harga satuan = harga dasar + delta modifier yang dipilih.
			unitPrice, unitCost, err := ApplyModifiers(product.Price, product.CostPrice, it.SelectedModifiers)
			if err != nil {
				return err
			}
			selectedJSON, err := MarshalSelectedModifiers(it.SelectedModifiers)
			if err != nil {
				return err
			}
			sub := unitPrice * int64(it.Quantity)
			subtotal += sub
			orderItems = append(orderItems, domain.OrderItem{
				ProductID:         it.ProductID,
				ProductName:       product.Name,
				Price:             unitPrice,
				CostPrice:         unitCost,
				SelectedModifiers: selectedJSON,
				UnitPrice:         unitPrice,
				UnitCost:          unitCost,
				Quantity:          it.Quantity,
				Notes:             strings.TrimSpace(it.Notes),
				Subtotal:          sub,
			})
		}
		var shippingCost int64
		var shippingAddress string
		switch in.OrderType {
		case "dine_in":
			shippingCost = 0
			shippingAddress = "Dine-In " + strings.TrimSpace(in.TableNumber)
		case "takeaway":
			shippingCost = 0
			shippingAddress = "Takeaway — diambil di tempat"
		case "delivery":
			shippingCost = 10000
			if subtotal >= 500000 {
				shippingCost = 0
			}
			a := in.DeliveryAddress
			shippingAddress = fmt.Sprintf("%s, %s, %s %s",
				strings.TrimSpace(a.Street), strings.TrimSpace(a.City),
				strings.TrimSpace(a.Province), strings.TrimSpace(a.PostalCode))
		}
		// Voucher (opsional): validasi + potongan + increment used_count
		// atomik di dalam transaksi yang sama. Ongkir gratis tetap dihitung
		// dari subtotal sebelum diskon (aturan lama).
		var discount int64
		var voucher *domain.Voucher
		if strings.TrimSpace(in.VoucherCode) != "" {
			v, d, err := s.voucherSvc.ApplyInTx(tx, in.VoucherCode, subtotal)
			if err != nil {
				return err
			}
			voucher = v
			discount = d
		}
		total := subtotal - discount + shippingCost
		orderNumber := generateOrderNumber()
		order := &domain.Order{
			UserID:          nil,
			OrderNumber:     orderNumber,
			CustomerName:    in.CustomerName,
			CustomerPhone:   strings.TrimSpace(in.CustomerPhone),
			OrderType:       in.OrderType,
			TableNumber:     strings.TrimSpace(in.TableNumber),
			ShippingName:    in.CustomerName,
			ShippingPhone:   strings.TrimSpace(in.CustomerPhone),
			ShippingAddress: shippingAddress,
			Status:          domain.OrderPending,
			Subtotal:        subtotal,
			DiscountAmount:  discount,
			ShippingCost:    shippingCost,
			Total:           total,
			PaymentMethod:   paymentMethod,
			Notes:           strings.TrimSpace(in.Notes),
			Items:           orderItems,
		}
		if voucher != nil {
			order.VoucherCode = voucher.Code
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
		order.Payment = pay
		result = order
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// TrackOrder mengembalikan order berdasarkan nomor order, tanpa login.
// Nomor order (ORD-...) berperan sebagai token rahasia pelanggan.
func (s *OrderService) TrackOrder(orderNumber string) (*domain.Order, error) {
	orderNumber = strings.TrimSpace(orderNumber)
	if orderNumber == "" {
		return nil, errors.New("order_number wajib diisi")
	}
	order, err := s.orderRepo.FindByOrderNumber(orderNumber)
	if err != nil {
		return nil, errors.New("order tidak ditemukan")
	}
	return order, nil
}

func (s *OrderService) GetOrder(userID, orderID uuid.UUID, isAdmin bool) (*domain.Order, error) {
	order, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		return nil, errors.New("order tidak ditemukan")
	}
	if !isAdmin && (order.UserID == nil || *order.UserID != userID) {
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

func (s *OrderService) UpdateOrderStatus(orderID uuid.UUID, target domain.OrderStatus, processorID *uuid.UUID, processorName string) (*domain.Order, error) {
	var updated *domain.Order
	err := s.db.Transaction(func(tx *gorm.DB) error {
		orderRepo := s.orderRepo.WithTx(tx)
		order, err := orderRepo.FindByID(orderID)
		if err != nil {
			return errors.New("order tidak ditemukan")
		}
		if !order.CanTransitionTo(target) {
			return fmt.Errorf("transisi status dari %s ke %s tidak diizinkan", order.Status, target)
		}
		if err := orderRepo.UpdateStatusByAdmin(tx, orderID, target, processorID, processorName); err != nil {
			return err
		}
		order.Status = target
		order.ProcessedByUserID = processorID
		order.ProcessedByName = processorName
		// Pesanan pertama kali lunas (paid/completed) -> +1 stamp loyalty.
		// AwardStampTx idempotent via flag loyalty_awarded di order.
		if target == domain.OrderPaid || target == domain.OrderCompleted {
			if err := s.loyaltySvc.AwardStampTx(tx, order); err != nil {
				return err
			}
		}
		updated = order
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// QueueItem adalah satu baris layar antrean TV (/queue).
type QueueItem struct {
	OrderNumber  string             `json:"order_number"`
	CustomerName string             `json:"customer_name"`
	OrderType    string             `json:"order_type"`
	TableNumber  string             `json:"table_number,omitempty"`
	Status       domain.OrderStatus `json:"status"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// GetQueue mengembalikan antrean hari ini (zona Asia/Jakarta) untuk layar TV:
// pesanan processing (sedang disiapkan) + completed yang selesai dalam
// 30 menit terakhir (siap diambil/diantar). Processing dulu, lalu completed;
// di dalam tiap status diurutkan dari yang paling lama dibuat.
func (s *OrderService) GetQueue() ([]QueueItem, error) {
	loc := wibLocation()
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	startUTC := today.UTC()
	endUTC := today.AddDate(0, 0, 1).UTC()
	readySince := time.Now().Add(-30 * time.Minute).UTC()
	var items []QueueItem
	err := s.db.Model(&domain.Order{}).
		Select("order_number, customer_name, order_type, table_number, status, created_at, updated_at").
		Where("(status = ? AND datetime(created_at) >= datetime(?) AND datetime(created_at) < datetime(?)) OR (status = ? AND datetime(updated_at) >= datetime(?))",
			domain.OrderProcessing, startUTC, endUTC, domain.OrderCompleted, readySince).
		Order("CASE WHEN status = 'processing' THEN 0 ELSE 1 END, created_at ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []QueueItem{}
	}
	return items, nil
}

func (s *OrderService) CancelOrder(userID, orderID uuid.UUID) (*domain.Order, error) {
	order, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		return nil, errors.New("order tidak ditemukan")
	}
	if order.UserID == nil || *order.UserID != userID {
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
