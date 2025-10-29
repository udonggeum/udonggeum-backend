package repository

import (
	"testing"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupCartTest(t *testing.T) (*gorm.DB, CartRepository, *model.User, *model.Product) {
	testDB, err := db.SetupTestDB()
	require.NoError(t, err)

	repo := NewCartRepository(testDB)

	// Create test user
	user := &model.User{
		Email:        "test@example.com",
		PasswordHash: "hash",
		Name:         "Test User",
		Role:         model.RoleUser,
	}
	testDB.Create(user)

	// Create test product
	product := &model.Product{
		Name:          "Test Product",
		Price:         100000,
		Category:      model.CategoryOther,
		Material:      model.MaterialGold,
		StockQuantity: 10,
	}
	testDB.Create(product)

	return testDB, repo, user, product
}

func TestCartRepository_Create(t *testing.T) {
	testDB, repo, user, product := setupCartTest(t)
	defer db.CleanupTestDB(testDB)

	cartItem := &model.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
	}

	err := repo.Create(cartItem)
	assert.NoError(t, err)
	assert.NotZero(t, cartItem.ID)
}

func TestCartRepository_FindByUserID(t *testing.T) {
	testDB, repo, user, product := setupCartTest(t)
	defer db.CleanupTestDB(testDB)

	// Create cart items
	item1 := &model.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 2}
	item2 := &model.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 1}

	repo.Create(item1)
	repo.Create(item2)

	// Find by user ID
	items, err := repo.FindByUserID(user.ID)
	assert.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestCartRepository_FindByID(t *testing.T) {
	testDB, repo, user, product := setupCartTest(t)
	defer db.CleanupTestDB(testDB)

	cartItem := &model.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  3,
	}
	repo.Create(cartItem)

	found, err := repo.FindByID(cartItem.ID)
	require.NoError(t, err)
	assert.Equal(t, cartItem.ID, found.ID)
	assert.Equal(t, 3, found.Quantity)
	assert.NotNil(t, found.Product)
}

func TestCartRepository_FindByUserAndProduct(t *testing.T) {
	testDB, repo, user, product := setupCartTest(t)
	defer db.CleanupTestDB(testDB)

	cartItem := &model.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
	}
	repo.Create(cartItem)

	found, err := repo.FindByUserAndProduct(user.ID, product.ID)
	require.NoError(t, err)
	assert.Equal(t, cartItem.ID, found.ID)
	assert.Equal(t, user.ID, found.UserID)
	assert.Equal(t, product.ID, found.ProductID)
}

func TestCartRepository_Update(t *testing.T) {
	testDB, repo, user, product := setupCartTest(t)
	defer db.CleanupTestDB(testDB)

	cartItem := &model.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
	}
	repo.Create(cartItem)

	// Update quantity
	cartItem.Quantity = 5
	err := repo.Update(cartItem)
	assert.NoError(t, err)

	// Verify
	updated, _ := repo.FindByID(cartItem.ID)
	assert.Equal(t, 5, updated.Quantity)
}

func TestCartRepository_Delete(t *testing.T) {
	testDB, repo, user, product := setupCartTest(t)
	defer db.CleanupTestDB(testDB)

	cartItem := &model.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
	}
	repo.Create(cartItem)

	err := repo.Delete(cartItem.ID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = repo.FindByID(cartItem.ID)
	assert.Error(t, err)
}

func TestCartRepository_DeleteByUserID(t *testing.T) {
	testDB, repo, user, product := setupCartTest(t)
	defer db.CleanupTestDB(testDB)

	// Create multiple items
	repo.Create(&model.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 1})
	repo.Create(&model.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 2})

	err := repo.DeleteByUserID(user.ID)
	assert.NoError(t, err)

	// Verify all deleted
	items, _ := repo.FindByUserID(user.ID)
	assert.Len(t, items, 0)
}
