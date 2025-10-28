package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/controller"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/db"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type TestServer struct {
	Router      *gin.Engine
	DB          *gorm.DB
	AuthService service.AuthService
}

func setupIntegrationTest(t *testing.T) *TestServer {
	gin.SetMode(gin.TestMode)

	// Setup database
	testDB, err := db.SetupTestDB()
	require.NoError(t, err)

	// Setup repositories
	userRepo := repository.NewUserRepository(testDB)
	productRepo := repository.NewProductRepository(testDB)
	productOptionRepo := repository.NewProductOptionRepository(testDB)
	orderRepo := repository.NewOrderRepository(testDB)
	cartRepo := repository.NewCartRepository(testDB)

	// Setup services
	authService := service.NewAuthService(
		userRepo,
		"test-secret",
		15*time.Minute,
		7*24*time.Hour,
	)
	productService := service.NewProductService(productRepo, productOptionRepo)
	cartService := service.NewCartService(cartRepo, productRepo, productOptionRepo)
	orderService := service.NewOrderService(orderRepo, cartRepo, productRepo, testDB, productOptionRepo)

	// Setup controllers
	authController := controller.NewAuthController(authService)
	productController := controller.NewProductController(productService)
	cartController := controller.NewCartController(cartService)
	orderController := controller.NewOrderController(orderService)

	// Setup middleware
	authMiddleware := middleware.NewAuthMiddleware("test-secret")

	// Setup router
	router := gin.New()

	// Auth routes
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/register", authController.Register)
		auth.POST("/login", authController.Login)
		auth.GET("/me", authMiddleware.Authenticate(), authController.GetMe)
	}

	// Product routes
	products := router.Group("/api/v1/products")
	{
		products.GET("", productController.GetAllProducts)
		products.GET("/:id", productController.GetProductByID)
		products.POST("", authMiddleware.Authenticate(), authMiddleware.RequireRole("admin"), productController.CreateProduct)
	}

	// Cart routes
	cart := router.Group("/api/v1/cart")
	cart.Use(authMiddleware.Authenticate())
	{
		cart.GET("", cartController.GetCart)
		cart.POST("", cartController.AddToCart)
		cart.PUT("/:id", cartController.UpdateCartItem)
		cart.DELETE("/:id", cartController.RemoveFromCart)
	}

	// Order routes
	orders := router.Group("/api/v1/orders")
	orders.Use(authMiddleware.Authenticate())
	{
		orders.GET("", orderController.GetOrders)
		orders.GET("/:id", orderController.GetOrderByID)
		orders.POST("", orderController.CreateOrder)
	}

	return &TestServer{
		Router:      router,
		DB:          testDB,
		AuthService: authService,
	}
}

func TestCompleteUserJourney(t *testing.T) {
	ts := setupIntegrationTest(t)
	defer db.CleanupTestDB(ts.DB)

	// 1. Register a new user
	t.Log("Step 1: Register user")
	registerReq := map[string]string{
		"email":    "buyer@example.com",
		"password": "password123",
		"name":     "Test Buyer",
		"phone":    "010-1234-5678",
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ts.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var registerResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &registerResp)
	tokens := registerResp["tokens"].(map[string]interface{})
	accessToken := tokens["access_token"].(string)

	// 2. Create products as admin
	t.Log("Step 2: Create products")
	// Register admin user (direct insert for test convenience)
	adminUser := &model.User{
		Email:        "admin@example.com",
		PasswordHash: "hash",
		Name:         "Admin",
		Role:         model.RoleAdmin,
	}
	ts.DB.Create(adminUser)

	// Create store and product directly in DB
	store := &model.Store{
		UserID:   adminUser.ID,
		Name:     "강남 본점",
		Region:   "서울특별시",
		District: "강남구",
		Address:  "서울시 강남구 테헤란로 1",
	}
	ts.DB.Create(store)

	product := &model.Product{
		Name:          "24K Gold Bar 100g",
		Description:   "Pure gold bar",
		Price:         8500000,
		Weight:        100,
		Purity:        "24K",
		Category:      model.CategoryGold,
		StockQuantity: 10,
		ImageURL:      "https://example.com/gold.jpg",
		StoreID:       store.ID,
	}
	ts.DB.Create(product)

	// 3. Get all products
	t.Log("Step 3: Browse products")
	req = httptest.NewRequest("GET", "/api/v1/products", nil)
	w = httptest.NewRecorder()

	ts.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var productsResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &productsResp)
	assert.NotNil(t, productsResp["products"])

	// 4. Add product to cart
	t.Log("Step 4: Add to cart")
	addToCartReq := map[string]interface{}{
		"product_id": product.ID,
		"quantity":   2,
	}
	body, _ = json.Marshal(addToCartReq)
	req = httptest.NewRequest("POST", "/api/v1/cart", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w = httptest.NewRecorder()

	ts.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// 5. View cart
	t.Log("Step 5: View cart")
	req = httptest.NewRequest("GET", "/api/v1/cart", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w = httptest.NewRecorder()

	ts.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var cartResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &cartResp)
	cartItems := cartResp["cart_items"].([]interface{})
	assert.Len(t, cartItems, 1)

	// 6. Create order
	t.Log("Step 6: Create order")
	createOrderReq := map[string]string{
		"shipping_address": "서울시 강남구 테헤란로 123",
	}
	body, _ = json.Marshal(createOrderReq)
	req = httptest.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w = httptest.NewRecorder()

	ts.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var orderResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &orderResp)
	order := orderResp["order"].(map[string]interface{})
	assert.NotNil(t, order)
	assert.Equal(t, "pending", order["status"])

	// 7. View orders
	t.Log("Step 7: View order history")
	req = httptest.NewRequest("GET", "/api/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w = httptest.NewRecorder()

	ts.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var ordersResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &ordersResp)
	orders := ordersResp["orders"].([]interface{})
	assert.Len(t, orders, 1)

	// 8. Verify cart is empty after order
	t.Log("Step 8: Verify cart is empty")
	req = httptest.NewRequest("GET", "/api/v1/cart", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w = httptest.NewRecorder()

	ts.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	json.Unmarshal(w.Body.Bytes(), &cartResp)
	cartItems = cartResp["cart_items"].([]interface{})
	assert.Len(t, cartItems, 0)

	// 9. Verify stock decreased
	t.Log("Step 9: Verify stock decreased")
	var updatedProduct model.Product
	ts.DB.First(&updatedProduct, product.ID)
	assert.Equal(t, 8, updatedProduct.StockQuantity) // 10 - 2 = 8

}

func TestAuthenticationFlow(t *testing.T) {
	ts := setupIntegrationTest(t)
	defer db.CleanupTestDB(ts.DB)

	// Register
	registerReq := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"name":     "Test User",
		"phone":    "010-1234-5678",
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ts.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var registerResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &registerResp)
	tokens := registerResp["tokens"].(map[string]interface{})
	accessToken := tokens["access_token"].(string)

	// Login
	loginReq := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}
	body, _ = json.Marshal(loginReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	ts.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Get user info
	req = httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w = httptest.NewRecorder()

	ts.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var meResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &meResp)
	user := meResp["user"].(map[string]interface{})
	assert.Equal(t, "test@example.com", user["email"])
	assert.Equal(t, "Test User", user["name"])
}

func TestUnauthorizedAccess(t *testing.T) {
	ts := setupIntegrationTest(t)
	defer db.CleanupTestDB(ts.DB)

	// Try to access protected routes without token
	protectedRoutes := []string{
		"/api/v1/auth/me",
		"/api/v1/cart",
		"/api/v1/orders",
	}

	for _, route := range protectedRoutes {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest("GET", route, nil)
			w := httptest.NewRecorder()

			ts.Router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}
