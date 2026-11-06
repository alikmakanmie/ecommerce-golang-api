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

type OrderHandler struct {
	svc     *service.OrderService
	userSvc *service.UserService
}

func NewOrderHandler(svc *service.OrderService, userSvc *service.UserService) *OrderHandler {
	return &OrderHandler{svc: svc, userSvc: userSvc}
}

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
		if r, ok := role.(domain.Role); ok &&
			(r == domain.RoleAdmin || r == domain.RoleSuperAdmin) {
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
		if r, ok := role.(domain.Role); ok &&
			(r == domain.RoleAdmin || r == domain.RoleSuperAdmin) {
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
	// Audit trail: catat siapa staf yang memproses (nama dari profil; jatuh
	// ke email bila profil tak ditemukan).
	var processorID *uuid.UUID
	processorName := ""
	if actorID, ok := middleware.CurrentUserID(c); ok {
		processorID = &actorID
		if u, err := h.userSvc.GetProfile(actorID); err == nil {
			processorName = u.Name
		}
		if processorName == "" {
			if email, ok := c.Get("email"); ok {
				processorName, _ = email.(string)
			}
		}
	}
	order, err := h.svc.UpdateOrderStatus(orderID, req.Status, processorID, processorName)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "status order diperbarui", order)
}

type GuestCheckoutRequest struct {
	Items           []service.GuestCheckoutItem   `json:"items" binding:"required,min=1"`
	CustomerName    string                        `json:"customer_name" binding:"required"`
	CustomerPhone   string                        `json:"customer_phone"`
	OrderType       string                        `json:"order_type" binding:"required"`
	TableNumber     string                        `json:"table_number"`
	PaymentMethod   string                        `json:"payment_method" binding:"required"`
	Notes           string                        `json:"notes"`
	DeliveryAddress *service.GuestDeliveryAddress `json:"delivery_address"`
}

// GuestCheckout adalah endpoint publik (tanpa JWT) untuk tamu.
func (h *OrderHandler) GuestCheckout(c *gin.Context) {
	var req GuestCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	order, err := h.svc.GuestCheckout(service.GuestCheckoutInput{
		Items:           req.Items,
		CustomerName:    req.CustomerName,
		CustomerPhone:   req.CustomerPhone,
		OrderType:       req.OrderType,
		TableNumber:     req.TableNumber,
		PaymentMethod:   req.PaymentMethod,
		Notes:           req.Notes,
		DeliveryAddress: req.DeliveryAddress,
	})
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusCreated, "guest checkout berhasil", order)
}

// TrackOrder adalah endpoint publik (tanpa JWT): lacak order via nomor order.
func (h *OrderHandler) TrackOrder(c *gin.Context) {
	order, err := h.svc.TrackOrder(c.Param("order_number"))
	if err != nil {
		utils.Error(c, http.StatusNotFound, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "order ditemukan", order)
}
