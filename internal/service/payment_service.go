package service

import (
	"errors"
	"strings"
	"time"

	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/repository"

	"gorm.io/gorm"
)

type PaymentService struct {
	db          *gorm.DB
	paymentRepo *repository.PaymentRepository
	orderRepo   *repository.OrderRepository
	productRepo *repository.ProductRepository
}

func NewPaymentService(db *gorm.DB, paymentRepo *repository.PaymentRepository, orderRepo *repository.OrderRepository, productRepo *repository.ProductRepository) *PaymentService {
	return &PaymentService{db: db, paymentRepo: paymentRepo, orderRepo: orderRepo, productRepo: productRepo}
}

func (s *PaymentService) HandleWebhook(transactionID, status string, rawPayload string) (*domain.Payment, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "settlement", "expire", "cancel", "failed", "pending":
	default:
		return nil, errors.New("status webhook tidak dikenal: " + status)
	}
	var result *domain.Payment
	err := s.db.Transaction(func(tx *gorm.DB) error {
		payRepo := s.paymentRepo.WithTx(tx)
		orderRepo := s.orderRepo.WithTx(tx)
		pay, err := payRepo.FindByTransactionID(transactionID)
		if err != nil {
			return errors.New("transaction_id tidak ditemukan")
		}
		// Idempotent: settlement already applied.
		if pay.Status == domain.PaymentSettlement && status == "settlement" {
			result = pay
			return nil
		}
		if pay.Status == domain.PaymentSettlement {
			// Already settled; ignore later non-settlement notifications.
			result = pay
			return nil
		}
		now := time.Now()
		switch status {
		case "settlement":
			if err := payRepo.UpdateStatus(tx, pay.ID, domain.PaymentSettlement, &now); err != nil {
				return err
			}
			pay.Status = domain.PaymentSettlement
			pay.PaidAt = &now
			order, err := orderRepo.FindByID(pay.OrderID)
			if err == nil && order.Status == domain.OrderPending {
				if err := orderRepo.UpdateStatus(tx, order.ID, domain.OrderPaid); err != nil {
					return err
				}
			}
		case "expire", "cancel":
			target := domain.PaymentExpire
			if status == "cancel" {
				target = domain.PaymentCancel
			}
			if err := payRepo.UpdateStatus(tx, pay.ID, target, nil); err != nil {
				return err
			}
			pay.Status = target
			order, err := orderRepo.FindByID(pay.OrderID)
			if err == nil && order.Status == domain.OrderPending {
				if err := orderRepo.UpdateStatus(tx, order.ID, domain.OrderCancelled); err != nil {
					return err
				}
				for _, item := range order.Items {
					if err := s.productRepo.IncreaseStock(tx, item.ProductID, item.Quantity); err != nil {
						return err
					}
				}
			}
		case "failed":
			if err := payRepo.UpdateStatus(tx, pay.ID, domain.PaymentFailed, nil); err != nil {
				return err
			}
			pay.Status = domain.PaymentFailed
		case "pending":
			if err := payRepo.UpdateStatus(tx, pay.ID, domain.PaymentPending, nil); err != nil {
				return err
			}
			pay.Status = domain.PaymentPending
		}
		if rawPayload != "" {
			_ = tx.Model(&domain.Payment{}).Where("id = ?", pay.ID).Update("raw_payload", rawPayload).Error
			pay.RawPayload = rawPayload
		}
		result = pay
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PaymentService) GetByTransactionID(txID string) (*domain.Payment, error) {
	return s.paymentRepo.FindByTransactionID(txID)
}
