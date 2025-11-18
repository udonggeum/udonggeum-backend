package repository

import (
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

type WishlistRepository interface {
	Create(item *model.WishlistItem) error
	FindByUserID(userID uint) ([]model.WishlistItem, error)
	FindByUserAndProduct(userID, productID uint) (*model.WishlistItem, error)
	Delete(userID, productID uint) error
}

type wishlistRepository struct {
	db *gorm.DB
}

func NewWishlistRepository(db *gorm.DB) WishlistRepository {
	return &wishlistRepository{db: db}
}

func (r *wishlistRepository) Create(item *model.WishlistItem) error {
	logger.Debug("Creating wishlist item in database", map[string]interface{}{
		"user_id":    item.UserID,
		"product_id": item.ProductID,
	})

	if err := r.db.Create(item).Error; err != nil {
		logger.Error("Failed to create wishlist item in database", err, map[string]interface{}{
			"user_id":    item.UserID,
			"product_id": item.ProductID,
		})
		return err
	}

	logger.Debug("Wishlist item created in database", map[string]interface{}{
		"wishlist_item_id": item.ID,
		"user_id":          item.UserID,
		"product_id":       item.ProductID,
	})
	return nil
}

func (r *wishlistRepository) FindByUserID(userID uint) ([]model.WishlistItem, error) {
	logger.Debug("Finding wishlist items by user ID in database", map[string]interface{}{
		"user_id": userID,
	})

	var items []model.WishlistItem
	err := r.db.Where("user_id = ?", userID).
		Preload("Product", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Store")
		}).
		Order("created_at DESC").
		Find(&items).Error
	if err != nil {
		logger.Error("Failed to find wishlist items by user ID in database", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	logger.Debug("Wishlist items found by user ID in database", map[string]interface{}{
		"user_id": userID,
		"count":   len(items),
	})
	return items, nil
}

func (r *wishlistRepository) FindByUserAndProduct(userID, productID uint) (*model.WishlistItem, error) {
	logger.Debug("Finding wishlist item by user and product", map[string]interface{}{
		"user_id":    userID,
		"product_id": productID,
	})

	var item model.WishlistItem
	err := r.db.Where("user_id = ? AND product_id = ?", userID, productID).First(&item).Error
	if err != nil {
		logger.Error("Failed to find wishlist item by user and product", err, map[string]interface{}{
			"user_id":    userID,
			"product_id": productID,
		})
		return nil, err
	}

	logger.Debug("Wishlist item found by user and product", map[string]interface{}{
		"wishlist_item_id": item.ID,
	})
	return &item, nil
}

func (r *wishlistRepository) Delete(userID, productID uint) error {
	logger.Debug("Deleting wishlist item from database", map[string]interface{}{
		"user_id":    userID,
		"product_id": productID,
	})

	if err := r.db.Where("user_id = ? AND product_id = ?", userID, productID).Delete(&model.WishlistItem{}).Error; err != nil {
		logger.Error("Failed to delete wishlist item from database", err, map[string]interface{}{
			"user_id":    userID,
			"product_id": productID,
		})
		return err
	}

	logger.Debug("Wishlist item deleted from database", map[string]interface{}{
		"user_id":    userID,
		"product_id": productID,
	})
	return nil
}
