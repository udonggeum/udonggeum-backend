package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/errors"
	"github.com/ikkim/udonggeum-backend/pkg/util"
)

// Context keys for user information
const (
	UserIDKey    = "user_id"
	UserEmailKey = "user_email"
	UserRoleKey  = "user_role"
)

type AuthMiddleware struct {
	jwtSecret string
}

func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: jwtSecret,
	}
}

// Authenticate validates JWT token (required)
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		log := GetLoggerFromContext(c)

		var token string

		// Try to get token from Authorization header first
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				log.Warn("Invalid authorization header format", map[string]interface{}{
					"path": c.Request.URL.Path,
				})
				errors.RespondWithError(c, http.StatusUnauthorized, errors.AuthTokenInvalid, "인증 형식이 올바르지 않습니다")
				c.Abort()
				return
			}
			token = parts[1]
		} else {
			// If no Authorization header, try to get token from query parameter (for WebSocket)
			token = c.Query("token")
			if token == "" {
				log.Warn("Missing authorization header", map[string]interface{}{
					"path": c.Request.URL.Path,
				})
				errors.Unauthorized(c, "로그인이 필요합니다")
				c.Abort()
				return
			}
			log.Debug("Using token from query parameter", map[string]interface{}{
				"path": c.Request.URL.Path,
			})
		}

		claims, err := util.ValidateToken(token, m.jwtSecret)
		if err != nil {
			log.Warn("Token validation failed", map[string]interface{}{
				"path":  c.Request.URL.Path,
				"error": err.Error(),
			})

			// 토큰 만료 에러인 경우 명확히 표시
			if err == util.ErrExpiredToken {
				errors.RespondWithError(c, http.StatusUnauthorized, errors.AuthTokenExpired, "로그인이 만료되었습니다")
			} else {
				errors.RespondWithError(c, http.StatusUnauthorized, errors.AuthTokenInvalid, "유효하지 않은 인증 토큰입니다")
			}
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", model.UserRole(claims.Role))

		log.Debug("User authenticated successfully", map[string]interface{}{
			"user_id": claims.UserID,
			"email":   claims.Email,
			"role":    claims.Role,
		})

		c.Next()
	}
}

// OptionalAuthenticate validates JWT token if present (optional)
// - If token is present and valid: sets user info in context
// - If token is missing or invalid: continues without user info
func (m *AuthMiddleware) OptionalAuthenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		log := GetLoggerFromContext(c)

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No token provided - continue as guest
			log.Debug("No authorization header - continuing as guest", map[string]interface{}{
				"path": c.Request.URL.Path,
			})
			c.Next()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			// Invalid format - continue as guest
			log.Debug("Invalid authorization header format - continuing as guest", map[string]interface{}{
				"path": c.Request.URL.Path,
			})
			c.Next()
			return
		}

		token := parts[1]
		claims, err := util.ValidateToken(token, m.jwtSecret)
		if err != nil {
			// Invalid or expired token - continue as guest
			log.Debug("Token validation failed - continuing as guest", map[string]interface{}{
				"path":  c.Request.URL.Path,
				"error": err.Error(),
			})
			c.Next()
			return
		}

		// Valid token - set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", model.UserRole(claims.Role))

		log.Debug("User authenticated successfully (optional)", map[string]interface{}{
			"user_id": claims.UserID,
			"email":   claims.Email,
			"role":    claims.Role,
		})

		c.Next()
	}
}

// RequireRole checks if user has required role
func (m *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := GetLoggerFromContext(c)

		userRole, exists := c.Get("user_role")
		if !exists {
			log.Warn("Role information not found in context", map[string]interface{}{
				"path": c.Request.URL.Path,
			})
			errors.RespondWithError(c, http.StatusForbidden, errors.AuthzRoleNotFound, "권한 정보를 찾을 수 없습니다")
			c.Abort()
			return
		}

		role := userRole.(model.UserRole)
		userID, _ := GetUserID(c)

		for _, r := range roles {
			if role == model.UserRole(r) {
				log.Debug("Role check passed", map[string]interface{}{
					"user_id":       userID,
					"user_role":     role,
					"required_role": r,
				})
				c.Next()
				return
			}
		}

		log.Warn("Insufficient permissions", map[string]interface{}{
			"user_id":        userID,
			"user_role":      role,
			"required_roles": roles,
			"path":           c.Request.URL.Path,
		})
		errors.Forbidden(c, "접근 권한이 없습니다")
		c.Abort()
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	return userID.(uint), true
}

// GetUserEmail extracts user email from context
func GetUserEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get("user_email")
	if !exists {
		return "", false
	}
	return email.(string), true
}

// GetUserRole extracts user role from context
func GetUserRole(c *gin.Context) (model.UserRole, bool) {
	role, exists := c.Get("user_role")
	if !exists {
		return "", false
	}
	return role.(model.UserRole), true
}
