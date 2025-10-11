package repository

import (
	"testing"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupProductTest(t *testing.T) (*gorm.DB, ProductRepository) {
	testDB, err := db.SetupTestDB()
	require.NoError(t, err)

	repo := NewProductRepository(testDB)
	return testDB, repo
}

func TestProductRepository_Create(t *testing.T) {
	testDB, repo := setupProductTest(t)
	defer db.CleanupTestDB(testDB)

	product := &model.Product{
		Name:          "24K Gold Bar",
		Description:   "Pure gold bar 100g",
		Price:         8500000,
		Weight:        100,
		Purity:        "24K",
		Category:      model.CategoryGold,
		StockQuantity: 10,
		ImageURL:      "https://example.com/gold.jpg",
	}

	err := repo.Create(product)
	assert.NoError(t, err)
	assert.NotZero(t, product.ID)
}

func TestProductRepository_FindAll(t *testing.T) {
	testDB, repo := setupProductTest(t)
	defer db.CleanupTestDB(testDB)

	// Create test products
	products := []model.Product{
		{
			Name:          "Gold Bar",
			Price:         1000000,
			Category:      model.CategoryGold,
			StockQuantity: 10,
		},
		{
			Name:          "Silver Ring",
			Price:         100000,
			Category:      model.CategorySilver,
			StockQuantity: 20,
		},
	}

	for i := range products {
		err := repo.Create(&products[i])
		require.NoError(t, err)
	}

	// Find all
	found, err := repo.FindAll()
	assert.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestProductRepository_FindByID(t *testing.T) {
	testDB, repo := setupProductTest(t)
	defer db.CleanupTestDB(testDB)

	product := &model.Product{
		Name:          "Gold Bar",
		Price:         1000000,
		Category:      model.CategoryGold,
		StockQuantity: 10,
	}
	err := repo.Create(product)
	require.NoError(t, err)

	tests := []struct {
		name    string
		id      uint
		wantErr bool
	}{
		{
			name:    "Existing product",
			id:      product.ID,
			wantErr: false,
		},
		{
			name:    "Non-existing product",
			id:      9999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := repo.FindByID(tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, found)
			} else {
				require.NoError(t, err)
				require.NotNil(t, found)
				assert.Equal(t, product.Name, found.Name)
			}
		})
	}
}

func TestProductRepository_FindByCategory(t *testing.T) {
	testDB, repo := setupProductTest(t)
	defer db.CleanupTestDB(testDB)

	// Create products with different categories
	goldProduct := &model.Product{
		Name:          "Gold Bar",
		Price:         1000000,
		Category:      model.CategoryGold,
		StockQuantity: 10,
	}
	silverProduct := &model.Product{
		Name:          "Silver Ring",
		Price:         100000,
		Category:      model.CategorySilver,
		StockQuantity: 20,
	}

	err := repo.Create(goldProduct)
	require.NoError(t, err)
	err = repo.Create(silverProduct)
	require.NoError(t, err)

	// Find by gold category
	goldProducts, err := repo.FindByCategory(model.CategoryGold)
	assert.NoError(t, err)
	assert.Len(t, goldProducts, 1)
	assert.Equal(t, "Gold Bar", goldProducts[0].Name)

	// Find by silver category
	silverProducts, err := repo.FindByCategory(model.CategorySilver)
	assert.NoError(t, err)
	assert.Len(t, silverProducts, 1)
	assert.Equal(t, "Silver Ring", silverProducts[0].Name)
}

func TestProductRepository_Update(t *testing.T) {
	testDB, repo := setupProductTest(t)
	defer db.CleanupTestDB(testDB)

	product := &model.Product{
		Name:          "Gold Bar",
		Price:         1000000,
		Category:      model.CategoryGold,
		StockQuantity: 10,
	}
	err := repo.Create(product)
	require.NoError(t, err)

	// Update product
	product.Price = 1100000
	product.StockQuantity = 15

	err = repo.Update(product)
	assert.NoError(t, err)

	// Verify update
	updated, err := repo.FindByID(product.ID)
	require.NoError(t, err)
	assert.Equal(t, float64(1100000), updated.Price)
	assert.Equal(t, 15, updated.StockQuantity)
}

func TestProductRepository_UpdateStock(t *testing.T) {
	testDB, repo := setupProductTest(t)
	defer db.CleanupTestDB(testDB)

	product := &model.Product{
		Name:          "Gold Bar",
		Price:         1000000,
		Category:      model.CategoryGold,
		StockQuantity: 10,
	}
	err := repo.Create(product)
	require.NoError(t, err)

	// Decrease stock
	err = repo.UpdateStock(product.ID, -3)
	assert.NoError(t, err)

	// Verify stock update
	updated, err := repo.FindByID(product.ID)
	require.NoError(t, err)
	assert.Equal(t, 7, updated.StockQuantity)

	// Increase stock
	err = repo.UpdateStock(product.ID, 5)
	assert.NoError(t, err)

	updated, err = repo.FindByID(product.ID)
	require.NoError(t, err)
	assert.Equal(t, 12, updated.StockQuantity)
}

func TestProductRepository_Delete(t *testing.T) {
	testDB, repo := setupProductTest(t)
	defer db.CleanupTestDB(testDB)

	product := &model.Product{
		Name:          "Gold Bar",
		Price:         1000000,
		Category:      model.CategoryGold,
		StockQuantity: 10,
	}
	err := repo.Create(product)
	require.NoError(t, err)

	// Delete product
	err = repo.Delete(product.ID)
	assert.NoError(t, err)

	// Verify deletion (soft delete)
	_, err = repo.FindByID(product.ID)
	assert.Error(t, err)
}
