package router

import (
	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/config"
	"github.com/ikkim/udonggeum-backend/internal/app/controller"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
)

type Router struct {
	authController    *controller.AuthController
	productController *controller.ProductController
	cartController    *controller.CartController
	orderController   *controller.OrderController
	authMiddleware    *middleware.AuthMiddleware
	config            *config.Config
}

func NewRouter(
	authController *controller.AuthController,
	productController *controller.ProductController,
	cartController *controller.CartController,
	orderController *controller.OrderController,
	authMiddleware *middleware.AuthMiddleware,
	cfg *config.Config,
) *Router {
	return &Router{
		authController:    authController,
		productController: productController,
		cartController:    cartController,
		orderController:   orderController,
		authMiddleware:    authMiddleware,
		config:            cfg,
	}
}

func (r *Router) Setup() *gin.Engine {
	// Set Gin mode
	gin.SetMode(r.config.Server.GinMode)

	// Create router without default middleware
	router := gin.New()

	// Recovery middleware - recovers from panics
	router.Use(gin.Recovery())

	// Custom logging middleware
	router.Use(middleware.LoggingMiddleware())

	// CORS middleware
	router.Use(corsMiddleware(r.config.CORS.AllowedOrigins))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"message": "UDONGGEUM API is running",
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", r.authController.Register)
			auth.POST("/login", r.authController.Login)
			auth.GET("/me", r.authMiddleware.Authenticate(), r.authController.GetMe)
		}

		// Product routes
		products := v1.Group("/products")
		{
			products.GET("", r.productController.GetAllProducts)
			products.GET("/:id", r.productController.GetProductByID)

			// Admin only routes
			products.POST("",
				r.authMiddleware.Authenticate(),
				r.authMiddleware.RequireRole("admin"),
				r.productController.CreateProduct,
			)
			products.PUT("/:id",
				r.authMiddleware.Authenticate(),
				r.authMiddleware.RequireRole("admin"),
				r.productController.UpdateProduct,
			)
			products.DELETE("/:id",
				r.authMiddleware.Authenticate(),
				r.authMiddleware.RequireRole("admin"),
				r.productController.DeleteProduct,
			)
		}

		// Cart routes (protected)
		cart := v1.Group("/cart")
		cart.Use(r.authMiddleware.Authenticate())
		{
			cart.GET("", r.cartController.GetCart)
			cart.POST("", r.cartController.AddToCart)
			cart.PUT("/:id", r.cartController.UpdateCartItem)
			cart.DELETE("/:id", r.cartController.RemoveFromCart)
			cart.DELETE("", r.cartController.ClearCart)
		}

		// Order routes (protected)
		orders := v1.Group("/orders")
		orders.Use(r.authMiddleware.Authenticate())
		{
			orders.GET("", r.orderController.GetOrders)
			orders.GET("/:id", r.orderController.GetOrderByID)
			orders.POST("", r.orderController.CreateOrder)

			// Admin only routes
			orders.PUT("/:id/status",
				r.authMiddleware.RequireRole("admin"),
				r.orderController.UpdateOrderStatus,
			)
			orders.PUT("/:id/payment", r.orderController.UpdatePaymentStatus)
		}
	}

	return router
}

// corsMiddleware handles CORS
func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin || allowedOrigin == "*" {
				allowed = true
				break
			}
		}

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
