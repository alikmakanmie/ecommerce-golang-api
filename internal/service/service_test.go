package service

import (
	"sync"
	"testing"
	"time"

	"ecommerce-api/internal/config"
	"ecommerce-api/internal/database"
	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/payment"
	"ecommerce-api/internal/repository"

	"github.com/google/uuid"
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
		// Limiter dibuka lebar agar tes menguji logika, bukan limiter.
		PublicRateLimit:    1000,
		PublicBurst:        1000,
		UploadMaxConcurrent: 20,
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
	voucher *VoucherService
	loyalty *LoyaltyService
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
	voucherRepo := repository.NewVoucherRepository(db)
	loyaltyRepo := repository.NewLoyaltyRepository(db)
	gateway := payment.NewMockGateway(cfg.PaymentMockBaseURL)
	voucherSvc := NewVoucherService(db, voucherRepo)
	loyaltySvc := NewLoyaltyService(db, loyaltyRepo)
	return &testServices{
		auth:    NewAuthService(userRepo, cfg),
		user:    NewUserService(userRepo, addressRepo),
		product: NewProductService(productRepo, categoryRepo),
		cart:    NewCartService(db, cartRepo, productRepo),
		order:   NewOrderService(db, orderRepo, cartRepo, productRepo, paymentRepo, addressRepo, gateway, cfg, voucherSvc, loyaltySvc),
		pay:     NewPaymentService(db, paymentRepo, orderRepo, productRepo, loyaltySvc),
		voucher: voucherSvc,
		loyalty: loyaltySvc,
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
	order, err := svc.order.Checkout(user.ID, addrs[0].ID, "qris", "", "")
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
	order, err := svc.order.Checkout(user.ID, addrs[0].ID, "qris", "", "")
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

func TestSimulatePayment(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)
	p := mustCategoryProduct(t, svc, 10, 100000)
	user := mustCustomerWithAddress(t, svc, "sim@example.com")

	order := mustCheckout(t, svc, user, p, 1)
	if order.Status != domain.OrderPending {
		t.Fatalf("order status = %s, want pending", order.Status)
	}

	// Simulasi: payment pending -> settlement, order pending -> paid.
	pay, updated, err := svc.pay.SimulatePayment(order.ID)
	if err != nil {
		t.Fatalf("simulate: %v", err)
	}
	if pay.Status != domain.PaymentSettlement {
		t.Fatalf("payment status = %s, want settlement", pay.Status)
	}
	if updated == nil || updated.Status != domain.OrderPaid {
		t.Fatalf("order status = %v, want paid", updated)
	}
	// Idempotent: simulasi kedua tidak error & tidak mengubah apa-apa.
	if _, _, err := svc.pay.SimulatePayment(order.ID); err != nil {
		t.Fatalf("simulate kedua harus idempotent, err: %v", err)
	}
	// Order tanpa pembayaran -> error.
	if _, _, err := svc.pay.SimulatePayment(uuid.New()); err == nil {
		t.Fatalf("simulate order tak dikenal harus error")
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

func TestAddItemConcurrent100(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)
	p := mustCategoryProduct(t, svc, 200, 100000)
	user, err := svc.auth.Register("Race", "race@example.com", "Password123", "081234567890")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	const n = 100
	var wg sync.WaitGroup
	errs := make(chan error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, err := svc.cart.AddItem(user.ID, p.ID, 1)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("AddItem konkuren error: %v", err)
		}
	}

	cart, err := svc.cart.GetCart(user.ID)
	if err != nil {
		t.Fatalf("get cart: %v", err)
	}
	if len(cart.Items) != 1 {
		t.Fatalf("jumlah baris item = %d, want 1 (upsert harus menggabung)", len(cart.Items))
	}
	if cart.Items[0].Quantity != n {
		t.Fatalf("quantity = %d, want %d (lost-update)", cart.Items[0].Quantity, n)
	}
}

func TestAddItemConcurrentStockLimit(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)
	p := mustCategoryProduct(t, svc, 10, 100000)
	user, err := svc.auth.Register("RaceStock", "racestock@example.com", "Password123", "081234567890")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	const n = 30
	var wg sync.WaitGroup
	var mu sync.Mutex
	succeeded := 0
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if _, err := svc.cart.AddItem(user.ID, p.ID, 1); err == nil {
				mu.Lock()
				succeeded++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if succeeded != 10 {
		t.Fatalf("AddItem sukses = %d, want 10 (stok harus dihormati)", succeeded)
	}
	cart, err := svc.cart.GetCart(user.ID)
	if err != nil {
		t.Fatalf("get cart: %v", err)
	}
	if cart.Items[0].Quantity != 10 {
		t.Fatalf("quantity = %d, want 10", cart.Items[0].Quantity)
	}
}

func TestAddItemAfterRemoveRevivesWithFreshQty(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)
	p := mustCategoryProduct(t, svc, 100, 100000)
	user, err := svc.auth.Register("Revive", "revive@example.com", "Password123", "081234567890")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	item, err := svc.cart.AddItem(user.ID, p.ID, 1)
	if err != nil {
		t.Fatalf("add pertama: %v", err)
	}
	if item.Quantity != 1 {
		t.Fatalf("qty pertama = %d, want 1", item.Quantity)
	}

	// Hapus item (soft-delete) lalu tambah lagi: qty harus 1, bukan 1+1.
	if err := svc.cart.RemoveItem(user.ID, item.ID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	item, err = svc.cart.AddItem(user.ID, p.ID, 1)
	if err != nil {
		t.Fatalf("add sesudah hapus: %v", err)
	}
	if item.Quantity != 1 {
		t.Fatalf("qty sesudah hapus+tambah = %d, want 1 (baris soft-deleted bangkit dengan qty lama)", item.Quantity)
	}

	// ClearCart (jalur checkout) lalu tambah lagi qty 2: harus 2, bukan 1+2.
	if err := svc.cart.ClearCart(user.ID); err != nil {
		t.Fatalf("clear: %v", err)
	}
	item, err = svc.cart.AddItem(user.ID, p.ID, 2)
	if err != nil {
		t.Fatalf("add sesudah clear: %v", err)
	}
	if item.Quantity != 2 {
		t.Fatalf("qty sesudah clear+tambah = %d, want 2", item.Quantity)
	}
}

func TestGuestCheckoutDineIn(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)
	p := mustCategoryProduct(t, svc, 10, 25000)

	order, err := svc.order.GuestCheckout(GuestCheckoutInput{
		Items: []GuestCheckoutItem{
			{ProductID: p.ID, Quantity: 2, Notes: "less sugar"},
		},
		CustomerName:  "Tamu DineIn",
		CustomerPhone: "",
		OrderType:     "dine_in",
		TableNumber:   "Meja 04",
		PaymentMethod: "qris",
	})
	if err != nil {
		t.Fatalf("guest checkout dine_in: %v", err)
	}
	if order.UserID != nil {
		t.Fatalf("UserID tamu harus nil, dapat %v", order.UserID)
	}
	if order.OrderType != "dine_in" || order.TableNumber != "Meja 04" {
		t.Fatalf("order_type/table = %s/%s, want dine_in/Meja 04", order.OrderType, order.TableNumber)
	}
	if order.Status != domain.OrderPending {
		t.Fatalf("status = %s, want pending", order.Status)
	}
	if order.ShippingCost != 0 {
		t.Fatalf("shipping dine_in = %d, want 0", order.ShippingCost)
	}
	if order.Total != 50000 {
		t.Fatalf("total = %d, want 50000", order.Total)
	}
	if len(order.Items) != 1 || order.Items[0].Notes != "less sugar" {
		t.Fatalf("notes item tidak tersimpan: %+v", order.Items)
	}
	if order.Payment == nil || order.Payment.QRString == "" {
		t.Fatalf("payment QRIS harus punya qr_string: %+v", order.Payment)
	}
	if order.Payment.PaymentURL == "" {
		t.Fatalf("payment_url kosong")
	}
	var after domain.Product
	if err := testDB.First(&after, "id = ?", p.ID).Error; err != nil {
		t.Fatalf("baca produk: %v", err)
	}
	if after.Stock != 8 {
		t.Fatalf("stok sesudah checkout = %d, want 8", after.Stock)
	}
	// Track publik harus menemukan order yang sama.
	tracked, err := svc.order.TrackOrder(order.OrderNumber)
	if err != nil {
		t.Fatalf("track: %v", err)
	}
	if tracked.ID != order.ID || tracked.Status != domain.OrderPending {
		t.Fatalf("track mengembalikan order yang salah: %s %s", tracked.OrderNumber, tracked.Status)
	}
}

func TestGuestCheckoutTakeaway(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)
	p := mustCategoryProduct(t, svc, 5, 12000)

	order, err := svc.order.GuestCheckout(GuestCheckoutInput{
		Items: []GuestCheckoutItem{
			{ProductID: p.ID, Quantity: 1},
		},
		CustomerName:  "Tamu Bungkus",
		CustomerPhone: "081234567890",
		OrderType:     "takeaway",
		PaymentMethod: "va",
	})
	if err != nil {
		t.Fatalf("guest checkout takeaway: %v", err)
	}
	if order.UserID != nil || order.OrderType != "takeaway" {
		t.Fatalf("order tamu salah: user=%v type=%s", order.UserID, order.OrderType)
	}
	if order.ShippingCost != 0 || order.Total != 12000 {
		t.Fatalf("shipping/total = %d/%d, want 0/12000", order.ShippingCost, order.Total)
	}
	if order.Payment == nil || order.Payment.Method != "va" || order.Payment.PaymentURL == "" {
		t.Fatalf("payment VA tidak lengkap: %+v", order.Payment)
	}
	if order.Payment.QRString != "" {
		t.Fatalf("payment VA tidak boleh punya qr_string: %s", order.Payment.QRString)
	}
}

func TestGuestCheckoutValidation(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)
	p := mustCategoryProduct(t, svc, 3, 10000)

	cases := []struct {
		name string
		in   GuestCheckoutInput
	}{
		{"items kosong", GuestCheckoutInput{CustomerName: "A", OrderType: "takeaway", CustomerPhone: "081", PaymentMethod: "qris"}},
		{"nama kosong", GuestCheckoutInput{Items: []GuestCheckoutItem{{ProductID: p.ID, Quantity: 1}}, OrderType: "takeaway", CustomerPhone: "081", PaymentMethod: "qris"}},
		{"order_type salah", GuestCheckoutInput{Items: []GuestCheckoutItem{{ProductID: p.ID, Quantity: 1}}, CustomerName: "A", CustomerPhone: "081", OrderType: "dine_out", PaymentMethod: "qris"}},
		{"dine_in tanpa meja", GuestCheckoutInput{Items: []GuestCheckoutItem{{ProductID: p.ID, Quantity: 1}}, CustomerName: "A", OrderType: "dine_in", PaymentMethod: "qris"}},
		{"takeaway tanpa phone", GuestCheckoutInput{Items: []GuestCheckoutItem{{ProductID: p.ID, Quantity: 1}}, CustomerName: "A", OrderType: "takeaway", PaymentMethod: "qris"}},
		{"delivery tanpa alamat", GuestCheckoutInput{Items: []GuestCheckoutItem{{ProductID: p.ID, Quantity: 1}}, CustomerName: "A", CustomerPhone: "081", OrderType: "delivery", PaymentMethod: "qris"}},
		{"payment_method salah", GuestCheckoutInput{Items: []GuestCheckoutItem{{ProductID: p.ID, Quantity: 1}}, CustomerName: "A", CustomerPhone: "081", OrderType: "takeaway", PaymentMethod: "cash"}},
		{"stok kurang", GuestCheckoutInput{Items: []GuestCheckoutItem{{ProductID: p.ID, Quantity: 99}}, CustomerName: "A", CustomerPhone: "081", OrderType: "takeaway", PaymentMethod: "qris"}},
		{"qty nol", GuestCheckoutInput{Items: []GuestCheckoutItem{{ProductID: p.ID, Quantity: 0}}, CustomerName: "A", CustomerPhone: "081", OrderType: "takeaway", PaymentMethod: "qris"}},
	}
	for _, tc := range cases {
		if _, err := svc.order.GuestCheckout(tc.in); err == nil {
			t.Errorf("%s: harus error, dapat sukses", tc.name)
		}
	}
	// Stok harus utuh sesudah semua penolakan (transaksi rollback).
	var after domain.Product
	if err := testDB.First(&after, "id = ?", p.ID).Error; err != nil {
		t.Fatalf("baca produk: %v", err)
	}
	if after.Stock != 3 {
		t.Fatalf("stok sesudah validasi gagal = %d, want 3", after.Stock)
	}
	// Track nomor yang tidak ada harus error.
	if _, err := svc.order.TrackOrder("ORD-20990101-ZZZZZZ"); err == nil {
		t.Errorf("track nomor palsu harus error")
	}
}

func TestProfitAnalytics(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)
	analytics := NewAnalyticsService(testDB)

	cat, err := svc.product.CreateCategory(CategoryInput{Name: "Makanan", Description: "makanan"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	mkProduct := func(name, sku string, price, cost int64, stock int) *domain.Product {
		t.Helper()
		p, err := svc.product.CreateProduct(ProductInput{
			CategoryID: cat.ID, Name: name, Price: price, CostPrice: cost,
			Stock: stock, SKU: sku, WeightGram: 200, IsActive: true,
		})
		if err != nil {
			t.Fatalf("create product %s: %v", name, err)
		}
		return p
	}
	prodA := mkProduct("Nasi Goreng", "TST-A-001", 10000, 6000, 100)
	prodB := mkProduct("Kopi Susu", "TST-B-001", 20000, 12000, 100)

	setStatus := func(orderID string, st domain.OrderStatus) {
		t.Helper()
		if err := testDB.Model(&domain.Order{}).Where("id = ?", orderID).Update("status", st).Error; err != nil {
			t.Fatalf("set status: %v", err)
		}
	}

	// Order1: dine-in qris 2xA = 20000, PAID.
	o1, err := svc.order.GuestCheckout(GuestCheckoutInput{
		Items: []GuestCheckoutItem{{ProductID: prodA.ID, Quantity: 2}},
		CustomerName: "Andi", OrderType: "dine_in", TableNumber: "Meja 04", PaymentMethod: "qris",
	})
	if err != nil {
		t.Fatalf("guest checkout 1: %v", err)
	}
	setStatus(o1.ID.String(), domain.OrderPaid)
	// Order2: takeaway va 1xB = 20000, PAID.
	o2, err := svc.order.GuestCheckout(GuestCheckoutInput{
		Items: []GuestCheckoutItem{{ProductID: prodB.ID, Quantity: 1}},
		CustomerName: "Budi", CustomerPhone: "081", OrderType: "takeaway", PaymentMethod: "va",
	})
	if err != nil {
		t.Fatalf("guest checkout 2: %v", err)
	}
	setStatus(o2.ID.String(), domain.OrderPaid)
	// Order3: delivery qris 1xA = 10000 + ongkir 10000 = 20000, PENDING.
	o3, err := svc.order.GuestCheckout(GuestCheckoutInput{
		Items: []GuestCheckoutItem{{ProductID: prodA.ID, Quantity: 1}},
		CustomerName: "Cici", CustomerPhone: "081", OrderType: "delivery", PaymentMethod: "qris",
		DeliveryAddress: &GuestDeliveryAddress{Street: "Jl. A 1", City: "Yogyakarta", Province: "DIY", PostalCode: "55165"},
	})
	if err != nil {
		t.Fatalf("guest checkout 3: %v", err)
	}
	_ = o3
	// Order4: dine-in qris 1xB = 20000, CANCELLED.
	o4, err := svc.order.GuestCheckout(GuestCheckoutInput{
		Items: []GuestCheckoutItem{{ProductID: prodB.ID, Quantity: 1}},
		CustomerName: "Dedi", OrderType: "dine_in", TableNumber: "Meja 02", PaymentMethod: "qris",
	})
	if err != nil {
		t.Fatalf("guest checkout 4: %v", err)
	}
	setStatus(o4.ID.String(), domain.OrderCancelled)

	res, err := analytics.GetProfitAnalytics("all", "", "")
	if err != nil {
		t.Fatalf("analytics all: %v", err)
	}
	s := res.Summary
	if s.GrossRevenue != 40000 {
		t.Errorf("gross_revenue = %d, want 40000", s.GrossRevenue)
	}
	if s.TotalCOGS != 24000 {
		t.Errorf("total_cogs = %d, want 24000", s.TotalCOGS)
	}
	if s.NetProfit != 16000 {
		t.Errorf("net_profit = %d, want 16000", s.NetProfit)
	}
	if s.ProfitMarginPercent != 40 {
		t.Errorf("margin = %v, want 40", s.ProfitMarginPercent)
	}
	if s.TotalPaidOrders != 2 {
		t.Errorf("total_paid_orders = %d, want 2", s.TotalPaidOrders)
	}
	if s.AverageOrderValue != 20000 {
		t.Errorf("aov = %d, want 20000", s.AverageOrderValue)
	}
	if s.PendingOrCancelledCount != 2 {
		t.Errorf("pending_or_cancelled = %d, want 2", s.PendingOrCancelledCount)
	}
	if len(res.DailySeries) != 1 {
		t.Fatalf("daily_series len = %d, want 1", len(res.DailySeries))
	}
	d := res.DailySeries[0]
	if d.Revenue != 40000 || d.Profit != 16000 || d.OrdersCount != 2 {
		t.Errorf("daily = %+v, want revenue 40000 profit 16000 orders 2", d)
	}
	byType := map[string]OrderTypeBreakdown{}
	for _, b := range res.ByOrderType {
		byType[b.OrderType] = b
	}
	if byType["dine_in"].Revenue != 20000 || byType["dine_in"].OrdersCount != 1 {
		t.Errorf("dine_in breakdown = %+v, want 20000/1", byType["dine_in"])
	}
	if byType["takeaway"].Revenue != 20000 || byType["takeaway"].OrdersCount != 1 {
		t.Errorf("takeaway breakdown = %+v, want 20000/1", byType["takeaway"])
	}
	if byType["delivery"].Revenue != 0 || byType["delivery"].OrdersCount != 0 {
		t.Errorf("delivery breakdown = %+v, want 0/0 (pending tidak dihitung)", byType["delivery"])
	}
	byPay := map[string]PaymentBreakdown{}
	for _, b := range res.ByPaymentMethod {
		byPay[b.PaymentMethod] = b
	}
	if byPay["qris"].Revenue != 20000 || byPay["qris"].OrdersCount != 1 {
		t.Errorf("qris breakdown = %+v, want 20000/1", byPay["qris"])
	}
	if byPay["va"].Revenue != 20000 || byPay["va"].OrdersCount != 1 {
		t.Errorf("va breakdown = %+v, want 20000/1", byPay["va"])
	}
	if len(res.TopProducts) != 2 {
		t.Fatalf("top_products len = %d, want 2", len(res.TopProducts))
	}
	top0 := res.TopProducts[0]
	if top0.ProductName != "Nasi Goreng" || top0.QuantitySold != 2 || top0.Revenue != 20000 || top0.Profit != 8000 {
		t.Errorf("top[0] = %+v, want Nasi Goreng qty 2 rev 20000 profit 8000", top0)
	}
	top1 := res.TopProducts[1]
	if top1.ProductName != "Kopi Susu" || top1.QuantitySold != 1 || top1.Revenue != 20000 || top1.Profit != 8000 {
		t.Errorf("top[1] = %+v, want Kopi Susu qty 1 rev 20000 profit 8000", top1)
	}
	if len(res.RecentTransactions) != 4 {
		t.Errorf("recent len = %d, want 4", len(res.RecentTransactions))
	}
	if len(res.RecentTransactions) > 0 && res.RecentTransactions[0].OrderNumber != o4.OrderNumber {
		t.Errorf("recent[0] = %s, want %s (terbaru dulu)", res.RecentTransactions[0].OrderNumber, o4.OrderNumber)
	}

	// Period "today" harus sama (semua order dibuat hari ini).
	resToday, err := analytics.GetProfitAnalytics("today", "", "")
	if err != nil {
		t.Fatalf("analytics today: %v", err)
	}
	if resToday.Summary.GrossRevenue != 40000 || len(resToday.DailySeries) != 1 {
		t.Errorf("today: revenue %d series %d, want 40000/1", resToday.Summary.GrossRevenue, len(resToday.DailySeries))
	}

	// Rentang tanggal yang tidak berisi order: semua nol.
	resEmpty, err := analytics.GetProfitAnalytics("", "2020-01-01", "2020-01-02")
	if err != nil {
		t.Fatalf("analytics range kosong: %v", err)
	}
	if resEmpty.Summary.GrossRevenue != 0 || resEmpty.Summary.TotalPaidOrders != 0 || len(resEmpty.DailySeries) != 2 {
		t.Errorf("range kosong: %+v series %d, want 0/0/2 hari", resEmpty.Summary, len(resEmpty.DailySeries))
	}

	// Validasi input.
	if _, err := analytics.GetProfitAnalytics("bogus", "", ""); err == nil {
		t.Errorf("period bogus harus error")
	}
	if _, err := analytics.GetProfitAnalytics("", "2020-01-05", "2020-01-01"); err == nil {
		t.Errorf("end sebelum start harus error")
	}
	if _, err := analytics.GetProfitAnalytics("", "01-01-2020", "2020-01-02"); err == nil {
		t.Errorf("format tanggal salah harus error")
	}
}
