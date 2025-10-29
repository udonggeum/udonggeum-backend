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
	"gorm.io/gorm"
)

func setupCartControllerTest(t *testing.T) (*CartController, *gin.Engine, *gorm.DB, *model.User, *model.Product) {
	testDB, err := db.SetupTestDB()
	require.NoError(t, err)
	t.Cleanup(func() {
		db.CleanupTestDB(testDB)
	})

	cartRepo := repository.NewCartRepository(testDB)
	productRepo := repository.NewProductRepository(testDB)
	productOptionRepo := repository.NewProductOptionRepository(testDB)
	cartService := service.NewCartService(cartRepo, productRepo, productOptionRepo)
	cartController := NewCartController(cartService)

	// Create test user
	user := &model.User{
		Email:        "test@example.com",
		PasswordHash: "hash",
		Name:         "Test User",
		Role:         model.RoleUser,
	}
	testDB.Create(user)

	store := &model.Store{
		UserID:   user.ID,
		Name:     "Test Store",
		Region:   "서울특별시",
		District: "강남구",
		Address:  "서울시 강남구 테스트로 1",
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

	gin.SetMode(gin.TestMode)
	router := gin.New()

	return cartController, router, testDB, user, product
}

// Helper function to set user ID in context
func setUserIDInContext(c *gin.Context, userID uint) {
	c.Set("user_id", userID)
}

func TestCartController_GetCart_Success(t *testing.T) {
	controller, router, testDB, user, product := setupCartControllerTest(t)

	// Add item to cart
	cartRepo := repository.NewCartRepository(testDB)
	cartRepo.Create(&model.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
	})

	router.GET("/cart", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.GetCart(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/cart", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(1), response["count"])
	assert.Equal(t, float64(200000), response["total"]) // 100000 * 2
}

func TestCartController_GetCart_Empty(t *testing.T) {
	controller, router, _, user, _ := setupCartControllerTest(t)

	router.GET("/cart", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.GetCart(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/cart", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(0), response["count"])
	assert.Equal(t, float64(0), response["total"])
}

func TestCartController_GetCart_Unauthorized(t *testing.T) {
	controller, router, _, _, _ := setupCartControllerTest(t)

	router.GET("/cart", controller.GetCart)

	req := httptest.NewRequest(http.MethodGet, "/cart", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "User not authenticated", response["error"])
}

func TestCartController_AddToCart_Success(t *testing.T) {
	controller, router, _, user, product := setupCartControllerTest(t)

	router.POST("/cart", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.AddToCart(c)
	})

	reqBody := AddToCartRequest{
		ProductID: product.ID,
		Quantity:  2,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/cart", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Item added to cart successfully", response["message"])
}

func TestCartController_AddToCart_Unauthorized(t *testing.T) {
	controller, router, _, _, product := setupCartControllerTest(t)

	router.POST("/cart", controller.AddToCart)

	reqBody := AddToCartRequest{
		ProductID: product.ID,
		Quantity:  2,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/cart", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "User not authenticated", response["error"])
}

func TestCartController_AddToCart_ProductNotFound(t *testing.T) {
	controller, router, _, user, _ := setupCartControllerTest(t)

	router.POST("/cart", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.AddToCart(c)
	})

	reqBody := AddToCartRequest{
		ProductID: 9999,
		Quantity:  2,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/cart", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Product not found", response["error"])
}

func TestCartController_AddToCart_InsufficientStock(t *testing.T) {
	controller, router, _, user, product := setupCartControllerTest(t)

	router.POST("/cart", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.AddToCart(c)
	})

	reqBody := AddToCartRequest{
		ProductID: product.ID,
		Quantity:  100, // Exceeds stock
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/cart", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Insufficient stock", response["error"])
}

func TestCartController_AddToCart_InvalidRequest(t *testing.T) {
	controller, router, _, user, _ := setupCartControllerTest(t)

	router.POST("/cart", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.AddToCart(c)
	})

	tests := []struct {
		name       string
		reqBody    map[string]interface{}
		wantStatus int
		wantError  string
	}{
		{
			name:       "Missing product_id",
			reqBody:    map[string]interface{}{"quantity": 2},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
		{
			name:       "Missing quantity",
			reqBody:    map[string]interface{}{"product_id": 1},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
		{
			name:       "Zero quantity",
			reqBody:    map[string]interface{}{"product_id": 1, "quantity": 0},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
		{
			name:       "Negative quantity",
			reqBody:    map[string]interface{}{"product_id": 1, "quantity": -1},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/cart", bytes.NewBuffer(jsonBody))
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

func TestCartController_UpdateCartItem_Success(t *testing.T) {
	controller, router, testDB, user, product := setupCartControllerTest(t)

	// Add item to cart
	cartRepo := repository.NewCartRepository(testDB)
	cartItem := &model.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
	}
	cartRepo.Create(cartItem)

	router.PUT("/cart/:id", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.UpdateCartItem(c)
	})

	reqBody := UpdateCartRequest{
		Quantity: 5,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/cart/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Cart item updated successfully", response["message"])
}

func TestCartController_UpdateCartItem_Unauthorized(t *testing.T) {
	controller, router, _, _, _ := setupCartControllerTest(t)

	router.PUT("/cart/:id", controller.UpdateCartItem)

	reqBody := UpdateCartRequest{
		Quantity: 5,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/cart/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "User not authenticated", response["error"])
}

func TestCartController_UpdateCartItem_NotFound(t *testing.T) {
	controller, router, _, user, _ := setupCartControllerTest(t)

	router.PUT("/cart/:id", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.UpdateCartItem(c)
	})

	reqBody := UpdateCartRequest{
		Quantity: 5,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/cart/9999", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Cart item not found", response["error"])
}

func TestCartController_UpdateCartItem_InvalidID(t *testing.T) {
	controller, router, _, user, _ := setupCartControllerTest(t)

	router.PUT("/cart/:id", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.UpdateCartItem(c)
	})

	reqBody := UpdateCartRequest{
		Quantity: 5,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/cart/invalid", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Invalid cart item ID", response["error"])
}

func TestCartController_UpdateCartItem_InsufficientStock(t *testing.T) {
	controller, router, testDB, user, product := setupCartControllerTest(t)

	// Add item to cart
	cartRepo := repository.NewCartRepository(testDB)
	cartItem := &model.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
	}
	cartRepo.Create(cartItem)

	router.PUT("/cart/:id", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.UpdateCartItem(c)
	})

	reqBody := UpdateCartRequest{
		Quantity: 100, // Exceeds stock
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/cart/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Insufficient stock", response["error"])
}

func TestCartController_UpdateCartItem_InvalidRequest(t *testing.T) {
	controller, router, _, user, _ := setupCartControllerTest(t)

	router.PUT("/cart/:id", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.UpdateCartItem(c)
	})

	tests := []struct {
		name       string
		reqBody    map[string]interface{}
		wantStatus int
		wantError  string
	}{
		{
			name:       "Missing quantity",
			reqBody:    map[string]interface{}{},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
		{
			name:       "Zero quantity",
			reqBody:    map[string]interface{}{"quantity": 0},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
		{
			name:       "Negative quantity",
			reqBody:    map[string]interface{}{"quantity": -1},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPut, "/cart/1", bytes.NewBuffer(jsonBody))
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

func TestCartController_RemoveFromCart_Success(t *testing.T) {
	controller, router, testDB, user, product := setupCartControllerTest(t)

	// Add item to cart
	cartRepo := repository.NewCartRepository(testDB)
	cartItem := &model.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
	}
	cartRepo.Create(cartItem)

	router.DELETE("/cart/:id", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.RemoveFromCart(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/cart/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Cart item removed successfully", response["message"])
}

func TestCartController_RemoveFromCart_Unauthorized(t *testing.T) {
	controller, router, _, _, _ := setupCartControllerTest(t)

	router.DELETE("/cart/:id", controller.RemoveFromCart)

	req := httptest.NewRequest(http.MethodDelete, "/cart/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "User not authenticated", response["error"])
}

func TestCartController_RemoveFromCart_NotFound(t *testing.T) {
	controller, router, _, user, _ := setupCartControllerTest(t)

	router.DELETE("/cart/:id", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.RemoveFromCart(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/cart/9999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Cart item not found", response["error"])
}

func TestCartController_RemoveFromCart_InvalidID(t *testing.T) {
	controller, router, _, user, _ := setupCartControllerTest(t)

	router.DELETE("/cart/:id", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.RemoveFromCart(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/cart/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Invalid cart item ID", response["error"])
}

func TestCartController_ClearCart_Success(t *testing.T) {
	controller, router, testDB, user, product := setupCartControllerTest(t)

	// Add items to cart
	cartRepo := repository.NewCartRepository(testDB)
	cartRepo.Create(&model.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 2})
	cartRepo.Create(&model.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 3})

	router.DELETE("/cart", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.ClearCart(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/cart", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Cart cleared successfully", response["message"])

	// Verify cart is empty
	items, _ := cartRepo.FindByUserID(user.ID)
	assert.Len(t, items, 0)
}

func TestCartController_ClearCart_Unauthorized(t *testing.T) {
	controller, router, _, _, _ := setupCartControllerTest(t)

	router.DELETE("/cart", controller.ClearCart)

	req := httptest.NewRequest(http.MethodDelete, "/cart", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "User not authenticated", response["error"])
}
