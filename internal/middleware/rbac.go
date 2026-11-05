package middleware

import (
	"ecommerce-api/internal/domain"
	"ecommerce-api/pkg/utils"

	"github.com/gin-gonic/gin"
)

func RequireRole(roles ...domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("role")
		if !ok {
			utils.Error(c, 403, "akses ditolak")
			c.Abort()
			return
		}
		role, ok := v.(domain.Role)
		if !ok {
			utils.Error(c, 403, "akses ditolak")
			c.Abort()
			return
		}
		for _, r := range roles {
			if r == role {
				c.Next()
				return
			}
		}
		utils.Error(c, 403, "akses ditolak: membutuhkan peran "+string(roles[0]))
		c.Abort()
	}
}
