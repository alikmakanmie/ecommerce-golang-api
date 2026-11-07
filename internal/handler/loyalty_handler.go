package handler

import (
	"net/http"

	"ecommerce-api/internal/service"
	"ecommerce-api/pkg/utils"

	"github.com/gin-gonic/gin"
)

type LoyaltyHandler struct{ svc *service.LoyaltyService }

func NewLoyaltyHandler(svc *service.LoyaltyService) *LoyaltyHandler { return &LoyaltyHandler{svc: svc} }

// GetLoyalty adalah endpoint publik: cek stamp/poin berdasarkan nomor HP.
// Nomor dinormalisasi di service (08xx dan 628xx dianggap pelanggan sama).
func (h *LoyaltyHandler) GetLoyalty(c *gin.Context) {
	status, err := h.svc.GetByPhone(c.Param("phone"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "status loyalty ditemukan", status)
}
