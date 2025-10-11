package service

import (
	"testing"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupProductServiceTest(t *testing.T) ProductService {
	testDB, err := db.SetupTestDB()
	require.NoError(t, err)

	productRepo := repository.NewProductRepository(testDB)
	return NewProductService(productRepo)
}

func TestProductService_GetAllProducts(t *testing.T) {
	productService := setupProductServiceTest(t)

	// Initially empty
	products, err := productService.GetAllProducts()
	assert.NoError(t, err)
	assert.Len(t, products, 0)

	// Create products
	product1 := &model.Product{Name: "Gold Bar", Price: 1000000, Category: model.CategoryGold, StockQuantity: 10}
	product2 := &model.Product{Name: "Silver Ring", Price: 100000, Category: model.CategorySilver, StockQuantity: 20}

	productService.CreateProduct(product1)
	productService.CreateProduct(product2)

	// Get all
	products, err = productService.GetAllProducts()
	assert.NoError(t, err)
	assert.Len(t, products, 2)
}

func TestProductService_GetProductByID(t *testing.T) {
	productService := setupProductServiceTest(t)

	product := &model.Product{
		Name:          "Gold Bar",
		Price:         1000000,
		Category:      model.CategoryGold,
		StockQuantity: 10,
	}
	productService.CreateProduct(product)

	tests := []struct {
		name    string
		id      uint
		wantErr error
	}{
		{
			name:    "Existing product",
			id:      product.ID,
			wantErr: nil,
		},
		{
			name:    "Non-existing product",
			id:      9999,
			wantErr: ErrProductNotFound,
		},
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
	productService := setupProductServiceTest(t)

	// Create products with different categories
	goldProduct := &model.Product{Name: "Gold Bar", Price: 1000000, Category: model.CategoryGold, StockQuantity: 10}
	silverProduct := &model.Product{Name: "Silver Ring", Price: 100000, Category: model.CategorySilver, StockQuantity: 20}

	productService.CreateProduct(goldProduct)
	productService.CreateProduct(silverProduct)

	// Get by gold category
	goldProducts, err := productService.GetProductsByCategory(model.CategoryGold)
	assert.NoError(t, err)
	assert.Len(t, goldProducts, 1)
	assert.Equal(t, "Gold Bar", goldProducts[0].Name)

	// Get by silver category
	silverProducts, err := productService.GetProductsByCategory(model.CategorySilver)
	assert.NoError(t, err)
	assert.Len(t, silverProducts, 1)
	assert.Equal(t, "Silver Ring", silverProducts[0].Name)
}

func TestProductService_CreateProduct(t *testing.T) {
	productService := setupProductServiceTest(t)

	product := &model.Product{
		Name:          "24K Gold Bar",
		Description:   "Pure gold",
		Price:         5000000,
		Weight:        100,
		Purity:        "24K",
		Category:      model.CategoryGold,
		StockQuantity: 5,
	}

	err := productService.CreateProduct(product)
	assert.NoError(t, err)
	assert.NotZero(t, product.ID)
}

func TestProductService_UpdateProduct(t *testing.T) {
	productService := setupProductServiceTest(t)

	product := &model.Product{
		Name:          "Gold Bar",
		Price:         1000000,
		Category:      model.CategoryGold,
		StockQuantity: 10,
	}
	productService.CreateProduct(product)

	// Update
	product.Price = 1100000
	product.StockQuantity = 15
	err := productService.UpdateProduct(product)
	assert.NoError(t, err)

	// Verify
	updated, _ := productService.GetProductByID(product.ID)
	assert.Equal(t, float64(1100000), updated.Price)
	assert.Equal(t, 15, updated.StockQuantity)
}

func TestProductService_UpdateProduct_NotFound(t *testing.T) {
	productService := setupProductServiceTest(t)

	product := &model.Product{
		ID:            9999,
		Name:          "Non-existing",
		Price:         1000000,
		Category:      model.CategoryGold,
		StockQuantity: 10,
	}

	err := productService.UpdateProduct(product)
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestProductService_DeleteProduct(t *testing.T) {
	productService := setupProductServiceTest(t)

	product := &model.Product{
		Name:          "Gold Bar",
		Price:         1000000,
		Category:      model.CategoryGold,
		StockQuantity: 10,
	}
	productService.CreateProduct(product)

	err := productService.DeleteProduct(product.ID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = productService.GetProductByID(product.ID)
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestProductService_DeleteProduct_NotFound(t *testing.T) {
	productService := setupProductServiceTest(t)

	err := productService.DeleteProduct(9999)
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestProductService_CheckStock(t *testing.T) {
	productService := setupProductServiceTest(t)

	product := &model.Product{
		Name:          "Gold Bar",
		Price:         1000000,
		Category:      model.CategoryGold,
		StockQuantity: 10,
	}
	productService.CreateProduct(product)

	tests := []struct {
		name      string
		productID uint
		quantity  int
		wantErr   error
	}{
		{
			name:      "Sufficient stock",
			productID: product.ID,
			quantity:  5,
			wantErr:   nil,
		},
		{
			name:      "Exact stock",
			productID: product.ID,
			quantity:  10,
			wantErr:   nil,
		},
		{
			name:      "Insufficient stock",
			productID: product.ID,
			quantity:  11,
			wantErr:   ErrInsufficientStock,
		},
		{
			name:      "Non-existing product",
			productID: 9999,
			quantity:  1,
			wantErr:   ErrProductNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := productService.CheckStock(tt.productID, tt.quantity)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
