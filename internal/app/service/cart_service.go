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
	AddToCart(userID, productID uint, productOptionID *uint, quantity int) error
	UpdateCartItem(userID, cartItemID uint, quantity int) error
	RemoveFromCart(userID, cartItemID uint) error
	ClearCart(userID uint) error
}

type cartService struct {
	cartRepo          repository.CartRepository
	productRepo       repository.ProductRepository
	productOptionRepo repository.ProductOptionRepository
}

func NewCartService(
	cartRepo repository.CartRepository,
	productRepo repository.ProductRepository,
	productOptionRepo ...repository.ProductOptionRepository,
) CartService {
	var optionRepo repository.ProductOptionRepository
	if len(productOptionRepo) > 0 {
		optionRepo = productOptionRepo[0]
	}
	return &cartService{
		cartRepo:          cartRepo,
		productRepo:       productRepo,
		productOptionRepo: optionRepo,
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

func (s *cartService) AddToCart(userID, productID uint, productOptionID *uint, quantity int) error {
	logger.Info("Adding item to cart", map[string]interface{}{
		"user_id":           userID,
		"product_id":        productID,
		"product_option_id": productOptionID,
		"quantity":          quantity,
	})

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

	var option *model.ProductOption
	if productOptionID != nil {
		if s.productOptionRepo == nil {
			logger.Warn("Cannot add to cart: product option repository unavailable", map[string]interface{}{
				"user_id":           userID,
				"product_id":        productID,
				"product_option_id": *productOptionID,
			})
			return ErrInvalidProductOption
		}
		opt, err := s.productOptionRepo.FindByID(*productOptionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Warn("Product option not found", map[string]interface{}{
					"product_option_id": *productOptionID,
				})
				return ErrInvalidProductOption
			}
			logger.Error("Failed to fetch product option", err, map[string]interface{}{
				"product_option_id": *productOptionID,
			})
			return err
		}

		if opt.ProductID != productID {
			logger.Warn("Product option mismatch", map[string]interface{}{
				"product_id":        productID,
				"product_option_id": *productOptionID,
			})
			return ErrInvalidProductOption
		}
		option = opt
	}

	existingItem, err := s.cartRepo.FindByUserProductOption(userID, productID, productOptionID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error("Failed to check existing cart item", err, map[string]interface{}{
			"user_id":    userID,
			"product_id": productID,
		})
		return err
	}

	requestedQuantity := quantity
	if existingItem != nil {
		requestedQuantity = existingItem.Quantity + quantity
	}

	if product.StockQuantity < requestedQuantity {
		logger.Warn("Cannot add to cart: insufficient product stock", map[string]interface{}{
			"user_id":    userID,
			"product_id": productID,
			"requested":  requestedQuantity,
			"available":  product.StockQuantity,
		})
		return ErrInsufficientStock
	}

	if option != nil && option.StockQuantity < requestedQuantity {
		logger.Warn("Cannot add to cart: insufficient option stock", map[string]interface{}{
			"user_id":           userID,
			"product_option_id": option.ID,
			"requested":         requestedQuantity,
			"available":         option.StockQuantity,
		})
		return ErrInsufficientStock
	}

	if existingItem != nil {
		logger.Debug("Updating existing cart item", map[string]interface{}{
			"cart_item_id": existingItem.ID,
			"old_qty":      existingItem.Quantity,
			"new_qty":      requestedQuantity,
		})
		existingItem.Quantity = requestedQuantity
		if err := s.cartRepo.Update(existingItem); err != nil {
			logger.Error("Failed to update cart item", err, map[string]interface{}{
				"cart_item_id": existingItem.ID,
			})
			return err
		}
		return nil
	}

	cartItem := &model.CartItem{
		UserID:          userID,
		ProductID:       productID,
		ProductOptionID: productOptionID,
		Quantity:        quantity,
	}

	if err := s.cartRepo.Create(cartItem); err != nil {
		logger.Error("Failed to create cart item", err, map[string]interface{}{
			"user_id":    userID,
			"product_id": productID,
		})
		return err
	}

	logger.Info("Cart item added successfully", map[string]interface{}{
		"cart_item_id": cartItem.ID,
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
				"cart_item_id": cartItemID,
			})
			return ErrCartItemNotFound
		}
		logger.Error("Failed to fetch cart item", err, map[string]interface{}{
			"cart_item_id": cartItemID,
		})
		return err
	}

	if cartItem.UserID != userID {
		logger.Warn("Cart item access denied: ownership mismatch", map[string]interface{}{
			"user_id":      userID,
			"cart_item_id": cartItemID,
			"owner_id":     cartItem.UserID,
		})
		return ErrCartItemNotFound
	}

	product, err := s.productRepo.FindByID(cartItem.ProductID)
	if err != nil {
		logger.Error("Failed to fetch product for stock check", err, map[string]interface{}{
			"cart_item_id": cartItemID,
			"product_id":   cartItem.ProductID,
		})
		return err
	}

	if product.StockQuantity < quantity {
		logger.Warn("Cannot update cart item: insufficient product stock", map[string]interface{}{
			"cart_item_id": cartItemID,
			"requested":    quantity,
			"available":    product.StockQuantity,
		})
		return ErrInsufficientStock
	}

	if cartItem.ProductOptionID != nil {
		option, err := s.productOptionRepo.FindByID(*cartItem.ProductOptionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Warn("Product option not found for stock check", map[string]interface{}{
					"cart_item_id":      cartItemID,
					"product_option_id": *cartItem.ProductOptionID,
				})
				return ErrInvalidProductOption
			}
			logger.Error("Failed to fetch product option for stock check", err, map[string]interface{}{
				"cart_item_id":      cartItemID,
				"product_option_id": *cartItem.ProductOptionID,
			})
			return err
		}

		if option.StockQuantity < quantity {
			logger.Warn("Cannot update cart item: insufficient option stock", map[string]interface{}{
				"cart_item_id":      cartItemID,
				"product_option_id": option.ID,
				"requested":         quantity,
				"available":         option.StockQuantity,
			})
			return ErrInsufficientStock
		}
	}

	cartItem.Quantity = quantity
	if err := s.cartRepo.Update(cartItem); err != nil {
		logger.Error("Failed to update cart item", err, map[string]interface{}{
			"cart_item_id": cartItemID,
		})
		return err
	}

	logger.Info("Cart item updated successfully", map[string]interface{}{
		"cart_item_id": cartItemID,
	})
	return nil
}

func (s *cartService) RemoveFromCart(userID, cartItemID uint) error {
	logger.Info("Removing cart item", map[string]interface{}{
		"user_id":      userID,
		"cart_item_id": cartItemID,
	})

	cartItem, err := s.cartRepo.FindByID(cartItemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Cart item not found for removal", map[string]interface{}{
				"cart_item_id": cartItemID,
			})
			return ErrCartItemNotFound
		}
		logger.Error("Failed to fetch cart item for removal", err, map[string]interface{}{
			"cart_item_id": cartItemID,
		})
		return err
	}

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
			"cart_item_id": cartItemID,
		})
		return err
	}

	logger.Info("Cart item removed", map[string]interface{}{
		"cart_item_id": cartItemID,
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

	logger.Info("User cart cleared", map[string]interface{}{
		"user_id": userID,
	})
	return nil
}
