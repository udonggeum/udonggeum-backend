package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ikkim/udonggeum-backend/config"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"github.com/ikkim/udonggeum-backend/pkg/payment/kakaopay"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrPaymentAlreadyProcessed = errors.New("payment already processed")
	ErrInvalidPaymentAmount    = errors.New("invalid payment amount")
	ErrPaymentNotFound         = errors.New("payment not found")
)

// PaymentInitResponse represents the response from payment initiation
type PaymentInitResponse struct {
	TID                   string `json:"tid"`
	NextRedirectAppURL    string `json:"next_redirect_app_url"`
	NextRedirectMobileURL string `json:"next_redirect_mobile_url"`
	NextRedirectPCURL     string `json:"next_redirect_pc_url"`
	AndroidAppScheme      string `json:"android_app_scheme"`
	IOSAppScheme          string `json:"ios_app_scheme"`
}

// PaymentApprovalResponse represents the response from payment approval
type PaymentApprovalResponse struct {
	OrderID       uint      `json:"order_id"`
	AID           string    `json:"aid"`
	TID           string    `json:"tid"`
	TotalAmount   float64   `json:"total_amount"`
	PaymentMethod string    `json:"payment_method"`
	ApprovedAt    time.Time `json:"approved_at"`
}

// PaymentCancelResponse represents the response from payment cancellation
type PaymentCancelResponse struct {
	OrderID             uint    `json:"order_id"`
	TID                 string  `json:"tid"`
	CanceledAmount      float64 `json:"canceled_amount"`
	RemainingAmount     float64 `json:"remaining_amount"`
	CanceledAt          time.Time `json:"canceled_at"`
}

// PaymentService defines the payment service interface
type PaymentService interface {
	InitiatePayment(ctx context.Context, userID, orderID uint) (*PaymentInitResponse, error)
	ApprovePayment(ctx context.Context, orderID uint, pgToken string) (*PaymentApprovalResponse, error)
	CancelPayment(ctx context.Context, userID, orderID uint, cancelAmount float64) (*PaymentCancelResponse, error)
	GetPaymentStatus(userID, orderID uint) (*model.Order, error)
	HandlePaymentCallback(ctx context.Context, orderID uint, success bool, pgToken string) error
}

type paymentService struct {
	orderRepo     repository.OrderRepository
	kakaoClient   *kakaopay.Client
	db            *gorm.DB
}

// NewPaymentService creates a new payment service
func NewPaymentService(
	orderRepo repository.OrderRepository,
	cfg *config.Config,
	db *gorm.DB,
) (PaymentService, error) {
	// Initialize Kakao Pay client
	kakaoConfig := kakaopay.Config{
		AdminKey:    cfg.Payment.KakaoPay.AdminKey,
		CID:         cfg.Payment.KakaoPay.CID,
		BaseURL:     cfg.Payment.KakaoPay.BaseURL,
		ApprovalURL: cfg.Payment.KakaoPay.ApprovalURL,
		FailURL:     cfg.Payment.KakaoPay.FailURL,
		CancelURL:   cfg.Payment.KakaoPay.CancelURL,
	}

	kakaoClient, err := kakaopay.NewClient(kakaoConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kakao pay client: %w", err)
	}

	return &paymentService{
		orderRepo:   orderRepo,
		kakaoClient: kakaoClient,
		db:          db,
	}, nil
}

// InitiatePayment initiates a payment process for an order
func (s *paymentService) InitiatePayment(ctx context.Context, userID, orderID uint) (*PaymentInitResponse, error) {
	log := logger.Get()

	// Get order with lock to prevent race conditions
	var order model.Order
	if err := s.db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("OrderItems.Product").
		Where("id = ? AND user_id = ?", orderID, userID).
		First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Check if payment is already processed
	if order.PaymentStatus == model.PaymentStatusCompleted {
		return nil, ErrPaymentAlreadyProcessed
	}

	// Validate payment amount
	if order.TotalAmount <= 0 {
		return nil, ErrInvalidPaymentAmount
	}

	// Prepare Kakao Pay request
	// Generate item name from order items
	itemName := "주문 결제"
	if len(order.OrderItems) > 0 {
		itemName = fmt.Sprintf("%s 외 %d건", order.OrderItems[0].Product.Name, len(order.OrderItems)-1)
	}

	totalAmountInt := int64(order.TotalAmount)

	// Append order_id to callback URLs so we know which order to process
	approvalURL := fmt.Sprintf("%s?order_id=%d", s.kakaoClient.GetConfig().ApprovalURL, order.ID)
	failURL := fmt.Sprintf("%s?order_id=%d", s.kakaoClient.GetConfig().FailURL, order.ID)
	cancelURL := fmt.Sprintf("%s?order_id=%d", s.kakaoClient.GetConfig().CancelURL, order.ID)

	req := kakaopay.ReadyRequest{
		PartnerOrderID: fmt.Sprintf("ORDER-%d", order.ID),
		PartnerUserID:  fmt.Sprintf("USER-%d", userID),
		ItemName:       itemName,
		Quantity:       len(order.OrderItems),
		TotalAmount:    totalAmountInt,
		TaxFreeAmount:  0, // Set to 0 for now, can be calculated if needed
		ApprovalURL:    approvalURL,
		FailURL:        failURL,
		CancelURL:      cancelURL,
	}

	// Call Kakao Pay Ready API
	resp, err := s.kakaoClient.Ready(ctx, req)
	if err != nil {
		log.Error("Failed to initiate kakao pay", err, map[string]interface{}{
			"order_id": orderID,
		})
		return nil, fmt.Errorf("failed to initiate payment: %w", err)
	}

	// Update order with payment TID and provider
	order.PaymentTID = resp.TID
	order.PaymentProvider = "kakaopay"
	order.PaymentStatus = model.PaymentStatusPending

	if err := s.db.Save(&order).Error; err != nil {
		log.Error("Failed to update order with payment info", err, map[string]interface{}{
			"order_id": orderID,
		})
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	log.Info("Payment initiated successfully", map[string]interface{}{
		"order_id": orderID,
		"tid":      resp.TID,
	})

	return &PaymentInitResponse{
		TID:                   resp.TID,
		NextRedirectAppURL:    resp.NextRedirectAppURL,
		NextRedirectMobileURL: resp.NextRedirectMobileURL,
		NextRedirectPCURL:     resp.NextRedirectPCURL,
		AndroidAppScheme:      resp.AndroidAppScheme,
		IOSAppScheme:          resp.IOSAppScheme,
	}, nil
}

// ApprovePayment approves a payment after user authentication
func (s *paymentService) ApprovePayment(ctx context.Context, orderID uint, pgToken string) (*PaymentApprovalResponse, error) {
	log := logger.Get()

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get order with lock
	var order model.Order
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", orderID).
		First(&order).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Check if already approved
	if order.PaymentStatus == model.PaymentStatusCompleted {
		tx.Rollback()
		return nil, ErrPaymentAlreadyProcessed
	}

	// Check if payment was initiated
	if order.PaymentTID == "" {
		tx.Rollback()
		return nil, ErrPaymentNotFound
	}

	// Call Kakao Pay Approve API
	req := kakaopay.ApproveRequest{
		TID:            order.PaymentTID,
		PartnerOrderID: fmt.Sprintf("ORDER-%d", order.ID),
		PartnerUserID:  fmt.Sprintf("USER-%d", order.UserID),
		PgToken:        pgToken,
	}

	resp, err := s.kakaoClient.Approve(ctx, req)
	if err != nil {
		tx.Rollback()
		log.Error("Failed to approve kakao pay", err, map[string]interface{}{
			"order_id": orderID,
		})

		// Update payment status to failed
		order.PaymentStatus = model.PaymentStatusFailed
		s.db.Save(&order)

		return nil, fmt.Errorf("failed to approve payment: %w", err)
	}

	// Update order with approval info
	now := time.Now()
	order.PaymentStatus = model.PaymentStatusCompleted
	order.PaymentAID = resp.AID
	order.PaymentApprovedAt = &now
	order.Status = model.OrderStatusConfirmed // Update order status to confirmed

	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		log.Error("Failed to update order after approval", err, map[string]interface{}{
			"order_id": orderID,
		})
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info("Payment approved successfully", map[string]interface{}{
		"order_id": orderID,
		"aid":      resp.AID,
		"tid":      resp.TID,
	})

	return &PaymentApprovalResponse{
		OrderID:       order.ID,
		AID:           resp.AID,
		TID:           resp.TID,
		TotalAmount:   float64(resp.Amount.Total),
		PaymentMethod: resp.PaymentMethodType,
		ApprovedAt:    resp.ApprovedAt,
	}, nil
}

// CancelPayment cancels a payment (full or partial refund)
func (s *paymentService) CancelPayment(ctx context.Context, userID, orderID uint, cancelAmount float64) (*PaymentCancelResponse, error) {
	log := logger.Get()

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get order with lock
	var order model.Order
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND user_id = ?", orderID, userID).
		First(&order).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Check if payment is completed
	if order.PaymentStatus != model.PaymentStatusCompleted {
		tx.Rollback()
		return nil, errors.New("payment not completed, cannot cancel")
	}

	// Check if payment TID exists
	if order.PaymentTID == "" {
		tx.Rollback()
		return nil, ErrPaymentNotFound
	}

	// Validate cancel amount
	if cancelAmount <= 0 || cancelAmount > order.TotalAmount {
		tx.Rollback()
		return nil, ErrInvalidPaymentAmount
	}

	// Call Kakao Pay Cancel API
	cancelAmountInt := int64(cancelAmount)
	req := kakaopay.CancelRequest{
		TID:                 order.PaymentTID,
		CancelAmount:        cancelAmountInt,
		CancelTaxFreeAmount: 0,
	}

	resp, err := s.kakaoClient.Cancel(ctx, req)
	if err != nil {
		tx.Rollback()
		log.Error("Failed to cancel kakao pay", err, map[string]interface{}{
			"order_id": orderID,
		})
		return nil, fmt.Errorf("failed to cancel payment: %w", err)
	}

	// Update order status
	// Only mark as fully refunded if the entire amount is cancelled
	if float64(resp.CancelAvailableAmount.Total) == 0 {
		// Full refund - no more amount available to cancel
		order.PaymentStatus = model.PaymentStatusRefunded
		order.Status = model.OrderStatusCancelled
	} else {
		// Partial refund - keep payment as completed but order might need custom status
		// For now, we'll keep the order as confirmed but log the partial refund
		log.Info("Partial refund processed", map[string]interface{}{
			"order_id":           orderID,
			"canceled_amount":    cancelAmountInt,
			"remaining_amount":   resp.CancelAvailableAmount.Total,
		})
	}

	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		log.Error("Failed to update order after cancellation", err, map[string]interface{}{
			"order_id": orderID,
		})
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info("Payment cancelled successfully", map[string]interface{}{
		"order_id":        orderID,
		"tid":             resp.TID,
		"canceled_amount": cancelAmountInt,
	})

	return &PaymentCancelResponse{
		OrderID:         order.ID,
		TID:             resp.TID,
		CanceledAmount:  float64(resp.CanceledAmount.Total),
		RemainingAmount: float64(resp.CancelAvailableAmount.Total),
		CanceledAt:      resp.CanceledAt,
	}, nil
}

// GetPaymentStatus retrieves the payment status for an order
func (s *paymentService) GetPaymentStatus(userID, orderID uint) (*model.Order, error) {
	var order model.Order
	if err := s.db.Preload("OrderItems.Product").
		Where("id = ? AND user_id = ?", orderID, userID).
		First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &order, nil
}

// HandlePaymentCallback handles payment callback from Kakao Pay
func (s *paymentService) HandlePaymentCallback(ctx context.Context, orderID uint, success bool, pgToken string) error {
	log := logger.Get()

	if !success {
		// Update order status to failed
		var order model.Order
		if err := s.db.Where("id = ?", orderID).First(&order).Error; err != nil {
			return fmt.Errorf("failed to get order: %w", err)
		}

		order.PaymentStatus = model.PaymentStatusFailed
		if err := s.db.Save(&order).Error; err != nil {
			log.Error("Failed to update order status to failed", err, map[string]interface{}{
				"order_id": orderID,
			})
			return fmt.Errorf("failed to update order: %w", err)
		}

		log.Info("Payment failed callback processed", map[string]interface{}{
			"order_id": orderID,
		})
		return nil
	}

	// For successful payment, approve it
	_, err := s.ApprovePayment(ctx, orderID, pgToken)
	return err
}
