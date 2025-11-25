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
	FindByStoreID(storeID uint, status string) ([]model.Order, error)
	FindByUserStores(userID uint) ([]model.Order, error)
	GetStatsByUserStores(userID uint) (map[string]interface{}, error)
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

func (r *orderRepository) FindByStoreID(storeID uint, status string) ([]model.Order, error) {
	logger.Debug("Finding orders by store ID in database", map[string]interface{}{
		"store_id": storeID,
		"status":   status,
	})

	query := r.db.Model(&model.Order{}).
		Joins("JOIN order_items ON order_items.order_id = orders.id").
		Where("order_items.store_id = ?", storeID).
		Group("orders.id")

	if status != "" {
		query = query.Where("orders.status = ?", status)
	}

	var orderIDs []uint
	if err := query.Pluck("orders.id", &orderIDs).Error; err != nil {
		logger.Error("Failed to find order IDs by store ID in database", err, map[string]interface{}{
			"store_id": storeID,
			"status":   status,
		})
		return nil, err
	}

	if len(orderIDs) == 0 {
		logger.Debug("No orders found for store", map[string]interface{}{
			"store_id": storeID,
		})
		return []model.Order{}, nil
	}

	var orders []model.Order
	if err := r.preloadOrder().Where("id IN ?", orderIDs).
		Order("created_at DESC").
		Find(&orders).Error; err != nil {
		logger.Error("Failed to find orders by store ID in database", err, map[string]interface{}{
			"store_id": storeID,
		})
		return nil, err
	}

	logger.Debug("Orders found by store ID in database", map[string]interface{}{
		"store_id": storeID,
		"count":    len(orders),
	})
	return orders, nil
}

func (r *orderRepository) FindByUserStores(userID uint) ([]model.Order, error) {
	logger.Debug("Finding orders by user stores in database", map[string]interface{}{
		"user_id": userID,
	})

	query := r.db.Model(&model.Order{}).
		Joins("JOIN order_items ON order_items.order_id = orders.id").
		Joins("JOIN stores ON stores.id = order_items.store_id").
		Where("stores.user_id = ?", userID).
		Group("orders.id")

	var orderIDs []uint
	if err := query.Pluck("orders.id", &orderIDs).Error; err != nil {
		logger.Error("Failed to find order IDs by user stores in database", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	if len(orderIDs) == 0 {
		logger.Debug("No orders found for user stores", map[string]interface{}{
			"user_id": userID,
		})
		return []model.Order{}, nil
	}

	var orders []model.Order
	if err := r.preloadOrder().Where("id IN ?", orderIDs).
		Order("created_at DESC").
		Find(&orders).Error; err != nil {
		logger.Error("Failed to find orders by user stores in database", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	logger.Debug("Orders found by user stores in database", map[string]interface{}{
		"user_id": userID,
		"count":   len(orders),
	})
	return orders, nil
}

func (r *orderRepository) GetStatsByUserStores(userID uint) (map[string]interface{}, error) {
	logger.Debug("Getting order statistics by user stores in database", map[string]interface{}{
		"user_id": userID,
	})

	var totalOrders int64
	var pendingOrders int64
	var confirmedOrders int64
	var shippingOrders int64
	var deliveredOrders int64
	var cancelledOrders int64
	var pickupOrders int64
	var deliveryOrders int64
	var totalRevenue float64
	var totalProducts int64
	var totalStores int64

	baseQuery := r.db.Model(&model.Order{}).
		Joins("JOIN order_items ON order_items.order_id = orders.id").
		Joins("JOIN stores ON stores.id = order_items.store_id").
		Where("stores.user_id = ?", userID)

	// Total orders
	if err := baseQuery.Session(&gorm.Session{}).
		Distinct("orders.id").
		Count(&totalOrders).Error; err != nil {
		logger.Error("Failed to count total orders", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	// Orders by status
	statusCounts := []struct {
		Status model.OrderStatus
		Count  int64
	}{}
	if err := baseQuery.Session(&gorm.Session{}).
		Select("orders.status, COUNT(DISTINCT orders.id) as count").
		Group("orders.status").
		Scan(&statusCounts).Error; err != nil {
		logger.Error("Failed to count orders by status", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	for _, sc := range statusCounts {
		switch sc.Status {
		case model.OrderStatusPending:
			pendingOrders = sc.Count
		case model.OrderStatusConfirmed:
			confirmedOrders = sc.Count
		case model.OrderStatusShipping:
			shippingOrders = sc.Count
		case model.OrderStatusDelivered:
			deliveredOrders = sc.Count
		case model.OrderStatusCancelled:
			cancelledOrders = sc.Count
		}
	}

	// Orders by fulfillment type
	fulfillmentCounts := []struct {
		FulfillmentType model.FulfillmentType
		Count           int64
	}{}
	if err := baseQuery.Session(&gorm.Session{}).
		Select("orders.fulfillment_type, COUNT(DISTINCT orders.id) as count").
		Group("orders.fulfillment_type").
		Scan(&fulfillmentCounts).Error; err != nil {
		logger.Error("Failed to count orders by fulfillment type", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	for _, fc := range fulfillmentCounts {
		switch fc.FulfillmentType {
		case model.FulfillmentPickup:
			pickupOrders = fc.Count
		case model.FulfillmentDelivery:
			deliveryOrders = fc.Count
		}
	}

	// Total revenue (sum of total_amount for delivered orders only)
	var revenueResult struct {
		TotalRevenue float64
	}
	if err := r.db.Model(&model.Order{}).
		Select("COALESCE(SUM(DISTINCT orders.total_amount), 0) as total_revenue").
		Joins("JOIN order_items ON order_items.order_id = orders.id").
		Joins("JOIN stores ON stores.id = order_items.store_id").
		Where("stores.user_id = ? AND orders.status = ?", userID, model.OrderStatusDelivered).
		Scan(&revenueResult).Error; err != nil {
		logger.Error("Failed to calculate total revenue", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}
	totalRevenue = revenueResult.TotalRevenue

	// Total products
	if err := r.db.Model(&model.Product{}).
		Joins("JOIN stores ON stores.id = products.store_id").
		Where("stores.user_id = ?", userID).
		Count(&totalProducts).Error; err != nil {
		logger.Error("Failed to count total products", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	// Total stores
	if err := r.db.Model(&model.Store{}).
		Where("user_id = ?", userID).
		Count(&totalStores).Error; err != nil {
		logger.Error("Failed to count total stores", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	stats := map[string]interface{}{
		"total_orders":     totalOrders,
		"pending_orders":   pendingOrders,
		"confirmed_orders": confirmedOrders,
		"shipping_orders":  shippingOrders,
		"delivered_orders": deliveredOrders,
		"cancelled_orders": cancelledOrders,
		"pickup_orders":    pickupOrders,
		"delivery_orders":  deliveryOrders,
		"total_revenue":    totalRevenue,
		"total_products":   totalProducts,
		"total_stores":     totalStores,
	}

	logger.Debug("Order statistics retrieved by user stores in database", map[string]interface{}{
		"user_id":       userID,
		"total_orders":  totalOrders,
		"total_revenue": totalRevenue,
	})

	return stats, nil
}
