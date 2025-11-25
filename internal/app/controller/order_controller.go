package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
)

type OrderController struct {
	orderService service.OrderService
}

func NewOrderController(orderService service.OrderService) *OrderController {
	return &OrderController{
		orderService: orderService,
	}
}

type OrderItemInput struct {
	ProductID       uint  `json:"product_id" binding:"required"`
	Quantity        int   `json:"quantity" binding:"required,min=1"`
	ProductOptionID *uint `json:"product_option_id"`
}

type CreateOrderRequest struct {
	Items           []OrderItemInput      `json:"items"`
	ShippingAddress string                `json:"shipping_address"`
	FulfillmentType model.FulfillmentType `json:"fulfillment_type"`
	PickupStoreID   *uint                 `json:"pickup_store_id"`
}

type UpdateOrderStatusRequest struct {
	Status model.OrderStatus `json:"status" binding:"required"`
}

type UpdatePaymentStatusRequest struct {
	Status model.PaymentStatus `json:"status" binding:"required"`
}

// GetOrders returns user's orders
// GET /api/v1/orders
func (ctrl *OrderController) GetOrders(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized access to orders", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	orders, err := ctrl.orderService.GetUserOrders(userID)
	if err != nil {
		log.Error("Failed to fetch orders", err, map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch orders",
		})
		return
	}

	log.Info("Orders fetched successfully", map[string]interface{}{
		"user_id": userID,
		"count":   len(orders),
	})

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"count":  len(orders),
	})
}

// GetOrderByID returns order by ID
// GET /api/v1/orders/:id
func (ctrl *OrderController) GetOrderByID(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized access to order", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid order ID format", map[string]interface{}{
			"user_id":  userID,
			"order_id": idStr,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	order, err := ctrl.orderService.GetOrderByID(userID, uint(id))
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			log.Warn("Order not found", map[string]interface{}{
				"user_id":  userID,
				"order_id": id,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Order not found",
			})
			return
		}
		log.Error("Failed to fetch order", err, map[string]interface{}{
			"user_id":  userID,
			"order_id": id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch order",
		})
		return
	}

	log.Info("Order fetched successfully", map[string]interface{}{
		"user_id":  userID,
		"order_id": order.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"order": order,
	})
}

// CreateOrder creates a new order from cart
// POST /api/v1/orders
func (ctrl *OrderController) CreateOrder(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized attempt to create order", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid create order request", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Creating order", map[string]interface{}{
		"user_id":          userID,
		"fulfillment_type": req.FulfillmentType,
		"pickup_store_id":  req.PickupStoreID,
		"has_items":        len(req.Items) > 0,
	})

	var order *model.Order
	var err error

	// 직접 주문 (items가 있는 경우) 또는 장바구니 주문
	if len(req.Items) > 0 {
		// Convert controller input to service input
		items := make([]service.OrderItemInput, len(req.Items))
		for i, item := range req.Items {
			items[i] = service.OrderItemInput{
				ProductID:       item.ProductID,
				Quantity:        item.Quantity,
				ProductOptionID: item.ProductOptionID,
			}
		}
		order, err = ctrl.orderService.CreateOrder(userID, items, req.ShippingAddress, req.FulfillmentType, req.PickupStoreID)
	} else {
		// 장바구니에서 주문 생성 (기존 로직)
		order, err = ctrl.orderService.CreateOrderFromCart(userID, req.ShippingAddress, req.FulfillmentType, req.PickupStoreID)
	}
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmptyCart):
			log.Warn("Order creation failed: empty cart", map[string]interface{}{
				"user_id": userID,
			})
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Cart is empty",
			})
			return
		case errors.Is(err, service.ErrInsufficientStock):
			log.Warn("Order creation failed: insufficient stock", map[string]interface{}{
				"user_id": userID,
			})
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Insufficient stock for one or more items",
			})
			return
		case errors.Is(err, service.ErrInvalidFulfillment):
			log.Warn("Order creation failed: invalid fulfillment", map[string]interface{}{
				"user_id":          userID,
				"fulfillment_type": req.FulfillmentType,
			})
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid fulfillment selection",
			})
			return
		case errors.Is(err, service.ErrInvalidProductOption):
			log.Warn("Order creation failed: invalid product option", map[string]interface{}{
				"user_id": userID,
			})
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid product option in cart",
			})
			return
		case errors.Is(err, service.ErrProductNotFound):
			log.Warn("Order creation failed: product not found", map[string]interface{}{
				"user_id": userID,
			})
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "One or more products are unavailable",
			})
			return
		default:
			log.Error("Failed to create order", err, map[string]interface{}{
				"user_id": userID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create order",
			})
			return
		}
	}

	log.Info("Order created successfully", map[string]interface{}{
		"user_id":      userID,
		"order_id":     order.ID,
		"total_amount": order.TotalAmount,
	})

	log.Info("Order created successfully", map[string]interface{}{
		"user_id":      userID,
		"order_id":     order.ID,
		"total_amount": order.TotalAmount,
	})

	c.JSON(http.StatusCreated, gin.H{
		"message": "Order created successfully",
		"order":   order,
	})
}

// UpdateOrderStatus updates order status (Admin only)
// PUT /api/v1/orders/:id/status
func (ctrl *OrderController) UpdateOrderStatus(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid order ID format", map[string]interface{}{
			"order_id": idStr,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	var req UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid update order status request", map[string]interface{}{
			"order_id": id,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Updating order status", map[string]interface{}{
		"order_id": id,
		"status":   req.Status,
	})

	err = ctrl.orderService.UpdateOrderStatus(uint(id), req.Status)
	if err != nil {
		log.Error("Failed to update order status", err, map[string]interface{}{
			"order_id": id,
			"status":   req.Status,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update order status",
		})
		return
	}

	log.Info("Order status updated successfully", map[string]interface{}{
		"order_id": id,
		"status":   req.Status,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Order status updated successfully",
	})
}

// UpdatePaymentStatus updates payment status
// PUT /api/v1/orders/:id/payment
func (ctrl *OrderController) UpdatePaymentStatus(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid order ID format", map[string]interface{}{
			"order_id": idStr,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	var req UpdatePaymentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid update payment status request", map[string]interface{}{
			"order_id": id,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Updating payment status", map[string]interface{}{
		"order_id": id,
		"status":   req.Status,
	})

	err = ctrl.orderService.UpdatePaymentStatus(uint(id), req.Status)
	if err != nil {
		log.Error("Failed to update payment status", err, map[string]interface{}{
			"order_id": id,
			"status":   req.Status,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update payment status",
		})
		return
	}

	log.Info("Payment status updated successfully", map[string]interface{}{
		"order_id": id,
		"status":   req.Status,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment status updated successfully",
	})
}
