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
		{Name: "Electronics", Slug: "electronics", Description: "Perangkat elektronik"},
		{Name: "Fashion", Slug: "fashion", Description: "Pakaian dan aksesoris"},
		{Name: "Home & Living", Slug: "home-living", Description: "Perabot rumah tangga"},
		{Name: "Books", Slug: "books", Description: "Buku dan literatur"},
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
		{CategoryID: catBySlug["electronics"].ID, Name: "Laptop Gaming", Slug: "laptop-gaming", Description: "Laptop gaming performa tinggi", Price: 15000000, Stock: 10, SKU: "ELEC-LAP-001", WeightGram: 2500, IsActive: true},
		{CategoryID: catBySlug["electronics"].ID, Name: "Smartphone", Slug: "smartphone", Description: "Smartphone layar AMOLED", Price: 5000000, Stock: 25, SKU: "ELEC-SMP-001", WeightGram: 200, IsActive: true},
		{CategoryID: catBySlug["electronics"].ID, Name: "Headphone Wireless", Slug: "headphone-wireless", Description: "Headphone wireless noise cancelling", Price: 850000, Stock: 50, SKU: "ELEC-HDP-001", WeightGram: 300, IsActive: true},
		{CategoryID: catBySlug["fashion"].ID, Name: "Kaos Polos", Slug: "kaos-polos", Description: "Kaos polos katun premium", Price: 120000, Stock: 100, SKU: "FSHN-KOS-001", WeightGram: 200, IsActive: true},
		{CategoryID: catBySlug["fashion"].ID, Name: "Sepatu Sneakers", Slug: "sepatu-sneakers", Description: "Sepatu sneakers nyaman", Price: 750000, Stock: 40, SKU: "FSHN-SPT-001", WeightGram: 800, IsActive: true},
		{CategoryID: catBySlug["home-living"].ID, Name: "Meja Kerja", Slug: "meja-kerja", Description: "Meja kerja kayu minimalis", Price: 1200000, Stock: 15, SKU: "HOME-MEJ-001", WeightGram: 15000, IsActive: true},
		{CategoryID: catBySlug["home-living"].ID, Name: "Lampu LED", Slug: "lampu-led", Description: "Lampu LED hemat energi", Price: 150000, Stock: 80, SKU: "HOME-LMP-001", WeightGram: 500, IsActive: true},
		{CategoryID: catBySlug["books"].ID, Name: "Buku Pemrograman Go", Slug: "buku-pemrograman-go", Description: "Buku belajar bahasa Go", Price: 250000, Stock: 60, SKU: "BOOK-GO-001", WeightGram: 600, IsActive: true},
	}
	for i := range products {
		if err := db.Create(&products[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

