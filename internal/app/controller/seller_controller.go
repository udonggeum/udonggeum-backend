package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
)

type SellerController struct {
	sellerService service.SellerService
	storeService  service.StoreService
}

func NewSellerController(sellerService service.SellerService, storeService service.StoreService) *SellerController {
	return &SellerController{
		sellerService: sellerService,
		storeService:  storeService,
	}
}

type SellerUpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// ListMyStores returns all stores owned by the authenticated seller
// GET /api/v1/seller/stores
func (ctrl *SellerController) ListMyStores(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized access to seller stores endpoint", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	log.Debug("Fetching stores for seller", map[string]interface{}{
		"user_id": userID,
	})

	stores, err := ctrl.storeService.GetStoresByUserID(userID)
	if err != nil {
		log.Error("Failed to fetch seller stores", err, map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch stores",
		})
		return
	}

	log.Info("Seller stores fetched", map[string]interface{}{
		"user_id": userID,
		"count":   len(stores),
	})

	c.JSON(http.StatusOK, gin.H{
		"stores": stores,
		"count":  len(stores),
	})
}

// GetDashboard returns seller dashboard statistics
// GET /api/v1/seller/dashboard
func (ctrl *SellerController) GetDashboard(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized access to seller dashboard endpoint", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	log.Debug("Fetching seller dashboard", map[string]interface{}{
		"user_id": userID,
	})

	stats, err := ctrl.sellerService.GetDashboard(userID)
	if err != nil {
		log.Error("Failed to fetch seller dashboard", err, map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch dashboard statistics",
		})
		return
	}

	log.Info("Seller dashboard fetched", map[string]interface{}{
		"user_id":       userID,
		"total_orders":  stats.TotalOrders,
		"total_revenue": stats.TotalRevenue,
	})

	c.JSON(http.StatusOK, gin.H{
		"dashboard": stats,
	})
}

// GetStoreOrders returns orders for a specific store owned by the seller
// GET /api/v1/seller/stores/:store_id/orders
func (ctrl *SellerController) GetStoreOrders(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized access to store orders endpoint", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	storeIDStr := c.Param("store_id")
	storeID, err := strconv.ParseUint(storeIDStr, 10, 32)
	if err != nil {
		log.Warn("Invalid store ID format", map[string]interface{}{
			"store_id": storeIDStr,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid store ID",
		})
		return
	}

	status := c.Query("status")

	log.Debug("Fetching orders for store", map[string]interface{}{
		"user_id":  userID,
		"store_id": storeID,
		"status":   status,
	})

	orders, err := ctrl.sellerService.GetStoreOrders(userID, uint(storeID), status)
	if err != nil {
		if errors.Is(err, service.ErrStoreNotFound) {
			log.Warn("Store not found", map[string]interface{}{
				"store_id": storeID,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Store not found",
			})
			return
		}
		if errors.Is(err, service.ErrNotStoreOwner) {
			log.Warn("User does not own the store", map[string]interface{}{
				"user_id":  userID,
				"store_id": storeID,
			})
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You do not own this store",
			})
			return
		}
		log.Error("Failed to fetch store orders", err, map[string]interface{}{
			"user_id":  userID,
			"store_id": storeID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch store orders",
		})
		return
	}

	log.Info("Store orders fetched", map[string]interface{}{
		"user_id":  userID,
		"store_id": storeID,
		"count":    len(orders),
	})

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"count":  len(orders),
	})
}

// UpdateOrderStatus updates the status of an order (seller only)
// PUT /api/v1/seller/orders/:id/status
func (ctrl *SellerController) UpdateOrderStatus(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized access to update order status endpoint", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	orderIDStr := c.Param("id")
	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		log.Warn("Invalid order ID format", map[string]interface{}{
			"order_id": orderIDStr,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	var req SellerUpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid update order status request", map[string]interface{}{
			"order_id": orderID,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Updating order status", map[string]interface{}{
		"user_id":  userID,
		"order_id": orderID,
		"status":   req.Status,
	})

	if err := ctrl.sellerService.UpdateOrderStatus(userID, uint(orderID), req.Status); err != nil {
		if errors.Is(err, service.ErrNotOrderOwner) {
			log.Warn("User does not own order", map[string]interface{}{
				"user_id":  userID,
				"order_id": orderID,
			})
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You do not have permission to update this order",
			})
			return
		}
		if err.Error() == "order not found" {
			log.Warn("Order not found for status update", map[string]interface{}{
				"order_id": orderID,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Order not found",
			})
			return
		}
		if err.Error() == "invalid order status" {
			log.Warn("Invalid order status provided", map[string]interface{}{
				"status": req.Status,
			})
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid order status. Valid statuses: pending, confirmed, shipping, delivered, cancelled",
			})
			return
		}
		log.Error("Failed to update order status", err, map[string]interface{}{
			"user_id":  userID,
			"order_id": orderID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update order status",
		})
		return
	}

	log.Info("Order status updated successfully", map[string]interface{}{
		"user_id":  userID,
		"order_id": orderID,
		"status":   req.Status,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Order status updated successfully",
	})
}
