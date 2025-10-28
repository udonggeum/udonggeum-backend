package repository

import (
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

type OrderRepository interface {
	Create(order *model.Order) error
	FindByID(id uint) (*model.Order, error)
	FindByUserID(userID uint) ([]model.Order, error)
	Update(order *model.Order) error
	UpdateStatus(id uint, status model.OrderStatus) error
	UpdatePaymentStatus(id uint, status model.PaymentStatus) error
}

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) preloadOrder() *gorm.DB {
	return r.db.Preload("OrderItems", func(db *gorm.DB) *gorm.DB {
		return db.Preload("Product", func(pdb *gorm.DB) *gorm.DB {
			return pdb.Preload("Store")
		}).Preload("ProductOption").Preload("Store")
	}).Preload("User").Preload("PickupStore")
}

func (r *orderRepository) Create(order *model.Order) error {
	logger.Debug("Creating order in database", map[string]interface{}{
		"user_id":          order.UserID,
		"total_amount":     order.TotalAmount,
		"fulfillment_type": order.FulfillmentType,
	})

	if err := r.db.Create(order).Error; err != nil {
		logger.Error("Failed to create order in database", err, map[string]interface{}{
			"user_id":          order.UserID,
			"total_amount":     order.TotalAmount,
			"fulfillment_type": order.FulfillmentType,
		})
		return err
	}

	logger.Debug("Order created in database", map[string]interface{}{
		"order_id":         order.ID,
		"user_id":          order.UserID,
		"total_amount":     order.TotalAmount,
		"fulfillment_type": order.FulfillmentType,
	})
	return nil
}

func (r *orderRepository) FindByID(id uint) (*model.Order, error) {
	logger.Debug("Finding order by ID in database", map[string]interface{}{
		"order_id": id,
	})

	var order model.Order
	if err := r.preloadOrder().First(&order, id).Error; err != nil {
		logger.Error("Failed to find order by ID in database", err, map[string]interface{}{
			"order_id": id,
		})
		return nil, err
	}

	logger.Debug("Order found by ID in database", map[string]interface{}{
		"order_id": order.ID,
		"user_id":  order.UserID,
		"status":   order.Status,
	})
	return &order, nil
}

func (r *orderRepository) FindByUserID(userID uint) ([]model.Order, error) {
	logger.Debug("Finding orders by user ID in database", map[string]interface{}{
		"user_id": userID,
	})

	var orders []model.Order
	if err := r.preloadOrder().Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&orders).Error; err != nil {
		logger.Error("Failed to find orders by user ID in database", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	logger.Debug("Orders found by user ID in database", map[string]interface{}{
		"user_id": userID,
		"count":   len(orders),
	})
	return orders, nil
}

func (r *orderRepository) Update(order *model.Order) error {
	logger.Debug("Updating order in database", map[string]interface{}{
		"order_id": order.ID,
		"user_id":  order.UserID,
		"status":   order.Status,
	})

	if err := r.db.Save(order).Error; err != nil {
		logger.Error("Failed to update order in database", err, map[string]interface{}{
			"order_id": order.ID,
			"user_id":  order.UserID,
		})
		return err
	}

	logger.Debug("Order updated in database", map[string]interface{}{
		"order_id": order.ID,
		"user_id":  order.UserID,
		"status":   order.Status,
	})
	return nil
}

func (r *orderRepository) UpdateStatus(id uint, status model.OrderStatus) error {
	logger.Debug("Updating order status in database", map[string]interface{}{
		"order_id": id,
		"status":   status,
	})

	if err := r.db.Model(&model.Order{}).Where("id = ?", id).
		Update("status", status).Error; err != nil {
		logger.Error("Failed to update order status in database", err, map[string]interface{}{
			"order_id": id,
			"status":   status,
		})
		return err
	}

	logger.Debug("Order status updated in database", map[string]interface{}{
		"order_id": id,
		"status":   status,
	})
	return nil
}

func (r *orderRepository) UpdatePaymentStatus(id uint, status model.PaymentStatus) error {
	logger.Debug("Updating order payment status in database", map[string]interface{}{
		"order_id":       id,
		"payment_status": status,
	})

	if err := r.db.Model(&model.Order{}).Where("id = ?", id).
		Update("payment_status", status).Error; err != nil {
		logger.Error("Failed to update order payment status in database", err, map[string]interface{}{
			"order_id":       id,
			"payment_status": status,
		})
		return err
	}

	logger.Debug("Order payment status updated in database", map[string]interface{}{
		"order_id":       id,
		"payment_status": status,
	})
	return nil
}
