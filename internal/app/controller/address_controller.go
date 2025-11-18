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

type AddressController struct {
	addressService service.AddressService
}

func NewAddressController(addressService service.AddressService) *AddressController {
	return &AddressController{
		addressService: addressService,
	}
}

type CreateAddressRequest struct {
	Name          string `json:"name" binding:"required"`
	Recipient     string `json:"recipient" binding:"required"`
	Phone         string `json:"phone" binding:"required"`
	ZipCode       string `json:"zip_code"`
	Address       string `json:"address" binding:"required"`
	DetailAddress string `json:"detail_address"`
	IsDefault     bool   `json:"is_default"`
}

type UpdateAddressRequest struct {
	Name          string `json:"name" binding:"required"`
	Recipient     string `json:"recipient" binding:"required"`
	Phone         string `json:"phone" binding:"required"`
	ZipCode       string `json:"zip_code"`
	Address       string `json:"address" binding:"required"`
	DetailAddress string `json:"detail_address"`
	IsDefault     bool   `json:"is_default"`
}

// ListAddresses returns user's addresses
// GET /api/v1/addresses
func (ctrl *AddressController) ListAddresses(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized access to addresses", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	addresses, err := ctrl.addressService.GetUserAddresses(userID)
	if err != nil {
		log.Error("Failed to fetch addresses", err, map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch addresses",
		})
		return
	}

	log.Info("Addresses fetched successfully", map[string]interface{}{
		"user_id": userID,
		"count":   len(addresses),
	})

	c.JSON(http.StatusOK, gin.H{
		"addresses": addresses,
		"count":     len(addresses),
	})
}

// CreateAddress creates a new address
// POST /api/v1/addresses
func (ctrl *AddressController) CreateAddress(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized attempt to create address", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req CreateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid create address request", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Creating address", map[string]interface{}{
		"user_id":   userID,
		"name":      req.Name,
		"recipient": req.Recipient,
	})

	address := &model.Address{
		Name:          req.Name,
		Recipient:     req.Recipient,
		Phone:         req.Phone,
		ZipCode:       req.ZipCode,
		Address:       req.Address,
		DetailAddress: req.DetailAddress,
		IsDefault:     req.IsDefault,
	}

	err := ctrl.addressService.CreateAddress(userID, address)
	if err != nil {
		log.Error("Failed to create address", err, map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create address",
		})
		return
	}

	log.Info("Address created successfully", map[string]interface{}{
		"user_id":    userID,
		"address_id": address.ID,
	})

	c.JSON(http.StatusCreated, gin.H{
		"message": "Address created successfully",
		"address": address,
	})
}

// UpdateAddress updates an existing address
// PUT /api/v1/addresses/:id
func (ctrl *AddressController) UpdateAddress(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized attempt to update address", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid address ID format", map[string]interface{}{
			"user_id":    userID,
			"address_id": idStr,
			"error":      err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid address ID",
		})
		return
	}

	var req UpdateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid update address request", map[string]interface{}{
			"user_id":    userID,
			"address_id": id,
			"error":      err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Updating address", map[string]interface{}{
		"user_id":    userID,
		"address_id": id,
	})

	updatedAddress := &model.Address{
		Name:          req.Name,
		Recipient:     req.Recipient,
		Phone:         req.Phone,
		ZipCode:       req.ZipCode,
		Address:       req.Address,
		DetailAddress: req.DetailAddress,
		IsDefault:     req.IsDefault,
	}

	err = ctrl.addressService.UpdateAddress(userID, uint(id), updatedAddress)
	if err != nil {
		if errors.Is(err, service.ErrAddressNotFound) {
			log.Warn("Address not found", map[string]interface{}{
				"user_id":    userID,
				"address_id": id,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Address not found",
			})
			return
		}
		if errors.Is(err, service.ErrUnauthorizedAccess) {
			log.Warn("Unauthorized access to address", map[string]interface{}{
				"user_id":    userID,
				"address_id": id,
			})
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Unauthorized access to address",
			})
			return
		}
		log.Error("Failed to update address", err, map[string]interface{}{
			"user_id":    userID,
			"address_id": id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update address",
		})
		return
	}

	log.Info("Address updated successfully", map[string]interface{}{
		"user_id":    userID,
		"address_id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Address updated successfully",
	})
}

// DeleteAddress deletes an address
// DELETE /api/v1/addresses/:id
func (ctrl *AddressController) DeleteAddress(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized attempt to delete address", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid address ID format", map[string]interface{}{
			"user_id":    userID,
			"address_id": idStr,
			"error":      err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid address ID",
		})
		return
	}

	log.Debug("Deleting address", map[string]interface{}{
		"user_id":    userID,
		"address_id": id,
	})

	err = ctrl.addressService.DeleteAddress(userID, uint(id))
	if err != nil {
		if errors.Is(err, service.ErrAddressNotFound) {
			log.Warn("Address not found for deletion", map[string]interface{}{
				"user_id":    userID,
				"address_id": id,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Address not found",
			})
			return
		}
		if errors.Is(err, service.ErrUnauthorizedAccess) {
			log.Warn("Unauthorized attempt to delete address", map[string]interface{}{
				"user_id":    userID,
				"address_id": id,
			})
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Unauthorized access to address",
			})
			return
		}
		log.Error("Failed to delete address", err, map[string]interface{}{
			"user_id":    userID,
			"address_id": id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete address",
		})
		return
	}

	log.Info("Address deleted successfully", map[string]interface{}{
		"user_id":    userID,
		"address_id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Address deleted successfully",
	})
}

// SetDefaultAddress sets an address as the default
// PUT /api/v1/addresses/:id/default
func (ctrl *AddressController) SetDefaultAddress(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized attempt to set default address", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid address ID format", map[string]interface{}{
			"user_id":    userID,
			"address_id": idStr,
			"error":      err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid address ID",
		})
		return
	}

	log.Debug("Setting default address", map[string]interface{}{
		"user_id":    userID,
		"address_id": id,
	})

	err = ctrl.addressService.SetDefaultAddress(userID, uint(id))
	if err != nil {
		if errors.Is(err, service.ErrAddressNotFound) {
			log.Warn("Address not found", map[string]interface{}{
				"user_id":    userID,
				"address_id": id,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Address not found",
			})
			return
		}
		if errors.Is(err, service.ErrUnauthorizedAccess) {
			log.Warn("Unauthorized attempt to set default address", map[string]interface{}{
				"user_id":    userID,
				"address_id": id,
			})
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Unauthorized access to address",
			})
			return
		}
		log.Error("Failed to set default address", err, map[string]interface{}{
			"user_id":    userID,
			"address_id": id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to set default address",
		})
		return
	}

	log.Info("Default address set successfully", map[string]interface{}{
		"user_id":    userID,
		"address_id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Default address set successfully",
	})
}
