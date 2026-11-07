package handler

import (
	"net/http"

	"ecommerce-api/internal/service"
	"ecommerce-api/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CategoryHandler struct{ svc *service.ProductService }

func NewCategoryHandler(svc *service.ProductService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

type CategoryRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
}

func (h *CategoryHandler) ListCategories(c *gin.Context) {
	list, err := h.svc.ListCategories()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "daftar kategori", list)
}

func (h *CategoryHandler) GetCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id kategori tidak valid")
		return
	}
	cat, err := h.svc.GetCategory(id)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "kategori tidak ditemukan")
		return
	}
	utils.Success(c, http.StatusOK, "kategori ditemukan", cat)
}

func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req CategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	cat, err := h.svc.CreateCategory(service.CategoryInput{Name: req.Name, Description: req.Description, ImageURL: req.ImageURL})
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusCreated, "kategori dibuat", cat)
}

func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id kategori tidak valid")
		return
	}
	var req CategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	cat, err := h.svc.UpdateCategory(id, service.CategoryInput{Name: req.Name, Description: req.Description, ImageURL: req.ImageURL})
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "kategori diperbarui", cat)
}

func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id kategori tidak valid")
		return
	}
	if err := h.svc.DeleteCategory(id); err != nil {
		utils.Error(c, http.StatusNotFound, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "kategori dihapus", nil)
}
