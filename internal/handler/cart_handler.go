package handler

import (
	"net/http"

	"ecommerce-api/internal/middleware"
	"ecommerce-api/internal/service"
	"ecommerce-api/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CartHandler struct{ svc *service.CartService }

func NewCartHandler(svc *service.CartService) *CartHandler { return &CartHandler{svc: svc} }

type AddCartItemRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" binding:"required,min=1"`
}

func (h *CartHandler) GetCart(c *gin.Context) {
	id, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "tidak terautentikasi")
		return
	}
	cart, err := h.svc.GetCart(id)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "keranjang ditemukan", gin.H{"cart": cart, "subtotal": cart.Subtotal(), "total_items": cart.TotalItems()})
}

func (h *CartHandler) AddItem(c *gin.Context) {
	id, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "tidak terautentikasi")
		return
	}
	var req AddCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	pid, err := uuid.Parse(req.ProductID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "product_id tidak valid")
		return
	}
	item, err := h.svc.AddItem(id, pid, req.Quantity)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusCreated, "item ditambahkan ke keranjang", item)
}

func (h *CartHandler) UpdateItemQty(c *gin.Context) {
	id, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "tidak terautentikasi")
		return
	}
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id item tidak valid")
		return
	}
	var req UpdateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.UpdateItemQty(id, itemID, req.Quantity); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "kuantitas diperbarui", nil)
}

func (h *CartHandler) RemoveItem(c *gin.Context) {
	id, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "tidak terautentikasi")
		return
	}
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id item tidak valid")
		return
	}
	if err := h.svc.RemoveItem(id, itemID); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "item dihapus dari keranjang", nil)
}
