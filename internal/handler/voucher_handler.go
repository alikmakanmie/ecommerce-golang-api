package handler

import (
	"net/http"
	"time"

	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/service"
	"ecommerce-api/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type VoucherHandler struct{ svc *service.VoucherService }

func NewVoucherHandler(svc *service.VoucherService) *VoucherHandler { return &VoucherHandler{svc: svc} }

type ValidateVoucherRequest struct {
	Code     string `json:"code" binding:"required"`
	Subtotal int64  `json:"subtotal" binding:"min=0"`
}

// ValidateVoucher adalah endpoint publik: cek kode + subtotal keranjang,
// kembalikan valid/discount_amount/message (selalu 200 untuk input yang
// terbentuk baik; valid=false bila voucher tidak bisa dipakai).
func (h *VoucherHandler) ValidateVoucher(c *gin.Context) {
	var req ValidateVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	_, discount, err := h.svc.Validate(req.Code, req.Subtotal)
	if err != nil {
		utils.Success(c, http.StatusOK, err.Error(), gin.H{
			"valid": false, "discount_amount": 0, "message": err.Error(),
			"code": service.NormalizeVoucherCode(req.Code),
		})
		return
	}
	utils.Success(c, http.StatusOK, "voucher valid", gin.H{
		"valid": true, "discount_amount": discount, "message": "voucher valid",
		"code": service.NormalizeVoucherCode(req.Code),
	})
}

type VoucherRequest struct {
	Code           string     `json:"code" binding:"required,max=50"`
	DiscountType   string     `json:"discount_type" binding:"required"`
	DiscountValue  int64      `json:"discount_value" binding:"min=0"`
	MinOrderAmount int64      `json:"min_order_amount" binding:"min=0"`
	MaxDiscount    int64      `json:"max_discount" binding:"min=0"`
	Quota          int        `json:"quota" binding:"min=0"`
	IsActive       *bool      `json:"is_active"`
	ExpiresAt      *time.Time `json:"expires_at"`
}

func (h *VoucherHandler) ListVouchers(c *gin.Context) {
	list, err := h.svc.List()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.Voucher{}
	}
	utils.Success(c, http.StatusOK, "daftar voucher", list)
}

func (h *VoucherHandler) CreateVoucher(c *gin.Context) {
	var req VoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	v, err := h.svc.Create(service.VoucherInput{
		Code: service.NormalizeVoucherCode(req.Code), DiscountType: req.DiscountType,
		DiscountValue: req.DiscountValue, MinOrderAmount: req.MinOrderAmount,
		MaxDiscount: req.MaxDiscount, Quota: req.Quota, IsActive: isActive,
		ExpiresAt: req.ExpiresAt,
	})
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusCreated, "voucher dibuat", v)
}

func (h *VoucherHandler) UpdateVoucher(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id voucher tidak valid")
		return
	}
	var req VoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	v, err := h.svc.Update(id, service.VoucherInput{
		Code: service.NormalizeVoucherCode(req.Code), DiscountType: req.DiscountType,
		DiscountValue: req.DiscountValue, MinOrderAmount: req.MinOrderAmount,
		MaxDiscount: req.MaxDiscount, Quota: req.Quota, IsActive: isActive,
		ExpiresAt: req.ExpiresAt,
	})
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "voucher diperbarui", v)
}

func (h *VoucherHandler) DeleteVoucher(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id voucher tidak valid")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		utils.Error(c, http.StatusNotFound, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "voucher dihapus", nil)
}
