package service

import (
	"errors"
	"strings"

	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/repository"
	"ecommerce-api/pkg/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryInput struct {
	Name        string
	Description string
	ImageURL    string
}

type ProductInput struct {
	CategoryID  uuid.UUID
	Name        string
	Slug        string
	Description string
	Price       int64
	Stock       int
	SKU         string
	ImageURL    string
	WeightGram  int
	IsActive    bool
}

type ProductService struct {
	productRepo  *repository.ProductRepository
	categoryRepo *repository.CategoryRepository
}

func NewProductService(productRepo *repository.ProductRepository, categoryRepo *repository.CategoryRepository) *ProductService {
	return &ProductService{productRepo: productRepo, categoryRepo: categoryRepo}
}

func (s *ProductService) CreateCategory(in CategoryInput) (*domain.Category, error) {
	slug := utils.Slugify(in.Name)
	if slug == "" {
		return nil, errors.New("nama kategori tidak valid")
	}
	if _, err := s.categoryRepo.FindBySlug(slug); err == nil {
		return nil, errors.New("kategori dengan slug tersebut sudah ada")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	cat := &domain.Category{Name: strings.TrimSpace(in.Name), Slug: slug, Description: in.Description, ImageURL: in.ImageURL}
	if err := s.categoryRepo.Create(cat); err != nil {
		return nil, err
	}
	return cat, nil
}

func (s *ProductService) ListCategories() ([]domain.Category, error) {
	return s.categoryRepo.List()
}

func (s *ProductService) GetCategory(id uuid.UUID) (*domain.Category, error) {
	return s.categoryRepo.FindByID(id)
}

func (s *ProductService) UpdateCategory(id uuid.UUID, in CategoryInput) (*domain.Category, error) {
	cat, err := s.categoryRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) != "" {
		cat.Name = strings.TrimSpace(in.Name)
		cat.Slug = utils.Slugify(in.Name)
	}
	cat.Description = in.Description
	cat.ImageURL = in.ImageURL
	if err := s.categoryRepo.Update(cat); err != nil {
		return nil, err
	}
	return cat, nil
}

func (s *ProductService) DeleteCategory(id uuid.UUID) error {
	if _, err := s.categoryRepo.FindByID(id); err != nil {
		return err
	}
	return s.categoryRepo.Delete(id)
}

func (s *ProductService) CreateProduct(in ProductInput) (*domain.Product, error) {
	if _, err := s.categoryRepo.FindByID(in.CategoryID); err != nil {
		return nil, errors.New("kategori tidak ditemukan")
	}
	if in.Price < 0 {
		return nil, errors.New("harga tidak boleh negatif")
	}
	if in.Stock < 0 {
		return nil, errors.New("stok tidak boleh negatif")
	}
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		slug = utils.Slugify(in.Name)
	}
	if slug == "" {
		return nil, errors.New("nama produk tidak valid")
	}
	if _, err := s.productRepo.FindBySlug(slug); err == nil {
		return nil, errors.New("produk dengan slug tersebut sudah ada")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if strings.TrimSpace(in.SKU) == "" {
		return nil, errors.New("SKU wajib diisi")
	}
	if _, err := s.productRepo.FindBySKU(in.SKU); err == nil {
		return nil, errors.New("SKU sudah digunakan")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	p := &domain.Product{
		CategoryID:  in.CategoryID,
		Name:        strings.TrimSpace(in.Name),
		Slug:        slug,
		Description: in.Description,
		Price:       in.Price,
		Stock:       in.Stock,
		SKU:         strings.TrimSpace(in.SKU),
		ImageURL:    in.ImageURL,
		WeightGram:  in.WeightGram,
		IsActive:    true,
	}
	// IsActive defaults to true; explicit false in input is respected only via UpdateProduct.
	// CreateProduct always activates a new product unless caller later deactivates it.
	if err := s.productRepo.Create(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProductService) UpdateProduct(id uuid.UUID, in ProductInput) (*domain.Product, error) {
	p, err := s.productRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if in.CategoryID != uuid.Nil {
		if _, err := s.categoryRepo.FindByID(in.CategoryID); err != nil {
			return nil, errors.New("kategori tidak ditemukan")
		}
		p.CategoryID = in.CategoryID
	}
	if strings.TrimSpace(in.Name) != "" {
		p.Name = strings.TrimSpace(in.Name)
	}
	if strings.TrimSpace(in.Slug) != "" {
		p.Slug = strings.TrimSpace(in.Slug)
	} else if strings.TrimSpace(in.Name) != "" {
		p.Slug = utils.Slugify(in.Name)
	}
	if in.Description != "" {
		p.Description = in.Description
	}
	if in.Price >= 0 {
		p.Price = in.Price
	}
	if in.Stock >= 0 {
		p.Stock = in.Stock
	}
	if strings.TrimSpace(in.SKU) != "" {
		p.SKU = strings.TrimSpace(in.SKU)
	}
	if in.ImageURL != "" {
		p.ImageURL = in.ImageURL
	}
	if in.WeightGram >= 0 {
		p.WeightGram = in.WeightGram
	}
	p.IsActive = in.IsActive
	if err := s.productRepo.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProductService) DeleteProduct(id uuid.UUID) error {
	if _, err := s.productRepo.FindByID(id); err != nil {
		return err
	}
	return s.productRepo.Delete(id)
}

func (s *ProductService) GetProduct(id uuid.UUID) (*domain.Product, error) {
	return s.productRepo.FindByID(id)
}

func (s *ProductService) ListProducts(search, categoryID string, minPrice, maxPrice int64, page, limit int) ([]domain.Product, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	return s.productRepo.List(search, categoryID, minPrice, maxPrice, page, limit)
}
