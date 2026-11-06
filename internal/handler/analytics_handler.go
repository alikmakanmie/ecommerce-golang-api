package handler

import (
	"net/http"

	"ecommerce-api/internal/service"
	"ecommerce-api/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct{ svc *service.AnalyticsService }

func NewAnalyticsHandler(svc *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc}
}

// GetProfit mengembalikan analitik profit admin.
// Query: ?period=today|7d|30d|all (default 30d) atau start_date & end_date (YYYY-MM-DD).
func (h *AnalyticsHandler) GetProfit(c *gin.Context) {
	data, err := h.svc.GetProfitAnalytics(c.Query("period"), c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "analitik profit", data)
}
