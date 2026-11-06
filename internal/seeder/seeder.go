package seeder

import (
	"ecommerce-api/internal/domain"
	"ecommerce-api/pkg/utils"

	"gorm.io/gorm"
)

func Run(db *gorm.DB) error {
	var count int64
	if err := db.Model(&domain.User{}).Where("email = ?", "admin@ecommerce.local").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	adminHash, err := utils.HashPassword("Admin123!")
	if err != nil {
		return err
	}
	customerHash, err := utils.HashPassword("Customer123!")
	if err != nil {
		return err
	}
	admin := &domain.User{Name: "Admin", Email: "admin@ecommerce.local", PasswordHash: adminHash, Phone: "081234567890", Role: domain.RoleAdmin, IsActive: true}
	customer := &domain.User{Name: "Customer", Email: "customer@ecommerce.local", PasswordHash: customerHash, Phone: "081234567891", Role: domain.RoleCustomer, IsActive: true}
	if err := db.Create(admin).Error; err != nil {
		return err
	}
	if err := db.Create(customer).Error; err != nil {
		return err
	}
	categories := []domain.Category{
		{Name: "Makanan Utama", Slug: "makanan-utama", Description: "Nasi, ayam, dan hidangan utama khas Slaras"},
		{Name: "Camilan & Roti", Slug: "camilan-roti", Description: "Camilan dan roti segar dari dapur Foodlab"},
		{Name: "Minuman", Slug: "minuman", Description: "Minuman segar pendamping santap"},
		{Name: "Paket & Hampers", Slug: "paket-hampers", Description: "Paket nasi box dan hampers untuk acara"},
	}
	for i := range categories {
		if err := db.Create(&categories[i]).Error; err != nil {
			return err
		}
	}
	catBySlug := map[string]domain.Category{}
	for _, c := range categories {
		catBySlug[c.Slug] = c
	}
	products := []domain.Product{
		{CategoryID: catBySlug["makanan-utama"].ID, Name: "Nasi Goreng Slaras", Slug: "nasi-goreng-slaras", Description: "Nasi goreng spesial Slaras dengan telur, ayam suwir, dan kerupuk", Price: 25000, Stock: 50, SKU: "SNF-NAS-001", WeightGram: 400, IsActive: true},
		{CategoryID: catBySlug["makanan-utama"].ID, Name: "Ayam Geprek Sambal Matah", Slug: "ayam-geprek-sambal-matah", Description: "Ayam geprek renyah dengan sambal matah segar khas Bali", Price: 18000, Stock: 60, SKU: "SNF-AYM-001", WeightGram: 300, IsActive: true},
		{CategoryID: catBySlug["makanan-utama"].ID, Name: "Rendang Sapi Slaras", Slug: "rendang-sapi-slaras", Description: "Rendang sapi dimasak perlahan hingga bumbu meresap", Price: 35000, Stock: 30, SKU: "SNF-RND-001", WeightGram: 350, IsActive: true},
		{CategoryID: catBySlug["makanan-utama"].ID, Name: "Nasi Liwet Solo", Slug: "nasi-liwet-solo", Description: "Nasi liwet gurih santan dengan ayam suwir dan labu siam", Price: 22000, Stock: 40, SKU: "SNF-LIW-001", WeightGram: 400, IsActive: true},
		{CategoryID: catBySlug["makanan-utama"].ID, Name: "Mie Ayam Slaras", Slug: "mie-ayam-slaras", Description: "Mie ayam dengan topping ayam kecap dan bakso", Price: 15000, Stock: 70, SKU: "SNF-MIE-001", WeightGram: 350, IsActive: true},
		{CategoryID: catBySlug["camilan-roti"].ID, Name: "Roti Bakar Coklat Keju", Slug: "roti-bakar-coklat-keju", Description: "Roti bakar lumer coklat dan keju, dipanggang renyah", Price: 12000, Stock: 80, SKU: "SNF-ROT-001", WeightGram: 150, IsActive: true},
		{CategoryID: catBySlug["camilan-roti"].ID, Name: "Pisang Goreng Gula Aren", Slug: "pisang-goreng-gula-aren", Description: "Pisang goreng manis gula aren dengan taburan keju", Price: 10000, Stock: 90, SKU: "SNF-PSG-001", WeightGram: 200, IsActive: true},
		{CategoryID: catBySlug["camilan-roti"].ID, Name: "Risoles Ragout Ayam", Slug: "risoles-ragout-ayam", Description: "Risoles isi ragout ayam creamy, kulit renyah", Price: 8000, Stock: 100, SKU: "SNF-RIS-001", WeightGram: 100, IsActive: true},
		{CategoryID: catBySlug["minuman"].ID, Name: "Es Teh Melati", Slug: "es-teh-melati", Description: "Teh melati diseduh segar, manisnya pas", Price: 6000, Stock: 120, SKU: "SNF-TEH-001", WeightGram: 300, IsActive: true},
		{CategoryID: catBySlug["minuman"].ID, Name: "Kopi Susu Gula Aren", Slug: "kopi-susu-gula-aren", Description: "Es kopi susu dengan gula aren asli, creamy", Price: 12000, Stock: 100, SKU: "SNF-KOP-001", WeightGram: 250, IsActive: true},
		{CategoryID: catBySlug["minuman"].ID, Name: "Es Jeruk Peras", Slug: "es-jeruk-peras", Description: "Jeruk peras asli dengan es batu, segar", Price: 7000, Stock: 110, SKU: "SNF-JER-001", WeightGram: 300, IsActive: true},
		{CategoryID: catBySlug["paket-hampers"].ID, Name: "Paket Nasi Box Slaras (isi 10)", Slug: "paket-nasi-box-slaras", Description: "Nasi box isi 10 porsi lengkap untuk rapat dan acara", Price: 250000, Stock: 20, SKU: "SNF-PKT-001", WeightGram: 4000, IsActive: true},
	}
	// Harga modal (HPP) default: 60% dari harga jual.
	for i := range products {
		products[i].CostPrice = products[i].Price * 60 / 100
		if err := db.Create(&products[i]).Error; err != nil {
			return err
		}
	}
	return nil
}
