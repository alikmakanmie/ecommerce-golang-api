package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

func ParsePagination(c *gin.Context) (page, limit, offset int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset = (page - 1) * limit
	return page, limit, offset
}

func NewPaginationMeta(page, limit int, total int64) PaginationMeta {
	totalPages := total / int64(limit)
	if total%int64(limit) != 0 {
		totalPages++
	}
	return PaginationMeta{Page: page, Limit: limit, Total: total, TotalPages: totalPages}
}
