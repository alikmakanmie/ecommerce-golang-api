package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestConcurrencyLimiter_SerializesSlowHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var inFlight int64
	var maxSeen int64
	slow := func(c *gin.Context) {
		cur := atomic.AddInt64(&inFlight, 1)
		for {
			old := atomic.LoadInt64(&maxSeen)
			if cur <= old || atomic.CompareAndSwapInt64(&maxSeen, old, cur) {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
		atomic.AddInt64(&inFlight, -1)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
	r := gin.New()
	r.POST("/upload", ConcurrencyLimiter(2, 50*time.Millisecond), slow)

	// 5 request serentak ke handler 100ms dengan batas 2 dan antrean 50ms:
	// 2 langsung jalan, 3 lainnya kehabisan antrean -> 503.
	var ok, busy int64
	done := make(chan struct{})
	for i := 0; i < 5; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodPost, "/upload", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code == http.StatusOK {
				atomic.AddInt64(&ok, 1)
			} else if rec.Code == http.StatusServiceUnavailable {
				atomic.AddInt64(&busy, 1)
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 5; i++ {
		<-done
	}
	if maxSeen > 2 {
		t.Fatalf("maksimal serentak = %d, want <= 2", maxSeen)
	}
	if ok != 2 {
		t.Fatalf("ok = %d, want 2", ok)
	}
	if busy != 3 {
		t.Fatalf("busy (503) = %d, want 3", busy)
	}
}
