package payment

import (
	"fmt"
	"strings"
	"time"
)

type MockGateway struct {
	BaseURL string
}

func NewMockGateway(baseURL string) *MockGateway {
	return &MockGateway{BaseURL: strings.TrimRight(baseURL, "/")}
}

func (g *MockGateway) CreateCharge(orderNumber string, amount int64, method string) (transactionID, paymentURL, qrString string, expiresAt time.Time) {
	transactionID = fmt.Sprintf("MOCK-%s-%d", orderNumber, time.Now().UnixNano())
	paymentURL = g.BaseURL + "/api/v1/mock-pay/" + transactionID
	if strings.EqualFold(method, "qris") {
		qrString = "MOCK-QRIS-" + transactionID
	}
	expiresAt = time.Now().Add(24 * time.Hour)
	return transactionID, paymentURL, qrString, expiresAt
}
