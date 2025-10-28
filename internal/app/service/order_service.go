package service

import (
	"errors"
	"fmt"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrEmptyCart          = errors.New("cart is empty")
	ErrInvalidFulfillment = errors.New("invalid fulfillment selection")
)

type OrderService interface {
	CreateOrderFromCart(userID uint, shippingAddress string, fulfillmentType model.FulfillmentType, pickupStoreID *uint) (*model.Order, error)
	GetUserOrders(userID uint) ([]model.Order, error)
	GetOrderByID(userID, orderID uint) (*model.Order, error)
	UpdateOrderStatus(orderID uint, status model.OrderStatus) error
	UpdatePaymentStatus(orderID uint, status model.PaymentStatus) error
}

type orderService struct {
	orderRepo         repository.OrderRepository
	cartRepo          repository.CartRepository
	productRepo       repository.ProductRepository
	productOptionRepo repository.ProductOptionRepository
	db                *gorm.DB
}

func NewOrderService(
	orderRepo repository.OrderRepository,
	cartRepo repository.CartRepository,
	productRepo repository.ProductRepository,
	db *gorm.DB,
	productOptionRepo ...repository.ProductOptionRepository,
) OrderService {
	var optionRepo repository.ProductOptionRepository
	if len(productOptionRepo) > 0 {
		optionRepo = productOptionRepo[0]
	}
	return &orderService{
		orderRepo:         orderRepo,
		cartRepo:          cartRepo,
		productRepo:       productRepo,
		productOptionRepo: optionRepo,
		db:                db,
	}
}

func (s *orderService) CreateOrderFromCart(userID uint, shippingAddress string, fulfillmentType model.FulfillmentType, pickupStoreID *uint) (*model.Order, error) {
	if fulfillmentType == "" {
		fulfillmentType = model.FulfillmentDelivery
	}

	logger.Info("Creating order from cart", map[string]interface{}{
		"user_id":          userID,
		"fulfillment_type": fulfillmentType,
		"pickup_store_id":  pickupStoreID,
	})

	if fulfillmentType == model.FulfillmentDelivery && shippingAddress == "" {
		logger.Warn("Delivery requires shipping address", map[string]interface{}{
			"user_id": userID,
		})
		return nil, ErrInvalidFulfillment
	}

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

	logger.Debug("Processing cart items for order", map[string]interface{}{
		"user_id":    userID,
		"item_count": len(cartItems),
	})

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error("Panic during order creation, rolling back", fmt.Errorf("panic: %v", r), map[string]interface{}{
				"user_id": userID,
			})
		}
	}()

	var (
		totalAmount      float64
		orderItems       []model.OrderItem
		resolvedPickupID *uint
		resolvedPickAddr string
	)

	for _, cartItem := range cartItems {
		var product model.Product
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Store").
			First(&product, cartItem.ProductID).Error; err != nil {
			tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Warn("Product not found during order creation", map[string]interface{}{
					"user_id":    userID,
					"product_id": cartItem.ProductID,
				})
				return nil, ErrProductNotFound
			}
			logger.Error("Failed to fetch product during order creation", err, map[string]interface{}{
				"user_id":    userID,
				"product_id": cartItem.ProductID,
			})
			return nil, err
		}

		if product.StockQuantity < cartItem.Quantity {
			tx.Rollback()
			logger.Warn("Order creation failed: insufficient product stock", map[string]interface{}{
				"user_id":    userID,
				"product_id": cartItem.ProductID,
				"requested":  cartItem.Quantity,
				"available":  product.StockQuantity,
			})
			return nil, ErrInsufficientStock
		}

		var option *model.ProductOption
		if cartItem.ProductOptionID != nil {
			var opt model.ProductOption
			if err := tx.
				Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&opt, *cartItem.ProductOptionID).Error; err != nil {
				tx.Rollback()
				if errors.Is(err, gorm.ErrRecordNotFound) {
					logger.Warn("Product option not found during order creation", map[string]interface{}{
						"user_id":           userID,
						"product_option_id": *cartItem.ProductOptionID,
					})
					return nil, ErrInvalidProductOption
				}
				logger.Error("Failed to fetch product option during order creation", err, map[string]interface{}{
					"user_id":           userID,
					"product_option_id": *cartItem.ProductOptionID,
				})
				return nil, err
			}
			if opt.ProductID != cartItem.ProductID {
				tx.Rollback()
				logger.Warn("Product option mismatch during order creation", map[string]interface{}{
					"user_id":           userID,
					"product_id":        cartItem.ProductID,
					"product_option_id": *cartItem.ProductOptionID,
				})
				return nil, ErrInvalidProductOption
			}
			if opt.StockQuantity < cartItem.Quantity {
				tx.Rollback()
				logger.Warn("Order creation failed: insufficient option stock", map[string]interface{}{
					"user_id":           userID,
					"product_option_id": opt.ID,
					"requested":         cartItem.Quantity,
					"available":         opt.StockQuantity,
				})
				return nil, ErrInsufficientStock
			}
			tmp := opt
			option = &tmp
		}

		if fulfillmentType == model.FulfillmentPickup {
			if resolvedPickupID == nil {
				if pickupStoreID != nil {
					resolvedPickupID = pickupStoreID
				} else {
					id := product.StoreID
					resolvedPickupID = &id
				}
				resolvedPickAddr = product.Store.Address
			}
			if product.StoreID != *resolvedPickupID {
				tx.Rollback()
				logger.Warn("Pickup order contains multiple stores", map[string]interface{}{
					"user_id":        userID,
					"existing_store": *resolvedPickupID,
					"item_store":     product.StoreID,
				})
				return nil, ErrInvalidFulfillment
			}
		}

		unitPrice := product.Price
		var optionSnapshot string
		if option != nil {
			unitPrice += option.AdditionalPrice
			optionSnapshot = fmt.Sprintf("%s: %s", option.Name, option.Value)
		}

		orderItems = append(orderItems, model.OrderItem{
			ProductID:       cartItem.ProductID,
			ProductOptionID: cartItem.ProductOptionID,
			StoreID:         product.StoreID,
			Quantity:        cartItem.Quantity,
			Price:           unitPrice,
			OptionSnapshot:  optionSnapshot,
		})
		totalAmount += unitPrice * float64(cartItem.Quantity)

		if err := tx.Model(&model.Product{}).
			Where("id = ?", product.ID).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Update("stock_quantity", gorm.Expr("stock_quantity - ?", cartItem.Quantity)).Error; err != nil {
			tx.Rollback()
			logger.Error("Failed to update product stock", err, map[string]interface{}{
				"user_id":    userID,
				"product_id": product.ID,
			})
			return nil, err
		}

		if option != nil {
			if err := tx.Model(&model.ProductOption{}).
				Where("id = ?", option.ID).
				Clauses(clause.Locking{Strength: "UPDATE"}).
				Update("stock_quantity", gorm.Expr("stock_quantity - ?", cartItem.Quantity)).Error; err != nil {
				tx.Rollback()
				logger.Error("Failed to update product option stock", err, map[string]interface{}{
					"user_id":           userID,
					"product_option_id": option.ID,
				})
				return nil, err
			}
		}
	}

	order := &model.Order{
		UserID:          userID,
		TotalAmount:     totalAmount,
		TotalPrice:      totalAmount,
		Status:          model.OrderStatusPending,
		PaymentStatus:   model.PaymentStatusPending,
		FulfillmentType: fulfillmentType,
		ShippingAddress: shippingAddress,
		OrderItems:      orderItems,
	}

	if fulfillmentType == model.FulfillmentPickup {
		order.ShippingAddress = resolvedPickAddr
		order.PickupStoreID = resolvedPickupID
	}

	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to create order", err, map[string]interface{}{
			"user_id":      userID,
			"total_amount": totalAmount,
		})
		return nil, err
	}

	if err := tx.Where("user_id = ?", userID).Delete(&model.CartItem{}).Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to clear cart after order creation", err, map[string]interface{}{
			"user_id": userID,
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
		"user_id":          userID,
		"order_id":         order.ID,
		"total_amount":     totalAmount,
		"item_count":       len(orderItems),
		"fulfillment_type": fulfillmentType,
	})

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
