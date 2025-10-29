package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupProductControllerTest 수정판: repository도 반환
func setupProductControllerTest(t *testing.T) (*ProductController, *gin.Engine, repository.ProductRepository, *model.User, *model.Store) {
	testDB, err := db.SetupTestDB()
	require.NoError(t, err)
	t.Cleanup(func() {
		db.CleanupTestDB(testDB)
	})

	productRepo := repository.NewProductRepository(testDB)
	productOptionRepo := repository.NewProductOptionRepository(testDB)
	productService := service.NewProductService(productRepo, productOptionRepo)
	productController := NewProductController(productService)

	owner := &model.User{
		Email:        fmt.Sprintf("owner-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hashed-password",
		Name:         "Store Owner",
		Role:         model.RoleAdmin,
	}
	require.NoError(t, testDB.Create(owner).Error)

	store := &model.Store{
		UserID:   owner.ID,
		Name:     "Test Store",
		Region:   "서울특별시",
		District: "강남구",
		Address:  "서울시 강남구 테스트로 1",
	}
	require.NoError(t, testDB.Create(store).Error)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", owner.ID)
		c.Set("user_role", string(owner.Role))
		c.Next()
	})

	return productController, router, productRepo, owner, store
}

func TestProductController_GetAllProducts_Success(t *testing.T) {
	controller, router, productRepo, _, store := setupProductControllerTest(t)

	// 테스트용 데이터 생성
	productRepo.Create(&model.Product{
		Name:          "Gold Necklace",
		Price:         500000,
		Category:      model.CategoryNecklace,
		Material:      model.MaterialGold,
		StockQuantity: 5,
		StoreID:       store.ID,
	})
	productRepo.Create(&model.Product{
		Name:          "Silver Ring",
		Price:         100000,
		Category:      model.CategoryRing,
		Material:      model.MaterialSilver,
		StockQuantity: 10,
		StoreID:       store.ID,
	})

	router.GET("/products", controller.GetAllProducts)

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	products := response["products"].([]interface{})
	assert.Len(t, products, 2)
	assert.Equal(t, float64(2), response["count"])
}

func TestProductController_GetProductFilters(t *testing.T) {
	controller, router, productRepo, _, store := setupProductControllerTest(t)

	productRepo.Create(&model.Product{
		Name:          "Gold Ring",
		Price:         300000,
		Category:      model.CategoryRing,
		Material:      model.MaterialGold,
		StockQuantity: 4,
		StoreID:       store.ID,
	})
	productRepo.Create(&model.Product{
		Name:          "Silver Bracelet",
		Price:         150000,
		Category:      model.CategoryBracelet,
		Material:      model.MaterialSilver,
		StockQuantity: 7,
		StoreID:       store.ID,
	})

	router.GET("/products/filters", controller.GetProductFilters)

	req := httptest.NewRequest(http.MethodGet, "/products/filters", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	categories := response["categories"].([]interface{})
	categoryValues := make([]string, 0, len(categories))
	for _, c := range categories {
		categoryValues = append(categoryValues, c.(string))
	}

	materials := response["materials"].([]interface{})
	materialValues := make([]string, 0, len(materials))
	for _, m := range materials {
		materialValues = append(materialValues, m.(string))
	}

	assert.ElementsMatch(t, []string{
		string(model.CategoryRing),
		string(model.CategoryBracelet),
	}, categoryValues)

	assert.ElementsMatch(t, []string{
		string(model.MaterialGold),
		string(model.MaterialSilver),
	}, materialValues)
}

func TestProductController_GetAllProducts_Empty(t *testing.T) {
	controller, router, _, _, _ := setupProductControllerTest(t)

	router.GET("/products", controller.GetAllProducts)

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	products := response["products"].([]interface{})
	assert.Len(t, products, 0)
	assert.Equal(t, float64(0), response["count"])
}

func TestProductController_GetProductByID_Success(t *testing.T) {
	controller, router, productRepo, _, store := setupProductControllerTest(t)

	// 테스트용 데이터 생성
	product := &model.Product{
		Name:          "Gold Ring",
		Price:         300000,
		Category:      model.CategoryRing,
		Material:      model.MaterialGold,
		StockQuantity: 3,
		StoreID:       store.ID,
	}
	productRepo.Create(product)

	router.GET("/products/:id", controller.GetProductByID)

	req := httptest.NewRequest(http.MethodGet, "/products/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	productData := response["product"].(map[string]interface{})
	assert.Equal(t, "Gold Ring", productData["name"])
	assert.Equal(t, float64(300000), productData["price"])
}

func TestProductController_GetProductByID_NotFound(t *testing.T) {
	controller, router, _, _, _ := setupProductControllerTest(t)

	router.GET("/products/:id", controller.GetProductByID)

	req := httptest.NewRequest(http.MethodGet, "/products/9999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Product not found", response["error"])
}

func TestProductController_GetProductByID_InvalidID(t *testing.T) {
	controller, router, _, _, _ := setupProductControllerTest(t)

	router.GET("/products/:id", controller.GetProductByID)

	req := httptest.NewRequest(http.MethodGet, "/products/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Invalid product ID", response["error"])
}

func TestProductController_CreateProduct_Success(t *testing.T) {
	controller, router, _, _, store := setupProductControllerTest(t)

	router.POST("/products", controller.CreateProduct)

	reqBody := CreateProductRequest{
		Name:          "Diamond Ring",
		Description:   "Beautiful diamond ring",
		Price:         1000000,
		Weight:        5.5,
		Purity:        "24K",
		Category:      model.CategoryRing,
		Material:      model.MaterialGold,
		StockQuantity: 2,
		ImageURL:      "http://example.com/diamond.jpg",
		StoreID:       store.ID,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Product created successfully", response["message"])

	productData := response["product"].(map[string]interface{})
	assert.Equal(t, "Diamond Ring", productData["name"])
	assert.Equal(t, float64(1000000), productData["price"])
	assert.Equal(t, string(model.CategoryRing), productData["category"])
	assert.Equal(t, string(model.MaterialGold), productData["material"])
}

func TestProductController_CreateProduct_InvalidRequest(t *testing.T) {
	controller, router, _, _, store := setupProductControllerTest(t)

	router.POST("/products", controller.CreateProduct)

	tests := []struct {
		name       string
		reqBody    map[string]interface{}
		wantStatus int
		wantError  string
	}{
		{
			name:       "Missing name",
			reqBody:    map[string]interface{}{"price": 100000, "category": string(model.CategoryRing), "material": string(model.MaterialGold), "store_id": store.ID},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
		{
			name:       "Missing price",
			reqBody:    map[string]interface{}{"name": "Ring", "category": string(model.CategoryRing), "material": string(model.MaterialGold), "store_id": store.ID},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
		{
			name:       "Invalid price (zero)",
			reqBody:    map[string]interface{}{"name": "Ring", "price": 0, "category": string(model.CategoryRing), "material": string(model.MaterialGold), "store_id": store.ID},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
		{
			name:       "Negative stock",
			reqBody:    map[string]interface{}{"name": "Ring", "price": 100000, "category": string(model.CategoryRing), "material": string(model.MaterialGold), "stock_quantity": -1, "store_id": store.ID},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
		{
			name:       "Missing category",
			reqBody:    map[string]interface{}{"name": "Ring", "price": 100000, "material": string(model.MaterialGold), "store_id": store.ID},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
		{
			name:       "Missing material",
			reqBody:    map[string]interface{}{"name": "Ring", "price": 100000, "category": string(model.CategoryRing), "store_id": store.ID},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
		{
			name:       "Missing store id",
			reqBody:    map[string]interface{}{"name": "Ring", "price": 100000, "category": string(model.CategoryRing), "material": string(model.MaterialGold)},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
		{
			name:       "Invalid category value",
			reqBody:    map[string]interface{}{"name": "Ring", "price": 100000, "category": "unknown", "material": string(model.MaterialGold), "store_id": store.ID},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid category",
		},
		{
			name:       "Invalid material value",
			reqBody:    map[string]interface{}{"name": "Ring", "price": 100000, "category": string(model.CategoryRing), "material": "플라스틱", "store_id": store.ID},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid material",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tt.wantError, response["error"])
		})
	}
}

// UpdateProduct, DeleteProduct 테스트도 동일하게 productRepo 직접 사용
func TestProductController_UpdateProduct_Success(t *testing.T) {
	controller, router, productRepo, _, store := setupProductControllerTest(t)

	product := &model.Product{
		Name:          "Old Name",
		Price:         100000,
		Category:      model.CategoryNecklace,
		Material:      model.MaterialGold,
		StockQuantity: 5,
		StoreID:       store.ID,
	}
	productRepo.Create(product)

	router.PUT("/products/:id", controller.UpdateProduct)

	reqBody := CreateProductRequest{
		Name:          "Updated Name",
		Description:   "Updated description",
		Price:         200000,
		Weight:        10.0,
		Purity:        "22K",
		Category:      model.CategoryNecklace,
		Material:      model.MaterialGold,
		StockQuantity: 10,
		ImageURL:      "http://example.com/updated.jpg",
		StoreID:       store.ID,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/products/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Product updated successfully", response["message"])

	productData := response["product"].(map[string]interface{})
	assert.Equal(t, "Updated Name", productData["name"])
	assert.Equal(t, float64(200000), productData["price"])
	assert.Equal(t, string(model.MaterialGold), productData["material"])
}

func TestProductController_DeleteProduct_Success(t *testing.T) {
	controller, router, productRepo, _, store := setupProductControllerTest(t)

	product := &model.Product{
		Name:          "To Be Deleted",
		Price:         100000,
		Category:      model.CategoryRing,
		Material:      model.MaterialOther,
		StockQuantity: 5,
		StoreID:       store.ID,
	}
	productRepo.Create(product)

	router.DELETE("/products/:id", controller.DeleteProduct)

	req := httptest.NewRequest(http.MethodDelete, "/products/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Product deleted successfully", response["message"])

	_, err = productRepo.FindByID(1)
	assert.Error(t, err)
}
