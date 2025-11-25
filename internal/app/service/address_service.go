package service

import (
	"errors"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

var (
	ErrAddressNotFound        = errors.New("address not found")
	ErrUnauthorizedAccess     = errors.New("unauthorized access to address")
)

type AddressService interface {
	GetUserAddresses(userID uint) ([]model.Address, error)
	CreateAddress(userID uint, address *model.Address) error
	UpdateAddress(userID, addressID uint, updatedAddress *model.Address) error
	DeleteAddress(userID, addressID uint) error
	SetDefaultAddress(userID, addressID uint) error
}

type addressService struct {
	addressRepo repository.AddressRepository
}

func NewAddressService(addressRepo repository.AddressRepository) AddressService {
	return &addressService{
		addressRepo: addressRepo,
	}
}

func (s *addressService) GetUserAddresses(userID uint) ([]model.Address, error) {
	logger.Debug("Fetching user addresses", map[string]interface{}{
		"user_id": userID,
	})

	addresses, err := s.addressRepo.FindByUserID(userID)
	if err != nil {
		logger.Error("Failed to fetch user addresses", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	logger.Info("User addresses fetched successfully", map[string]interface{}{
		"user_id": userID,
		"count":   len(addresses),
	})
	return addresses, nil
}

func (s *addressService) CreateAddress(userID uint, address *model.Address) error {
	logger.Info("Creating address", map[string]interface{}{
		"user_id":   userID,
		"name":      address.Name,
		"recipient": address.Recipient,
	})

	// Set the user ID
	address.UserID = userID

	// If this is the first address, make it default
	existingAddresses, err := s.addressRepo.FindByUserID(userID)
	if err != nil {
		logger.Error("Failed to check existing addresses", err, map[string]interface{}{
			"user_id": userID,
		})
		return err
	}

	if len(existingAddresses) == 0 {
		address.IsDefault = true
		logger.Debug("Setting first address as default", map[string]interface{}{
			"user_id": userID,
		})
	}

	// If address is set as default, unset other defaults
	if address.IsDefault {
		if err := s.addressRepo.SetDefault(userID, 0); err != nil {
			logger.Error("Failed to unset default addresses", err, map[string]interface{}{
				"user_id": userID,
			})
			return err
		}
	}

	if err := s.addressRepo.Create(address); err != nil {
		logger.Error("Failed to create address", err, map[string]interface{}{
			"user_id": userID,
		})
		return err
	}

	logger.Info("Address created successfully", map[string]interface{}{
		"address_id": address.ID,
		"user_id":    userID,
	})
	return nil
}

func (s *addressService) UpdateAddress(userID, addressID uint, updatedAddress *model.Address) error {
	logger.Info("Updating address", map[string]interface{}{
		"user_id":    userID,
		"address_id": addressID,
	})

	// Fetch existing address
	address, err := s.addressRepo.FindByID(addressID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Address not found", map[string]interface{}{
				"address_id": addressID,
			})
			return ErrAddressNotFound
		}
		logger.Error("Failed to fetch address", err, map[string]interface{}{
			"address_id": addressID,
		})
		return err
	}

	// Check ownership
	if address.UserID != userID {
		logger.Warn("Unauthorized access to address", map[string]interface{}{
			"user_id":    userID,
			"address_id": addressID,
			"owner_id":   address.UserID,
		})
		return ErrUnauthorizedAccess
	}

	// Update fields
	address.Name = updatedAddress.Name
	address.Recipient = updatedAddress.Recipient
	address.Phone = updatedAddress.Phone
	address.ZipCode = updatedAddress.ZipCode
	address.Address = updatedAddress.Address
	address.DetailAddress = updatedAddress.DetailAddress

	// Handle default status change
	if updatedAddress.IsDefault && !address.IsDefault {
		if err := s.addressRepo.SetDefault(userID, addressID); err != nil {
			logger.Error("Failed to set address as default", err, map[string]interface{}{
				"user_id":    userID,
				"address_id": addressID,
			})
			return err
		}
		address.IsDefault = true
	} else {
		address.IsDefault = updatedAddress.IsDefault
	}

	if err := s.addressRepo.Update(address); err != nil {
		logger.Error("Failed to update address", err, map[string]interface{}{
			"address_id": addressID,
		})
		return err
	}

	logger.Info("Address updated successfully", map[string]interface{}{
		"address_id": addressID,
	})
	return nil
}

func (s *addressService) DeleteAddress(userID, addressID uint) error {
	logger.Info("Deleting address", map[string]interface{}{
		"user_id":    userID,
		"address_id": addressID,
	})

	// Fetch existing address
	address, err := s.addressRepo.FindByID(addressID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Address not found for deletion", map[string]interface{}{
				"address_id": addressID,
			})
			return ErrAddressNotFound
		}
		logger.Error("Failed to fetch address for deletion", err, map[string]interface{}{
			"address_id": addressID,
		})
		return err
	}

	// Check ownership
	if address.UserID != userID {
		logger.Warn("Unauthorized attempt to delete address", map[string]interface{}{
			"user_id":    userID,
			"address_id": addressID,
			"owner_id":   address.UserID,
		})
		return ErrUnauthorizedAccess
	}

	if err := s.addressRepo.Delete(addressID); err != nil {
		logger.Error("Failed to delete address", err, map[string]interface{}{
			"address_id": addressID,
		})
		return err
	}

	logger.Info("Address deleted successfully", map[string]interface{}{
		"address_id": addressID,
	})
	return nil
}

func (s *addressService) SetDefaultAddress(userID, addressID uint) error {
	logger.Info("Setting default address", map[string]interface{}{
		"user_id":    userID,
		"address_id": addressID,
	})

	// Fetch address to verify ownership
	address, err := s.addressRepo.FindByID(addressID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Address not found", map[string]interface{}{
				"address_id": addressID,
			})
			return ErrAddressNotFound
		}
		logger.Error("Failed to fetch address", err, map[string]interface{}{
			"address_id": addressID,
		})
		return err
	}

	// Check ownership
	if address.UserID != userID {
		logger.Warn("Unauthorized attempt to set default address", map[string]interface{}{
			"user_id":    userID,
			"address_id": addressID,
			"owner_id":   address.UserID,
		})
		return ErrUnauthorizedAccess
	}

	if err := s.addressRepo.SetDefault(userID, addressID); err != nil {
		logger.Error("Failed to set default address", err, map[string]interface{}{
			"user_id":    userID,
			"address_id": addressID,
		})
		return err
	}

	logger.Info("Default address set successfully", map[string]interface{}{
		"user_id":    userID,
		"address_id": addressID,
	})
	return nil
}
