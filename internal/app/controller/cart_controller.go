package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
)

type CartController struct {
	cartService service.CartService
}

func NewCartController(cartService service.CartService) *CartController {
	return &CartController{
		cartService: cartService,
	}
}

type AddToCartRequest struct {
	ProductID       uint  `json:"product_id" binding:"required"`
	ProductOptionID *uint `json:"product_option_id"`
	Quantity        int   `json:"quantity" binding:"required,gt=0"`
}

type UpdateCartRequest struct {
	Quantity        int   `json:"quantity" binding:"required,gt=0"`
	ProductOptionID *uint `json:"product_option_id"`
}

// GetCart returns user's cart
// GET /api/v1/cart
func (ctrl *CartController) GetCart(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized access to cart", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	cartItems, err := ctrl.cartService.GetUserCart(userID)
	if err != nil {
		log.Error("Failed to fetch cart", err, map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch cart",
		})
		return
	}

	// Calculate total
	var total float64
	for _, item := range cartItems {
		price := item.Product.Price
		if item.ProductOptionID != nil {
			price += item.ProductOption.AdditionalPrice
		}
		total += price * float64(item.Quantity)
	}

	log.Info("Cart fetched successfully", map[string]interface{}{
		"user_id": userID,
		"count":   len(cartItems),
		"total":   total,
	})

	c.JSON(http.StatusOK, gin.H{
		"cart_items": cartItems,
		"count":      len(cartItems),
		"total":      total,
	})
}

// AddToCart adds item to cart
// POST /api/v1/cart
func (ctrl *CartController) AddToCart(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized attempt to add to cart", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req AddToCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid add to cart request", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Adding item to cart", map[string]interface{}{
		"user_id":           userID,
		"product_id":        req.ProductID,
		"product_option_id": req.ProductOptionID,
		"quantity":          req.Quantity,
	})

	err := ctrl.cartService.AddToCart(userID, req.ProductID, req.ProductOptionID, req.Quantity)
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			log.Warn("Product not found for cart", map[string]interface{}{
				"user_id":    userID,
				"product_id": req.ProductID,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Product not found",
			})
			return
		}
		if errors.Is(err, service.ErrInvalidProductOption) {
			log.Warn("Invalid product option for cart item", map[string]interface{}{
				"user_id":           userID,
				"product_id":        req.ProductID,
				"product_option_id": req.ProductOptionID,
			})
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid product option",
			})
			return
		}
		if errors.Is(err, service.ErrInsufficientStock) {
			log.Warn("Insufficient stock for cart item", map[string]interface{}{
				"user_id":    userID,
				"product_id": req.ProductID,
				"quantity":   req.Quantity,
			})
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Insufficient stock",
			})
			return
		}
		log.Error("Failed to add item to cart", err, map[string]interface{}{
			"user_id":           userID,
			"product_id":        req.ProductID,
			"product_option_id": req.ProductOptionID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to add item to cart",
		})
		return
	}

	log.Info("Item added to cart successfully", map[string]interface{}{
		"user_id":           userID,
		"product_id":        req.ProductID,
		"product_option_id": req.ProductOptionID,
		"quantity":          req.Quantity,
	})

	c.JSON(http.StatusCreated, gin.H{
		"message": "Item added to cart successfully",
	})
}

// UpdateCartItem updates cart item quantity
// PUT /api/v1/cart/:id
func (ctrl *CartController) UpdateCartItem(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized attempt to update cart item", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid cart item ID format", map[string]interface{}{
			"user_id":      userID,
			"cart_item_id": idStr,
			"error":        err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid cart item ID",
		})
		return
	}

	var raw map[string]interface{}
	if err := c.ShouldBindBodyWith(&raw, binding.JSON); err != nil {
		log.Warn("Invalid update cart request payload", map[string]interface{}{
			"user_id":      userID,
			"cart_item_id": id,
			"error":        err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	var req UpdateCartRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		log.Warn("Invalid update cart request", map[string]interface{}{
			"user_id":      userID,
			"cart_item_id": id,
			"error":        err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	_, optionProvided := raw["product_option_id"]

	log.Debug("Updating cart item", map[string]interface{}{
		"user_id":           userID,
		"cart_item_id":      id,
		"quantity":          req.Quantity,
		"product_option_id": req.ProductOptionID,
		"update_option":     optionProvided,
	})

	err = ctrl.cartService.UpdateCartItem(userID, uint(id), req.Quantity, req.ProductOptionID, optionProvided)
	if err != nil {
		if errors.Is(err, service.ErrCartItemNotFound) {
			log.Warn("Cart item not found", map[string]interface{}{
				"user_id":      userID,
				"cart_item_id": id,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Cart item not found",
			})
			return
		}
		if errors.Is(err, service.ErrInvalidProductOption) {
			log.Warn("Invalid product option for cart update", map[string]interface{}{
				"user_id":           userID,
				"cart_item_id":      id,
				"product_option_id": req.ProductOptionID,
			})
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid product option",
			})
			return
		}
		if errors.Is(err, service.ErrInsufficientStock) {
			log.Warn("Insufficient stock for cart update", map[string]interface{}{
				"user_id":      userID,
				"cart_item_id": id,
				"quantity":     req.Quantity,
			})
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Insufficient stock",
			})
			return
		}
		log.Error("Failed to update cart item", err, map[string]interface{}{
			"user_id":      userID,
			"cart_item_id": id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update cart item",
		})
		return
	}

	log.Info("Cart item updated successfully", map[string]interface{}{
		"user_id":           userID,
		"cart_item_id":      id,
		"quantity":          req.Quantity,
		"product_option_id": req.ProductOptionID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Cart item updated successfully",
	})
}

// RemoveFromCart removes item from cart
// DELETE /api/v1/cart/:id
func (ctrl *CartController) RemoveFromCart(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized attempt to remove cart item", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid cart item ID format", map[string]interface{}{
			"user_id":      userID,
			"cart_item_id": idStr,
			"error":        err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid cart item ID",
		})
		return
	}

	log.Debug("Removing cart item", map[string]interface{}{
		"user_id":      userID,
		"cart_item_id": id,
	})

	err = ctrl.cartService.RemoveFromCart(userID, uint(id))
	if err != nil {
		if errors.Is(err, service.ErrCartItemNotFound) {
			log.Warn("Cart item not found for removal", map[string]interface{}{
				"user_id":      userID,
				"cart_item_id": id,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Cart item not found",
			})
			return
		}
		log.Error("Failed to remove cart item", err, map[string]interface{}{
			"user_id":      userID,
			"cart_item_id": id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to remove cart item",
		})
		return
	}

	log.Info("Cart item removed successfully", map[string]interface{}{
		"user_id":      userID,
		"cart_item_id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Cart item removed successfully",
	})
}

// ClearCart clears all items from cart
// DELETE /api/v1/cart
func (ctrl *CartController) ClearCart(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized attempt to clear cart", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	log.Debug("Clearing cart", map[string]interface{}{
		"user_id": userID,
	})

	err := ctrl.cartService.ClearCart(userID)
	if err != nil {
		log.Error("Failed to clear cart", err, map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to clear cart",
		})
		return
	}

	log.Info("Cart cleared successfully", map[string]interface{}{
		"user_id": userID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Cart cleared successfully",
	})
}
