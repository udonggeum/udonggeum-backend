package service

import (
	"errors"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

var (
	ErrOrderNotFound = errors.New("order not found")
	ErrEmptyCart     = errors.New("cart is empty")
)

type OrderService interface {
	CreateOrderFromCart(userID uint, shippingAddress string) (*model.Order, error)
	GetUserOrders(userID uint) ([]model.Order, error)
	GetOrderByID(userID, orderID uint) (*model.Order, error)
	UpdateOrderStatus(orderID uint, status model.OrderStatus) error
	UpdatePaymentStatus(orderID uint, status model.PaymentStatus) error
}

type orderService struct {
	orderRepo   repository.OrderRepository
	cartRepo    repository.CartRepository
	productRepo repository.ProductRepository
	db          *gorm.DB
}

func NewOrderService(
	orderRepo repository.OrderRepository,
	cartRepo repository.CartRepository,
	productRepo repository.ProductRepository,
	db *gorm.DB,
) OrderService {
	return &orderService{
		orderRepo:   orderRepo,
		cartRepo:    cartRepo,
		productRepo: productRepo,
		db:          db,
	}
}

func (s *orderService) CreateOrderFromCart(userID uint, shippingAddress string) (*model.Order, error) {
	logger.Info("Creating order from cart", map[string]interface{}{
		"user_id":          userID,
		"shipping_address": shippingAddress,
	})

	// Get cart items
	cartItems, err := s.cartRepo.FindByUserID(userID)
	if err != nil {
		logger.Error("Failed to fetch cart items", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	if len(cartItems) == 0 {
		logger.Warn("Cannot create order: cart is empty", map[string]interface{}{
			"user_id": userID,
		})
		return nil, ErrEmptyCart
	}

	logger.Debug("Processing cart items", map[string]interface{}{
		"user_id":    userID,
		"item_count": len(cartItems),
	})

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error("Panic during order creation, rolling back", nil, map[string]interface{}{
				"user_id": userID,
				"panic":   r,
			})
		}
	}()

	// Calculate total and create order items
	var totalAmount float64
	var orderItems []model.OrderItem

	for _, cartItem := range cartItems {
		// Check stock
		product, err := s.productRepo.FindByID(cartItem.ProductID)
		if err != nil {
			tx.Rollback()
			logger.Error("Failed to fetch product during order creation", err, map[string]interface{}{
				"user_id":    userID,
				"product_id": cartItem.ProductID,
			})
			return nil, err
		}

		if product.StockQuantity < cartItem.Quantity {
			tx.Rollback()
			logger.Warn("Order creation failed: insufficient stock", map[string]interface{}{
				"user_id":         userID,
				"product_id":      cartItem.ProductID,
				"requested":       cartItem.Quantity,
				"available_stock": product.StockQuantity,
			})
			return nil, ErrInsufficientStock
		}

		// Create order item
		orderItem := model.OrderItem{
			ProductID: cartItem.ProductID,
			Quantity:  cartItem.Quantity,
			Price:     product.Price,
		}
		orderItems = append(orderItems, orderItem)
		totalAmount += product.Price * float64(cartItem.Quantity)

		logger.Debug("Processing order item", map[string]interface{}{
			"user_id":    userID,
			"product_id": cartItem.ProductID,
			"quantity":   cartItem.Quantity,
			"price":      product.Price,
		})

		// Update stock
		err = s.productRepo.UpdateStock(product.ID, -cartItem.Quantity)
		if err != nil {
			tx.Rollback()
			logger.Error("Failed to update product stock", err, map[string]interface{}{
				"user_id":    userID,
				"product_id": product.ID,
				"quantity":   -cartItem.Quantity,
			})
			return nil, err
		}
	}

	// Create order
	order := &model.Order{
		UserID:          userID,
		TotalAmount:     totalAmount,
		Status:          model.OrderStatusPending,
		PaymentStatus:   model.PaymentStatusPending,
		ShippingAddress: shippingAddress,
		OrderItems:      orderItems,
	}

	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to create order", err, map[string]interface{}{
			"user_id":      userID,
			"total_amount": totalAmount,
		})
		return nil, err
	}

	logger.Debug("Order created, clearing cart", map[string]interface{}{
		"user_id":  userID,
		"order_id": order.ID,
	})

	// Clear cart
	if err := tx.Where("user_id = ?", userID).Delete(&model.CartItem{}).Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to clear cart after order creation", err, map[string]interface{}{
			"user_id":  userID,
			"order_id": order.ID,
		})
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		logger.Error("Failed to commit order transaction", err, map[string]interface{}{
			"user_id":  userID,
			"order_id": order.ID,
		})
		return nil, err
	}

	logger.Info("Order created successfully", map[string]interface{}{
		"user_id":       userID,
		"order_id":      order.ID,
		"total_amount":  totalAmount,
		"item_count":    len(orderItems),
		"order_status":  order.Status,
		"payment_status": order.PaymentStatus,
	})

	// Reload order with associations
	return s.orderRepo.FindByID(order.ID)
}

func (s *orderService) GetUserOrders(userID uint) ([]model.Order, error) {
	logger.Debug("Fetching user orders", map[string]interface{}{
		"user_id": userID,
	})

	orders, err := s.orderRepo.FindByUserID(userID)
	if err != nil {
		logger.Error("Failed to fetch user orders", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	logger.Info("User orders fetched successfully", map[string]interface{}{
		"user_id": userID,
		"count":   len(orders),
	})
	return orders, nil
}

func (s *orderService) GetOrderByID(userID, orderID uint) (*model.Order, error) {
	logger.Debug("Fetching order by ID", map[string]interface{}{
		"user_id":  userID,
		"order_id": orderID,
	})

	order, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Order not found", map[string]interface{}{
				"user_id":  userID,
				"order_id": orderID,
			})
			return nil, ErrOrderNotFound
		}
		logger.Error("Failed to fetch order", err, map[string]interface{}{
			"user_id":  userID,
			"order_id": orderID,
		})
		return nil, err
	}

	// Verify ownership
	if order.UserID != userID {
		logger.Warn("Order access denied: ownership mismatch", map[string]interface{}{
			"user_id":  userID,
			"order_id": orderID,
			"owner_id": order.UserID,
		})
		return nil, ErrOrderNotFound
	}

	logger.Debug("Order fetched successfully", map[string]interface{}{
		"user_id":        userID,
		"order_id":       orderID,
		"order_status":   order.Status,
		"payment_status": order.PaymentStatus,
	})
	return order, nil
}

func (s *orderService) UpdateOrderStatus(orderID uint, status model.OrderStatus) error {
	logger.Info("Updating order status", map[string]interface{}{
		"order_id":   orderID,
		"new_status": status,
	})

	if err := s.orderRepo.UpdateStatus(orderID, status); err != nil {
		logger.Error("Failed to update order status", err, map[string]interface{}{
			"order_id":   orderID,
			"new_status": status,
		})
		return err
	}

	logger.Info("Order status updated successfully", map[string]interface{}{
		"order_id": orderID,
		"status":   status,
	})
	return nil
}

func (s *orderService) UpdatePaymentStatus(orderID uint, status model.PaymentStatus) error {
	logger.Info("Updating payment status", map[string]interface{}{
		"order_id":   orderID,
		"new_status": status,
	})

	if err := s.orderRepo.UpdatePaymentStatus(orderID, status); err != nil {
		logger.Error("Failed to update payment status", err, map[string]interface{}{
			"order_id":   orderID,
			"new_status": status,
		})
		return err
	}

	logger.Info("Payment status updated successfully", map[string]interface{}{
		"order_id": orderID,
		"status":   status,
	})
	return nil
}
