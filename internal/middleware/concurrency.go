package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ConcurrencyLimiter membatasi berapa request yang boleh dieksekusi
// SERENTAK pada satu handler berat (mis. upload file). Request yang
// datang saat slot penuh menunggu dalam antrean sampai waitTimeout;
// bila antrean juga penuh (timeout habis), server menjawab 503 agar
// klien retry nanti — daripada menerima ribuan upload sekaligus sampai
// CPU/RAM/disk VPS 1 vCPU lockup (insiden benchmark 6 Nov 2026: domain
// 530 selama ~40 menit).
func ConcurrencyLimiter(maxConcurrent int, waitTimeout time.Duration) gin.HandlerFunc {
	sem := make(chan struct{}, maxConcurrent)
	return func(c *gin.Context) {
		timer := time.NewTimer(waitTimeout)
		defer timer.Stop()
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			c.Next()
		case <-timer.C:
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"error":   "server sedang sibuk, coba lagi sebentar lagi",
			})
		case <-c.Request.Context().Done():
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"error":   "permintaan dibatalkan",
			})
		}
	}
}
