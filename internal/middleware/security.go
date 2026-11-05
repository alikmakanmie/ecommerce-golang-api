package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders sets common security response headers on every response.
// Note: Go's net/http server does not emit a Server header by default, so
// there is nothing to strip; we simply never set one ourselves.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		c.Next()
	}
}
