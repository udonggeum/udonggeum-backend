package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/db"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
	"github.com/ikkim/udonggeum-backend/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuthControllerTest(t *testing.T) (*gin.Engine, *AuthController, service.AuthService) {
	gin.SetMode(gin.TestMode)

	testDB, err := db.SetupTestDB(t)
	require.NoError(t, err)

	userRepo := repository.NewUserRepository(testDB)
	passwordResetRepo := repository.NewPasswordResetRepository(testDB)
	authService := service.NewAuthService(
		userRepo,
		"test-secret",
		15*time.Minute,
		7*24*time.Hour,
		"test-kakao-client-id",
		"test-kakao-client-secret",
		"http://localhost:8080/api/v1/auth/kakao/callback",
	)
	passwordResetService := service.NewPasswordResetService(passwordResetRepo, userRepo)

	ctrl := NewAuthController(authService, passwordResetService)
	authMiddleware := middleware.NewAuthMiddleware("test-secret")

	router := gin.New()
	router.POST("/register", ctrl.Register)
	router.POST("/login", ctrl.Login)
	router.GET("/me", authMiddleware.Authenticate(), ctrl.GetMe)

	return router, ctrl, authService
}

func TestAuthController_Register_Success(t *testing.T) {
	router, _, _ := setupAuthControllerTest(t)

	reqBody := RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
		Phone:    "010-1234-5678",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "User registered successfully", response["message"])
	assert.NotNil(t, response["user"])
	assert.NotNil(t, response["tokens"])
}

func TestAuthController_Register_InvalidEmail(t *testing.T) {
	router, _, _ := setupAuthControllerTest(t)

	reqBody := RegisterRequest{
		Email:    "invalid-email",
		Password: "password123",
		Name:     "Test User",
		Phone:    "010-1234-5678",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthController_Register_DuplicateEmail(t *testing.T) {
	router, _, authService := setupAuthControllerTest(t)

	// Register first user
	_, _, err := authService.Register("test@example.com", "password123", "Test User", "010-1234-5678")
	require.NoError(t, err)

	// Try to register with same email
	reqBody := RegisterRequest{
		Email:    "test@example.com",
		Password: "password456",
		Name:     "Another User",
		Phone:    "010-8765-4321",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "Email already exists")
}

func TestAuthController_Login_Success(t *testing.T) {
	router, _, authService := setupAuthControllerTest(t)

	// Register a user first
	email := "test@example.com"
	password := "password123"
	_, _, err := authService.Register(email, password, "Test User", "010-1234-5678")
	require.NoError(t, err)

	// Login
	reqBody := LoginRequest{
		Email:    email,
		Password: password,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Login successful", response["message"])
	assert.NotNil(t, response["user"])
	assert.NotNil(t, response["tokens"])
}

func TestAuthController_Login_WrongPassword(t *testing.T) {
	router, _, authService := setupAuthControllerTest(t)

	// Register a user
	_, _, err := authService.Register("test@example.com", "password123", "Test User", "010-1234-5678")
	require.NoError(t, err)

	// Login with wrong password
	reqBody := LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid email or password")
}

func TestAuthController_GetMe_Success(t *testing.T) {
	router, _, authService := setupAuthControllerTest(t)

	// Register and get token
	user, tokens, err := authService.Register("test@example.com", "password123", "Test User", "010-1234-5678")
	require.NoError(t, err)

	// Get user info
	req := httptest.NewRequest("GET", "/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	userMap := response["user"].(map[string]interface{})
	assert.Equal(t, user.Email, userMap["email"])
	assert.Equal(t, user.Name, userMap["name"])
}

func TestAuthController_GetMe_Unauthorized(t *testing.T) {
	router, _, _ := setupAuthControllerTest(t)

	req := httptest.NewRequest("GET", "/me", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthController_GetMe_InvalidToken(t *testing.T) {
	router, _, _ := setupAuthControllerTest(t)

	req := httptest.NewRequest("GET", "/me", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthController_Register_MissingFields(t *testing.T) {
	router, _, _ := setupAuthControllerTest(t)

	tests := []struct {
		name    string
		reqBody RegisterRequest
	}{
		{
			name: "Missing email",
			reqBody: RegisterRequest{
				Password: "password123",
				Name:     "Test User",
			},
		},
		{
			name: "Missing password",
			reqBody: RegisterRequest{
				Email: "test@example.com",
				Name:  "Test User",
			},
		},
		{
			name: "Missing name",
			reqBody: RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
		},
		{
			name: "Short password",
			reqBody: RegisterRequest{
				Email:    "test@example.com",
				Password: "123",
				Name:     "Test User",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestAuthController_TokensAreDifferent(t *testing.T) {
	router, _, _ := setupAuthControllerTest(t)

	reqBody := RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
		Phone:    "010-1234-5678",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	tokens := response["tokens"].(map[string]interface{})
	accessToken := tokens["access_token"].(string)
	refreshToken := tokens["refresh_token"].(string)

	assert.NotEqual(t, accessToken, refreshToken)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)

	// Validate tokens
	claims, err := util.ValidateToken(accessToken, "test-secret")
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", claims.Email)
}
