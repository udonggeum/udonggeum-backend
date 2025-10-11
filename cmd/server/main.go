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
	"github.com/ikkim/udonggeum-backend/pkg/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", err)
	}

	// Initialize logger
	logLevel := "info"
	if cfg.Server.Environment == "development" {
		logLevel = "debug"
	}
	logger.Initialize(logger.Config{
		Level:       logLevel,
		Format:      "console", // Use "json" for production
		EnableColor: true,
	})

	logger.Info("Starting UDONGGEUM Backend Server", map[string]interface{}{
		"environment": cfg.Server.Environment,
		"port":        cfg.Server.Port,
		"log_level":   logLevel,
	})

	// Initialize database
	if err := db.Initialize(&cfg.Database); err != nil {
		logger.Fatal("Failed to initialize database", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database connection", err)
		}
	}()

	// Run migrations
	if err := db.Migrate(); err != nil {
		logger.Fatal("Failed to run migrations", err)
	}

	// Seed database (optional)
	if err := db.Seed(); err != nil {
		logger.Warn("Failed to seed database", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db.GetDB())
	productRepo := repository.NewProductRepository(db.GetDB())
	orderRepo := repository.NewOrderRepository(db.GetDB())
	cartRepo := repository.NewCartRepository(db.GetDB())

	// Initialize services
	authService := service.NewAuthService(
		userRepo,
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpiry,
		cfg.JWT.RefreshTokenExpiry,
	)
	productService := service.NewProductService(productRepo)
	cartService := service.NewCartService(cartRepo, productRepo)
	orderService := service.NewOrderService(orderRepo, cartRepo, productRepo, db.GetDB())

	// Initialize controllers
	authController := controller.NewAuthController(authService)
	productController := controller.NewProductController(productService)
	cartController := controller.NewCartController(cartService)
	orderController := controller.NewOrderController(orderService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWT.Secret)

	// Setup router
	r := router.NewRouter(
		authController,
		productController,
		cartController,
		orderController,
		authMiddleware,
		cfg,
	)
	engine := r.Setup()

	// Start server in a goroutine
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

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server gracefully...")
	logger.Info("Server stopped successfully")
}
