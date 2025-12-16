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
	"github.com/ikkim/udonggeum-backend/internal/scheduler"
	"github.com/ikkim/udonggeum-backend/internal/storage"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	redisClient "github.com/ikkim/udonggeum-backend/pkg/redis"
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

	if err := redisClient.Init(&cfg.Redis); err != nil {
		logger.Fatal("Failed to initialize Redis", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("Failed to close Redis connection", err)
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
	passwordResetRepo := repository.NewPasswordResetRepository(dbConn)
	goldPriceRepo := repository.NewGoldPriceRepository(dbConn)
	communityRepo := repository.NewCommunityRepository(dbConn)
	reviewRepo := repository.NewReviewRepository(dbConn)

	authService := service.NewAuthService(
		userRepo,
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpiry,
		cfg.JWT.RefreshTokenExpiry,
		cfg.Kakao.ClientID,
		cfg.Kakao.ClientSecret,
		cfg.Kakao.RedirectURI,
	)
	passwordResetService := service.NewPasswordResetService(passwordResetRepo, userRepo)
	storeService := service.NewStoreService(storeRepo, userRepo)

	goldPriceAPI := service.NewDefaultGoldPriceAPI(cfg.GoldPrice.APIURL, cfg.GoldPrice.APIKey)
	goldPriceService := service.NewGoldPriceService(goldPriceRepo, goldPriceAPI)

	communityService := service.NewCommunityService(communityRepo, userRepo)
	reviewService := service.NewReviewService(reviewRepo, storeRepo)

	// Initialize S3 storage
	s3Storage := storage.NewS3Storage(
		cfg.S3.Region,
		cfg.S3.Bucket,
		cfg.S3.AccessKeyID,
		cfg.S3.SecretAccessKey,
		cfg.S3.BaseURL,
	)

	authController := controller.NewAuthController(authService, passwordResetService)
	storeController := controller.NewStoreController(storeService)
	goldPriceController := controller.NewGoldPriceController(goldPriceService)
	communityController := controller.NewCommunityController(communityService)
	reviewController := controller.NewReviewController(reviewService)
	uploadController := controller.NewUploadController(s3Storage)

	authMiddleware := middleware.NewAuthMiddleware(cfg.JWT.Secret)

	r := router.NewRouter(
		authController,
		storeController,
		goldPriceController,
		communityController,
		reviewController,
		uploadController,
		authMiddleware,
		cfg,
	)
	engine := r.Setup()

	// 금 시세 자동 업데이트 스케줄러 시작
	goldPriceScheduler := scheduler.NewGoldPriceScheduler(goldPriceService)
	if err := goldPriceScheduler.Start(); err != nil {
		logger.Fatal("Failed to start gold price scheduler", err)
	}
	defer goldPriceScheduler.Stop()

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
