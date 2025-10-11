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

func setupOrderControllerTest(t *testing.T) (*OrderController, *gin.Engine, *gorm.DB, *model.User, *model.Product) {
	testDB, err := db.SetupTestDB()
	require.NoError(t, err)

	orderRepo := repository.NewOrderRepository(testDB)
	cartRepo := repository.NewCartRepository(testDB)
	productRepo := repository.NewProductRepository(testDB)
	orderService := service.NewOrderService(orderRepo, cartRepo, productRepo, testDB)
	orderController := NewOrderController(orderService)

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
		Category:      model.CategoryGold,
		StockQuantity: 10,
	}
	testDB.Create(product)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	return orderController, router, testDB, user, product
}

func TestOrderController_GetOrders_Success(t *testing.T) {
	controller, router, testDB, user, _ := setupOrderControllerTest(t)

	// Create test orders
	orderRepo := repository.NewOrderRepository(testDB)
	orderRepo.Create(&model.Order{
		UserID:          user.ID,
		TotalAmount:     100000,
		Status:          model.OrderStatusPending,
		PaymentStatus:   model.PaymentStatusPending,
		ShippingAddress: "서울시 강남구",
	})
	orderRepo.Create(&model.Order{
		UserID:          user.ID,
		TotalAmount:     200000,
		Status:          model.OrderStatusConfirmed,
		PaymentStatus:   model.PaymentStatusCompleted,
		ShippingAddress: "서울시 서초구",
	})

	router.GET("/orders", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.GetOrders(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(2), response["count"])
	orders := response["orders"].([]interface{})
	assert.Len(t, orders, 2)
}

func TestOrderController_GetOrders_Empty(t *testing.T) {
	controller, router, _, user, _ := setupOrderControllerTest(t)

	router.GET("/orders", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.GetOrders(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(0), response["count"])
	orders := response["orders"].([]interface{})
	assert.Len(t, orders, 0)
}

func TestOrderController_GetOrders_Unauthorized(t *testing.T) {
	controller, router, _, _, _ := setupOrderControllerTest(t)

	router.GET("/orders", controller.GetOrders)

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "User not authenticated", response["error"])
}

func TestOrderController_GetOrderByID_Success(t *testing.T) {
	controller, router, testDB, user, _ := setupOrderControllerTest(t)

	// Create test order
	orderRepo := repository.NewOrderRepository(testDB)
	order := &model.Order{
		UserID:          user.ID,
		TotalAmount:     100000,
		Status:          model.OrderStatusPending,
		PaymentStatus:   model.PaymentStatusPending,
		ShippingAddress: "서울시 강남구",
	}
	orderRepo.Create(order)

	router.GET("/orders/:id", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.GetOrderByID(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	orderData := response["order"].(map[string]interface{})
	assert.Equal(t, float64(100000), orderData["total_amount"])
	assert.Equal(t, "pending", orderData["status"])
}

func TestOrderController_GetOrderByID_Unauthorized(t *testing.T) {
	controller, router, _, _, _ := setupOrderControllerTest(t)

	router.GET("/orders/:id", controller.GetOrderByID)

	req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "User not authenticated", response["error"])
}

func TestOrderController_GetOrderByID_NotFound(t *testing.T) {
	controller, router, _, user, _ := setupOrderControllerTest(t)

	router.GET("/orders/:id", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.GetOrderByID(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/orders/9999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Order not found", response["error"])
}

func TestOrderController_GetOrderByID_InvalidID(t *testing.T) {
	controller, router, _, user, _ := setupOrderControllerTest(t)

	router.GET("/orders/:id", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.GetOrderByID(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/orders/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Invalid order ID", response["error"])
}

func TestOrderController_CreateOrder_Success(t *testing.T) {
	controller, router, testDB, user, product := setupOrderControllerTest(t)

	// Add item to cart
	cartRepo := repository.NewCartRepository(testDB)
	cartRepo.Create(&model.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
	})

	router.POST("/orders", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.CreateOrder(c)
	})

	reqBody := CreateOrderRequest{
		ShippingAddress: "서울시 강남구 테헤란로 123",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Order created successfully", response["message"])

	orderData := response["order"].(map[string]interface{})
	assert.Equal(t, float64(200000), orderData["total_amount"]) // 100000 * 2
}

func TestOrderController_CreateOrder_Unauthorized(t *testing.T) {
	controller, router, _, _, _ := setupOrderControllerTest(t)

	router.POST("/orders", controller.CreateOrder)

	reqBody := CreateOrderRequest{
		ShippingAddress: "서울시 강남구",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "User not authenticated", response["error"])
}

func TestOrderController_CreateOrder_EmptyCart(t *testing.T) {
	controller, router, _, user, _ := setupOrderControllerTest(t)

	router.POST("/orders", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.CreateOrder(c)
	})

	reqBody := CreateOrderRequest{
		ShippingAddress: "서울시 강남구",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Cart is empty", response["error"])
}

func TestOrderController_CreateOrder_InsufficientStock(t *testing.T) {
	controller, router, testDB, user, product := setupOrderControllerTest(t)

	// Add item with quantity exceeding stock
	cartRepo := repository.NewCartRepository(testDB)
	cartRepo.Create(&model.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  100, // Exceeds stock
	})

	router.POST("/orders", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.CreateOrder(c)
	})

	reqBody := CreateOrderRequest{
		ShippingAddress: "서울시 강남구",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Insufficient stock for one or more items", response["error"])
}

func TestOrderController_CreateOrder_InvalidRequest(t *testing.T) {
	controller, router, _, user, _ := setupOrderControllerTest(t)

	router.POST("/orders", func(c *gin.Context) {
		setUserIDInContext(c, user.ID)
		controller.CreateOrder(c)
	})

	tests := []struct {
		name       string
		reqBody    map[string]interface{}
		wantStatus int
		wantError  string
	}{
		{
			name:       "Missing shipping address",
			reqBody:    map[string]interface{}{},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
		{
			name:       "Empty shipping address",
			reqBody:    map[string]interface{}{"shipping_address": ""},
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(jsonBody))
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

func TestOrderController_UpdateOrderStatus_Success(t *testing.T) {
	controller, router, testDB, user, _ := setupOrderControllerTest(t)

	// Create test order
	orderRepo := repository.NewOrderRepository(testDB)
	order := &model.Order{
		UserID:          user.ID,
		TotalAmount:     100000,
		Status:          model.OrderStatusPending,
		PaymentStatus:   model.PaymentStatusPending,
		ShippingAddress: "서울시 강남구",
	}
	orderRepo.Create(order)

	router.PUT("/orders/:id/status", controller.UpdateOrderStatus)

	reqBody := UpdateOrderStatusRequest{
		Status: model.OrderStatusConfirmed,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/orders/1/status", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Order status updated successfully", response["message"])

	// Verify status updated
	updatedOrder, _ := orderRepo.FindByID(1)
	assert.Equal(t, model.OrderStatusConfirmed, updatedOrder.Status)
}

func TestOrderController_UpdateOrderStatus_InvalidID(t *testing.T) {
	controller, router, _, _, _ := setupOrderControllerTest(t)

	router.PUT("/orders/:id/status", controller.UpdateOrderStatus)

	reqBody := UpdateOrderStatusRequest{
		Status: model.OrderStatusConfirmed,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/orders/invalid/status", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Invalid order ID", response["error"])
}

func TestOrderController_UpdateOrderStatus_InvalidRequest(t *testing.T) {
	controller, router, _, _, _ := setupOrderControllerTest(t)

	router.PUT("/orders/:id/status", controller.UpdateOrderStatus)

	reqBody := map[string]interface{}{}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/orders/1/status", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Invalid request data", response["error"])
}

func TestOrderController_UpdatePaymentStatus_Success(t *testing.T) {
	controller, router, testDB, user, _ := setupOrderControllerTest(t)

	// Create test order
	orderRepo := repository.NewOrderRepository(testDB)
	order := &model.Order{
		UserID:          user.ID,
		TotalAmount:     100000,
		Status:          model.OrderStatusPending,
		PaymentStatus:   model.PaymentStatusPending,
		ShippingAddress: "서울시 강남구",
	}
	orderRepo.Create(order)

	router.PUT("/orders/:id/payment", controller.UpdatePaymentStatus)

	reqBody := UpdatePaymentStatusRequest{
		Status: model.PaymentStatusCompleted,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/orders/1/payment", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Payment status updated successfully", response["message"])

	// Verify status updated
	updatedOrder, _ := orderRepo.FindByID(1)
	assert.Equal(t, model.PaymentStatusCompleted, updatedOrder.PaymentStatus)
}

func TestOrderController_UpdatePaymentStatus_InvalidID(t *testing.T) {
	controller, router, _, _, _ := setupOrderControllerTest(t)

	router.PUT("/orders/:id/payment", controller.UpdatePaymentStatus)

	reqBody := UpdatePaymentStatusRequest{
		Status: model.PaymentStatusCompleted,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/orders/invalid/payment", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Invalid order ID", response["error"])
}

func TestOrderController_UpdatePaymentStatus_InvalidRequest(t *testing.T) {
	controller, router, _, _, _ := setupOrderControllerTest(t)

	router.PUT("/orders/:id/payment", controller.UpdatePaymentStatus)

	reqBody := map[string]interface{}{}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/orders/1/payment", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Invalid request data", response["error"])
}
