package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
)

type StoreController struct {
	storeService service.StoreService
}

func NewStoreController(storeService service.StoreService) *StoreController {
	return &StoreController{storeService: storeService}
}

type StoreRequest struct {
	Name        string   `json:"name" binding:"required"`
	Region      string   `json:"region" binding:"required"`
	District    string   `json:"district" binding:"required"`
	Address     string   `json:"address"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	PhoneNumber string   `json:"phone_number"`
	ImageURL    string   `json:"image_url"`
	Description string   `json:"description"`
	OpenTime    string   `json:"open_time"`
	CloseTime   string   `json:"close_time"`
}

func (ctrl *StoreController) ListStores(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	includeProducts := strings.EqualFold(c.DefaultQuery("include_products", "false"), "true")
	buyingGold := strings.EqualFold(c.DefaultQuery("buying", "false"), "true")
	opts := service.StoreListOptions{
		Region:          c.Query("region"),
		District:        c.Query("district"),
		Search:          c.Query("search"),
		IncludeProducts: includeProducts,
		BuyingGold:      buyingGold,
	}

	stores, err := ctrl.storeService.ListStores(opts)
	if err != nil {
		log.Error("Failed to list stores", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch stores",
		})
		return
	}

	log.Info("Stores listed", map[string]interface{}{
		"count": len(stores),
	})

	c.JSON(http.StatusOK, gin.H{
		"stores": stores,
		"count":  len(stores),
	})
}

func (ctrl *StoreController) GetStoreByID(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid store ID", map[string]interface{}{
			"store_id": idStr,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid store ID",
		})
		return
	}

	includeProducts := strings.EqualFold(c.DefaultQuery("include_products", "false"), "true")

	store, err := ctrl.storeService.GetStoreByID(uint(id), includeProducts)
	if err != nil {
		if err == service.ErrStoreNotFound {
			log.Warn("Store not found", map[string]interface{}{
				"store_id": id,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Store not found",
			})
			return
		}
		log.Error("Failed to fetch store", err, map[string]interface{}{
			"store_id": id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch store",
		})
		return
	}

	log.Info("Store fetched", map[string]interface{}{
		"store_id": store.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"store": store,
	})
}

func (ctrl *StoreController) CreateStore(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("User ID not found in context for store creation", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	var req StoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid store creation request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	store := &model.Store{
		UserID:      userID,
		Name:        req.Name,
		Region:      req.Region,
		District:    req.District,
		Address:     req.Address,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		PhoneNumber: req.PhoneNumber,
		ImageURL:    req.ImageURL,
		Description: req.Description,
		OpenTime:    req.OpenTime,
		CloseTime:   req.CloseTime,
	}

	created, err := ctrl.storeService.CreateStore(store)
	if err != nil {
		log.Error("Failed to create store", err, map[string]interface{}{
			"user_id": userID,
			"name":    req.Name,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create store",
		})
		return
	}

	log.Info("Store created", map[string]interface{}{
		"store_id": created.ID,
		"user_id":  userID,
	})

	c.JSON(http.StatusCreated, gin.H{
		"message": "Store created successfully",
		"store":   created,
	})
}

func (ctrl *StoreController) UpdateStore(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("User ID not found in context for store update", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	idStr := c.Param("id")
	storeID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid store ID format for update", map[string]interface{}{
			"store_id": idStr,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid store ID",
		})
		return
	}

	var req StoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid store update request", map[string]interface{}{
			"store_id": storeID,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	updated, err := ctrl.storeService.UpdateStore(userID, uint(storeID), service.StoreMutation{
		Name:        req.Name,
		Region:      req.Region,
		District:    req.District,
		Address:     req.Address,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		PhoneNumber: req.PhoneNumber,
		ImageURL:    req.ImageURL,
		Description: req.Description,
		OpenTime:    req.OpenTime,
		CloseTime:   req.CloseTime,
	})
	if err != nil {
		switch err {
		case service.ErrStoreNotFound:
			log.Warn("Cannot update store: not found", map[string]interface{}{
				"store_id": storeID,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Store not found",
			})
			return
		case service.ErrStoreAccessDenied:
			log.Warn("Store update forbidden", map[string]interface{}{
				"store_id": storeID,
				"user_id":  userID,
			})
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
			})
			return
		default:
			log.Error("Failed to update store", err, map[string]interface{}{
				"store_id": storeID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update store",
			})
			return
		}
	}

	log.Info("Store updated", map[string]interface{}{
		"store_id": storeID,
		"user_id":  userID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Store updated successfully",
		"store":   updated,
	})
}

func (ctrl *StoreController) DeleteStore(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("User ID not found in context for store deletion", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	idStr := c.Param("id")
	storeID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid store ID format for delete", map[string]interface{}{
			"store_id": idStr,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid store ID",
		})
		return
	}

	if err := ctrl.storeService.DeleteStore(userID, uint(storeID)); err != nil {
		switch err {
		case service.ErrStoreNotFound:
			log.Warn("Cannot delete store: not found", map[string]interface{}{
				"store_id": storeID,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Store not found",
			})
			return
		case service.ErrStoreAccessDenied:
			log.Warn("Store deletion forbidden", map[string]interface{}{
				"store_id": storeID,
				"user_id":  userID,
			})
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
			})
			return
		default:
			log.Error("Failed to delete store", err, map[string]interface{}{
				"store_id": storeID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to delete store",
			})
			return
		}
	}

	log.Info("Store deleted", map[string]interface{}{
		"store_id": storeID,
		"user_id":  userID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Store deleted successfully",
	})
}

func (ctrl *StoreController) ListLocations(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	locations, err := ctrl.storeService.ListLocations()
	if err != nil {
		log.Error("Failed to list store locations", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch store locations",
		})
		return
	}

	log.Info("Store locations listed", map[string]interface{}{
		"count": len(locations),
	})

	c.JSON(http.StatusOK, gin.H{
		"locations": locations,
		"count":     len(locations),
	})
}
