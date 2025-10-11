package service

import (
	"errors"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

var (
	ErrCartItemNotFound = errors.New("cart item not found")
)

type CartService interface {
	GetUserCart(userID uint) ([]model.CartItem, error)
	AddToCart(userID, productID uint, quantity int) error
	UpdateCartItem(userID, cartItemID uint, quantity int) error
	RemoveFromCart(userID, cartItemID uint) error
	ClearCart(userID uint) error
}

type cartService struct {
	cartRepo    repository.CartRepository
	productRepo repository.ProductRepository
}

func NewCartService(
	cartRepo repository.CartRepository,
	productRepo repository.ProductRepository,
) CartService {
	return &cartService{
		cartRepo:    cartRepo,
		productRepo: productRepo,
	}
}

func (s *cartService) GetUserCart(userID uint) ([]model.CartItem, error) {
	logger.Debug("Fetching user cart", map[string]interface{}{
		"user_id": userID,
	})

	cartItems, err := s.cartRepo.FindByUserID(userID)
	if err != nil {
		logger.Error("Failed to fetch user cart", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	logger.Info("User cart fetched successfully", map[string]interface{}{
		"user_id": userID,
		"count":   len(cartItems),
	})
	return cartItems, nil
}

func (s *cartService) AddToCart(userID, productID uint, quantity int) error {
	logger.Info("Adding item to cart", map[string]interface{}{
		"user_id":    userID,
		"product_id": productID,
		"quantity":   quantity,
	})

	// Check if product exists and has sufficient stock
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Cannot add to cart: product not found", map[string]interface{}{
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

	if product.StockQuantity < quantity {
		logger.Warn("Cannot add to cart: insufficient stock", map[string]interface{}{
			"user_id":          userID,
			"product_id":       productID,
			"requested":        quantity,
			"available_stock":  product.StockQuantity,
		})
		return ErrInsufficientStock
	}

	// Check if item already in cart
	existingItem, err := s.cartRepo.FindByUserAndProduct(userID, productID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error("Failed to check existing cart item", err, map[string]interface{}{
			"user_id":    userID,
			"product_id": productID,
		})
		return err
	}

	if existingItem != nil {
		// Update quantity
		logger.Debug("Updating existing cart item", map[string]interface{}{
			"user_id":       userID,
			"cart_item_id":  existingItem.ID,
			"old_quantity":  existingItem.Quantity,
			"new_quantity":  existingItem.Quantity + quantity,
		})
		existingItem.Quantity += quantity
		if err := s.cartRepo.Update(existingItem); err != nil {
			logger.Error("Failed to update cart item", err, map[string]interface{}{
				"user_id":      userID,
				"cart_item_id": existingItem.ID,
			})
			return err
		}
		logger.Info("Cart item updated successfully", map[string]interface{}{
			"user_id":      userID,
			"cart_item_id": existingItem.ID,
			"product_id":   productID,
			"quantity":     existingItem.Quantity,
		})
		return nil
	}

	// Create new cart item
	cartItem := &model.CartItem{
		UserID:    userID,
		ProductID: productID,
		Quantity:  quantity,
	}

	if err := s.cartRepo.Create(cartItem); err != nil {
		logger.Error("Failed to create cart item", err, map[string]interface{}{
			"user_id":    userID,
			"product_id": productID,
			"quantity":   quantity,
		})
		return err
	}

	logger.Info("Cart item added successfully", map[string]interface{}{
		"user_id":      userID,
		"cart_item_id": cartItem.ID,
		"product_id":   productID,
		"quantity":     quantity,
	})
	return nil
}

func (s *cartService) UpdateCartItem(userID, cartItemID uint, quantity int) error {
	logger.Info("Updating cart item", map[string]interface{}{
		"user_id":      userID,
		"cart_item_id": cartItemID,
		"quantity":     quantity,
	})

	cartItem, err := s.cartRepo.FindByID(cartItemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Cart item not found", map[string]interface{}{
				"user_id":      userID,
				"cart_item_id": cartItemID,
			})
			return ErrCartItemNotFound
		}
		logger.Error("Failed to fetch cart item", err, map[string]interface{}{
			"user_id":      userID,
			"cart_item_id": cartItemID,
		})
		return err
	}

	// Verify ownership
	if cartItem.UserID != userID {
		logger.Warn("Cart item access denied: ownership mismatch", map[string]interface{}{
			"user_id":      userID,
			"cart_item_id": cartItemID,
			"owner_id":     cartItem.UserID,
		})
		return ErrCartItemNotFound
	}

	// Check stock
	product, err := s.productRepo.FindByID(cartItem.ProductID)
	if err != nil {
		logger.Error("Failed to fetch product for stock check", err, map[string]interface{}{
			"user_id":      userID,
			"cart_item_id": cartItemID,
			"product_id":   cartItem.ProductID,
		})
		return err
	}

	if product.StockQuantity < quantity {
		logger.Warn("Cannot update cart item: insufficient stock", map[string]interface{}{
			"user_id":         userID,
			"cart_item_id":    cartItemID,
			"product_id":      cartItem.ProductID,
			"requested":       quantity,
			"available_stock": product.StockQuantity,
		})
		return ErrInsufficientStock
	}

	logger.Debug("Updating cart item quantity", map[string]interface{}{
		"user_id":      userID,
		"cart_item_id": cartItemID,
		"old_quantity": cartItem.Quantity,
		"new_quantity": quantity,
	})

	cartItem.Quantity = quantity
	if err := s.cartRepo.Update(cartItem); err != nil {
		logger.Error("Failed to update cart item", err, map[string]interface{}{
			"user_id":      userID,
			"cart_item_id": cartItemID,
		})
		return err
	}

	logger.Info("Cart item updated successfully", map[string]interface{}{
		"user_id":      userID,
		"cart_item_id": cartItemID,
		"product_id":   cartItem.ProductID,
		"quantity":     quantity,
	})
	return nil
}

func (s *cartService) RemoveFromCart(userID, cartItemID uint) error {
	logger.Info("Removing item from cart", map[string]interface{}{
		"user_id":      userID,
		"cart_item_id": cartItemID,
	})

	cartItem, err := s.cartRepo.FindByID(cartItemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Cart item not found", map[string]interface{}{
				"user_id":      userID,
				"cart_item_id": cartItemID,
			})
			return ErrCartItemNotFound
		}
		logger.Error("Failed to fetch cart item", err, map[string]interface{}{
			"user_id":      userID,
			"cart_item_id": cartItemID,
		})
		return err
	}

	// Verify ownership
	if cartItem.UserID != userID {
		logger.Warn("Cart item removal denied: ownership mismatch", map[string]interface{}{
			"user_id":      userID,
			"cart_item_id": cartItemID,
			"owner_id":     cartItem.UserID,
		})
		return ErrCartItemNotFound
	}

	if err := s.cartRepo.Delete(cartItemID); err != nil {
		logger.Error("Failed to delete cart item", err, map[string]interface{}{
			"user_id":      userID,
			"cart_item_id": cartItemID,
		})
		return err
	}

	logger.Info("Cart item removed successfully", map[string]interface{}{
		"user_id":      userID,
		"cart_item_id": cartItemID,
		"product_id":   cartItem.ProductID,
	})
	return nil
}

func (s *cartService) ClearCart(userID uint) error {
	logger.Info("Clearing user cart", map[string]interface{}{
		"user_id": userID,
	})

	if err := s.cartRepo.DeleteByUserID(userID); err != nil {
		logger.Error("Failed to clear cart", err, map[string]interface{}{
			"user_id": userID,
		})
		return err
	}

	logger.Info("Cart cleared successfully", map[string]interface{}{
		"user_id": userID,
	})
	return nil
}
