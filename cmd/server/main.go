package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ikkim/udonggeum-backend/config"
	"github.com/ikkim/udonggeum-backend/internal/app/controller"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/db"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
	"github.com/ikkim/udonggeum-backend/internal/router"
	"github.com/ikkim/udonggeum-backend/internal/storage"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", err)
	}

	logLevel := "info"
	if cfg.Server.Environment == "development" {
		logLevel = "debug"
	}
	logger.Initialize(logger.Config{
		Level:       logLevel,
		Format:      "console",
		EnableColor: true,
	})

	logger.Info("Starting UDONGGEUM Backend Server", map[string]interface{}{
		"environment": cfg.Server.Environment,
		"port":        cfg.Server.Port,
		"log_level":   logLevel,
	})

	if err := db.Initialize(&cfg.Database); err != nil {
		logger.Fatal("Failed to initialize database", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database connection", err)
		}
	}()

	if err := db.Migrate(); err != nil {
		logger.Fatal("Failed to run migrations", err)
	}

	if err := db.Seed(); err != nil {
		logger.Warn("Failed to seed database", map[string]interface{}{
			"error": err.Error(),
		})
	}

	dbConn := db.GetDB()

	userRepo := repository.NewUserRepository(dbConn)
	storeRepo := repository.NewStoreRepository(dbConn)
	productRepo := repository.NewProductRepository(dbConn)
	productOptionRepo := repository.NewProductOptionRepository(dbConn)
	orderRepo := repository.NewOrderRepository(dbConn)
	cartRepo := repository.NewCartRepository(dbConn)
	wishlistRepo := repository.NewWishlistRepository(dbConn)
	addressRepo := repository.NewAddressRepository(dbConn)
	passwordResetRepo := repository.NewPasswordResetRepository(dbConn)

	authService := service.NewAuthService(
		userRepo,
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpiry,
		cfg.JWT.RefreshTokenExpiry,
	)
	passwordResetService := service.NewPasswordResetService(passwordResetRepo, userRepo)
	storeService := service.NewStoreService(storeRepo)
	productService := service.NewProductService(productRepo, productOptionRepo)
	cartService := service.NewCartService(cartRepo, productRepo, productOptionRepo)
	orderService := service.NewOrderService(orderRepo, cartRepo, productRepo, dbConn, productOptionRepo)

	paymentService, err := service.NewPaymentService(orderRepo, cfg, dbConn)
	if err != nil {
		logger.Fatal("Failed to initialize payment service", err)
	}
	wishlistService := service.NewWishlistService(wishlistRepo, productRepo)
	addressService := service.NewAddressService(addressRepo)
	sellerService := service.NewSellerService(orderRepo, storeRepo)

	authController := controller.NewAuthController(authService, passwordResetService)
	storeController := controller.NewStoreController(storeService)
	productController := controller.NewProductController(productService)
	cartController := controller.NewCartController(cartService)
	orderController := controller.NewOrderController(orderService)
	paymentController := controller.NewPaymentController(paymentService)
	wishlistController := controller.NewWishlistController(wishlistService)
	addressController := controller.NewAddressController(addressService)
	sellerController := controller.NewSellerController(sellerService, storeService)

	s3Storage := storage.NewS3Storage(
		cfg.S3.Region,
		cfg.S3.Bucket,
		cfg.S3.AccessKeyID,
		cfg.S3.SecretAccessKey,
		cfg.S3.BaseURL,
	)
	uploadController := controller.NewUploadController(s3Storage)

	authMiddleware := middleware.NewAuthMiddleware(cfg.JWT.Secret)

	r := router.NewRouter(
		authController,
		storeController,
		productController,
		cartController,
		orderController,
		paymentController,
		wishlistController,
		addressController,
		sellerController,
		uploadController,
		authMiddleware,
		cfg,
	)
	engine := r.Setup()

	go func() {
		addr := fmt.Sprintf(":%s", cfg.Server.Port)
		logger.Info("Server started successfully", map[string]interface{}{
			"address": addr,
			"pid":     os.Getpid(),
		})
		if err := engine.Run(addr); err != nil {
			logger.Fatal("Failed to start server", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server gracefully...")
	logger.Info("Server stopped successfully")
}
