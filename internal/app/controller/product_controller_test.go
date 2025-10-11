package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupProductControllerTest 수정판: repository도 반환
func setupProductControllerTest(t *testing.T) (*ProductController, *gin.Engine, repository.ProductRepository) {
	testDB, err := db.SetupTestDB()
	require.NoError(t, err)

	productRepo := repository.NewProductRepository(testDB)
	productService := service.NewProductService(productRepo)
	productController := NewProductController(productService)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	return productController, router, productRepo
}

func TestProductController_GetAllProducts_Success(t *testing.T) {
	controller, router, productRepo := setupProductControllerTest(t)

	// 테스트용 데이터 생성
	productRepo.Create(&model.Product{
		Name:          "Gold Necklace",
		Price:         500000,
		Category:      model.CategoryGold,
		StockQuantity: 5,
	})
	productRepo.Create(&model.Product{
		Name:          "Silver Ring",
		Price:         100000,
		Category:      model.CategorySilver,
		StockQuantity: 10,
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

func TestProductController_GetAllProducts_Empty(t *testing.T) {
	controller, router, _ := setupProductControllerTest(t)

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
	controller, router, productRepo := setupProductControllerTest(t)

	// 테스트용 데이터 생성
	product := &model.Product{
		Name:          "Gold Ring",
		Price:         300000,
		Category:      model.CategoryGold,
		StockQuantity: 3,
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
	controller, router, _ := setupProductControllerTest(t)

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
	controller, router, _ := setupProductControllerTest(t)

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
	controller, router, _ := setupProductControllerTest(t)

	router.POST("/products", controller.CreateProduct)

	reqBody := CreateProductRequest{
		Name:          "Diamond Ring",
		Description:   "Beautiful diamond ring",
		Price:         1000000,
		Weight:        5.5,
		Purity:        "24K",
		Category:      model.CategoryGold,
		StockQuantity: 2,
		ImageURL:      "http://example.com/diamond.jpg",
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
	assert.Equal(t, "gold", productData["category"])
}

func TestProductController_CreateProduct_InvalidRequest(t *testing.T) {
	controller, router, _ := setupProductControllerTest(t)

	router.POST("/products", controller.CreateProduct)

	tests := []struct {
		name       string
		reqBody    map[string]interface{}
		wantStatus int
		wantError  string
	}{
		{
			name:       "Missing name",
			reqBody:    map[string]interface{}{"price": 100000, "category": "gold"},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
		{
			name:       "Missing price",
			reqBody:    map[string]interface{}{"name": "Ring", "category": "gold"},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
		{
			name:       "Invalid price (zero)",
			reqBody:    map[string]interface{}{"name": "Ring", "price": 0, "category": "gold"},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
		{
			name:       "Negative stock",
			reqBody:    map[string]interface{}{"name": "Ring", "price": 100000, "category": "gold", "stock_quantity": -1},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
		{
			name:       "Missing category",
			reqBody:    map[string]interface{}{"name": "Ring", "price": 100000},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
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
	controller, router, productRepo := setupProductControllerTest(t)

	product := &model.Product{
		Name:          "Old Name",
		Price:         100000,
		Category:      model.CategoryGold,
		StockQuantity: 5,
	}
	productRepo.Create(product)

	router.PUT("/products/:id", controller.UpdateProduct)

	reqBody := CreateProductRequest{
		Name:          "Updated Name",
		Description:   "Updated description",
		Price:         200000,
		Weight:        10.0,
		Purity:        "22K",
		Category:      model.CategoryGold,
		StockQuantity: 10,
		ImageURL:      "http://example.com/updated.jpg",
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
}

func TestProductController_DeleteProduct_Success(t *testing.T) {
	controller, router, productRepo := setupProductControllerTest(t)

	product := &model.Product{
		Name:          "To Be Deleted",
		Price:         100000,
		Category:      model.CategoryGold,
		StockQuantity: 5,
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
