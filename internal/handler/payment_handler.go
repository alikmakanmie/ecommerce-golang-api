package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"ecommerce-api/internal/service"
	"ecommerce-api/pkg/utils"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	svc           *service.PaymentService
	webhookSecret string
}

func NewPaymentHandler(svc *service.PaymentService, webhookSecret string) *PaymentHandler {
	return &PaymentHandler{svc: svc, webhookSecret: webhookSecret}
}

func (h *PaymentHandler) verifySignature(body []byte, sigHeader string) bool {
	if h.webhookSecret == "" {
		return false
	}
	sig := strings.TrimSpace(sigHeader)
	sig = strings.TrimPrefix(sig, "sha256=")
	expectedMAC := hmac.New(sha256.New, []byte(h.webhookSecret))
	expectedMAC.Write(body)
	expected, err := hex.DecodeString(sig)
	if err != nil {
		return false
	}
	return hmac.Equal(expected, expectedMAC.Sum(nil))
}

func (h *PaymentHandler) Webhook(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "body tidak valid")
		return
	}
	if h.webhookSecret == "" {
		utils.Error(c, http.StatusUnauthorized, "webhook secret belum dikonfigurasi")
		return
	}
	if !h.verifySignature(body, c.GetHeader("X-Webhook-Signature")) {
		utils.Error(c, http.StatusUnauthorized, "signature webhook tidak valid")
		return
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		utils.Error(c, http.StatusBadRequest, "body JSON tidak valid")
		return
	}
	txID, _ := raw["transaction_id"].(string)
	status, _ := raw["transaction_status"].(string)
	if status == "" {
		status, _ = raw["status"].(string)
	}
	if txID == "" || status == "" {
		utils.Error(c, http.StatusBadRequest, "transaction_id dan transaction_status wajib diisi")
		return
	}
	pay, err := h.svc.HandleWebhook(txID, status, string(body))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "webhook diproses", pay)
}

func (h *PaymentHandler) MockPayPage(c *gin.Context) {
	txID := c.Param("transaction_id")
	pay, err := h.svc.GetByTransactionID(txID)
	if err != nil {
		c.String(http.StatusNotFound, "transaksi tidak ditemukan: %s", txID)
		return
	}
	html := fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>Mock Payment</title></head><body>
<h1>Mock Payment Gateway</h1>
<p>Transaction ID: <strong>%s</strong></p>
<p>Amount: <strong>%d</strong></p>
<p>Method: <strong>%s</strong></p>
<p>Status: <strong>%s</strong></p>
<p>QR: <strong>%s</strong></p>
<p>Ini halaman simulasi. Untuk mensimulasikan pembayaran, kirim webhook dengan header X-Webhook-Signature berisi HMAC-SHA256 hex dari body (kunci: WEBHOOK_SECRET):</p>
<pre>BODY='{"transaction_id":"%s","transaction_status":"settlement"}'
SIG=$(printf '%%s' "$BODY" | openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" | cut -d' ' -f2)
curl -X POST http://localhost:8080/api/v1/payments/webhook -H 'Content-Type: application/json' -H "X-Webhook-Signature: $SIG" -d "$BODY"</pre>
</body></html>`, pay.TransactionID, pay.Amount, pay.Method, pay.Status, pay.QRString, pay.TransactionID)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
