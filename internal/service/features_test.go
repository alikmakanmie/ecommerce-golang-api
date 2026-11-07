package service

import (
	"testing"
	"time"

	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/seeder"
	"ecommerce-api/pkg/utils"

	"github.com/google/uuid"
)

// mustVoucher membuat voucher tes (gagal bila error).
func mustVoucher(t *testing.T, svc *testServices, in VoucherInput) *domain.Voucher {
	t.Helper()
	v, err := svc.voucher.Create(in)
	if err != nil {
		t.Fatalf("create voucher %s: %v", in.Code, err)
	}
	return v
}

func mustGuestOrder(t *testing.T, svc *testServices, in GuestCheckoutInput) *domain.Order {
	t.Helper()
	o, err := svc.order.GuestCheckout(in)
	if err != nil {
		t.Fatalf("guest checkout: %v", err)
	}
	return o
}

func setOrderStatus(t *testing.T, svc *testServices, id uuid.UUID, st domain.OrderStatus) {
	t.Helper()
	if _, err := svc.order.UpdateOrderStatus(id, st, nil, ""); err != nil {
		t.Fatalf("set status %s: %v", st, err)
	}
}

// (a) Perhitungan harga modifier: unit_price/unit_cost + penolakan delta negatif.
func TestModifierPricing(t *testing.T) {
	unitPrice, unitCost, err := ApplyModifiers(18000, 10000, []domain.SelectedModifier{
		{Name: "Iced"},
		{Name: "Less Sugar"},
		{Name: "Extra Espresso Shot", PriceDelta: 5000, CostDelta: 2500},
		{Name: "Upgrade Oat Milk", PriceDelta: 8000, CostDelta: 4500},
	})
	if err != nil {
		t.Fatalf("ApplyModifiers: %v", err)
	}
	if unitPrice != 31000 {
		t.Errorf("unit_price = %d, want 31000 (18000+5000+8000)", unitPrice)
	}
	if unitCost != 17000 {
		t.Errorf("unit_cost = %d, want 17000 (10000+2500+4500)", unitCost)
	}
	// Tanpa modifier: harga dasar utuh.
	up, uc, err := ApplyModifiers(18000, 10000, nil)
	if err != nil || up != 18000 || uc != 10000 {
		t.Errorf("tanpa modifier: %d/%d err=%v, want 18000/10000", up, uc, err)
	}
	// Delta negatif harus ditolak.
	if _, _, err := ApplyModifiers(18000, 10000, []domain.SelectedModifier{{Name: "Nakal", PriceDelta: -1000}}); err == nil {
		t.Errorf("delta negatif harus error")
	}
	if _, _, err := ApplyModifiers(18000, 10000, []domain.SelectedModifier{{Name: "", PriceDelta: 1000}}); err == nil {
		t.Errorf("modifier tanpa nama harus error")
	}
}

// (b) Validasi voucher: berhasil, cap max_discount, di bawah min,
// expired, quota habis, kode tidak dikenal, normalisasi huruf kecil.
func TestVoucherValidation(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)

	mustVoucher(t, svc, VoucherInput{Code: "SLARASHEMAT", DiscountType: "percentage", DiscountValue: 20, MinOrderAmount: 30000, MaxDiscount: 15000, IsActive: true})
	mustVoucher(t, svc, VoucherInput{Code: "KOPIPAGI", DiscountType: "fixed", DiscountValue: 10000, MinOrderAmount: 25000, IsActive: true})
	expired := time.Now().Add(-time.Hour)
	mustVoucher(t, svc, VoucherInput{Code: "KADALUARSA", DiscountType: "fixed", DiscountValue: 5000, IsActive: true, ExpiresAt: &expired})
	mustVoucher(t, svc, VoucherInput{Code: "KUOTA1", DiscountType: "fixed", DiscountValue: 5000, Quota: 1, IsActive: true})
	mustVoucher(t, svc, VoucherInput{Code: "NONAKTIF", DiscountType: "fixed", DiscountValue: 5000, IsActive: false})

	// Percentage 20% dari 100000 = 20000 -> dicap 15000.
	if _, d, err := svc.voucher.Validate("SLARASHEMAT", 100000); err != nil || d != 15000 {
		t.Errorf("SLARASHEMAT 100000: discount=%d err=%v, want 15000", d, err)
	}
	// Percentage 20% dari 50000 = 10000 (di bawah cap).
	if _, d, err := svc.voucher.Validate("slarashemat", 50000); err != nil || d != 10000 {
		t.Errorf("slarashemat lowercase 50000: discount=%d err=%v, want 10000", d, err)
	}
	// Fixed 10000, subtotal 30000 -> 10000.
	if _, d, err := svc.voucher.Validate("KOPIPAGI", 30000); err != nil || d != 10000 {
		t.Errorf("KOPIPAGI 30000: discount=%d err=%v, want 10000", d, err)
	}
	// Di bawah min belanja.
	if _, _, err := svc.voucher.Validate("KOPIPAGI", 20000); err == nil {
		t.Errorf("KOPIPAGI 20000 harus error (min 25000)")
	}
	if _, _, err := svc.voucher.Validate("SLARASHEMAT", 29999); err == nil {
		t.Errorf("SLARASHEMAT 29999 harus error (min 30000)")
	}
	// Expired / nonaktif / tidak dikenal.
	if _, _, err := svc.voucher.Validate("KADALUARSA", 50000); err == nil {
		t.Errorf("voucher expired harus error")
	}
	if _, _, err := svc.voucher.Validate("NONAKTIF", 50000); err == nil {
		t.Errorf("voucher nonaktif harus error")
	}
	if _, _, err := svc.voucher.Validate("TIDAKADA", 50000); err == nil {
		t.Errorf("kode tidak dikenal harus error")
	}
	// Kuota 1: sekali pakai lewat checkout, validasi kedua harus gagal.
	p := mustCategoryProduct(t, svc, 10, 10000)
	mustGuestOrder(t, svc, GuestCheckoutInput{
		Items: []GuestCheckoutItem{{ProductID: p.ID, Quantity: 1}},
		CustomerName: "Kuota", CustomerPhone: "081234567890",
		OrderType: "takeaway", PaymentMethod: "qris", VoucherCode: "KUOTA1",
	})
	if _, _, err := svc.voucher.Validate("KUOTA1", 50000); err == nil {
		t.Errorf("KUOTA1 sesudah 1x pakai harus error (kuota habis)")
	}
}

// (c) Toggle availability: produk habis tidak bisa di-checkout, stok utuh.
func TestToggleAvailability(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)
	p := mustCategoryProduct(t, svc, 10, 10000)

	after, err := svc.product.SetAvailability(p.ID, false)
	if err != nil {
		t.Fatalf("set availability false: %v", err)
	}
	if after.IsAvailable {
		t.Fatalf("is_available harus false")
	}
	if _, err := svc.order.GuestCheckout(GuestCheckoutInput{
		Items: []GuestCheckoutItem{{ProductID: p.ID, Quantity: 1}},
		CustomerName: "Tamu", CustomerPhone: "081234567890",
		OrderType: "takeaway", PaymentMethod: "qris",
	}); err == nil {
		t.Fatalf("checkout produk habis harus error")
	}
	// Stok tidak berkurang karena penolakan.
	var prod domain.Product
	if err := testDB.First(&prod, "id = ?", p.ID).Error; err != nil {
		t.Fatalf("baca produk: %v", err)
	}
	if prod.Stock != 10 {
		t.Fatalf("stok = %d, want 10", prod.Stock)
	}
	// Nyalakan lagi: checkout berhasil.
	if _, err := svc.product.SetAvailability(p.ID, true); err != nil {
		t.Fatalf("set availability true: %v", err)
	}
	if _, err := svc.order.GuestCheckout(GuestCheckoutInput{
		Items: []GuestCheckoutItem{{ProductID: p.ID, Quantity: 1}},
		CustomerName: "Tamu", CustomerPhone: "081234567890",
		OrderType: "takeaway", PaymentMethod: "qris",
	}); err != nil {
		t.Fatalf("checkout sesudah tersedia: %v", err)
	}
}

// (d) Queue: processing hari ini muncul, completed >30 menit tidak,
// order kemarin tidak muncul; processing diurutkan sebelum completed.
func TestOrderQueue(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)
	p := mustCategoryProduct(t, svc, 20, 10000)

	mk := func(name string) *domain.Order {
		return mustGuestOrder(t, svc, GuestCheckoutInput{
			Items: []GuestCheckoutItem{{ProductID: p.ID, Quantity: 1}},
			CustomerName: name, CustomerPhone: "081234567890",
			OrderType: "dine_in", TableNumber: "Meja 01", PaymentMethod: "qris",
		})
	}
	// o1: processing hari ini -> muncul.
	o1 := mk("Satu")
	setOrderStatus(t, svc, o1.ID, domain.OrderPaid)
	setOrderStatus(t, svc, o1.ID, domain.OrderProcessing)
	// o2: completed 40 menit lalu -> tidak muncul.
	o2 := mk("Dua")
	setOrderStatus(t, svc, o2.ID, domain.OrderPaid)
	setOrderStatus(t, svc, o2.ID, domain.OrderProcessing)
	setOrderStatus(t, svc, o2.ID, domain.OrderShipped)
	setOrderStatus(t, svc, o2.ID, domain.OrderCompleted)
	past := time.Now().Add(-40 * time.Minute)
	if err := testDB.Model(&domain.Order{}).Where("id = ?", o2.ID).
		UpdateColumns(map[string]any{"updated_at": past}).Error; err != nil {
		t.Fatalf("geser updated_at o2: %v", err)
	}
	// o3: completed barusan -> muncul (setelah processing).
	o3 := mk("Tiga")
	setOrderStatus(t, svc, o3.ID, domain.OrderPaid)
	setOrderStatus(t, svc, o3.ID, domain.OrderProcessing)
	setOrderStatus(t, svc, o3.ID, domain.OrderShipped)
	setOrderStatus(t, svc, o3.ID, domain.OrderCompleted)
	// o4: processing tapi kemarin -> tidak muncul.
	o4 := mk("Empat")
	setOrderStatus(t, svc, o4.ID, domain.OrderPaid)
	setOrderStatus(t, svc, o4.ID, domain.OrderProcessing)
	yesterday := time.Now().AddDate(0, 0, -1)
	if err := testDB.Model(&domain.Order{}).Where("id = ?", o4.ID).
		UpdateColumns(map[string]any{"created_at": yesterday, "updated_at": yesterday}).Error; err != nil {
		t.Fatalf("geser tanggal o4: %v", err)
	}
	// o5: pending hari ini -> tidak muncul.
	_ = mk("Lima")

	items, err := svc.order.GetQueue()
	if err != nil {
		t.Fatalf("GetQueue: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("queue len = %d, want 2 (o1 processing + o3 completed): %+v", len(items), items)
	}
	if items[0].OrderNumber != o1.OrderNumber || items[0].Status != domain.OrderProcessing {
		t.Errorf("queue[0] = %s/%s, want %s/processing", items[0].OrderNumber, items[0].Status, o1.OrderNumber)
	}
	if items[1].OrderNumber != o3.OrderNumber || items[1].Status != domain.OrderCompleted {
		t.Errorf("queue[1] = %s/%s, want %s/completed", items[1].OrderNumber, items[1].Status, o3.OrderNumber)
	}
	if items[0].TableNumber != "Meja 01" || items[0].CustomerName != "Satu" {
		t.Errorf("queue[0] field salah: %+v", items[0])
	}
}

// (e) Guest checkout dengan voucher + modifier: discount benar,
// used_count naik, unit_price/unit_cost tersimpan, profit analytics akurat.
func TestGuestCheckoutVoucherModifier(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)
	analytics := NewAnalyticsService(testDB)
	p := mustCategoryProduct(t, svc, 10, 18000)
	// HPP produk tes diset 10000 (CreateProduct default 60% bila 0;
	// mustCategoryProduct tidak mengisi cost -> perbaiki langsung).
	if err := testDB.Model(&domain.Product{}).Where("id = ?", p.ID).
		Update("cost_price", 10000).Error; err != nil {
		t.Fatalf("set cost_price: %v", err)
	}
	mustVoucher(t, svc, VoucherInput{Code: "SLARASHEMAT", DiscountType: "percentage", DiscountValue: 20, MinOrderAmount: 30000, MaxDiscount: 15000, IsActive: true})

	order := mustGuestOrder(t, svc, GuestCheckoutInput{
		Items: []GuestCheckoutItem{{
			ProductID: p.ID, Quantity: 2,
			SelectedModifiers: []domain.SelectedModifier{
				{Name: "Iced"},
				{Name: "Extra Espresso Shot", PriceDelta: 5000, CostDelta: 2500},
			},
		}},
		CustomerName:  "Tamu Voucher",
		CustomerPhone: "081234567890",
		OrderType:     "dine_in",
		TableNumber:   "Meja 05",
		PaymentMethod: "qris",
		VoucherCode:   "slarashemat", // huruf kecil harus tetap jalan
	})
	// unit 18000+5000=23000 x2 = 46000; diskon 20% = 9200; total 36800.
	if order.Subtotal != 46000 {
		t.Errorf("subtotal = %d, want 46000", order.Subtotal)
	}
	if order.DiscountAmount != 9200 {
		t.Errorf("discount_amount = %d, want 9200", order.DiscountAmount)
	}
	if order.VoucherCode != "SLARASHEMAT" {
		t.Errorf("voucher_code = %s, want SLARASHEMAT", order.VoucherCode)
	}
	if order.Total != 36800 {
		t.Errorf("total = %d, want 36800", order.Total)
	}
	if len(order.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(order.Items))
	}
	it := order.Items[0]
	if it.UnitPrice != 23000 || it.Price != 23000 {
		t.Errorf("unit_price/price = %d/%d, want 23000/23000", it.UnitPrice, it.Price)
	}
	if it.UnitCost != 12500 || it.CostPrice != 12500 {
		t.Errorf("unit_cost/cost_price = %d/%d, want 12500/12500", it.UnitCost, it.CostPrice)
	}
	if it.Subtotal != 46000 {
		t.Errorf("item subtotal = %d, want 46000", it.Subtotal)
	}
	var row domain.Voucher
	if err := testDB.First(&row, "code = ?", "SLARASHEMAT").Error; err != nil {
		t.Fatalf("baca voucher row: %v", err)
	}
	if row.UsedCount != 1 {
		t.Errorf("used_count = %d, want 1", row.UsedCount)
	}
	// Profit analytics: order lunas -> revenue 36800, COGS 2x12500=25000.
	setOrderStatus(t, svc, order.ID, domain.OrderPaid)
	res, err := analytics.GetProfitAnalytics("all", "", "")
	if err != nil {
		t.Fatalf("analytics: %v", err)
	}
	if res.Summary.GrossRevenue != 36800 {
		t.Errorf("gross_revenue = %d, want 36800", res.Summary.GrossRevenue)
	}
	if res.Summary.TotalCOGS != 25000 {
		t.Errorf("total_cogs = %d, want 25000", res.Summary.TotalCOGS)
	}
	if res.Summary.NetProfit != 11800 {
		t.Errorf("net_profit = %d, want 11800", res.Summary.NetProfit)
	}
}

// Loyalty: +1 stamp saat lunas (simulate-payment & patch status),
// tidak double-count, normalisasi 08xx/628xx, reward di 10 stamp.
func TestLoyaltyStamps(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)
	p := mustCategoryProduct(t, svc, 20, 10000)

	if got := utils.NormalizePhone("0812-3456-7890"); got != "6281234567890" {
		t.Errorf("NormalizePhone 08xx = %s, want 6281234567890", got)
	}
	if got := utils.NormalizePhone("+6281234567890"); got != "6281234567890" {
		t.Errorf("NormalizePhone +62 = %s, want 6281234567890", got)
	}
	if got := utils.NormalizePhone("81234567890"); got != "6281234567890" {
		t.Errorf("NormalizePhone tanpa prefix = %s, want 6281234567890", got)
	}

	mkPaid := func(phone string) *domain.Order {
		o := mustGuestOrder(t, svc, GuestCheckoutInput{
			Items: []GuestCheckoutItem{{ProductID: p.ID, Quantity: 1}},
			CustomerName: "Loyal", CustomerPhone: phone,
			OrderType: "takeaway", PaymentMethod: "qris",
		})
		if _, _, err := svc.pay.SimulatePayment(o.ID); err != nil {
			t.Fatalf("simulate payment: %v", err)
		}
		return o
	}
	// Belum pernah pesan -> 0 stamp.
	st, err := svc.loyalty.GetByPhone("081234567890")
	if err != nil || st.Points != 0 || st.RewardEligible {
		t.Fatalf("loyalty awal: %+v err=%v, want 0/false", st, err)
	}
	// 1 pesanan lunas via simulate-payment -> 1 stamp.
	mkPaid("081234567890")
	st, _ = svc.loyalty.GetByPhone("6281234567890")
	if st.Points != 1 {
		t.Errorf("stamps sesudah 1 lunas = %d, want 1", st.Points)
	}
	// Simulate kedua (idempotent) tidak menambah stamp.
	o := mustGuestOrder(t, svc, GuestCheckoutInput{
		Items: []GuestCheckoutItem{{ProductID: p.ID, Quantity: 1}},
		CustomerName: "Loyal", CustomerPhone: "081234567890",
		OrderType: "takeaway", PaymentMethod: "qris",
	})
	if _, _, err := svc.pay.SimulatePayment(o.ID); err != nil {
		t.Fatalf("simulate: %v", err)
	}
	if _, _, err := svc.pay.SimulatePayment(o.ID); err != nil {
		t.Fatalf("simulate kedua: %v", err)
	}
	st, _ = svc.loyalty.GetByPhone("081234567890")
	if st.Points != 2 {
		t.Errorf("stamps sesudah simulate ganda = %d, want 2", st.Points)
	}
	// Patch status manual ke paid (pending -> paid) juga memberi stamp.
	o2 := mustGuestOrder(t, svc, GuestCheckoutInput{
		Items: []GuestCheckoutItem{{ProductID: p.ID, Quantity: 1}},
		CustomerName: "Loyal", CustomerPhone: "081234567890",
		OrderType: "takeaway", PaymentMethod: "qris",
	})
	setOrderStatus(t, svc, o2.ID, domain.OrderPaid)
	st, _ = svc.loyalty.GetByPhone("081234567890")
	if st.Points != 3 {
		t.Errorf("stamps sesudah patch paid = %d, want 3", st.Points)
	}
	// Lanjut ke completed: tidak ada stamp kedua untuk order yang sama.
	setOrderStatus(t, svc, o2.ID, domain.OrderProcessing)
	setOrderStatus(t, svc, o2.ID, domain.OrderShipped)
	setOrderStatus(t, svc, o2.ID, domain.OrderCompleted)
	st, _ = svc.loyalty.GetByPhone("081234567890")
	if st.Points != 3 {
		t.Errorf("stamps sesudah completed = %d, want 3 (tanpa double-count)", st.Points)
	}
	// Ambang reward: 9 stamp + 1 lunas lagi = 10 -> reward_eligible.
	if err := testDB.Model(&domain.CustomerLoyalty{}).Where("customer_phone = ?", "6281234567890").
		Update("points", 9).Error; err != nil {
		t.Fatalf("set points 9: %v", err)
	}
	mkPaid("081234567890")
	st, _ = svc.loyalty.GetByPhone("081234567890")
	if st.Points != 10 || !st.RewardEligible || st.NextRewardAt != 10 {
		t.Errorf("reward: %+v, want points 10 eligible true next 10", st)
	}
	// Nomor tidak valid -> error.
	if _, err := svc.loyalty.GetByPhone(""); err == nil {
		t.Errorf("nomor kosong harus error")
	}
}

// Voucher CRUD admin: buat, ubah, hapus, dan validasi input.
func TestVoucherCRUD(t *testing.T) {
	cfg := setupTestDB(t)
	svc := newTestServices(t, cfg)

	v := mustVoucher(t, svc, VoucherInput{Code: "PROMO10", DiscountType: "percentage", DiscountValue: 10, MinOrderAmount: 0, MaxDiscount: 5000, Quota: 100, IsActive: true})
	list, err := svc.voucher.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %d err=%v, want 1", len(list), err)
	}
	updated, err := svc.voucher.Update(v.ID, VoucherInput{Code: "PROMO10", DiscountType: "percentage", DiscountValue: 25, MinOrderAmount: 0, MaxDiscount: 5000, Quota: 100, IsActive: false})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.DiscountValue != 25 || updated.IsActive {
		t.Errorf("update salah: %+v", updated)
	}
	if _, _, err := svc.voucher.Validate("PROMO10", 100000); err == nil {
		t.Errorf("voucher nonaktif sesudah update harus error")
	}
	if err := svc.voucher.Delete(v.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, _ = svc.voucher.List()
	if len(list) != 0 {
		t.Errorf("list sesudah delete = %d, want 0", len(list))
	}
	// Input tidak valid.
	if _, err := svc.voucher.Create(VoucherInput{Code: "X", DiscountType: "percentage", DiscountValue: 150}); err == nil {
		t.Errorf("percentage >100 harus error")
	}
	if _, err := svc.voucher.Create(VoucherInput{Code: "X", DiscountType: "bogus", DiscountValue: 10}); err == nil {
		t.Errorf("discount_type bogus harus error")
	}
	// Duplikat kode.
	mustVoucher(t, svc, VoucherInput{Code: "DUPE", DiscountType: "fixed", DiscountValue: 5000, IsActive: true})
	if _, err := svc.voucher.Create(VoucherInput{Code: "dupe", DiscountType: "fixed", DiscountValue: 5000, IsActive: true}); err == nil {
		t.Errorf("kode duplikat (case-insensitive) harus error")
	}
}

// Seeder voucher: 3 voucher aktif tertanam idempotent.
func TestSeedVouchers(t *testing.T) {
	cfg := setupTestDB(t)
	_ = cfg
	db := testDB
	if err := seeder.Run(db); err != nil {
		t.Fatalf("seeder: %v", err)
	}
	var count int64
	if err := db.Model(&domain.Voucher{}).Count(&count).Error; err != nil {
		t.Fatalf("count voucher: %v", err)
	}
	if count != 3 {
		t.Fatalf("voucher seed = %d, want 3", count)
	}
	// Query tiap voucher dengan struct baru: GORM First memakai primary key
	// yang sudah terisi di struct sebagai kondisi tambahan.
	var vs domain.Voucher
	if err := db.First(&vs, "code = ?", "SLARASHEMAT").Error; err != nil {
		t.Fatalf("SLARASHEMAT tidak ada: %v", err)
	}
	if vs.DiscountType != "percentage" || vs.DiscountValue != 20 || vs.MinOrderAmount != 30000 || vs.MaxDiscount != 15000 {
		t.Errorf("SLARASHEMAT salah: %+v", vs)
	}
	var vk domain.Voucher
	if err := db.First(&vk, "code = ?", "KOPIPAGI").Error; err != nil {
		t.Fatalf("KOPIPAGI tidak ada: %v", err)
	}
	if vk.DiscountType != "fixed" || vk.DiscountValue != 10000 || vk.MinOrderAmount != 25000 {
		t.Errorf("KOPIPAGI salah: %+v", vk)
	}
	var v2 domain.Voucher
	if err := db.First(&v2, "code = ?", "FOODLAB50").Error; err != nil {
		t.Fatalf("FOODLAB50 tidak ada: %v", err)
	}
	if v2.DiscountType != "percentage" || v2.DiscountValue != 50 || v2.MaxDiscount != 20000 {
		t.Errorf("FOODLAB50 salah: %+v", v2)
	}
	// Jalankan seeder lagi: tetap 3 (idempotent).
	if err := seeder.Run(db); err != nil {
		t.Fatalf("seeder kedua: %v", err)
	}
	if err := db.Model(&domain.Voucher{}).Count(&count).Error; err != nil {
		t.Fatalf("count kedua: %v", err)
	}
	if count != 3 {
		t.Errorf("voucher sesudah seeder kedua = %d, want 3", count)
	}
	// Produk minuman hasil seeder punya modifier Temperature + Add-Ons.
	var p domain.Product
	if err := db.First(&p, "sku = ?", "SNF-SIG-001").Error; err != nil {
		t.Fatalf("produk seed tidak ada: %v", err)
	}
	if !p.IsAvailable {
		t.Errorf("produk seed is_available harus true")
	}
	if len(p.Modifiers) < 4 {
		t.Fatalf("modifiers kopi susu = %d grup, want >=4: %+v", len(p.Modifiers), p.Modifiers)
	}
	found := map[string]bool{}
	for _, g := range p.Modifiers {
		found[g.Group] = true
		for _, o := range g.Options {
			if o.Name == "Extra Espresso Shot" && (o.PriceDelta != 5000 || o.CostDelta != 2500) {
				t.Errorf("Extra Espresso Shot delta salah: %+v", o)
			}
		}
	}
	for _, g := range []string{"Temperature", "Sugar Level", "Ice Level", "Add-Ons"} {
		if !found[g] {
			t.Errorf("grup modifier %s tidak ada", g)
		}
	}
}
