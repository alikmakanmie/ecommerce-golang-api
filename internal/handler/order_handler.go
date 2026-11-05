package handler

import (
	"net/http"

	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/middleware"
	"ecommerce-api/internal/service"
	"ecommerce-api/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrderHandler struct{ svc *service.OrderService }

func NewOrderHandler(svc *service.OrderService) *OrderHandler { return &OrderHandler{svc: svc} }

type CheckoutRequest struct {
	AddressID     string `json:"address_id" binding:"required"`
	PaymentMethod string `json:"payment_method"`
	Notes         string `json:"notes"`
}

type UpdateOrderStatusRequest struct {
	Status domain.OrderStatus `json:"status" binding:"required"`
}

func (h *OrderHandler) Checkout(c *gin.Context) {
	id, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "tidak terautentikasi")
		return
	}
	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	addrID, err := uuid.Parse(req.AddressID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "address_id tidak valid")
		return
	}
	order, err := h.svc.Checkout(id, addrID, req.PaymentMethod, req.Notes)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusCreated, "checkout berhasil", order)
}

func (h *OrderHandler) ListOrders(c *gin.Context) {
	id, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "tidak terautentikasi")
		return
	}
	isAdmin := false
	if role, exists := c.Get("role"); exists {
		if r, ok := role.(domain.Role); ok && r == domain.RoleAdmin {
			isAdmin = true
		}
	}
	page, limit, _ := utils.ParsePagination(c)
	// Admin listing is via /admin/orders; this endpoint lists own orders unless admin.
	list, total, err := h.svc.ListOrders(id, false, page, limit)
	_ = isAdmin
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.SuccessWithMeta(c, http.StatusOK, "daftar order", list, utils.NewPaginationMeta(page, limit, total))
}

func (h *OrderHandler) ListAllOrders(c *gin.Context) {
	id, _ := middleware.CurrentUserID(c)
	page, limit, _ := utils.ParsePagination(c)
	list, total, err := h.svc.ListOrders(id, true, page, limit)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.SuccessWithMeta(c, http.StatusOK, "daftar semua order", list, utils.NewPaginationMeta(page, limit, total))
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	id, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "tidak terautentikasi")
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id order tidak valid")
		return
	}
	isAdmin := false
	if role, exists := c.Get("role"); exists {
		if r, ok := role.(domain.Role); ok && r == domain.RoleAdmin {
			isAdmin = true
		}
	}
	order, err := h.svc.GetOrder(id, orderID, isAdmin)
	if err != nil {
		utils.Error(c, http.StatusNotFound, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "order ditemukan", order)
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "tidak terautentikasi")
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id order tidak valid")
		return
	}
	order, err := h.svc.CancelOrder(id, orderID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "order dibatalkan", order)
}

func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id order tidak valid")
		return
	}
	var req UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	order, err := h.svc.UpdateOrderStatus(orderID, req.Status)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "status order diperbarui", order)
}
