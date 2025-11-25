package service

import (
	"errors"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

var (
	ErrWishlistItemAlreadyExists = errors.New("product already in wishlist")
	ErrWishlistItemNotFound      = errors.New("wishlist item not found")
)

type WishlistService interface {
	GetUserWishlist(userID uint) ([]model.WishlistItem, error)
	AddToWishlist(userID, productID uint) error
	RemoveFromWishlist(userID, productID uint) error
}

type wishlistService struct {
	wishlistRepo repository.WishlistRepository
	productRepo  repository.ProductRepository
}

func NewWishlistService(
	wishlistRepo repository.WishlistRepository,
	productRepo repository.ProductRepository,
) WishlistService {
	return &wishlistService{
		wishlistRepo: wishlistRepo,
		productRepo:  productRepo,
	}
}

func (s *wishlistService) GetUserWishlist(userID uint) ([]model.WishlistItem, error) {
	logger.Debug("Fetching user wishlist", map[string]interface{}{
		"user_id": userID,
	})

	items, err := s.wishlistRepo.FindByUserID(userID)
	if err != nil {
		logger.Error("Failed to fetch user wishlist", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	logger.Info("User wishlist fetched successfully", map[string]interface{}{
		"user_id": userID,
		"count":   len(items),
	})
	return items, nil
}

func (s *wishlistService) AddToWishlist(userID, productID uint) error {
	logger.Info("Adding item to wishlist", map[string]interface{}{
		"user_id":    userID,
		"product_id": productID,
	})

	// Check if product exists
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Cannot add to wishlist: product not found", map[string]interface{}{
				"user_id":    userID,
				"product_id": productID,
			})
			return ErrProductNotFound
		}
		logger.Error("Failed to fetch product", err, map[string]interface{}{
			"user_id":    userID,
			"product_id": productID,
		})
		return err
	}

	// Check if item already in wishlist
	existingItem, err := s.wishlistRepo.FindByUserAndProduct(userID, productID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error("Failed to check existing wishlist item", err, map[string]interface{}{
			"user_id":    userID,
			"product_id": productID,
		})
		return err
	}

	if existingItem != nil {
		logger.Warn("Product already in wishlist", map[string]interface{}{
			"user_id":    userID,
			"product_id": productID,
		})
		return ErrWishlistItemAlreadyExists
	}

	// Create wishlist item
	item := &model.WishlistItem{
		UserID:    userID,
		ProductID: productID,
	}

	if err := s.wishlistRepo.Create(item); err != nil {
		logger.Error("Failed to create wishlist item", err, map[string]interface{}{
			"user_id":    userID,
			"product_id": productID,
		})
		return err
	}

	logger.Info("Item added to wishlist successfully", map[string]interface{}{
		"wishlist_item_id": item.ID,
		"user_id":          userID,
		"product_id":       product.ID,
	})
	return nil
}

func (s *wishlistService) RemoveFromWishlist(userID, productID uint) error {
	logger.Info("Removing item from wishlist", map[string]interface{}{
		"user_id":    userID,
		"product_id": productID,
	})

	// Check if item exists in wishlist
	existingItem, err := s.wishlistRepo.FindByUserAndProduct(userID, productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Wishlist item not found", map[string]interface{}{
				"user_id":    userID,
				"product_id": productID,
			})
			return ErrWishlistItemNotFound
		}
		logger.Error("Failed to find wishlist item", err, map[string]interface{}{
			"user_id":    userID,
			"product_id": productID,
		})
		return err
	}

	if err := s.wishlistRepo.Delete(userID, productID); err != nil {
		logger.Error("Failed to delete wishlist item", err, map[string]interface{}{
			"user_id":    userID,
			"product_id": productID,
		})
		return err
	}

	logger.Info("Item removed from wishlist successfully", map[string]interface{}{
		"wishlist_item_id": existingItem.ID,
		"user_id":          userID,
		"product_id":       productID,
	})
	return nil
}
