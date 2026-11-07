package handler

import (
	"net/http"
	"strconv"

	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/service"
	"ecommerce-api/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProductHandler struct{ svc *service.ProductService }

func NewProductHandler(svc *service.ProductService) *ProductHandler { return &ProductHandler{svc: svc} }

type ProductRequest struct {
	CategoryID  string                  `json:"category_id" binding:"required"`
	Name        string                  `json:"name" binding:"required,min=2,max=150"`
	Slug        string                  `json:"slug" binding:"max=170"`
	Description string                  `json:"description"`
	Price       int64                   `json:"price" binding:"min=0"`
	CostPrice   int64                   `json:"cost_price" binding:"min=0"`
	Stock       int                     `json:"stock" binding:"min=0"`
	SKU         string                  `json:"sku" binding:"required,max=50"`
	ImageURL    string                  `json:"image_url"`
	WeightGram  int                     `json:"weight_gram" binding:"min=0"`
	IsActive    *bool                   `json:"is_active"`
	IsAvailable *bool                   `json:"is_available"`
	Modifiers   []domain.ModifierGroup  `json:"modifiers"`
}

func (h *ProductHandler) ListProducts(c *gin.Context) {
	page, limit, _ := utils.ParsePagination(c)
	minPrice, _ := strconv.ParseInt(c.DefaultQuery("min_price", "0"), 10, 64)
	maxPrice, _ := strconv.ParseInt(c.DefaultQuery("max_price", "0"), 10, 64)
	products, total, err := h.svc.ListProducts(c.Query("search"), c.Query("category_id"), minPrice, maxPrice, page, limit)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if products == nil {
		products = []domain.Product{}
	}
	utils.SuccessWithMeta(c, http.StatusOK, "daftar produk", products, utils.NewPaginationMeta(page, limit, total))
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	raw := c.Param("id")
	if id, err := uuid.Parse(raw); err == nil {
		p, err := h.svc.GetProduct(id)
		if err != nil {
			utils.Error(c, http.StatusNotFound, "produk tidak ditemukan")
			return
		}
		utils.Success(c, http.StatusOK, "produk ditemukan", p)
		return
	}
	utils.Error(c, http.StatusBadRequest, "id produk tidak valid")
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req ProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	catID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "category_id tidak valid")
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	p, err := h.svc.CreateProduct(service.ProductInput{
		CategoryID: catID, Name: req.Name, Slug: req.Slug, Description: req.Description,
		Price: req.Price, CostPrice: req.CostPrice, Stock: req.Stock, SKU: req.SKU, ImageURL: req.ImageURL,
		WeightGram: req.WeightGram, IsActive: isActive, IsAvailable: req.IsAvailable,
		Modifiers: req.Modifiers,
	})
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusCreated, "produk dibuat", p)
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id produk tidak valid")
		return
	}
	var req ProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	existing, err := h.svc.GetProduct(id)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "produk tidak ditemukan")
		return
	}
	catID := existing.CategoryID
	if req.CategoryID != "" {
		parsed, err := uuid.Parse(req.CategoryID)
		if err != nil {
			utils.Error(c, http.StatusBadRequest, "category_id tidak valid")
			return
		}
		catID = parsed
	}
	isActive := existing.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	name := existing.Name
	if req.Name != "" {
		name = req.Name
	}
	sku := existing.SKU
	if req.SKU != "" {
		sku = req.SKU
	}
	p, err := h.svc.UpdateProduct(id, service.ProductInput{
		CategoryID: catID, Name: name, Slug: req.Slug, Description: req.Description,
		Price: req.Price, CostPrice: req.CostPrice, Stock: req.Stock, SKU: sku, ImageURL: req.ImageURL,
		WeightGram: req.WeightGram, IsActive: isActive, IsAvailable: req.IsAvailable,
		Modifiers: req.Modifiers,
	})
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "produk diperbarui", p)
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id produk tidak valid")
		return
	}
	if err := h.svc.DeleteProduct(id); err != nil {
		utils.Error(c, http.StatusNotFound, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "produk dihapus", nil)
}

type AvailabilityRequest struct {
	IsAvailable bool `json:"is_available"`
}

// SetAvailability adalah toggle 1-klik "Habis Hari Ini" (admin/superadmin):
// mengubah is_available tanpa menyentuh stok.
func (h *ProductHandler) SetAvailability(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id produk tidak valid")
		return
	}
	var req AvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.svc.SetAvailability(id, req.IsAvailable)
	if err != nil {
		utils.Error(c, http.StatusNotFound, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "ketersediaan produk diperbarui", p)
}
