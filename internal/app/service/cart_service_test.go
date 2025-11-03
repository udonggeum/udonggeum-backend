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

func setupCartServiceTest(t *testing.T) (CartService, *model.User, *model.Product, repository.ProductOptionRepository, *gorm.DB) {
	testDB, err := db.SetupTestDB()
	require.NoError(t, err)
	t.Cleanup(func() {
		db.CleanupTestDB(testDB)
	})

	cartRepo := repository.NewCartRepository(testDB)
	productRepo := repository.NewProductRepository(testDB)
	productOptionRepo := repository.NewProductOptionRepository(testDB)
	cartService := NewCartService(cartRepo, productRepo, productOptionRepo)

	// Create test user
	user := &model.User{
		Email:        "test@example.com",
		PasswordHash: "hash",
		Name:         "Test User",
		Role:         model.RoleUser,
	}
	testDB.Create(user)

	// Create store
	store := &model.Store{
		UserID:   user.ID,
		Name:     "Test Store",
		Region:   "서울특별시",
		District: "강남구",
		Address:  "서울특별시 강남구 테스트로 1",
	}
	testDB.Create(store)

	// Create test product
	product := &model.Product{
		Name:          "Test Product",
		Price:         100000,
		Category:      model.CategoryRing,
		Material:      model.MaterialGold,
		StockQuantity: 10,
		StoreID:       store.ID,
	}
	testDB.Create(product)

	return cartService, user, product, productOptionRepo, testDB
}

func TestCartService_GetUserCart(t *testing.T) {
	cartService, user, product, _, _ := setupCartServiceTest(t)

	// Initially empty
	items, err := cartService.GetUserCart(user.ID)
	assert.NoError(t, err)
	assert.Len(t, items, 0)

	// Add item
	err = cartService.AddToCart(user.ID, product.ID, nil, 2)
	require.NoError(t, err)

	// Get cart
	items, err = cartService.GetUserCart(user.ID)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, 2, items[0].Quantity)
}

func TestCartService_AddToCart_Success(t *testing.T) {
	cartService, user, product, _, _ := setupCartServiceTest(t)

	err := cartService.AddToCart(user.ID, product.ID, nil, 3)
	assert.NoError(t, err)

	// Verify
	items, _ := cartService.GetUserCart(user.ID)
	assert.Len(t, items, 1)
	assert.Equal(t, 3, items[0].Quantity)
}

func TestCartService_AddToCart_ProductNotFound(t *testing.T) {
	cartService, user, _, _, _ := setupCartServiceTest(t)

	err := cartService.AddToCart(user.ID, 9999, nil, 1)
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestCartService_AddToCart_InsufficientStock(t *testing.T) {
	cartService, user, product, _, _ := setupCartServiceTest(t)

	err := cartService.AddToCart(user.ID, product.ID, nil, 100)
	assert.ErrorIs(t, err, ErrInsufficientStock)
}

func TestCartService_AddToCart_ExistingItem(t *testing.T) {
	cartService, user, product, _, _ := setupCartServiceTest(t)

	// Add first time
	cartService.AddToCart(user.ID, product.ID, nil, 2)

	// Add again (should increment)
	err := cartService.AddToCart(user.ID, product.ID, nil, 3)
	assert.NoError(t, err)

	// Verify quantity is summed
	items, _ := cartService.GetUserCart(user.ID)
	assert.Len(t, items, 1)
	assert.Equal(t, 5, items[0].Quantity)
}

func TestCartService_UpdateCartItem_Success(t *testing.T) {
	cartService, user, product, _, _ := setupCartServiceTest(t)

	// Add item
	cartService.AddToCart(user.ID, product.ID, nil, 2)
	items, _ := cartService.GetUserCart(user.ID)
	cartItemID := items[0].ID

	// Update quantity
	err := cartService.UpdateCartItem(user.ID, cartItemID, 5, nil, false)
	assert.NoError(t, err)

	// Verify
	items, _ = cartService.GetUserCart(user.ID)
	assert.Equal(t, 5, items[0].Quantity)
}

func TestCartService_UpdateCartItem_NotFound(t *testing.T) {
	cartService, user, _, _, _ := setupCartServiceTest(t)

	err := cartService.UpdateCartItem(user.ID, 9999, 5, nil, false)
	assert.ErrorIs(t, err, ErrCartItemNotFound)
}

func TestCartService_UpdateCartItem_WrongUser(t *testing.T) {
	cartService, user, product, _, _ := setupCartServiceTest(t)

	// Add item
	cartService.AddToCart(user.ID, product.ID, nil, 2)
	items, _ := cartService.GetUserCart(user.ID)
	cartItemID := items[0].ID

	// Try to update with different user
	err := cartService.UpdateCartItem(user.ID+1, cartItemID, 5, nil, false)
	assert.ErrorIs(t, err, ErrCartItemNotFound)
}

func TestCartService_UpdateCartItem_InsufficientStock(t *testing.T) {
	cartService, user, product, _, _ := setupCartServiceTest(t)

	// Add item
	cartService.AddToCart(user.ID, product.ID, nil, 2)
	items, _ := cartService.GetUserCart(user.ID)
	cartItemID := items[0].ID

	// Try to update to quantity exceeding stock
	err := cartService.UpdateCartItem(user.ID, cartItemID, 100, nil, false)
	assert.ErrorIs(t, err, ErrInsufficientStock)
}

func TestCartService_UpdateCartItem_ChangeProductOption(t *testing.T) {
	cartService, user, product, _, testDB := setupCartServiceTest(t)

	// Create product option
	option := &model.ProductOption{
		ProductID:       product.ID,
		Name:            "사이즈",
		Value:           "11호",
		AdditionalPrice: 10000,
		StockQuantity:   5,
	}
	testDB.Create(option)

	// Add item without option
	require.NoError(t, cartService.AddToCart(user.ID, product.ID, nil, 1))
	items, _ := cartService.GetUserCart(user.ID)
	cartItemID := items[0].ID

	// Update to use the option
	err := cartService.UpdateCartItem(user.ID, cartItemID, 1, &option.ID, true)
	assert.NoError(t, err)

	// Verify option updated
	items, _ = cartService.GetUserCart(user.ID)
	require.NotNil(t, items[0].ProductOptionID)
	assert.Equal(t, option.ID, *items[0].ProductOptionID)
}

func TestCartService_UpdateCartItem_RemoveProductOption(t *testing.T) {
	cartService, user, product, _, testDB := setupCartServiceTest(t)

	option := &model.ProductOption{
		ProductID:       product.ID,
		Name:            "색상",
		Value:           "골드",
		AdditionalPrice: 5000,
		StockQuantity:   3,
	}
	testDB.Create(option)

	// Add item with option
	require.NoError(t, cartService.AddToCart(user.ID, product.ID, &option.ID, 1))
	items, _ := cartService.GetUserCart(user.ID)
	cartItemID := items[0].ID
	require.NotNil(t, items[0].ProductOptionID)

	// Remove option
	err := cartService.UpdateCartItem(user.ID, cartItemID, 1, nil, true)
	assert.NoError(t, err)

	items, _ = cartService.GetUserCart(user.ID)
	assert.Nil(t, items[0].ProductOptionID)
}

func TestCartService_UpdateCartItem_InvalidProductOption(t *testing.T) {
	cartService, user, product, _, testDB := setupCartServiceTest(t)

	// Add item
	require.NoError(t, cartService.AddToCart(user.ID, product.ID, nil, 1))
	items, _ := cartService.GetUserCart(user.ID)
	cartItemID := items[0].ID

	// Create option for different product
	otherProduct := &model.Product{
		Name:          "Other Product",
		Price:         50000,
		Category:      model.CategoryRing,
		Material:      model.MaterialGold,
		StockQuantity: 10,
		StoreID:       product.StoreID,
	}
	testDB.Create(otherProduct)

	invalidOption := &model.ProductOption{
		ProductID:       otherProduct.ID,
		Name:            "사이즈",
		Value:           "13호",
		AdditionalPrice: 15000,
		StockQuantity:   5,
	}
	testDB.Create(invalidOption)

	err := cartService.UpdateCartItem(user.ID, cartItemID, 1, &invalidOption.ID, true)
	assert.ErrorIs(t, err, ErrInvalidProductOption)
}

func TestCartService_RemoveFromCart_Success(t *testing.T) {
	cartService, user, product, _, _ := setupCartServiceTest(t)

	// Add item
	cartService.AddToCart(user.ID, product.ID, nil, 2)
	items, _ := cartService.GetUserCart(user.ID)
	cartItemID := items[0].ID

	// Remove
	err := cartService.RemoveFromCart(user.ID, cartItemID)
	assert.NoError(t, err)

	// Verify
	items, _ = cartService.GetUserCart(user.ID)
	assert.Len(t, items, 0)
}

func TestCartService_RemoveFromCart_NotFound(t *testing.T) {
	cartService, user, _, _, _ := setupCartServiceTest(t)

	err := cartService.RemoveFromCart(user.ID, 9999)
	assert.ErrorIs(t, err, ErrCartItemNotFound)
}

func TestCartService_RemoveFromCart_WrongUser(t *testing.T) {
	cartService, user, product, _, _ := setupCartServiceTest(t)

	// Add item
	cartService.AddToCart(user.ID, product.ID, nil, 2)
	items, _ := cartService.GetUserCart(user.ID)
	cartItemID := items[0].ID

	// Try to remove with different user
	err := cartService.RemoveFromCart(user.ID+1, cartItemID)
	assert.ErrorIs(t, err, ErrCartItemNotFound)
}

func TestCartService_ClearCart(t *testing.T) {
	cartService, user, product, _, _ := setupCartServiceTest(t)

	// Add multiple items
	cartService.AddToCart(user.ID, product.ID, nil, 2)
	cartService.AddToCart(user.ID, product.ID, nil, 3)

	// Clear
	err := cartService.ClearCart(user.ID)
	assert.NoError(t, err)

	// Verify
	items, _ := cartService.GetUserCart(user.ID)
	assert.Len(t, items, 0)
}
