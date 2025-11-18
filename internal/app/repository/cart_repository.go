package repository

import (
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

type CartRepository interface {
	Create(cartItem *model.CartItem) error
	FindByUserID(userID uint) ([]model.CartItem, error)
	FindByID(id uint) (*model.CartItem, error)
	FindByUserProductOption(userID, productID uint, productOptionID *uint) (*model.CartItem, error)
	FindByUserAndProduct(userID, productID uint) (*model.CartItem, error)
	Update(cartItem *model.CartItem) error
	Delete(id uint) error
	DeleteByUserID(userID uint) error
}

type cartRepository struct {
	db *gorm.DB
}

func NewCartRepository(db *gorm.DB) CartRepository {
	return &cartRepository{db: db}
}

func (r *cartRepository) Create(cartItem *model.CartItem) error {
	logger.Debug("Creating cart item in database", map[string]interface{}{
		"user_id":           cartItem.UserID,
		"product_id":        cartItem.ProductID,
		"product_option_id": cartItem.ProductOptionID,
		"quantity":          cartItem.Quantity,
	})

	if err := r.db.Create(cartItem).Error; err != nil {
		logger.Error("Failed to create cart item in database", err, map[string]interface{}{
			"user_id":           cartItem.UserID,
			"product_id":        cartItem.ProductID,
			"product_option_id": cartItem.ProductOptionID,
			"quantity":          cartItem.Quantity,
		})
		return err
	}

	logger.Debug("Cart item created in database", map[string]interface{}{
		"cart_item_id": cartItem.ID,
		"user_id":      cartItem.UserID,
		"product_id":   cartItem.ProductID,
	})
	return nil
}

func (r *cartRepository) FindByUserID(userID uint) ([]model.CartItem, error) {
	logger.Debug("Finding cart items by user ID in database", map[string]interface{}{
		"user_id": userID,
	})

	var cartItems []model.CartItem
	err := r.db.Where("user_id = ?", userID).
		Preload("Product", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Store").Preload("Options")
		}).
		Preload("ProductOption").
		Find(&cartItems).Error
	if err != nil {
		logger.Error("Failed to find cart items by user ID in database", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	logger.Debug("Cart items found by user ID in database", map[string]interface{}{
		"user_id": userID,
		"count":   len(cartItems),
	})
	return cartItems, nil
}

func (r *cartRepository) FindByID(id uint) (*model.CartItem, error) {
	logger.Debug("Finding cart item by ID in database", map[string]interface{}{
		"cart_item_id": id,
	})

	var cartItem model.CartItem
	err := r.db.Preload("Product", func(db *gorm.DB) *gorm.DB {
		return db.Preload("Store").Preload("Options")
	}).
		Preload("ProductOption").
		First(&cartItem, id).Error
	if err != nil {
		logger.Error("Failed to find cart item by ID in database", err, map[string]interface{}{
			"cart_item_id": id,
		})
		return nil, err
	}

	logger.Debug("Cart item found by ID in database", map[string]interface{}{
		"cart_item_id": cartItem.ID,
		"user_id":      cartItem.UserID,
		"product_id":   cartItem.ProductID,
	})
	return &cartItem, nil
}

func (r *cartRepository) FindByUserAndProduct(userID, productID uint) (*model.CartItem, error) {
	return r.FindByUserProductOption(userID, productID, nil)
}

func (r *cartRepository) FindByUserProductOption(userID, productID uint, productOptionID *uint) (*model.CartItem, error) {
	logger.Debug("Finding cart item by user, product, and option", map[string]interface{}{
		"user_id":           userID,
		"product_id":        productID,
		"product_option_id": productOptionID,
	})

	query := r.db.Where("user_id = ? AND product_id = ?", userID, productID)
	if productOptionID == nil {
		query = query.Where("product_option_id IS NULL")
	} else {
		query = query.Where("product_option_id = ?", *productOptionID)
	}

	var cartItem model.CartItem
	err := query.First(&cartItem).Error
	if err != nil {
		logger.Error("Failed to find cart item by user/product/option", err, map[string]interface{}{
			"user_id":           userID,
			"product_id":        productID,
			"product_option_id": productOptionID,
		})
		return nil, err
	}

	logger.Debug("Cart item found by user/product/option", map[string]interface{}{
		"cart_item_id": cartItem.ID,
	})
	return &cartItem, nil
}

func (r *cartRepository) Update(cartItem *model.CartItem) error {
	logger.Debug("Updating cart item in database", map[string]interface{}{
		"cart_item_id":      cartItem.ID,
		"user_id":           cartItem.UserID,
		"product_id":        cartItem.ProductID,
		"quantity":          cartItem.Quantity,
		"product_option_id": cartItem.ProductOptionID,
	})

	// Use Updates to properly handle pointer fields (like ProductOptionID)
	updates := map[string]interface{}{
		"quantity": cartItem.Quantity,
	}

	// Explicitly set product_option_id (handles both nil and non-nil cases)
	if cartItem.ProductOptionID == nil {
		updates["product_option_id"] = nil
	} else {
		updates["product_option_id"] = *cartItem.ProductOptionID
	}

	if err := r.db.Model(&model.CartItem{}).Where("id = ?", cartItem.ID).Updates(updates).Error; err != nil {
		logger.Error("Failed to update cart item in database", err, map[string]interface{}{
			"cart_item_id": cartItem.ID,
			"user_id":      cartItem.UserID,
			"product_id":   cartItem.ProductID,
		})
		return err
	}

	logger.Debug("Cart item updated in database", map[string]interface{}{
		"cart_item_id":      cartItem.ID,
		"user_id":           cartItem.UserID,
		"product_id":        cartItem.ProductID,
		"product_option_id": cartItem.ProductOptionID,
	})
	return nil
}

func (r *cartRepository) Delete(id uint) error {
	logger.Debug("Deleting cart item from database", map[string]interface{}{
		"cart_item_id": id,
	})

	if err := r.db.Delete(&model.CartItem{}, id).Error; err != nil {
		logger.Error("Failed to delete cart item from database", err, map[string]interface{}{
			"cart_item_id": id,
		})
		return err
	}

	logger.Debug("Cart item deleted from database", map[string]interface{}{
		"cart_item_id": id,
	})
	return nil
}

func (r *cartRepository) DeleteByUserID(userID uint) error {
	logger.Debug("Deleting cart items by user ID from database", map[string]interface{}{
		"user_id": userID,
	})

	if err := r.db.Where("user_id = ?", userID).Delete(&model.CartItem{}).Error; err != nil {
		logger.Error("Failed to delete cart items by user ID from database", err, map[string]interface{}{
			"user_id": userID,
		})
		return err
	}

	logger.Debug("Cart items deleted by user ID from database", map[string]interface{}{
		"user_id": userID,
	})
	return nil
}
