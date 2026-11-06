package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/middleware"
	"ecommerce-api/internal/service"
	"ecommerce-api/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SuperAdminHandler melayani manajemen user di bawah /api/v1/superadmin/users.
// Seluruh route-nya dijaga middleware RequireRole(RoleSuperAdmin).
type SuperAdminHandler struct{ svc *service.UserService }

func NewSuperAdminHandler(svc *service.UserService) *SuperAdminHandler {
	return &SuperAdminHandler{svc: svc}
}

type CreateUserRequest struct {
	Name     string      `json:"name" binding:"required,max=100"`
	Email    string      `json:"email" binding:"required,max=150"`
	Password string      `json:"password" binding:"required,min=8"`
	Phone    string      `json:"phone" binding:"max=20"`
	Role     domain.Role `json:"role" binding:"required"`
}

type UpdateRoleRequest struct {
	Role domain.Role `json:"role" binding:"required"`
}

type UpdateStatusRequest struct {
	IsActive *bool `json:"is_active" binding:"required"`
}

func (h *SuperAdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	search := strings.TrimSpace(c.Query("search"))
	role := domain.Role(strings.TrimSpace(c.Query("role")))
	var isActive *bool
	if v := c.Query("is_active"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			isActive = &b
		}
	}
	users, total, err := h.svc.ListUsers(page, perPage, search, role, isActive)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	utils.SuccessWithMeta(c, http.StatusOK, "daftar pengguna ditemukan", users, gin.H{
		"page": page, "per_page": perPage, "total": total, "total_pages": totalPages,
	})
}

func (h *SuperAdminHandler) GetUserStats(c *gin.Context) {
	stats, err := h.svc.UserStats()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "statistik pengguna ditemukan", stats)
}

func (h *SuperAdminHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	user, err := h.svc.CreateUser(req.Name, req.Email, req.Password, req.Phone, req.Role)
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			utils.Error(c, http.StatusConflict, err.Error())
			return
		}
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusCreated, "pengguna dibuat", user)
}

func (h *SuperAdminHandler) UpdateUserRole(c *gin.Context) {
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "tidak terautentikasi")
		return
	}
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id pengguna tidak valid")
		return
	}
	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	user, err := h.svc.UpdateUserRole(actorID, targetID, req.Role)
	if err != nil {
		h.mapUserError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "role pengguna diubah", user)
}

func (h *SuperAdminHandler) UpdateUserStatus(c *gin.Context) {
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "tidak terautentikasi")
		return
	}
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id pengguna tidak valid")
		return
	}
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	user, err := h.svc.UpdateUserStatus(actorID, targetID, *req.IsActive)
	if err != nil {
		h.mapUserError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "status pengguna diubah", user)
}

func (h *SuperAdminHandler) DeleteUser(c *gin.Context) {
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "tidak terautentikasi")
		return
	}
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "id pengguna tidak valid")
		return
	}
	if err := h.svc.DeleteUser(actorID, targetID); err != nil {
		h.mapUserError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "pengguna dihapus", gin.H{"id": targetID.String()})
}

func (h *SuperAdminHandler) mapUserError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrSelfAction):
		utils.Error(c, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrUserNotFound):
		utils.Error(c, http.StatusNotFound, err.Error())
	default:
		utils.Error(c, http.StatusBadRequest, err.Error())
	}
}
