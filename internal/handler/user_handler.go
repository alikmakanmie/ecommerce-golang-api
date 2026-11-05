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

type UserHandler struct{ svc *service.UserService }

func NewUserHandler(svc *service.UserService) *UserHandler { return &UserHandler{svc: svc} }

type UpdateProfileRequest struct {
	Name  string `json:"name" binding:"max=100"`
	Phone string `json:"phone" binding:"max=20"`
}

type CreateAddressRequest struct {
	Label         string `json:"label" binding:"required,max=50"`
	RecipientName string `json:"recipient_name" binding:"required,max=100"`
	Phone         string `json:"phone" binding:"required,max=20"`
	Street        string `json:"street" binding:"required,max=255"`
	City          string `json:"city" binding:"required,max=100"`
	Province      string `json:"province" binding:"required,max=100"`
	PostalCode    string `json:"postal_code" binding:"required,max=10"`
	IsDefault     bool   `json:"is_default"`
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	id, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "tidak terautentikasi")
		return
	}
	user, err := h.svc.GetProfile(id)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "pengguna tidak ditemukan")
		return
	}
	utils.Success(c, http.StatusOK, "profil ditemukan", user)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	id, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "tidak terautentikasi")
		return
	}
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	user, err := h.svc.UpdateProfile(id, req.Name, req.Phone)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "profil diperbarui", user)
}

func (h *UserHandler) ListAddresses(c *gin.Context) {
	id, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "tidak terautentikasi")
		return
	}
	list, err := h.svc.ListAddresses(id)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.Address{}
	}
	utils.Success(c, http.StatusOK, "daftar alamat", list)
}

func (h *UserHandler) CreateAddress(c *gin.Context) {
	id, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "tidak terautentikasi")
		return
	}
	var req CreateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	addr := &domain.Address{
		Label: req.Label, RecipientName: req.RecipientName, Phone: req.Phone,
		Street: req.Street, City: req.City, Province: req.Province,
		PostalCode: req.PostalCode, IsDefault: req.IsDefault,
	}
	created, err := h.svc.CreateAddress(id, addr)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusCreated, "alamat dibuat", created)
}

func (h *UserHandler) DeleteAddress(c *gin.Context) {
	id, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "tidak terautentikasi")
		return
	}
	addrID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id alamat tidak valid")
		return
	}
	if err := h.svc.DeleteAddress(id, addrID); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "alamat dihapus", nil)
}

func (h *UserHandler) SetDefaultAddress(c *gin.Context) {
	id, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "tidak terautentikasi")
		return
	}
	addrID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id alamat tidak valid")
		return
	}
	addr, err := h.svc.SetDefaultAddress(id, addrID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "alamat default diperbarui", addr)
}
