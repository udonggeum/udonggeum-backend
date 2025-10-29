package repository

import (
	"testing"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupOrderTest(t *testing.T) (*gorm.DB, OrderRepository, *model.User, *model.Product) {
	testDB, err := db.SetupTestDB()
	require.NoError(t, err)

	repo := NewOrderRepository(testDB)

	user := &model.User{
		Email:        "test@example.com",
		PasswordHash: "hash",
		Name:         "Test User",
		Role:         model.RoleUser,
	}
	testDB.Create(user)

	product := &model.Product{
		Name:          "Test Product",
		Price:         100000,
		Category:      model.CategoryNecklace,
		Material:      model.MaterialGold,
		StockQuantity: 10,
	}
	testDB.Create(product)

	return testDB, repo, user, product
}

func TestOrderRepository_Create(t *testing.T) {
	testDB, repo, user, product := setupOrderTest(t)
	defer db.CleanupTestDB(testDB)

	order := &model.Order{
		UserID:          user.ID,
		TotalAmount:     200000,
		Status:          model.OrderStatusPending,
		PaymentStatus:   model.PaymentStatusPending,
		ShippingAddress: "서울시 강남구",
		OrderItems: []model.OrderItem{
			{
				ProductID: product.ID,
				Quantity:  2,
				Price:     100000,
			},
		},
	}

	err := repo.Create(order)
	assert.NoError(t, err)
	assert.NotZero(t, order.ID)
	assert.Len(t, order.OrderItems, 1)
}

func TestOrderRepository_FindByID(t *testing.T) {
	testDB, repo, user, _ := setupOrderTest(t)
	defer db.CleanupTestDB(testDB)

	order := &model.Order{
		UserID:          user.ID,
		TotalAmount:     100000,
		Status:          model.OrderStatusPending,
		PaymentStatus:   model.PaymentStatusPending,
		ShippingAddress: "서울시 강남구",
	}
	repo.Create(order)

	found, err := repo.FindByID(order.ID)
	require.NoError(t, err)
	assert.Equal(t, order.ID, found.ID)
	assert.Equal(t, user.ID, found.UserID)
	assert.NotNil(t, found.User)
}

func TestOrderRepository_FindByUserID(t *testing.T) {
	testDB, repo, user, _ := setupOrderTest(t)
	defer db.CleanupTestDB(testDB)

	// Create multiple orders
	for i := 0; i < 3; i++ {
		order := &model.Order{
			UserID:          user.ID,
			TotalAmount:     float64((i + 1) * 100000),
			Status:          model.OrderStatusPending,
			PaymentStatus:   model.PaymentStatusPending,
			ShippingAddress: "서울시 강남구",
		}
		repo.Create(order)
	}

	orders, err := repo.FindByUserID(user.ID)
	assert.NoError(t, err)
	assert.Len(t, orders, 3)
	// Should be ordered by created_at DESC
	assert.True(t, orders[0].TotalAmount > orders[1].TotalAmount)
}

func TestOrderRepository_Update(t *testing.T) {
	testDB, repo, user, _ := setupOrderTest(t)
	defer db.CleanupTestDB(testDB)

	order := &model.Order{
		UserID:          user.ID,
		TotalAmount:     100000,
		Status:          model.OrderStatusPending,
		PaymentStatus:   model.PaymentStatusPending,
		ShippingAddress: "서울시 강남구",
	}
	repo.Create(order)

	// Update
	order.Status = model.OrderStatusConfirmed
	order.PaymentStatus = model.PaymentStatusCompleted
	err := repo.Update(order)
	assert.NoError(t, err)

	// Verify
	updated, _ := repo.FindByID(order.ID)
	assert.Equal(t, model.OrderStatusConfirmed, updated.Status)
	assert.Equal(t, model.PaymentStatusCompleted, updated.PaymentStatus)
}

func TestOrderRepository_UpdateStatus(t *testing.T) {
	testDB, repo, user, _ := setupOrderTest(t)
	defer db.CleanupTestDB(testDB)

	order := &model.Order{
		UserID:          user.ID,
		TotalAmount:     100000,
		Status:          model.OrderStatusPending,
		PaymentStatus:   model.PaymentStatusPending,
		ShippingAddress: "서울시 강남구",
	}
	repo.Create(order)

	err := repo.UpdateStatus(order.ID, model.OrderStatusShipping)
	assert.NoError(t, err)

	// Verify
	updated, _ := repo.FindByID(order.ID)
	assert.Equal(t, model.OrderStatusShipping, updated.Status)
}

func TestOrderRepository_UpdatePaymentStatus(t *testing.T) {
	testDB, repo, user, _ := setupOrderTest(t)
	defer db.CleanupTestDB(testDB)

	order := &model.Order{
		UserID:          user.ID,
		TotalAmount:     100000,
		Status:          model.OrderStatusPending,
		PaymentStatus:   model.PaymentStatusPending,
		ShippingAddress: "서울시 강남구",
	}
	repo.Create(order)

	err := repo.UpdatePaymentStatus(order.ID, model.PaymentStatusCompleted)
	assert.NoError(t, err)

	// Verify
	updated, _ := repo.FindByID(order.ID)
	assert.Equal(t, model.PaymentStatusCompleted, updated.PaymentStatus)
}

func TestOrderRepository_WithOrderItems(t *testing.T) {
	testDB, repo, user, product := setupOrderTest(t)
	defer db.CleanupTestDB(testDB)

	order := &model.Order{
		UserID:          user.ID,
		TotalAmount:     300000,
		Status:          model.OrderStatusPending,
		PaymentStatus:   model.PaymentStatusPending,
		ShippingAddress: "서울시 강남구",
		OrderItems: []model.OrderItem{
			{ProductID: product.ID, Quantity: 2, Price: 100000},
			{ProductID: product.ID, Quantity: 1, Price: 100000},
		},
	}
	repo.Create(order)

	// Find with preloaded order items
	found, err := repo.FindByID(order.ID)
	require.NoError(t, err)
	assert.Len(t, found.OrderItems, 2)
	assert.NotNil(t, found.OrderItems[0].Product)
}
