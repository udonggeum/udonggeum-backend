package router

import (
	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/config"
	"github.com/ikkim/udonggeum-backend/internal/app/controller"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
)

type Router struct {
	authController     *controller.AuthController
	storeController    *controller.StoreController
	productController  *controller.ProductController
	cartController     *controller.CartController
	orderController    *controller.OrderController
	wishlistController *controller.WishlistController
	addressController  *controller.AddressController
	sellerController   *controller.SellerController
	uploadController   *controller.UploadController
	authMiddleware     *middleware.AuthMiddleware
	config             *config.Config
}

func NewRouter(
	authController *controller.AuthController,
	storeController *controller.StoreController,
	productController *controller.ProductController,
	cartController *controller.CartController,
	orderController *controller.OrderController,
	wishlistController *controller.WishlistController,
	addressController *controller.AddressController,
	sellerController *controller.SellerController,
	uploadController *controller.UploadController,
	authMiddleware *middleware.AuthMiddleware,
	cfg *config.Config,
) *Router {
	return &Router{
		authController:     authController,
		storeController:    storeController,
		productController:  productController,
		cartController:     cartController,
		orderController:    orderController,
		wishlistController: wishlistController,
		addressController:  addressController,
		sellerController:   sellerController,
		uploadController:   uploadController,
		authMiddleware:     authMiddleware,
		config:             cfg,
	}
}

func (r *Router) Setup() *gin.Engine {
	gin.SetMode(r.config.Server.GinMode)

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(middleware.LoggingMiddleware())
	router.Use(corsMiddleware(r.config.CORS.AllowedOrigins))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"message": "UDONGGEUM API is running",
		})
	})

	// Serve static files from uploads directory
	router.Static("/uploads", "./uploads")

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", r.authController.Register)
			auth.POST("/login", r.authController.Login)
			auth.POST("/forgot-password", r.authController.ForgotPassword)
			auth.POST("/reset-password", r.authController.ResetPassword)
			auth.GET("/me", r.authMiddleware.Authenticate(), r.authController.GetMe)
			auth.PUT("/me", r.authMiddleware.Authenticate(), r.authController.UpdateMe)
		}

		stores := v1.Group("/stores")
		{
			stores.GET("", r.storeController.ListStores)
			stores.GET("/locations", r.storeController.ListLocations)
			stores.GET("/:id", r.storeController.GetStoreByID)
			stores.POST("",
				r.authMiddleware.Authenticate(),
				r.authMiddleware.RequireRole("seller", "admin"),
				r.storeController.CreateStore,
			)
			stores.PUT("/:id",
				r.authMiddleware.Authenticate(),
				r.authMiddleware.RequireRole("seller", "admin"),
				r.storeController.UpdateStore,
			)
			stores.DELETE("/:id",
				r.authMiddleware.Authenticate(),
				r.authMiddleware.RequireRole("seller", "admin"),
				r.storeController.DeleteStore,
			)
		}

		products := v1.Group("/products")
		{
			products.GET("", r.productController.GetAllProducts)
			products.GET("/filters", r.productController.GetProductFilters)
			products.GET("/popular", r.productController.GetPopularProducts)
			products.GET("/:id", r.productController.GetProductByID)

			products.POST("",
				r.authMiddleware.Authenticate(),
				r.authMiddleware.RequireRole("seller", "admin"),
				r.productController.CreateProduct,
			)
			products.PUT("/:id",
				r.authMiddleware.Authenticate(),
				r.authMiddleware.RequireRole("seller", "admin"),
				r.productController.UpdateProduct,
			)
			products.DELETE("/:id",
				r.authMiddleware.Authenticate(),
				r.authMiddleware.RequireRole("seller", "admin"),
				r.productController.DeleteProduct,
			)
		}

		cart := v1.Group("/cart")
		cart.Use(r.authMiddleware.Authenticate())
		{
			cart.GET("", r.cartController.GetCart)
			cart.POST("", r.cartController.AddToCart)
			cart.PUT("/:id", r.cartController.UpdateCartItem)
			cart.DELETE("/:id", r.cartController.RemoveFromCart)
			cart.DELETE("", r.cartController.ClearCart)
		}

		orders := v1.Group("/orders")
		orders.Use(r.authMiddleware.Authenticate())
		{
			orders.GET("", r.orderController.GetOrders)
			orders.GET("/:id", r.orderController.GetOrderByID)
			orders.POST("", r.orderController.CreateOrder)

			orders.PUT("/:id/status",
				r.authMiddleware.RequireRole("admin"),
				r.orderController.UpdateOrderStatus,
			)
			orders.PUT("/:id/payment", r.orderController.UpdatePaymentStatus)
		}

		wishlist := v1.Group("/wishlist")
		wishlist.Use(r.authMiddleware.Authenticate())
		{
			wishlist.GET("", r.wishlistController.GetWishlist)
			wishlist.POST("", r.wishlistController.AddToWishlist)
			wishlist.DELETE("/:product_id", r.wishlistController.RemoveFromWishlist)
		}

		addresses := v1.Group("/addresses")
		addresses.Use(r.authMiddleware.Authenticate())
		{
			addresses.GET("", r.addressController.ListAddresses)
			addresses.POST("", r.addressController.CreateAddress)
			addresses.PUT("/:id", r.addressController.UpdateAddress)
			addresses.DELETE("/:id", r.addressController.DeleteAddress)
			addresses.PUT("/:id/default", r.addressController.SetDefaultAddress)
		}

		seller := v1.Group("/seller")
		seller.Use(r.authMiddleware.Authenticate())
		{
			seller.GET("/stores", r.sellerController.ListMyStores)
			seller.GET("/dashboard", r.sellerController.GetDashboard)
			seller.GET("/stores/:store_id/orders", r.sellerController.GetStoreOrders)
			seller.PUT("/orders/:id/status", r.sellerController.UpdateOrderStatus)
		}

		upload := v1.Group("/upload")
		upload.Use(r.authMiddleware.Authenticate())
		{
			upload.POST("/image", r.uploadController.UploadImage)
		}
	}

	return router
}

func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

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
