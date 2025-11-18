package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
)

type WishlistController struct {
	wishlistService service.WishlistService
}

func NewWishlistController(wishlistService service.WishlistService) *WishlistController {
	return &WishlistController{
		wishlistService: wishlistService,
	}
}

type AddToWishlistRequest struct {
	ProductID uint `json:"product_id" binding:"required"`
}

// GetWishlist returns user's wishlist
// GET /api/v1/wishlist
func (ctrl *WishlistController) GetWishlist(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized access to wishlist", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	items, err := ctrl.wishlistService.GetUserWishlist(userID)
	if err != nil {
		log.Error("Failed to fetch wishlist", err, map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch wishlist",
		})
		return
	}

	log.Info("Wishlist fetched successfully", map[string]interface{}{
		"user_id": userID,
		"count":   len(items),
	})

	c.JSON(http.StatusOK, gin.H{
		"wishlist_items": items,
		"count":          len(items),
	})
}

// AddToWishlist adds product to wishlist
// POST /api/v1/wishlist
func (ctrl *WishlistController) AddToWishlist(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized attempt to add to wishlist", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req AddToWishlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid add to wishlist request", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Adding item to wishlist", map[string]interface{}{
		"user_id":    userID,
		"product_id": req.ProductID,
	})

	err := ctrl.wishlistService.AddToWishlist(userID, req.ProductID)
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			log.Warn("Product not found for wishlist", map[string]interface{}{
				"user_id":    userID,
				"product_id": req.ProductID,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Product not found",
			})
			return
		}
		if errors.Is(err, service.ErrWishlistItemAlreadyExists) {
			log.Warn("Product already in wishlist", map[string]interface{}{
				"user_id":    userID,
				"product_id": req.ProductID,
			})
			c.JSON(http.StatusConflict, gin.H{
				"error": "Product already in wishlist",
			})
			return
		}
		log.Error("Failed to add item to wishlist", err, map[string]interface{}{
			"user_id":    userID,
			"product_id": req.ProductID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to add item to wishlist",
		})
		return
	}

	log.Info("Item added to wishlist successfully", map[string]interface{}{
		"user_id":    userID,
		"product_id": req.ProductID,
	})

	c.JSON(http.StatusCreated, gin.H{
		"message": "Item added to wishlist successfully",
	})
}

// RemoveFromWishlist removes product from wishlist
// DELETE /api/v1/wishlist/:product_id
func (ctrl *WishlistController) RemoveFromWishlist(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized attempt to remove from wishlist", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	productIDStr := c.Param("product_id")
	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		log.Warn("Invalid product ID format", map[string]interface{}{
			"user_id":    userID,
			"product_id": productIDStr,
			"error":      err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	log.Debug("Removing item from wishlist", map[string]interface{}{
		"user_id":    userID,
		"product_id": productID,
	})

	err = ctrl.wishlistService.RemoveFromWishlist(userID, uint(productID))
	if err != nil {
		if errors.Is(err, service.ErrWishlistItemNotFound) {
			log.Warn("Wishlist item not found", map[string]interface{}{
				"user_id":    userID,
				"product_id": productID,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Wishlist item not found",
			})
			return
		}
		log.Error("Failed to remove item from wishlist", err, map[string]interface{}{
			"user_id":    userID,
			"product_id": productID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to remove item from wishlist",
		})
		return
	}

	log.Info("Item removed from wishlist successfully", map[string]interface{}{
		"user_id":    userID,
		"product_id": productID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Item removed from wishlist successfully",
	})
}
