package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testJWTSecret = "test-jwt-secret-for-middleware"

func setupMiddlewareTest() (*gin.Engine, *AuthMiddleware) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	middleware := NewAuthMiddleware(testJWTSecret)
	return router, middleware
}

func generateTestToken(t *testing.T, userID uint, email, role string) string {
	tokens, err := util.GenerateTokenPair(
		userID,
		email,
		role,
		testJWTSecret,
		15*time.Minute,
		7*24*time.Hour,
	)
	require.NoError(t, err)
	return tokens.AccessToken
}

func TestAuthMiddleware_Authenticate_Success(t *testing.T) {
	router, authMiddleware := setupMiddlewareTest()

	// Generate valid token
	token := generateTestToken(t, 1, "test@example.com", "user")

	router.GET("/test", authMiddleware.Authenticate(), func(c *gin.Context) {
		userID, _ := GetUserID(c)
		email, _ := GetUserEmail(c)
		role, _ := GetUserRole(c)

		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"email":   email,
			"role":    role,
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_Authenticate_NoToken(t *testing.T) {
	router, authMiddleware := setupMiddlewareTest()

	router.GET("/test", authMiddleware.Authenticate(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authorization header is required")
}

func TestAuthMiddleware_Authenticate_InvalidFormat(t *testing.T) {
	router, authMiddleware := setupMiddlewareTest()

	router.GET("/test", authMiddleware.Authenticate(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "Missing Bearer prefix",
			header: "invalid-token",
		},
		{
			name:   "Wrong prefix",
			header: "Basic token123",
		},
		{
			name:   "Empty token",
			header: "Bearer ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tt.header)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestAuthMiddleware_Authenticate_InvalidToken(t *testing.T) {
	router, authMiddleware := setupMiddlewareTest()

	router.GET("/test", authMiddleware.Authenticate(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid or expired token")
}

func TestAuthMiddleware_RequireRole_Success(t *testing.T) {
	router, authMiddleware := setupMiddlewareTest()

	// Generate admin token
	token := generateTestToken(t, 1, "admin@example.com", "admin")

	router.GET("/admin",
		authMiddleware.Authenticate(),
		authMiddleware.RequireRole("admin"),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
		},
	)

	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_RequireRole_Forbidden(t *testing.T) {
	router, authMiddleware := setupMiddlewareTest()

	// Generate user token (not admin)
	token := generateTestToken(t, 1, "user@example.com", "user")

	router.GET("/admin",
		authMiddleware.Authenticate(),
		authMiddleware.RequireRole("admin"),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
		},
	)

	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Insufficient permissions")
}

func TestAuthMiddleware_RequireRole_MultipleRoles(t *testing.T) {
	router, authMiddleware := setupMiddlewareTest()

	router.GET("/moderator",
		authMiddleware.Authenticate(),
		authMiddleware.RequireRole("admin", "moderator"),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "access granted"})
		},
	)

	tests := []struct {
		name           string
		role           string
		expectedStatus int
	}{
		{
			name:           "Admin role",
			role:           "admin",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Moderator role",
			role:           "moderator",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "User role",
			role:           "user",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := generateTestToken(t, 1, "test@example.com", tt.role)

			req := httptest.NewRequest("GET", "/moderator", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Without setting user_id
	userID, exists := GetUserID(c)
	assert.False(t, exists)
	assert.Equal(t, uint(0), userID)

	// After setting user_id
	c.Set("user_id", uint(123))
	userID, exists = GetUserID(c)
	assert.True(t, exists)
	assert.Equal(t, uint(123), userID)
}

func TestGetUserEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Without setting user_email
	email, exists := GetUserEmail(c)
	assert.False(t, exists)
	assert.Empty(t, email)

	// After setting user_email
	c.Set("user_email", "test@example.com")
	email, exists = GetUserEmail(c)
	assert.True(t, exists)
	assert.Equal(t, "test@example.com", email)
}

func TestGetUserRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Without setting user_role
	role, exists := GetUserRole(c)
	assert.False(t, exists)
	assert.Empty(t, role)

	// After setting user_role
	c.Set("user_role", "admin")
	role, exists = GetUserRole(c)
	assert.True(t, exists)
	assert.Equal(t, "admin", role)
}
