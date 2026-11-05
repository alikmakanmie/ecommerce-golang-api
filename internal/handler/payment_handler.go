package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"ecommerce-api/internal/service"
	"ecommerce-api/pkg/utils"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct{ svc *service.PaymentService }

func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler { return &PaymentHandler{svc: svc} }

func (h *PaymentHandler) Webhook(c *gin.Context) {
	var raw map[string]interface{}
	body, err := c.GetRawData()
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "body tidak valid")
		return
	}
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
<p>Ini halaman simulasi. Untuk mensimulasikan pembayaran, kirim webhook:</p>
<pre>curl -X POST http://localhost:8080/api/v1/payments/webhook -H 'Content-Type: application/json' -d '{"transaction_id":"%s","transaction_status":"settlement"}'</pre>
</body></html>`, pay.TransactionID, pay.Amount, pay.Method, pay.Status, pay.QRString, pay.TransactionID)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
