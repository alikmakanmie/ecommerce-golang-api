package seeder

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"ecommerce-api/internal/domain"
	"ecommerce-api/pkg/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func unsplash(id string) string {
	return fmt.Sprintf("https://images.unsplash.com/photo-%s?auto=format&fit=crop&w=800&q=80", id)
}

// Run bersifat idempotent: user/kategori/produk di-upsert per kunci unik,
// order demo hanya dibuat sekali (gerbang nama pelanggan demo).
func Run(db *gorm.DB) error {
	if err := seedUsers(db); err != nil {
		return err
	}
	if err := seedCatalog(db); err != nil {
		return err
	}
	if err := seedDemoOrders(db); err != nil {
		return err
	}
	return nil
}

func seedUsers(db *gorm.DB) error {
	seedUsers := []struct {
		name, email, password, phone string
		role                         domain.Role
	}{
		{"Superadmin Slaras", "superadmin@slaras.local", "SuperAdmin123!", "081234567890", domain.RoleSuperAdmin},
		{"Admin Slaras", "admin@ecommerce.local", "Admin123!", "081234567890", domain.RoleAdmin},
		{"Barista Slaras", "barista@slaras.local", "Barista123!", "081234567892", domain.RoleAdmin},
		{"Customer Slaras", "customer@ecommerce.local", "Customer123!", "081234567891", domain.RoleCustomer},
		{"Budi Santoso", "budi@slaras.local", "Customer123!", "081234567893", domain.RoleCustomer},
		{"Siti Rahmawati", "siti@slaras.local", "Customer123!", "081234567894", domain.RoleCustomer},
	}
	for _, su := range seedUsers {
		var count int64
		if err := db.Model(&domain.User{}).Where("email = ?", su.email).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		hash, err := utils.HashPassword(su.password)
		if err != nil {
			return err
		}
		u := &domain.User{Name: su.name, Email: su.email, PasswordHash: hash, Phone: su.phone, Role: su.role, IsActive: true}
		if err := db.Create(u).Error; err != nil {
			return err
		}
	}
	return nil
}

type seedCategory struct {
	name, slug, desc, img string
}

type seedProduct struct {
	catSlug, name, slug, desc, sku, img string
	price, costPrice                    int64
	stock, weight                       int
}

var seedCategories = []seedCategory{
	{"Signature Coffee", "signature-coffee", "Espresso & racikan kopi khas Foodlab", unsplash("1497935586351-b67a49e012bf")},
	{"Non-Coffee & Artisanal Tea", "non-coffee-tea", "Matcha, cokelat & teh racikan tanpa kopi", unsplash("1564890369478-c89ca6d9cde9")},
	{"Artisanal Pastry & Croissant", "pastry-croissant", "Roti & pastry panggang segar setiap pagi", unsplash("1509440159596-0249088772ff")},
	{"Main Course / Rice Bowl", "main-course", "Rice bowl & hidangan utama pengenyang", unsplash("1414235077428-338989a2e8c0")},
	{"Coffee Beans & Merchandise", "beans-merch", "Biji kopi & merchandise resmi Slaras", unsplash("1447933601403-0c6688de566e")},
}

var seedProducts = []seedProduct{
	{"signature-coffee", "Kopi Susu Slaras Aren", "kopi-susu-slaras-aren", "Espresso double shot, susu segar, dan gula aren asli — manis legit khas Slaras.", "SNF-SIG-001", unsplash("1495474472287-4d71bcdd2085"), 18000, 10000, 80, 250},
	{"signature-coffee", "Cold Brew Nusantara", "cold-brew-nusantara", "Biji arabika Gayo diseduh dingin 18 jam — halus, rendah asam, wangi rempah.", "SNF-SIG-002", unsplash("1498804103079-a6351b050096"), 22000, 12000, 60, 300},
	{"signature-coffee", "Espresso Double Shot", "espresso-double-shot", "Dua shot espresso pekat dari blend house Slaras — bold, crema tebal.", "SNF-SIG-003", unsplash("1514432324607-a09d9b4aefdd"), 15000, 8000, 100, 60},
	{"signature-coffee", "Charcoal Latte Hitam", "charcoal-latte-hitam", "Latte hitam pekat dari arang bambu aktif, susu creamy, dan sentuhan vanila.", "SNF-SIG-004", unsplash("1511920170033-f8396924c348"), 24000, 14000, 50, 300},
	{"signature-coffee", "Butterscotch Sea Salt Latte", "butterscotch-sea-salt-latte", "Latte butterscotch karamel dengan taburan garam laut — manis gurih seimbang.", "SNF-SIG-005", unsplash("1541167760496-1628856ab772"), 26000, 15000, 50, 300},
	{"non-coffee-tea", "Matcha Ceremonial Latte", "matcha-ceremonial-latte", "Matcha grade seremonial, susu oat, dan madu hutan — hijau pekat, menenangkan.", "SNF-NCF-001", unsplash("1572442388796-11668a67e53d"), 25000, 14000, 60, 300},
	{"non-coffee-tea", "Dark Chocolate Foodlab", "dark-chocolate-foodlab", "Cokelat hitam 70% dilelehkan dengan susu segar — pekat, tidak enek.", "SNF-NCF-002", unsplash("1551024506-0bccd828d307"), 23000, 13000, 60, 300},
	{"non-coffee-tea", "Hibiscus Berry Fizz", "hibiscus-berry-fizz", "Teh hibiscus, berry segar, dan soda — asam segar, warna merah merona.", "SNF-NCF-003", unsplash("1565958011703-44f9829ba187"), 20000, 11000, 70, 350},
	{"non-coffee-tea", "Lychee Yakult Refresher", "lychee-yakult-refresher", "Leci segar, Yakult, dan es batu — asam manis menyegarkan.", "SNF-NCF-004", unsplash("1522992319-0365e5f11656"), 19000, 10000, 70, 350},
	{"pastry-croissant", "Charcoal Croissant Almond", "charcoal-croissant-almond", "Croissant arang renyah berlapis, isian krim almond panggang.", "SNF-PAS-001", unsplash("1555507036-ab1f4038808a"), 21000, 12000, 40, 120},
	{"pastry-croissant", "Pain au Chocolat", "pain-au-chocolat", "Roti Prancis berlapis dengan batangan cokelat hitam di tengahnya.", "SNF-PAS-002", unsplash("1550617931-e17a7b70dce2"), 18000, 10000, 40, 110},
	{"pastry-croissant", "Cinnamon Roll Glaze", "cinnamon-roll-glaze", "Roti gulung kayu manis hangat dengan glaze gula lembut.", "SNF-PAS-003", unsplash("1484723091739-30a097e8f929"), 17000, 9000, 40, 130},
	{"pastry-croissant", "Basque Burnt Cheesecake", "basque-burnt-cheesecake", "Cheesecake panggang gosong khas Basque — karamel di luar, lembut di dalam.", "SNF-PAS-004", unsplash("1578985545062-69928b1d9587"), 28000, 16000, 30, 150},
	{"main-course", "Sei Sapi Sambal Luat Bowl", "sei-sapi-sambal-luat-bowl", "Sei sapi asap, sambal luat pedas khas NTT, nasi hangat, dan telur.", "SNF-MNC-001", unsplash("1512058564366-18510be2db19"), 35000, 20000, 30, 450},
	{"main-course", "Chicken Nanban Black Tare", "chicken-nanban-black-tare", "Ayam nanban renyah dengan saus tare hitam manis dan tartar.", "SNF-MNC-002", unsplash("1504674900247-0877df9cc836"), 32000, 18000, 30, 400},
	{"main-course", "Truffle Mushroom Pasta", "truffle-mushroom-pasta", "Pasta krim jamur dengan minyak truffle dan parmesan.", "SNF-MNC-003", unsplash("1546069901-ba9599a7e63c"), 38000, 22000, 25, 350},
	{"beans-merch", "Biji Kopi Arabika Gayo 250g", "biji-kopi-arabika-gayo-250g", "Biji arabika Gayo single origin, medium roast — cokelat, rempah, bersih.", "SNF-MER-001", unsplash("1461023058943-07fcbe16d735"), 85000, 50000, 20, 250},
	{"beans-merch", "Tumbler Hitam Matte Slaras", "tumbler-hitam-matte-slaras", "Tumbler stainless 500ml hitam matte, tahan panas 8 jam, logo Slaras.", "SNF-MER-002", unsplash("1509042239860-f550ce710b93"), 120000, 70000, 15, 500},
}

// legacyProductSlugs adalah menu lama (pra-kopi) yang dinonaktifkan agar
// katalog menampilkan menu kopi baru; riwayat order lama tetap utuh.
var legacyProductSlugs = []string{
	"nasi-goreng-slaras", "ayam-geprek-sambal-matah", "rendang-sapi-slaras",
	"nasi-liwet-solo", "mie-ayam-slaras", "roti-bakar-coklat-keju",
	"pisang-goreng-gula-aren", "risoles-ragout-ayam", "es-teh-melati",
	"kopi-susu-gula-aren", "es-jeruk-peras", "paket-nasi-box-slaras",
}

func seedCatalog(db *gorm.DB) error {
	catBySlug := map[string]domain.Category{}
	for _, sc := range seedCategories {
		var cat domain.Category
		if err := db.Where("slug = ?", sc.slug).First(&cat).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			cat = domain.Category{Name: sc.name, Slug: sc.slug, Description: sc.desc, ImageURL: sc.img}
			if err := db.Create(&cat).Error; err != nil {
				return err
			}
		} else {
			// Perbarui gambar/deskripsi bila seeder diperbaiki belakangan.
			cat.ImageURL = sc.img
			cat.Description = sc.desc
			if err := db.Save(&cat).Error; err != nil {
				return err
			}
		}
		catBySlug[sc.slug] = cat
	}
	for _, sp := range seedProducts {
		cat, ok := catBySlug[sp.catSlug]
		if !ok {
			return fmt.Errorf("kategori seeder tidak ditemukan: %s", sp.catSlug)
		}
		var p domain.Product
		if err := db.Where("sku = ?", sp.sku).First(&p).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			p = domain.Product{CategoryID: cat.ID, Name: sp.name, Slug: sp.slug, Description: sp.desc,
				Price: sp.price, CostPrice: sp.costPrice, Stock: sp.stock, SKU: sp.sku,
				ImageURL: sp.img, WeightGram: sp.weight, IsActive: true}
			if err := db.Create(&p).Error; err != nil {
				return err
			}
		} else {
			// Perbarui data menu; stok TIDAK di-reset (milik operasional toko).
			p.CategoryID = cat.ID
			p.Name = sp.name
			p.Slug = sp.slug
			p.Description = sp.desc
			p.Price = sp.price
			p.CostPrice = sp.costPrice
			p.ImageURL = sp.img
			p.WeightGram = sp.weight
			p.IsActive = true
			if err := db.Save(&p).Error; err != nil {
				return err
			}
		}
	}
	// Nonaktifkan menu lama (bukan hapus: order lama tetap bisa dibaca).
	if err := db.Model(&domain.Product{}).Where("slug IN ?", legacyProductSlugs).
		Update("is_active", false).Error; err != nil {
		return err
	}
	return nil
}

// demoCustomers adalah nama pelanggan pada 20 order demo (dipakai gerbang idempotensi).
var demoCustomers = []string{"Budi Santoso", "Siti Rahmawati", "Andi Wijaya", "Rina Marlina", "Dewi Lestari"}

func seedDemoOrders(db *gorm.DB) error {
	var existing int64
	if err := db.Model(&domain.Order{}).Where("customer_name IN ?", demoCustomers).Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}
	// Peta pembantu: user demo & produk baru.
	usersByEmail := map[string]domain.User{}
	var users []domain.User
	if err := db.Where("email IN ?", []string{"budi@slaras.local", "siti@slaras.local", "customer@ecommerce.local", "admin@ecommerce.local", "barista@slaras.local"}).Find(&users).Error; err != nil {
		return err
	}
	for _, u := range users {
		usersByEmail[u.Email] = u
	}
	var products []domain.Product
	if err := db.Where("is_active = ?", true).Order("sku ASC").Find(&products).Error; err != nil {
		return err
	}
	if len(products) == 0 {
		return fmt.Errorf("tidak ada produk aktif untuk order demo")
	}
	prodBySKU := map[string]domain.Product{}
	for _, p := range products {
		prodBySKU[p.SKU] = p
	}

	loc, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(loc)
	rng := rand.New(rand.NewSource(20261106))

	// 20 order selama 14 hari terakhir (minimal 1 order/hari agar grafik terisi).
	// daysAgo: 0=hari ini. status: 14 completed, 2 paid, 2 processing, 2 pending
	// (pending hanya di hari yang tetap punya order beromzet lain).
	type demoOrder struct {
		daysAgo  int
		typ      string
		method   string
		status   domain.OrderStatus
		customer string
		phone    string
	}
	plan := []demoOrder{
		{0, "dine_in", "qris", domain.OrderCompleted, "Budi Santoso", "081234567893"},
		{0, "takeaway", "qris", domain.OrderPending, "Siti Rahmawati", "081234567894"},
		{1, "delivery", "va", domain.OrderCompleted, "Andi Wijaya", "081234567895"},
		{1, "dine_in", "qris", domain.OrderPending, "Rina Marlina", "081234567896"},
		{2, "takeaway", "qris", domain.OrderCompleted, "Budi Santoso", "081234567893"},
		{3, "dine_in", "qris", domain.OrderCompleted, "Dewi Lestari", "081234567897"},
		{3, "delivery", "qris", domain.OrderPaid, "Siti Rahmawati", "081234567894"},
		{4, "takeaway", "va", domain.OrderCompleted, "Andi Wijaya", "081234567895"},
		{5, "dine_in", "va", domain.OrderCompleted, "Rina Marlina", "081234567896"},
		{5, "delivery", "qris", domain.OrderProcessing, "Budi Santoso", "081234567893"},
		{6, "dine_in", "qris", domain.OrderCompleted, "Dewi Lestari", "081234567897"},
		{7, "takeaway", "qris", domain.OrderCompleted, "Siti Rahmawati", "081234567894"},
		{7, "delivery", "va", domain.OrderPaid, "Andi Wijaya", "081234567895"},
		{8, "dine_in", "qris", domain.OrderCompleted, "Rina Marlina", "081234567896"},
		{9, "takeaway", "qris", domain.OrderCompleted, "Budi Santoso", "081234567893"},
		{10, "delivery", "qris", domain.OrderCompleted, "Dewi Lestari", "081234567897"},
		{10, "dine_in", "va", domain.OrderProcessing, "Siti Rahmawati", "081234567894"},
		{11, "takeaway", "qris", domain.OrderCompleted, "Andi Wijaya", "081234567895"},
		{12, "dine_in", "qris", domain.OrderCompleted, "Rina Marlina", "081234567896"},
		{13, "delivery", "va", domain.OrderCompleted, "Budi Santoso", "081234567893"},
	}

	// Nomor order unik: ORD-YYYYMMDD-XXXXXX (huruf acak).
	usedNumbers := map[string]bool{}
	newOrderNumber := func(t time.Time) string {
		const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		for {
			b := make([]byte, 6)
			for i := range b {
				b[i] = letters[rng.Intn(len(letters))]
			}
			n := fmt.Sprintf("ORD-%s-%s", t.Format("20060102"), string(b))
			if !usedNumbers[n] {
				usedNumbers[n] = true
				return n
			}
		}
	}

	// Rotasi item: tiap order 1-3 menu, semua produk kebagian tampil.
	skus := make([]string, 0, len(seedProducts))
	for _, sp := range seedProducts {
		skus = append(skus, sp.sku)
	}
	itemCursor := 0
	nextItems := func() []struct {
		sku string
		qty int
	} {
		n := 1 + rng.Intn(3)
		items := make([]struct {
			sku string
			qty int
		}, 0, n)
		for i := 0; i < n; i++ {
			items = append(items, struct {
				sku string
				qty int
			}{skus[itemCursor%len(skus)], 1 + rng.Intn(2)})
			itemCursor++
		}
		return items
	}

	admin := usersByEmail["admin@ecommerce.local"]
	barista := usersByEmail["barista@slaras.local"]
	processors := []domain.User{admin, barista}

	for i, d := range plan {
		createdAt := time.Date(now.Year(), now.Month(), now.Day(), 9+rng.Intn(12), rng.Intn(60), 0, 0, loc).AddDate(0, 0, -d.daysAgo)
		items := nextItems()
		var subtotal int64
		var orderItems []domain.OrderItem
		for _, it := range items {
			p := prodBySKU[it.sku]
			line := p.Price * int64(it.qty)
			subtotal += line
			orderItems = append(orderItems, domain.OrderItem{
				ProductID: p.ID, ProductName: p.Name, Price: p.Price,
				CostPrice: p.CostPrice, Quantity: it.qty, Subtotal: line,
				CreatedAt: createdAt, UpdatedAt: createdAt,
			})
		}
		var shipping int64
		table := ""
		addrText := ""
		if d.typ == "delivery" {
			shipping = 10000
			if subtotal >= 500000 {
				shipping = 0
			}
			addrText = "Jl. Demo No. 1, Yogyakarta, DIY 55165"
		}
		if d.typ == "dine_in" {
			table = fmt.Sprintf("Meja %02d", 1+rng.Intn(12))
		}
		total := subtotal + shipping

		var userID *uuid.UUID
		switch d.customer {
		case "Budi Santoso":
			if u, ok := usersByEmail["budi@slaras.local"]; ok {
				uid := u.ID
				userID = &uid
			}
		case "Siti Rahmawati":
			if u, ok := usersByEmail["siti@slaras.local"]; ok {
				uid := u.ID
				userID = &uid
			}
		}

		var procID *uuid.UUID
		procName := ""
		if d.status != domain.OrderPending {
			proc := processors[i%len(processors)]
			pid := proc.ID
			procID = &pid
			procName = proc.Name
		}

		order := &domain.Order{
			UserID: userID, OrderNumber: newOrderNumber(createdAt),
			CustomerName: d.customer, CustomerPhone: d.phone,
			OrderType: d.typ, TableNumber: table,
			ShippingName: d.customer, ShippingPhone: d.phone, ShippingAddress: addrText,
			Status: d.status, Subtotal: subtotal, ShippingCost: shipping, Total: total,
			PaymentMethod: d.method, ProcessedByUserID: procID, ProcessedByName: procName,
			CreatedAt: createdAt, UpdatedAt: createdAt,
		}
		if err := db.Create(order).Error; err != nil {
			return err
		}
		for j := range orderItems {
			orderItems[j].OrderID = order.ID
			if err := db.Create(&orderItems[j]).Error; err != nil {
				return err
			}
		}
		paid := d.status == domain.OrderPaid || d.status == domain.OrderProcessing || d.status == domain.OrderCompleted
		payStatus := domain.PaymentPending
		var paidAt *time.Time
		if paid {
			payStatus = domain.PaymentSettlement
			t := createdAt.Add(5 * time.Minute)
			paidAt = &t
		}
		exp := createdAt.Add(30 * time.Minute)
		qr := ""
		if d.method == "qris" {
			qr = "QRIS-SLARAS-" + order.OrderNumber
		}
		pay := &domain.Payment{
			OrderID: order.ID, TransactionID: "TRX-DEMO-" + strings.ReplaceAll(order.OrderNumber[4:], "-", ""),
			Gateway: "mock", Method: d.method, Amount: total, Status: payStatus,
			PaymentURL: "https://api.alikmakanmie.my.id/api/v1/mock-pay/" + "TRX-DEMO-" + strings.ReplaceAll(order.OrderNumber[4:], "-", ""),
			QRString:   qr, ExpiresAt: &exp, PaidAt: paidAt,
			CreatedAt: createdAt, UpdatedAt: createdAt,
		}
		if err := db.Create(pay).Error; err != nil {
			return err
		}
	}
	return nil
}
