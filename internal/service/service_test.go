package service

import (
	"testing"
	"time"

	"ecommerce-api/internal/config"
	"ecommerce-api/internal/database"
	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/payment"
	"ecommerce-api/internal/repository"

	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *config.Config {
	t.Helper()
	cfg := &config.Config{
		AppEnv:            "test",
		AppPort:           "8080",
		DBDriver:          "sqlite",
		SQLitePath:        ":memory:",
		JWTAccessSecret:   "test-access-secret-min-32-chars-0123456789",
		JWTRefreshSecret:  "test-refresh-secret-min-32-chars-0123456789",
		JWTAccessTTL:      15 * time.Minute,
		JWTRefreshTTL:     7 * 24 * time.Hour,
		PaymentMockBaseURL: "http://localhost:8080",
	}
	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	testDB = db
	return cfg
}

type testServices struct {
	auth    *AuthService
	user    *UserService
	product *ProductService
	cart    *CartService
	order   *OrderService
	pay     *PaymentService
}

func newTestServices(t *testing.T, cfg *config.Config) *testServices {
	t.Helper()
	db := testDB
	userRepo := repository.NewUserRepository(db)
	addressRepo := repository.NewAddressRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	productRepo := repository.NewProductRepository(db)
	cartRepo := repository.NewCartRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	gateway := payment.NewMockGateway(cfg.PaymentMockBaseURL)
	return &testServices{
		auth:    NewAuthService(userRepo, cfg),
		user:    NewUserService(userRepo, addressRepo),
		product: NewProductService(productRepo, categoryRepo),
		cart:    NewCartService(cartRepo, productRepo),
		order:   NewOrderService(db, orderRepo, cartRepo, productRepo, paymentRepo, addressRepo, gateway, cfg),
		pay:     NewPaymentService(db, paymentRepo, orderRepo, productRepo),
	}
}

// testDB is set by setupTestDB; single DB per test (each test calls setupTestDB fresh).
var testDB *gorm.DB

func TestAuthRegisterLogin(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)

	user, err := svc.auth.Register("Budi", "budi@example.com", "Password123", "081234567890")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Role != domain.RoleCustomer {
		t.Fatalf("role = %s, want customer", user.Role)
	}
	if _, err := svc.auth.Register("Budi2", "budi@example.com", "Password123", ""); err == nil {
		t.Fatalf("register email duplikat harus error")
	}
	access, refresh, logged, err := svc.auth.Login("budi@example.com", "Password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if access == "" || refresh == "" {
		t.Fatalf("token kosong")
	}
	if logged.ID != user.ID {
		t.Fatalf("login user id mismatch")
	}
	if _, _, _, err := svc.auth.Login("budi@example.com", "Salah123"); err == nil {
		t.Fatalf("login password salah harus error")
	}
	// Refresh rotation
	newAccess, newRefresh, err := svc.auth.Refresh(refresh)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if newAccess == "" || newRefresh == "" {
		t.Fatalf("refresh token kosong")
	}
	// Old refresh token must now be revoked
	if _, _, err := svc.auth.Refresh(refresh); err == nil {
		t.Fatalf("refresh token lama harus sudah dicabut")
	}
}

func mustCategoryProduct(t *testing.T, svc *testServices, stock int, price int64) *domain.Product {
	t.Helper()
	cat, err := svc.product.CreateCategory(CategoryInput{Name: "Electronics", Description: "elektronik"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	p, err := svc.product.CreateProduct(ProductInput{
		CategoryID: cat.ID, Name: "Smartphone", Description: "hp", Price: price, Stock: stock,
		SKU: "TEST-SMP-001", WeightGram: 200, IsActive: true,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	return p
}

func mustCustomerWithAddress(t *testing.T, svc *testServices, email string) *domain.User {
	t.Helper()
	user, err := svc.auth.Register("Customer", email, "Password123", "081234567890")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	addr := &domain.Address{
		Label: "Rumah", RecipientName: "Customer", Phone: "081234567890",
		Street: "Jl. Merdeka 1", City: "Yogyakarta", Province: "DIY", PostalCode: "55165", IsDefault: true,
	}
	if _, err := svc.user.CreateAddress(user.ID, addr); err != nil {
		t.Fatalf("create address: %v", err)
	}
	return user
}

func mustCheckout(t *testing.T, svc *testServices, user *domain.User, product *domain.Product, qty int) *domain.Order {
	t.Helper()
	addrs, err := svc.user.ListAddresses(user.ID)
	if err != nil || len(addrs) == 0 {
		t.Fatalf("list addresses: %v len=%d", err, len(addrs))
	}
	if _, err := svc.cart.AddItem(user.ID, product.ID, qty); err != nil {
		t.Fatalf("add item: %v", err)
	}
	order, err := svc.order.Checkout(user.ID, addrs[0].ID, "qris", "")
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	return order
}

func TestCartCheckoutFlow(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)
	p := mustCategoryProduct(t, svc, 10, 100000)
	user := mustCustomerWithAddress(t, svc, "cart@example.com")

	if _, err := svc.cart.AddItem(user.ID, p.ID, 2); err != nil {
		t.Fatalf("add item: %v", err)
	}
	cart, err := svc.cart.GetCart(user.ID)
	if err != nil {
		t.Fatalf("get cart: %v", err)
	}
	if cart.Subtotal() != 200000 {
		t.Fatalf("subtotal = %d, want 200000", cart.Subtotal())
	}
	addrs, _ := svc.user.ListAddresses(user.ID)
	order, err := svc.order.Checkout(user.ID, addrs[0].ID, "qris", "")
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if order.Status != domain.OrderPending {
		t.Fatalf("order status = %s, want pending", order.Status)
	}
	if order.Total != 210000 {
		t.Fatalf("total = %d, want 210000 (200000+10000 ongkir)", order.Total)
	}
	if order.Payment == nil || order.Payment.TransactionID == "" || order.Payment.PaymentURL == "" || order.Payment.QRString == "" {
		t.Fatalf("payment tidak lengkap: %+v", order.Payment)
	}
	after, err := svc.product.GetProduct(p.ID)
	if err != nil {
		t.Fatalf("get product: %v", err)
	}
	if after.Stock != 8 {
		t.Fatalf("stok = %d, want 8", after.Stock)
	}
	cart2, err := svc.cart.GetCart(user.ID)
	if err != nil {
		t.Fatalf("get cart after checkout: %v", err)
	}
	if len(cart2.Items) != 0 {
		t.Fatalf("cart harus kosong setelah checkout, isi %d", len(cart2.Items))
	}
}

func TestPaymentWebhookSettlement(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)
	p := mustCategoryProduct(t, svc, 10, 100000)
	user := mustCustomerWithAddress(t, svc, "pay@example.com")

	order := mustCheckout(t, svc, user, p, 2)
	txID := order.Payment.TransactionID

	pay, err := svc.pay.HandleWebhook(txID, "settlement", `{"transaction_id":"`+txID+`","transaction_status":"settlement"}`)
	if err != nil {
		t.Fatalf("webhook settlement: %v", err)
	}
	if pay.Status != domain.PaymentSettlement {
		t.Fatalf("payment status = %s, want settlement", pay.Status)
	}
	updated, err := svc.order.GetOrder(user.ID, order.ID, false)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if updated.Status != domain.OrderPaid {
		t.Fatalf("order status = %s, want paid", updated.Status)
	}
	// Idempotent second settlement
	if _, err := svc.pay.HandleWebhook(txID, "settlement", "{}"); err != nil {
		t.Fatalf("webhook settlement kedua harus idempotent, err: %v", err)
	}
	updated2, _ := svc.order.GetOrder(user.ID, order.ID, false)
	if updated2.Status != domain.OrderPaid {
		t.Fatalf("order status setelah webhook kedua = %s, want paid", updated2.Status)
	}

	// Expire on second order cancels it and restores stock
	order2 := mustCheckout(t, svc, user, p, 1)
	if _, err := svc.pay.HandleWebhook(order2.Payment.TransactionID, "expire", "{}"); err != nil {
		t.Fatalf("webhook expire: %v", err)
	}
	updated3, _ := svc.order.GetOrder(user.ID, order2.ID, false)
	if updated3.Status != domain.OrderCancelled {
		t.Fatalf("order2 status = %s, want cancelled", updated3.Status)
	}
	after, _ := svc.product.GetProduct(p.ID)
	// stock: 10 -2 (order1) -1 (order2) +1 (expire restore) = 8
	if after.Stock != 8 {
		t.Fatalf("stok setelah expire = %d, want 8", after.Stock)
	}
	if _, err := svc.pay.HandleWebhook(txID, "bogus-status", "{}"); err == nil {
		t.Fatalf("webhook status tidak dikenal harus error")
	}
}

func TestCheckoutInsufficientStock(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)
	p := mustCategoryProduct(t, svc, 1, 100000)
	user := mustCustomerWithAddress(t, svc, "stock@example.com")

	// First checkout consumes the only stock.
	_ = mustCheckout(t, svc, user, p, 1)
	// Second attempt must fail: stock is 0.
	if _, err := svc.cart.AddItem(user.ID, p.ID, 1); err == nil {
		t.Fatalf("add item dengan stok 0 harus error")
	}
	after, _ := svc.product.GetProduct(p.ID)
	if after.Stock != 0 {
		t.Fatalf("stok = %d, want 0", after.Stock)
	}
}
