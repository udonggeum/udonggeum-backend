package service

import (
	"errors"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

var (
	ErrNotStoreOwner = errors.New("user does not own this store")
	ErrNotOrderOwner = errors.New("user does not own a store associated with this order")
)

type DashboardStats struct {
	TotalOrders     int64   `json:"total_orders"`
	PendingOrders   int64   `json:"pending_orders"`
	ConfirmedOrders int64   `json:"confirmed_orders"`
	ShippingOrders  int64   `json:"shipping_orders"`
	DeliveredOrders int64   `json:"delivered_orders"`
	CancelledOrders int64   `json:"cancelled_orders"`
	PickupOrders    int64   `json:"pickup_orders"`
	DeliveryOrders  int64   `json:"delivery_orders"`
	TotalRevenue    float64 `json:"total_revenue"`
	TotalProducts   int64   `json:"total_products"`
	TotalStores     int64   `json:"total_stores"`
}

type SellerService interface {
	GetDashboard(userID uint) (*DashboardStats, error)
	GetStoreOrders(userID, storeID uint, status string) ([]model.Order, error)
	UpdateOrderStatus(userID, orderID uint, status string) error
}

type sellerService struct {
	orderRepo repository.OrderRepository
	storeRepo repository.StoreRepository
}

func NewSellerService(
	orderRepo repository.OrderRepository,
	storeRepo repository.StoreRepository,
) SellerService {
	return &sellerService{
		orderRepo: orderRepo,
		storeRepo: storeRepo,
	}
}

func (s *sellerService) GetDashboard(userID uint) (*DashboardStats, error) {
	logger.Info("Fetching seller dashboard statistics", map[string]interface{}{
		"user_id": userID,
	})

	statsMap, err := s.orderRepo.GetStatsByUserStores(userID)
	if err != nil {
		logger.Error("Failed to fetch dashboard statistics", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	stats := &DashboardStats{
		TotalOrders:     statsMap["total_orders"].(int64),
		PendingOrders:   statsMap["pending_orders"].(int64),
		ConfirmedOrders: statsMap["confirmed_orders"].(int64),
		ShippingOrders:  statsMap["shipping_orders"].(int64),
		DeliveredOrders: statsMap["delivered_orders"].(int64),
		CancelledOrders: statsMap["cancelled_orders"].(int64),
		PickupOrders:    statsMap["pickup_orders"].(int64),
		DeliveryOrders:  statsMap["delivery_orders"].(int64),
		TotalRevenue:    statsMap["total_revenue"].(float64),
		TotalProducts:   statsMap["total_products"].(int64),
		TotalStores:     statsMap["total_stores"].(int64),
	}

	logger.Info("Seller dashboard statistics fetched", map[string]interface{}{
		"user_id":       userID,
		"total_orders":  stats.TotalOrders,
		"total_revenue": stats.TotalRevenue,
		"total_stores":  stats.TotalStores,
	})

	return stats, nil
}

func (s *sellerService) GetStoreOrders(userID, storeID uint, status string) ([]model.Order, error) {
	logger.Info("Fetching orders for store", map[string]interface{}{
		"user_id":  userID,
		"store_id": storeID,
		"status":   status,
	})

	// Verify store ownership
	store, err := s.storeRepo.FindByID(storeID, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Store not found for order retrieval", map[string]interface{}{
				"store_id": storeID,
			})
			return nil, ErrStoreNotFound
		}
		logger.Error("Failed to find store for order retrieval", err, map[string]interface{}{
			"store_id": storeID,
		})
		return nil, err
	}

	if store.UserID != userID {
		logger.Warn("User does not own the store", map[string]interface{}{
			"user_id":  userID,
			"store_id": storeID,
			"owner_id": store.UserID,
		})
		return nil, ErrNotStoreOwner
	}

	// Fetch orders for the store
	orders, err := s.orderRepo.FindByStoreID(storeID, status)
	if err != nil {
		logger.Error("Failed to fetch orders for store", err, map[string]interface{}{
			"store_id": storeID,
		})
		return nil, err
	}

	logger.Info("Orders fetched for store", map[string]interface{}{
		"user_id":  userID,
		"store_id": storeID,
		"count":    len(orders),
	})

	return orders, nil
}

func (s *sellerService) UpdateOrderStatus(userID, orderID uint, status string) error {
	logger.Info("Updating order status", map[string]interface{}{
		"user_id":  userID,
		"order_id": orderID,
		"status":   status,
	})

	// Validate status
	validStatuses := map[string]model.OrderStatus{
		"pending":   model.OrderStatusPending,
		"confirmed": model.OrderStatusConfirmed,
		"shipping":  model.OrderStatusShipping,
		"delivered": model.OrderStatusDelivered,
		"cancelled": model.OrderStatusCancelled,
	}

	orderStatus, ok := validStatuses[status]
	if !ok {
		logger.Warn("Invalid order status provided", map[string]interface{}{
			"status": status,
		})
		return errors.New("invalid order status")
	}

	// Fetch order to verify ownership
	order, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Order not found for status update", map[string]interface{}{
				"order_id": orderID,
			})
			return errors.New("order not found")
		}
		logger.Error("Failed to find order for status update", err, map[string]interface{}{
			"order_id": orderID,
		})
		return err
	}

	// Verify that the user owns at least one store associated with the order
	ownsStore := false
	for _, item := range order.OrderItems {
		store, err := s.storeRepo.FindByID(item.StoreID, false)
		if err == nil && store.UserID == userID {
			ownsStore = true
			break
		}
	}

	if !ownsStore {
		logger.Warn("User does not own any store associated with the order", map[string]interface{}{
			"user_id":  userID,
			"order_id": orderID,
		})
		return ErrNotOrderOwner
	}

	// Update order status
	if err := s.orderRepo.UpdateStatus(orderID, orderStatus); err != nil {
		logger.Error("Failed to update order status", err, map[string]interface{}{
			"order_id": orderID,
			"status":   status,
		})
		return err
	}

	logger.Info("Order status updated successfully", map[string]interface{}{
		"user_id":  userID,
		"order_id": orderID,
		"status":   status,
	})

	return nil
}
