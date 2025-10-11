package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
)

// LoggingMiddleware logs HTTP requests with structured logging
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		startTime := time.Now()

		// Get request ID (if exists from context)
		requestID := c.GetString("request_id")
		if requestID == "" {
			requestID = generateRequestID()
			c.Set("request_id", requestID)
		}

		// Create logger with request context
		log := logger.WithContext(map[string]interface{}{
			"request_id": requestID,
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"ip":         c.ClientIP(),
		})

		// Log incoming request
		log.Info("Incoming request", map[string]interface{}{
			"user_agent": c.Request.UserAgent(),
			"query":      c.Request.URL.RawQuery,
		})

		// Store logger in context for use in handlers
		c.Set("logger", log)

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)
		statusCode := c.Writer.Status()

		// Determine log level based on status code
		fields := map[string]interface{}{
			"status_code": statusCode,
			"latency_ms":  latency.Milliseconds(),
			"latency":     latency.String(),
			"body_size":   c.Writer.Size(),
		}

		// Add error if exists
		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		// Log based on status code
		msg := "Request completed"
		if statusCode >= 500 {
			log.Error(msg, nil, fields)
		} else if statusCode >= 400 {
			log.Warn(msg, fields)
		} else {
			log.Info(msg, fields)
		}
	}
}

// generateRequestID generates a simple request ID
// In production, consider using UUID or similar
func generateRequestID() string {
	return time.Now().Format("20060102150405.000")
}

// GetLoggerFromContext retrieves the logger from gin context
func GetLoggerFromContext(c *gin.Context) *logger.Logger {
	if log, exists := c.Get("logger"); exists {
		if l, ok := log.(*logger.Logger); ok {
			return l
		}
	}
	// Return global logger as fallback
	return logger.Get()
}
