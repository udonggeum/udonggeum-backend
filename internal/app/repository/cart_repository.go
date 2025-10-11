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
		"user_id":    cartItem.UserID,
		"product_id": cartItem.ProductID,
		"quantity":   cartItem.Quantity,
	})

	if err := r.db.Create(cartItem).Error; err != nil {
		logger.Error("Failed to create cart item in database", err, map[string]interface{}{
			"user_id":    cartItem.UserID,
			"product_id": cartItem.ProductID,
			"quantity":   cartItem.Quantity,
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
		Preload("Product").
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
	err := r.db.Preload("Product").First(&cartItem, id).Error
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
	logger.Debug("Finding cart item by user and product in database", map[string]interface{}{
		"user_id":    userID,
		"product_id": productID,
	})

	var cartItem model.CartItem
	err := r.db.Where("user_id = ? AND product_id = ?", userID, productID).
		First(&cartItem).Error
	if err != nil {
		logger.Error("Failed to find cart item by user and product in database", err, map[string]interface{}{
			"user_id":    userID,
			"product_id": productID,
		})
		return nil, err
	}

	logger.Debug("Cart item found by user and product in database", map[string]interface{}{
		"cart_item_id": cartItem.ID,
		"user_id":      userID,
		"product_id":   productID,
	})
	return &cartItem, nil
}

func (r *cartRepository) Update(cartItem *model.CartItem) error {
	logger.Debug("Updating cart item in database", map[string]interface{}{
		"cart_item_id": cartItem.ID,
		"user_id":      cartItem.UserID,
		"product_id":   cartItem.ProductID,
		"quantity":     cartItem.Quantity,
	})

	if err := r.db.Save(cartItem).Error; err != nil {
		logger.Error("Failed to update cart item in database", err, map[string]interface{}{
			"cart_item_id": cartItem.ID,
			"user_id":      cartItem.UserID,
			"product_id":   cartItem.ProductID,
		})
		return err
	}

	logger.Debug("Cart item updated in database", map[string]interface{}{
		"cart_item_id": cartItem.ID,
		"user_id":      cartItem.UserID,
		"product_id":   cartItem.ProductID,
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
