package service

import (
	"testing"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupOrderServiceTest(t *testing.T) (OrderService, *gorm.DB, *model.User, *model.Product, *model.Store) {
	testDB, err := db.SetupTestDB()
	require.NoError(t, err)
	t.Cleanup(func() {
		db.CleanupTestDB(testDB)
	})

	orderRepo := repository.NewOrderRepository(testDB)
	cartRepo := repository.NewCartRepository(testDB)
	productRepo := repository.NewProductRepository(testDB)
	productOptionRepo := repository.NewProductOptionRepository(testDB)
	orderService := NewOrderService(orderRepo, cartRepo, productRepo, testDB, productOptionRepo)

	user := &model.User{
		Email:        "test@example.com",
		PasswordHash: "hash",
		Name:         "Test User",
		Role:         model.RoleUser,
	}
	testDB.Create(user)

	store := &model.Store{
		UserID:   user.ID,
		Name:     "테스트 매장",
		Region:   "서울특별시",
		District: "강남구",
		Address:  "서울시 강남구 테스트로 1",
	}
	testDB.Create(store)

	product := &model.Product{
		Name:          "Test Product",
		Price:         100000,
		Category:      model.CategoryRing,
		Material:      model.MaterialGold,
		StockQuantity: 10,
		StoreID:       store.ID,
	}
	testDB.Create(product)

	return orderService, testDB, user, product, store
}

func TestOrderService_CreateOrderFromCart_Success(t *testing.T) {
	orderService, testDB, user, product, _ := setupOrderServiceTest(t)

	// Add items to cart
	cartRepo := repository.NewCartRepository(testDB)
	cartRepo.Create(&model.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
	})

	// Create order
	order, err := orderService.CreateOrderFromCart(user.ID, "서울시 강남구 테헤란로 123", model.FulfillmentDelivery, nil)
	require.NoError(t, err)
	assert.NotZero(t, order.ID)
	assert.Equal(t, user.ID, order.UserID)
	assert.Equal(t, float64(200000), order.TotalAmount)
	assert.Equal(t, model.OrderStatusPending, order.Status)
	assert.Equal(t, model.PaymentStatusPending, order.PaymentStatus)
	assert.Len(t, order.OrderItems, 1)

	// Verify stock decreased
	var updatedProduct model.Product
	testDB.First(&updatedProduct, product.ID)
	assert.Equal(t, 8, updatedProduct.StockQuantity)

	// Verify cart is empty
	items, _ := cartRepo.FindByUserID(user.ID)
	assert.Len(t, items, 0)
}

func TestOrderService_CreateOrderFromCart_EmptyCart(t *testing.T) {
	orderService, _, user, _, _ := setupOrderServiceTest(t)

	order, err := orderService.CreateOrderFromCart(user.ID, "서울시 강남구", model.FulfillmentDelivery, nil)
	assert.ErrorIs(t, err, ErrEmptyCart)
	assert.Nil(t, order)
}

func TestOrderService_CreateOrderFromCart_InsufficientStock(t *testing.T) {
	orderService, testDB, user, product, _ := setupOrderServiceTest(t)

	// Add item with quantity exceeding stock
	cartRepo := repository.NewCartRepository(testDB)
	cartRepo.Create(&model.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  100,
	})

	order, err := orderService.CreateOrderFromCart(user.ID, "서울시 강남구", model.FulfillmentDelivery, nil)
	assert.ErrorIs(t, err, ErrInsufficientStock)
	assert.Nil(t, order)

	// Verify stock unchanged
	var updatedProduct model.Product
	testDB.First(&updatedProduct, product.ID)
	assert.Equal(t, 10, updatedProduct.StockQuantity)

	// Verify cart unchanged
	items, _ := cartRepo.FindByUserID(user.ID)
	assert.Len(t, items, 1)
}

func TestOrderService_GetUserOrders(t *testing.T) {
	orderService, testDB, user, _, _ := setupOrderServiceTest(t)

	// Create multiple orders
	orderRepo := repository.NewOrderRepository(testDB)
	for i := 0; i < 3; i++ {
		order := &model.Order{
			UserID:          user.ID,
			TotalAmount:     float64((i + 1) * 100000),
			Status:          model.OrderStatusPending,
			PaymentStatus:   model.PaymentStatusPending,
			ShippingAddress: "서울시 강남구",
		}
		orderRepo.Create(order)
	}

	orders, err := orderService.GetUserOrders(user.ID)
	assert.NoError(t, err)
	assert.Len(t, orders, 3)
}

func TestOrderService_GetOrderByID_Success(t *testing.T) {
	orderService, testDB, user, _, _ := setupOrderServiceTest(t)

	// Create order
	orderRepo := repository.NewOrderRepository(testDB)
	order := &model.Order{
		UserID:          user.ID,
		TotalAmount:     100000,
		Status:          model.OrderStatusPending,
		PaymentStatus:   model.PaymentStatusPending,
		ShippingAddress: "서울시 강남구",
	}
	orderRepo.Create(order)

	// Get order
	found, err := orderService.GetOrderByID(user.ID, order.ID)
	require.NoError(t, err)
	assert.Equal(t, order.ID, found.ID)
}

func TestOrderService_GetOrderByID_NotFound(t *testing.T) {
	orderService, _, user, _, _ := setupOrderServiceTest(t)

	order, err := orderService.GetOrderByID(user.ID, 9999)
	assert.ErrorIs(t, err, ErrOrderNotFound)
	assert.Nil(t, order)
}

func TestOrderService_GetOrderByID_WrongUser(t *testing.T) {
	orderService, testDB, user, _, _ := setupOrderServiceTest(t)

	// Create order
	orderRepo := repository.NewOrderRepository(testDB)
	order := &model.Order{
		UserID:          user.ID,
		TotalAmount:     100000,
		Status:          model.OrderStatusPending,
		PaymentStatus:   model.PaymentStatusPending,
		ShippingAddress: "서울시 강남구",
	}
	orderRepo.Create(order)

	// Try to get with different user
	found, err := orderService.GetOrderByID(user.ID+1, order.ID)
	assert.ErrorIs(t, err, ErrOrderNotFound)
	assert.Nil(t, found)
}

func TestOrderService_UpdateOrderStatus(t *testing.T) {
	orderService, testDB, user, _, _ := setupOrderServiceTest(t)

	orderRepo := repository.NewOrderRepository(testDB)
	order := &model.Order{
		UserID:          user.ID,
		TotalAmount:     100000,
		Status:          model.OrderStatusPending,
		PaymentStatus:   model.PaymentStatusPending,
		ShippingAddress: "서울시 강남구",
	}
	orderRepo.Create(order)

	err := orderService.UpdateOrderStatus(order.ID, model.OrderStatusConfirmed)
	assert.NoError(t, err)

	// Verify
	updated, _ := orderRepo.FindByID(order.ID)
	assert.Equal(t, model.OrderStatusConfirmed, updated.Status)
}

func TestOrderService_UpdatePaymentStatus(t *testing.T) {
	orderService, testDB, user, _, _ := setupOrderServiceTest(t)

	orderRepo := repository.NewOrderRepository(testDB)
	order := &model.Order{
		UserID:          user.ID,
		TotalAmount:     100000,
		Status:          model.OrderStatusPending,
		PaymentStatus:   model.PaymentStatusPending,
		ShippingAddress: "서울시 강남구",
	}
	orderRepo.Create(order)

	err := orderService.UpdatePaymentStatus(order.ID, model.PaymentStatusCompleted)
	assert.NoError(t, err)

	// Verify
	updated, _ := orderRepo.FindByID(order.ID)
	assert.Equal(t, model.PaymentStatusCompleted, updated.PaymentStatus)
}

func TestOrderService_CreateOrder_WithMultipleItems(t *testing.T) {
	orderService, testDB, user, product, store := setupOrderServiceTest(t)

	// Create another product
	product2 := &model.Product{
		Name:          "Test Product 2",
		Price:         50000,
		Category:      model.CategoryBracelet,
		Material:      model.MaterialSilver,
		StockQuantity: 20,
		StoreID:       store.ID,
	}
	testDB.Create(product2)

	// Add multiple items to cart
	cartRepo := repository.NewCartRepository(testDB)
	cartRepo.Create(&model.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 2})
	cartRepo.Create(&model.CartItem{UserID: user.ID, ProductID: product2.ID, Quantity: 3})

	// Create order
	order, err := orderService.CreateOrderFromCart(user.ID, "서울시 강남구", model.FulfillmentDelivery, nil)
	require.NoError(t, err)
	assert.Equal(t, float64(350000), order.TotalAmount) // (100000*2) + (50000*3)
	assert.Len(t, order.OrderItems, 2)

	// Verify stock decreased for both products
	var p1, p2 model.Product
	testDB.First(&p1, product.ID)
	testDB.First(&p2, product2.ID)
	assert.Equal(t, 8, p1.StockQuantity)
	assert.Equal(t, 17, p2.StockQuantity)
}
