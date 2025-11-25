package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
	"github.com/ikkim/udonggeum-backend/pkg/payment/kakaopay"
)

type PaymentController struct {
	paymentService service.PaymentService
}

func NewPaymentController(paymentService service.PaymentService) *PaymentController {
	return &PaymentController{
		paymentService: paymentService,
	}
}

// InitiatePaymentRequest represents the request to initiate a payment
type InitiatePaymentRequest struct {
	OrderID uint `json:"order_id" binding:"required"`
}

// CancelPaymentRequest represents the request to cancel a payment
type CancelPaymentRequest struct {
	CancelAmount float64 `json:"cancel_amount" binding:"required,gt=0"`
}

// InitiatePayment initiates a payment process
// POST /api/v1/payments/ready
func (ctrl *PaymentController) InitiatePayment(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized access to payment initiation", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req InitiatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	resp, err := ctrl.paymentService.InitiatePayment(c.Request.Context(), userID, req.OrderID)
	if err != nil {
		log.Error("Failed to initiate payment", err, map[string]interface{}{
			"user_id":  userID,
			"order_id": req.OrderID,
		})

		status := http.StatusInternalServerError
		message := "Failed to initiate payment"

		if errors.Is(err, service.ErrOrderNotFound) {
			status = http.StatusNotFound
			message = "Order not found"
		} else if errors.Is(err, service.ErrPaymentAlreadyProcessed) {
			status = http.StatusConflict
			message = "Payment already processed"
		} else if errors.Is(err, service.ErrInvalidPaymentAmount) {
			status = http.StatusBadRequest
			message = "Invalid payment amount"
		}

		c.JSON(status, gin.H{
			"error": message,
		})
		return
	}

	log.Info("Payment initiated successfully", map[string]interface{}{
		"user_id":  userID,
		"order_id": req.OrderID,
		"tid":      resp.TID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment initiated successfully",
		"data":    resp,
	})
}

// PaymentSuccess handles successful payment callback
// GET /api/v1/payments/success
func (ctrl *PaymentController) PaymentSuccess(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	// Parse query parameters
	pgToken := c.Query("pg_token")
	orderIDStr := c.Query("order_id")

	if pgToken == "" || orderIDStr == "" {
		log.Warn("Missing required parameters", nil)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required parameters",
		})
		return
	}

	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		log.Warn("Invalid order ID", map[string]interface{}{
			"error":    err.Error(),
			"order_id": orderIDStr,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	// Approve the payment
	resp, err := ctrl.paymentService.ApprovePayment(c.Request.Context(), uint(orderID), pgToken)
	if err != nil {
		log.Error("Failed to approve payment", err, map[string]interface{}{
			"order_id": orderID,
		})

		status := http.StatusInternalServerError
		message := "Failed to approve payment"

		if errors.Is(err, service.ErrOrderNotFound) {
			status = http.StatusNotFound
			message = "Order not found"
		} else if errors.Is(err, service.ErrPaymentAlreadyProcessed) {
			status = http.StatusConflict
			message = "Payment already processed"
		} else if errors.Is(err, kakaopay.ErrPaymentFailed) {
			status = http.StatusPaymentRequired
			message = "Payment failed"
		}

		c.JSON(status, gin.H{
			"error": message,
		})
		return
	}

	log.Info("Payment approved successfully", map[string]interface{}{
		"order_id": orderID,
		"aid":      resp.AID,
		"tid":      resp.TID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment completed successfully",
		"data":    resp,
	})
}

// PaymentFail handles failed payment callback
// GET /api/v1/payments/fail
func (ctrl *PaymentController) PaymentFail(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	orderIDStr := c.Query("order_id")
	errorMsg := c.DefaultQuery("error_msg", "Payment failed")

	if orderIDStr == "" {
		log.Warn("Missing order_id parameter", nil)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing order_id parameter",
		})
		return
	}

	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		log.Warn("Invalid order ID", map[string]interface{}{
			"error":    err.Error(),
			"order_id": orderIDStr,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	// Handle the failed payment
	err = ctrl.paymentService.HandlePaymentCallback(c.Request.Context(), uint(orderID), false, "")
	if err != nil {
		log.Error("Failed to handle payment failure", err, map[string]interface{}{
			"order_id": orderID,
		})
	}

	log.Info("Payment failure processed", map[string]interface{}{
		"order_id": orderID,
		"error":    errorMsg,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment failed",
		"error":   errorMsg,
	})
}

// PaymentCancel handles cancelled payment callback
// GET /api/v1/payments/cancel
func (ctrl *PaymentController) PaymentCancel(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	orderIDStr := c.Query("order_id")

	if orderIDStr == "" {
		log.Warn("Missing order_id parameter", nil)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing order_id parameter",
		})
		return
	}

	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		log.Warn("Invalid order ID", map[string]interface{}{
			"error":    err.Error(),
			"order_id": orderIDStr,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	// Handle the cancelled payment
	err = ctrl.paymentService.HandlePaymentCallback(c.Request.Context(), uint(orderID), false, "")
	if err != nil {
		log.Error("Failed to handle payment cancellation", err, map[string]interface{}{
			"order_id": orderID,
		})
	}

	log.Info("Payment cancellation processed", map[string]interface{}{
		"order_id": orderID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment cancelled by user",
	})
}

// RefundPayment handles payment refund/cancellation
// POST /api/v1/payments/:orderID/refund
func (ctrl *PaymentController) RefundPayment(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized access to payment refund", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	orderIDStr := c.Param("orderID")
	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		log.Warn("Invalid order ID", map[string]interface{}{
			"error":    err.Error(),
			"order_id": orderIDStr,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	var req CancelPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	resp, err := ctrl.paymentService.CancelPayment(c.Request.Context(), userID, uint(orderID), req.CancelAmount)
	if err != nil {
		log.Error("Failed to cancel payment", err, map[string]interface{}{
			"user_id":  userID,
			"order_id": orderID,
		})

		status := http.StatusInternalServerError
		message := "Failed to cancel payment"

		if errors.Is(err, service.ErrOrderNotFound) {
			status = http.StatusNotFound
			message = "Order not found"
		} else if errors.Is(err, service.ErrInvalidPaymentAmount) {
			status = http.StatusBadRequest
			message = "Invalid cancel amount"
		} else if errors.Is(err, kakaopay.ErrInsufficientAmount) {
			status = http.StatusBadRequest
			message = "Cancel amount exceeds approved amount"
		}

		c.JSON(status, gin.H{
			"error": message,
		})
		return
	}

	log.Info("Payment cancelled successfully", map[string]interface{}{
		"user_id":         userID,
		"order_id":        orderID,
		"canceled_amount": req.CancelAmount,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment cancelled successfully",
		"data":    resp,
	})
}

// GetPaymentStatus retrieves payment status for an order
// GET /api/v1/payments/status/:orderID
func (ctrl *PaymentController) GetPaymentStatus(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized access to payment status", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	orderIDStr := c.Param("orderID")
	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		log.Warn("Invalid order ID", map[string]interface{}{
			"error":    err.Error(),
			"order_id": orderIDStr,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	order, err := ctrl.paymentService.GetPaymentStatus(userID, uint(orderID))
	if err != nil {
		log.Error("Failed to get payment status", err, map[string]interface{}{
			"user_id":  userID,
			"order_id": orderID,
		})

		status := http.StatusInternalServerError
		message := "Failed to get payment status"

		if errors.Is(err, service.ErrOrderNotFound) {
			status = http.StatusNotFound
			message = "Order not found"
		}

		c.JSON(status, gin.H{
			"error": message,
		})
		return
	}

	log.Info("Payment status retrieved successfully", map[string]interface{}{
		"user_id":  userID,
		"order_id": orderID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment status retrieved successfully",
		"data": gin.H{
			"order_id":            order.ID,
			"payment_status":      order.PaymentStatus,
			"payment_provider":    order.PaymentProvider,
			"payment_tid":         order.PaymentTID,
			"payment_aid":         order.PaymentAID,
			"payment_approved_at": order.PaymentApprovedAt,
			"total_amount":        order.TotalAmount,
		},
	})
}
