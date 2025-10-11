package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/pkg/util"
)

type AuthMiddleware struct {
	jwtSecret string
}

func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: jwtSecret,
	}
}

// Authenticate validates JWT token
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		log := GetLoggerFromContext(c)

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Warn("Missing authorization header", map[string]interface{}{
				"path": c.Request.URL.Path,
			})
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Warn("Invalid authorization header format", map[string]interface{}{
				"path": c.Request.URL.Path,
			})
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := util.ValidateToken(token, m.jwtSecret)
		if err != nil {
			log.Warn("Token validation failed", map[string]interface{}{
				"path":  c.Request.URL.Path,
				"error": err.Error(),
			})
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)

		log.Debug("User authenticated successfully", map[string]interface{}{
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
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Role information not found",
			})
			c.Abort()
			return
		}

		role := userRole.(string)
		userID, _ := GetUserID(c)

		for _, r := range roles {
			if role == r {
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
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Insufficient permissions",
		})
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
func GetUserRole(c *gin.Context) (string, bool) {
	role, exists := c.Get("user_role")
	if !exists {
		return "", false
	}
	return role.(string), true
}
