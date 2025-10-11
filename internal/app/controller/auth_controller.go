package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
)

type AuthController struct {
	authService service.AuthService
}

func NewAuthController(authService service.AuthService) *AuthController {
	return &AuthController{
		authService: authService,
	}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required"`
	Phone    string `json:"phone"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Register handles user registration
// POST /api/v1/auth/register
func (ctrl *AuthController) Register(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid registration request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Processing registration", map[string]interface{}{
		"email": req.Email,
		"name":  req.Name,
	})

	user, tokens, err := ctrl.authService.Register(req.Email, req.Password, req.Name, req.Phone)
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			log.Warn("Registration failed: email already exists", map[string]interface{}{
				"email": req.Email,
			})
			c.JSON(http.StatusConflict, gin.H{
				"error": "Email already exists",
			})
			return
		}
		log.Error("Registration failed", err, map[string]interface{}{
			"email": req.Email,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to register user",
		})
		return
	}

	log.Info("User registered successfully", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"phone": user.Phone,
			"role":  user.Role,
		},
		"tokens": tokens,
	})
}

// Login handles user login
// POST /api/v1/auth/login
func (ctrl *AuthController) Login(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid login request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Processing login", map[string]interface{}{
		"email": req.Email,
	})

	user, tokens, err := ctrl.authService.Login(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			log.Warn("Login failed: invalid credentials", map[string]interface{}{
				"email": req.Email,
			})
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid email or password",
			})
			return
		}
		log.Error("Login failed", err, map[string]interface{}{
			"email": req.Email,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to login",
		})
		return
	}

	log.Info("Login successful", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"phone": user.Phone,
			"role":  user.Role,
		},
		"tokens": tokens,
	})
}

// GetMe returns current user information
// GET /api/v1/auth/me
func (ctrl *AuthController) GetMe(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized access to GetMe endpoint", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	user, err := ctrl.authService.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("User not found", map[string]interface{}{
				"user_id": userID,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "User not found",
			})
			return
		}
		log.Error("Failed to get user information", err, map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user information",
		})
		return
	}

	log.Info("User information retrieved", map[string]interface{}{
		"user_id": user.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"phone": user.Phone,
			"role":  user.Role,
		},
	})
}
