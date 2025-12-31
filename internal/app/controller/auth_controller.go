package controller

import (
	"errors"
	"net/http"
	"strings"

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
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	Nickname     string `json:"nickname"`
	Address      string `json:"address"`
	ProfileImage string `json:"profile_image"` // S3 URL from upload API
}

type CheckNicknameRequest struct {
	Nickname string `json:"nickname" binding:"required,min=2,max=20"`
}

type CheckEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
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
			"id":             user.ID,
			"email":          user.Email,
			"name":           user.Name,
			"nickname":       user.Nickname,
			"phone":          user.Phone,
			"phone_verified": user.PhoneVerified,
			"address":        user.Address,
			"role":           user.Role,
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
			"id":             user.ID,
			"email":          user.Email,
			"name":           user.Name,
			"nickname":       user.Nickname,
			"phone":          user.Phone,
			"phone_verified": user.PhoneVerified,
			"address":        user.Address,
			"role":           user.Role,
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
			"id":             user.ID,
			"email":          user.Email,
			"name":           user.Name,
			"nickname":       user.Nickname,
			"phone":          user.Phone,
			"phone_verified": user.PhoneVerified,
			"address":        user.Address,
			"profile_image":  user.ProfileImage,
			"role":           user.Role,
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
		"user_id":       userID,
		"name":          req.Name,
		"nickname":      req.Nickname,
		"profile_image": req.ProfileImage,
	})

	user, err := ctrl.authService.UpdateProfile(userID, req.Name, req.Phone, req.Nickname, req.Address, req.ProfileImage)
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
		if errors.Is(err, service.ErrNicknameAlreadyExists) {
			log.Warn("Nickname already exists", map[string]interface{}{
				"user_id":  userID,
				"nickname": req.Nickname,
			})
			c.JSON(http.StatusConflict, gin.H{
				"error": "Nickname already exists",
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
			"id":             user.ID,
			"email":          user.Email,
			"name":           user.Name,
			"nickname":       user.Nickname,
			"phone":          user.Phone,
			"phone_verified": user.PhoneVerified,
			"address":        user.Address,
			"profile_image":  user.ProfileImage,
			"role":           user.Role,
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

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Logout handles user logout
// POST /api/v1/auth/logout
func (ctrl *AuthController) Logout(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid logout request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	userID, exists := middleware.GetUserID(c)
	if exists {
		log.Info("User logged out", map[string]interface{}{
			"user_id": userID,
		})
	} else {
		log.Debug("Logout called without authenticated user")
	}

	// Revoke the refresh token by adding it to blacklist
	if err := ctrl.authService.RevokeToken(req.RefreshToken); err != nil {
		log.Error("Failed to revoke token during logout", err, nil)
		// Don't fail the request, logout should always succeed from user perspective
	}

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
		if errors.Is(err, service.ErrInvalidToken) || errors.Is(err, service.ErrExpiredToken) || errors.Is(err, service.ErrTokenRevoked) {
			log.Warn("Token refresh failed: invalid, expired, or revoked token", map[string]interface{}{
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

// CheckNickname checks if a nickname is available
// POST /api/v1/auth/check-nickname
func (ctrl *AuthController) CheckNickname(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	var req CheckNicknameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid check nickname request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Checking nickname availability", map[string]interface{}{
		"nickname": req.Nickname,
	})

	isAvailable, err := ctrl.authService.CheckNickname(req.Nickname)
	if err != nil {
		log.Error("Failed to check nickname availability", err, map[string]interface{}{
			"nickname": req.Nickname,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check nickname availability",
		})
		return
	}

	log.Info("Nickname availability checked", map[string]interface{}{
		"nickname":    req.Nickname,
		"is_available": isAvailable,
	})

	c.JSON(http.StatusOK, gin.H{
		"is_available": isAvailable,
	})
}

// CheckEmailAvailability checks if an email is available for registration
// POST /api/v1/auth/check-email
func (ctrl *AuthController) CheckEmailAvailability(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	var req CheckEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid check email request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Checking email availability", map[string]interface{}{
		"email": req.Email,
	})

	isAvailable, err := ctrl.authService.CheckEmailAvailability(req.Email)
	if err != nil {
		log.Error("Failed to check email availability", err, map[string]interface{}{
			"email": req.Email,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check email availability",
		})
		return
	}

	log.Info("Email availability checked", map[string]interface{}{
		"email":        req.Email,
		"is_available": isAvailable,
	})

	c.JSON(http.StatusOK, gin.H{
		"is_available": isAvailable,
	})
}

// GetKakaoLoginURL returns the Kakao OAuth login URL
// GET /api/v1/auth/kakao/login
func (ctrl *AuthController) GetKakaoLoginURL(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	log.Debug("Generating Kakao login URL")

	loginURL := ctrl.authService.GetKakaoLoginURL()

	log.Info("Kakao login URL generated", map[string]interface{}{
		"url": loginURL,
	})

	c.JSON(http.StatusOK, gin.H{
		"login_url": loginURL,
	})
}

// KakaoCallback handles Kakao OAuth callback
// GET /api/v1/auth/kakao/callback
func (ctrl *AuthController) KakaoCallback(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	code := c.Query("code")
	if code == "" {
		log.Warn("Kakao callback without authorization code", nil)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Authorization code is required",
		})
		return
	}

	log.Debug("Processing Kakao callback", map[string]interface{}{
		"code": code,
	})

	user, tokens, err := ctrl.authService.KakaoLogin(code)
	if err != nil {
		log.Error("Kakao login failed", err, map[string]interface{}{
			"code": code,
		})

		// Provide more specific error messages
		errorMsg := "Failed to login with Kakao"
		statusCode := http.StatusInternalServerError

		errStr := err.Error()
		if errors.Is(err, service.ErrUserNotFound) ||
		   strings.Contains(errStr, "email consent required") ||
		   strings.Contains(errStr, "missing email") {
			errorMsg = "Email consent is required for Kakao login"
			statusCode = http.StatusBadRequest
		} else if strings.Contains(errStr, "status 401") ||
		          strings.Contains(errStr, "status 400") {
			errorMsg = "Invalid Kakao authorization - please try again"
			statusCode = http.StatusUnauthorized
		}

		c.JSON(statusCode, gin.H{
			"error": errorMsg,
		})
		return
	}

	log.Info("Kakao login successful", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Kakao login successful",
		"user": gin.H{
			"id":             user.ID,
			"email":          user.Email,
			"name":           user.Name,
			"nickname":       user.Nickname,
			"phone":          user.Phone,
			"phone_verified": user.PhoneVerified,
			"profile_image":  user.ProfileImage,
			"role":           user.Role,
		},
		"tokens": tokens,
	})
}

// === 이메일/휴대폰 인증 관련 API ===

type SendEmailVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

type SendPhoneVerificationRequest struct {
	Phone string `json:"phone" binding:"required"`
}

type VerifyPhoneRequest struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required,len=6"`
}

// SendEmailVerification sends email verification code
// POST /api/v1/auth/send-email-verification
func (ctrl *AuthController) SendEmailVerification(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	var req SendEmailVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
		})
		return
	}

	err := ctrl.authService.SendEmailVerification(req.Email)
	if err != nil {
		log.Error("Failed to send email verification", err, map[string]interface{}{
			"email": req.Email,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to send verification email",
		})
		return
	}

	log.Info("Email verification sent", map[string]interface{}{
		"email": req.Email,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Verification code sent to your email",
	})
}

// VerifyEmail verifies email with code
// POST /api/v1/auth/verify-email
func (ctrl *AuthController) VerifyEmail(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
		})
		return
	}

	err := ctrl.authService.VerifyEmail(req.Email, req.Code)
	if err != nil {
		if errors.Is(err, service.ErrInvalidVerificationCode) {
			log.Warn("Invalid verification code", map[string]interface{}{
				"email": req.Email,
			})
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid or expired verification code",
			})
			return
		}

		log.Error("Email verification failed", err, map[string]interface{}{
			"email": req.Email,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify email",
		})
		return
	}

	log.Info("Email verified successfully", map[string]interface{}{
		"email": req.Email,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Email verified successfully",
	})
}

// SendPhoneVerification sends phone verification code
// POST /api/v1/auth/send-phone-verification
func (ctrl *AuthController) SendPhoneVerification(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	var req SendPhoneVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
		})
		return
	}

	// Get user ID from context (must be authenticated)
	userID, exists := c.Get("user_id")
	if !exists {
		log.Warn("User not authenticated", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	err := ctrl.authService.SendPhoneVerification(userID.(uint), req.Phone)
	if err != nil {
		log.Error("Failed to send phone verification", err, map[string]interface{}{
			"phone": req.Phone,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to send verification SMS",
		})
		return
	}

	log.Info("Phone verification sent", map[string]interface{}{
		"phone": req.Phone,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Verification code sent to your phone",
	})
}

// VerifyPhone verifies phone with code
// POST /api/v1/auth/verify-phone
func (ctrl *AuthController) VerifyPhone(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	var req VerifyPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
		})
		return
	}

	// Get user ID from context (must be authenticated)
	userID, exists := c.Get("user_id")
	if !exists {
		log.Warn("User not authenticated", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	err := ctrl.authService.VerifyPhone(userID.(uint), req.Phone, req.Code)
	if err != nil {
		if errors.Is(err, service.ErrInvalidVerificationCode) {
			log.Warn("Invalid verification code", map[string]interface{}{
				"phone": req.Phone,
			})
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid or expired verification code",
			})
			return
		}

		log.Error("Phone verification failed", err, map[string]interface{}{
			"phone": req.Phone,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify phone",
		})
		return
	}

	log.Info("Phone verified successfully", map[string]interface{}{
		"phone": req.Phone,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Phone verified successfully",
	})
}
