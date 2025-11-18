package repository

import (
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

type AddressRepository interface {
	Create(address *model.Address) error
	FindByUserID(userID uint) ([]model.Address, error)
	FindByID(id uint) (*model.Address, error)
	Update(address *model.Address) error
	Delete(id uint) error
	SetDefault(userID, addressID uint) error
}

type addressRepository struct {
	db *gorm.DB
}

func NewAddressRepository(db *gorm.DB) AddressRepository {
	return &addressRepository{db: db}
}

func (r *addressRepository) Create(address *model.Address) error {
	logger.Debug("Creating address in database", map[string]interface{}{
		"user_id":   address.UserID,
		"name":      address.Name,
		"recipient": address.Recipient,
	})

	if err := r.db.Create(address).Error; err != nil {
		logger.Error("Failed to create address in database", err, map[string]interface{}{
			"user_id":   address.UserID,
			"name":      address.Name,
			"recipient": address.Recipient,
		})
		return err
	}

	logger.Debug("Address created in database", map[string]interface{}{
		"address_id": address.ID,
		"user_id":    address.UserID,
		"name":       address.Name,
	})
	return nil
}

func (r *addressRepository) FindByUserID(userID uint) ([]model.Address, error) {
	logger.Debug("Finding addresses by user ID in database", map[string]interface{}{
		"user_id": userID,
	})

	var addresses []model.Address
	err := r.db.Where("user_id = ?", userID).
		Order("is_default DESC, created_at DESC").
		Find(&addresses).Error
	if err != nil {
		logger.Error("Failed to find addresses by user ID in database", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	logger.Debug("Addresses found by user ID in database", map[string]interface{}{
		"user_id": userID,
		"count":   len(addresses),
	})
	return addresses, nil
}

func (r *addressRepository) FindByID(id uint) (*model.Address, error) {
	logger.Debug("Finding address by ID in database", map[string]interface{}{
		"address_id": id,
	})

	var address model.Address
	err := r.db.First(&address, id).Error
	if err != nil {
		logger.Error("Failed to find address by ID in database", err, map[string]interface{}{
			"address_id": id,
		})
		return nil, err
	}

	logger.Debug("Address found by ID in database", map[string]interface{}{
		"address_id": address.ID,
		"user_id":    address.UserID,
		"name":       address.Name,
	})
	return &address, nil
}

func (r *addressRepository) Update(address *model.Address) error {
	logger.Debug("Updating address in database", map[string]interface{}{
		"address_id": address.ID,
		"user_id":    address.UserID,
		"name":       address.Name,
	})

	if err := r.db.Save(address).Error; err != nil {
		logger.Error("Failed to update address in database", err, map[string]interface{}{
			"address_id": address.ID,
			"user_id":    address.UserID,
		})
		return err
	}

	logger.Debug("Address updated in database", map[string]interface{}{
		"address_id": address.ID,
		"user_id":    address.UserID,
		"name":       address.Name,
	})
	return nil
}

func (r *addressRepository) Delete(id uint) error {
	logger.Debug("Deleting address from database", map[string]interface{}{
		"address_id": id,
	})

	if err := r.db.Delete(&model.Address{}, id).Error; err != nil {
		logger.Error("Failed to delete address from database", err, map[string]interface{}{
			"address_id": id,
		})
		return err
	}

	logger.Debug("Address deleted from database", map[string]interface{}{
		"address_id": id,
	})
	return nil
}

func (r *addressRepository) SetDefault(userID, addressID uint) error {
	logger.Debug("Setting default address", map[string]interface{}{
		"user_id":    userID,
		"address_id": addressID,
	})

	// Start transaction
	tx := r.db.Begin()
	if tx.Error != nil {
		logger.Error("Failed to begin transaction for setting default address", tx.Error, map[string]interface{}{
			"user_id":    userID,
			"address_id": addressID,
		})
		return tx.Error
	}

	// Unset all default addresses for this user
	if err := tx.Model(&model.Address{}).Where("user_id = ?", userID).Update("is_default", false).Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to unset default addresses", err, map[string]interface{}{
			"user_id": userID,
		})
		return err
	}

	// Set the specified address as default
	if err := tx.Model(&model.Address{}).Where("id = ? AND user_id = ?", addressID, userID).Update("is_default", true).Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to set address as default", err, map[string]interface{}{
			"user_id":    userID,
			"address_id": addressID,
		})
		return err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		logger.Error("Failed to commit transaction for setting default address", err, map[string]interface{}{
			"user_id":    userID,
			"address_id": addressID,
		})
		return err
	}

	logger.Debug("Default address set successfully", map[string]interface{}{
		"user_id":    userID,
		"address_id": addressID,
	})
	return nil
}
