package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
)

type AuthController struct {
	authService          service.AuthService
	passwordResetService service.PasswordResetService
}

func NewAuthController(authService service.AuthService, passwordResetService service.PasswordResetService) *AuthController {
	return &AuthController{
		authService:          authService,
		passwordResetService: passwordResetService,
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

type UpdateProfileRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
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

// UpdateMe updates current user's profile
// PUT /api/v1/auth/me
func (ctrl *AuthController) UpdateMe(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Unauthorized access to UpdateMe endpoint", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid update profile request", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Processing profile update", map[string]interface{}{
		"user_id": userID,
		"name":    req.Name,
	})

	user, err := ctrl.authService.UpdateProfile(userID, req.Name, req.Phone)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("User not found for profile update", map[string]interface{}{
				"user_id": userID,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "User not found",
			})
			return
		}
		log.Error("Failed to update user profile", err, map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update user profile",
		})
		return
	}

	log.Info("User profile updated successfully", map[string]interface{}{
		"user_id": user.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"phone": user.Phone,
			"role":  user.Role,
		},
	})
}

// ForgotPassword handles password reset requests
// POST /api/v1/auth/forgot-password
func (ctrl *AuthController) ForgotPassword(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid forgot password request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Processing forgot password request", map[string]interface{}{
		"email": req.Email,
	})

	if err := ctrl.passwordResetService.RequestReset(req.Email); err != nil {
		log.Error("Failed to process password reset request", err, map[string]interface{}{
			"email": req.Email,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process password reset request",
		})
		return
	}

	// Always return success to prevent user enumeration
	log.Info("Password reset request processed", map[string]interface{}{
		"email": req.Email,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "If the email exists, a password reset link has been sent",
	})
}

// ResetPassword handles password reset with token
// POST /api/v1/auth/reset-password
func (ctrl *AuthController) ResetPassword(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid reset password request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Processing password reset with token")

	if err := ctrl.passwordResetService.ResetPassword(req.Token, req.NewPassword); err != nil {
		if errors.Is(err, service.ErrInvalidResetToken) ||
			errors.Is(err, service.ErrResetTokenExpired) ||
			errors.Is(err, service.ErrResetTokenUsed) {
			log.Warn("Password reset failed: invalid or expired token", map[string]interface{}{
				"error": err.Error(),
			})
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		log.Error("Failed to reset password", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to reset password",
		})
		return
	}

	log.Info("Password reset successful")

	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset successful",
	})
}

// Logout handles user logout
// POST /api/v1/auth/logout
// Note: Since we're using stateless JWT, logout is handled client-side
// This endpoint is provided for consistency and future token blacklisting
func (ctrl *AuthController) Logout(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if exists {
		log.Info("User logged out", map[string]interface{}{
			"user_id": userID,
		})
	} else {
		log.Debug("Logout called without authenticated user")
	}

	// In a stateless JWT system, logout is primarily handled client-side
	// by removing the tokens from storage. This endpoint is provided for:
	// 1. Logging/auditing purposes
	// 2. Future implementation of token blacklisting
	// 3. API consistency with frontend expectations

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// RefreshToken handles token refresh
// POST /api/v1/auth/refresh
func (ctrl *AuthController) RefreshToken(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid refresh token request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Processing token refresh")

	tokens, err := ctrl.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) || errors.Is(err, service.ErrExpiredToken) {
			log.Warn("Token refresh failed: invalid or expired token", map[string]interface{}{
				"error": err.Error(),
			})
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired refresh token",
			})
			return
		}
		log.Error("Failed to refresh token", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to refresh token",
		})
		return
	}

	log.Info("Token refreshed successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Token refreshed successfully",
		"tokens":  tokens,
	})
}
