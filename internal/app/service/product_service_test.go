package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupProductServiceTest(t *testing.T) (ProductService, *gorm.DB, *model.User, *model.Store) {
	testDB, err := db.SetupTestDB()
	require.NoError(t, err)
	t.Cleanup(func() {
		db.CleanupTestDB(testDB)
	})

	productRepo := repository.NewProductRepository(testDB)
	productOptionRepo := repository.NewProductOptionRepository(testDB)
	service := NewProductService(productRepo, productOptionRepo)

	user := &model.User{
		Email:        fmt.Sprintf("admin-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hashed-password",
		Name:         "Test Admin",
		Role:         model.RoleAdmin,
	}
	require.NoError(t, testDB.Create(user).Error)

	store := &model.Store{
		UserID:   user.ID,
		Name:     "Test Store",
		Region:   "서울특별시",
		District: "강남구",
		Address:  "서울시 강남구 테스트로 1",
	}
	require.NoError(t, testDB.Create(store).Error)

	return service, testDB, user, store
}

func TestProductService_ListProducts(t *testing.T) {
	productService, _, _, store := setupProductServiceTest(t)

	products, err := productService.ListProducts(ProductListOptions{})
	assert.NoError(t, err)
	assert.Len(t, products, 0)

	product1 := &model.Product{Name: "Gold Ring", Price: 1000000, Category: model.CategoryRing, Material: model.MaterialGold, StockQuantity: 10, StoreID: store.ID}
	product2 := &model.Product{Name: "Silver Bracelet", Price: 100000, Category: model.CategoryBracelet, Material: model.MaterialSilver, StockQuantity: 20, StoreID: store.ID}

	productService.CreateProduct(product1)
	productService.CreateProduct(product2)

	products, err = productService.ListProducts(ProductListOptions{})
	assert.NoError(t, err)
	assert.Len(t, products, 2)
}

func TestProductService_GetProductByID(t *testing.T) {
	productService, _, _, store := setupProductServiceTest(t)

	product := &model.Product{
		Name:          "Gold Necklace",
		Price:         1000000,
		Category:      model.CategoryNecklace,
		Material:      model.MaterialGold,
		StockQuantity: 10,
		StoreID:       store.ID,
	}
	productService.CreateProduct(product)

	tests := []struct {
		name    string
		id      uint
		wantErr error
	}{
		{name: "Existing product", id: product.ID, wantErr: nil},
		{name: "Non-existing product", id: 9999, wantErr: ErrProductNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := productService.GetProductByID(tt.id)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, found)
			} else {
				require.NoError(t, err)
				assert.Equal(t, product.Name, found.Name)
			}
		})
	}
}

func TestProductService_GetProductsByCategory(t *testing.T) {
	productService, _, _, store := setupProductServiceTest(t)

	ringProduct := &model.Product{Name: "Gold Ring", Price: 1000000, Category: model.CategoryRing, Material: model.MaterialGold, StockQuantity: 10, StoreID: store.ID}
	braceletProduct := &model.Product{Name: "Silver Bracelet", Price: 100000, Category: model.CategoryBracelet, Material: model.MaterialSilver, StockQuantity: 20, StoreID: store.ID}

	productService.CreateProduct(ringProduct)
	productService.CreateProduct(braceletProduct)

	ringProducts, err := productService.GetProductsByCategory(model.CategoryRing)
	assert.NoError(t, err)
	assert.Len(t, ringProducts, 1)
	assert.Equal(t, "Gold Ring", ringProducts[0].Name)

	braceletProducts, err := productService.GetProductsByCategory(model.CategoryBracelet)
	assert.NoError(t, err)
	assert.Len(t, braceletProducts, 1)
	assert.Equal(t, "Silver Bracelet", braceletProducts[0].Name)
}

func TestProductService_GetAvailableFilters(t *testing.T) {
	productService, _, _, store := setupProductServiceTest(t)

	productService.CreateProduct(&model.Product{
		Name:          "Gold Ring",
		Price:         300000,
		Category:      model.CategoryRing,
		Material:      model.MaterialGold,
		StockQuantity: 5,
		StoreID:       store.ID,
	})
	productService.CreateProduct(&model.Product{
		Name:          "Silver Necklace",
		Price:         250000,
		Category:      model.CategoryNecklace,
		Material:      model.MaterialSilver,
		StockQuantity: 3,
		StoreID:       store.ID,
	})

	filters, err := productService.GetAvailableFilters()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []model.ProductCategory{
		model.CategoryRing,
		model.CategoryNecklace,
	}, filters.Categories)
	assert.ElementsMatch(t, []model.ProductMaterial{
		model.MaterialGold,
		model.MaterialSilver,
	}, filters.Materials)
}

func TestProductService_CreateProduct(t *testing.T) {
	productService, _, _, store := setupProductServiceTest(t)

	product := &model.Product{
		Name:          "24K Gold Pendant",
		Description:   "Pure gold",
		Price:         5000000,
		Weight:        100,
		Purity:        "24K",
		Category:      model.CategoryNecklace,
		Material:      model.MaterialGold,
		StockQuantity: 5,
		StoreID:       store.ID,
	}

	err := productService.CreateProduct(product)
	assert.NoError(t, err)
	assert.NotZero(t, product.ID)
}

func TestProductService_UpdateProduct(t *testing.T) {
	productService, _, user, store := setupProductServiceTest(t)

	product := &model.Product{
		Name:          "Gold Bangle",
		Price:         1000000,
		Category:      model.CategoryBracelet,
		Material:      model.MaterialGold,
		StockQuantity: 10,
		StoreID:       store.ID,
	}
	productService.CreateProduct(product)

	product.Price = 1100000
	product.StockQuantity = 15
	err := productService.UpdateProduct(user.ID, product)
	assert.NoError(t, err)

	updated, _ := productService.GetProductByID(product.ID)
	assert.Equal(t, float64(1100000), updated.Price)
	assert.Equal(t, 15, updated.StockQuantity)
}

func TestProductService_UpdateProduct_NotFound(t *testing.T) {
	productService, _, user, store := setupProductServiceTest(t)

	product := &model.Product{
		ID:            9999,
		Name:          "Non-existing",
		Price:         1000000,
		Category:      model.CategoryOther,
		Material:      model.MaterialOther,
		StockQuantity: 10,
		StoreID:       store.ID,
	}

	err := productService.UpdateProduct(user.ID, product)
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestProductService_UpdateProduct_AccessDenied(t *testing.T) {
	productService, dbConn, user, _ := setupProductServiceTest(t)

	otherUser := &model.User{
		Email:        fmt.Sprintf("other-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hashed-password",
		Name:         "Other Admin",
		Role:         model.RoleAdmin,
	}
	require.NoError(t, dbConn.Create(otherUser).Error)

	otherStore := &model.Store{
		UserID:   otherUser.ID,
		Name:     "Other Store",
		Region:   "서울특별시",
		District: "서초구",
		Address:  "서울시 서초구 테스트로 2",
	}
	require.NoError(t, dbConn.Create(otherStore).Error)

	product := &model.Product{
		Name:          "Other Gold Necklace",
		Price:         1200000,
		Category:      model.CategoryNecklace,
		Material:      model.MaterialGold,
		StockQuantity: 8,
		StoreID:       otherStore.ID,
	}
	require.NoError(t, productService.CreateProduct(product))

	product.Price = 1400000
	err := productService.UpdateProduct(user.ID, product)
	assert.ErrorIs(t, err, ErrProductAccessDenied)
}

func TestProductService_DeleteProduct(t *testing.T) {
	productService, _, user, store := setupProductServiceTest(t)

	product := &model.Product{
		Name:          "Gold Pendant",
		Price:         1000000,
		Category:      model.CategoryNecklace,
		Material:      model.MaterialGold,
		StockQuantity: 10,
		StoreID:       store.ID,
	}
	productService.CreateProduct(product)

	err := productService.DeleteProduct(user.ID, product.ID)
	assert.NoError(t, err)

	_, err = productService.GetProductByID(product.ID)
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestProductService_DeleteProduct_NotFound(t *testing.T) {
	productService, _, user, _ := setupProductServiceTest(t)

	err := productService.DeleteProduct(user.ID, 9999)
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestProductService_DeleteProduct_AccessDenied(t *testing.T) {
	productService, dbConn, user, _ := setupProductServiceTest(t)

	otherUser := &model.User{
		Email:        fmt.Sprintf("delete-other-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hashed-password",
		Name:         "Delete Admin",
		Role:         model.RoleAdmin,
	}
	require.NoError(t, dbConn.Create(otherUser).Error)

	otherStore := &model.Store{
		UserID:   otherUser.ID,
		Name:     "Delete Store",
		Region:   "서울특별시",
		District: "중구",
		Address:  "서울시 중구 테스트로 3",
	}
	require.NoError(t, dbConn.Create(otherStore).Error)

	product := &model.Product{
		Name:          "Delete Gold Ring",
		Price:         1500000,
		Category:      model.CategoryRing,
		Material:      model.MaterialGold,
		StockQuantity: 7,
		StoreID:       otherStore.ID,
	}
	require.NoError(t, productService.CreateProduct(product))

	err := productService.DeleteProduct(user.ID, product.ID)
	assert.ErrorIs(t, err, ErrProductAccessDenied)
}

func TestProductService_CheckStock(t *testing.T) {
	productService, _, _, store := setupProductServiceTest(t)

	product := &model.Product{
		Name:          "Gold Earrings",
		Price:         1000000,
		Category:      model.CategoryEarring,
		Material:      model.MaterialGold,
		StockQuantity: 10,
		StoreID:       store.ID,
	}
	productService.CreateProduct(product)

	tests := []struct {
		name      string
		productID uint
		quantity  int
		wantErr   error
	}{
		{name: "Sufficient stock", productID: product.ID, quantity: 5, wantErr: nil},
		{name: "Exact stock", productID: product.ID, quantity: 10, wantErr: nil},
		{name: "Insufficient stock", productID: product.ID, quantity: 11, wantErr: ErrInsufficientStock},
		{name: "Non-existing product", productID: 9999, quantity: 1, wantErr: ErrProductNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := productService.CheckStock(tt.productID, nil, tt.quantity)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
