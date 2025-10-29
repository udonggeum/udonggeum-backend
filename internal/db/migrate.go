package db

import (
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"github.com/ikkim/udonggeum-backend/pkg/util"
)

// Migrate runs database migrations
func Migrate() error {
	logger.Info("Running database migrations...")

	models := []interface{}{
		&model.Store{},
		&model.User{},
		&model.Product{},
		&model.ProductOption{},
		&model.Order{},
		&model.OrderItem{},
		&model.CartItem{},
	}

	err := DB.AutoMigrate(models...)
	if err != nil {
		logger.Error("Failed to run migrations", err)
		return err
	}

	logger.Info("Database migrations completed successfully", map[string]interface{}{
		"models_count": len(models),
	})
	return nil
}

// Seed adds initial data to the database (optional)
func Seed() error {
	logger.Info("Seeding database...")

	var count int64
	DB.Model(&model.Product{}).Count(&count)
	if count > 0 {
		logger.Info("Database already seeded, skipping...", map[string]interface{}{
			"existing_products": count,
		})
		return nil
	}

	adminPassword, err := util.HashPassword("password123!")
	if err != nil {
		logger.Error("Failed to hash admin password", err)
		return err
	}

	admin := model.User{
		Email:        "admin@example.com",
		PasswordHash: adminPassword,
		Name:         "관리자",
		Role:         model.RoleAdmin,
	}

	if err := DB.Where("email = ?", admin.Email).FirstOrCreate(&admin).Error; err != nil {
		logger.Error("Failed to seed admin user", err)
		return err
	}

	stores := []model.Store{
		{
			UserID:      admin.ID,
			Name:        "강동 우동금 주얼리",
			Region:      "서울특별시",
			District:    "강동구",
			Address:     "서울특별시 강동구 천호대로 1075",
			PhoneNumber: "02-1234-5678",
			ImageURL:    "https://example.com/images/stores/gangdong-main.jpg",
			Description: "강동구 대표 귀금속 전문 매장",
		},
		{
			UserID:      admin.ID,
			Name:        "강남 우동금",
			Region:      "서울특별시",
			District:    "강남구",
			Address:     "서울특별시 강남구 테헤란로 231",
			PhoneNumber: "02-9876-5432",
			ImageURL:    "https://example.com/images/stores/gangnam-main.jpg",
			Description: "강남구 프리미엄 금은방",
		},
	}

	if err := DB.Create(&stores).Error; err != nil {
		logger.Error("Failed to seed stores", err)
		return err
	}

	products := []model.Product{
		{
			Name:            "24K 순금 반지",
			Description:     "서울 강동구 프리미엄 순금 반지",
			Price:           950000,
			Weight:          7.5,
			Purity:          "24K",
			Category:        model.CategoryRing,
			Material:        model.MaterialGold,
			StockQuantity:   12,
			ImageURL:        "https://example.com/images/products/gangdong-gold-ring.jpg",
			StoreID:         stores[0].ID,
			PopularityScore: 92,
		},
		{
			Name:            "18K 골드 목걸이",
			Description:     "데일리 착용하기 좋은 18K 목걸이",
			Price:           1280000,
			Weight:          9.8,
			Purity:          "18K",
			Category:        model.CategoryNecklace,
			Material:        model.MaterialGold,
			StockQuantity:   8,
			ImageURL:        "https://example.com/images/products/gangdong-gold-necklace.jpg",
			StoreID:         stores[0].ID,
			PopularityScore: 87,
		},
		{
			Name:            "실버 커플링",
			Description:     "심플한 디자인의 925 실버 커플링",
			Price:           180000,
			Weight:          5.2,
			Purity:          "925",
			Category:        model.CategoryRing,
			Material:        model.MaterialSilver,
			StockQuantity:   25,
			ImageURL:        "https://example.com/images/products/gangnam-silver-ring.jpg",
			StoreID:         stores[1].ID,
			PopularityScore: 75,
		},
	}

	if err := DB.Create(&products).Error; err != nil {
		logger.Error("Failed to seed products", err)
		return err
	}

	options := []model.ProductOption{
		{
			ProductID:       products[0].ID,
			Name:            "사이즈",
			Value:           "9호",
			AdditionalPrice: 0,
			StockQuantity:   4,
			IsDefault:       true,
		},
		{
			ProductID:       products[0].ID,
			Name:            "사이즈",
			Value:           "11호",
			AdditionalPrice: 20000,
			StockQuantity:   4,
		},
		{
			ProductID:       products[0].ID,
			Name:            "사이즈",
			Value:           "13호",
			AdditionalPrice: 30000,
			StockQuantity:   4,
		},
		{
			ProductID:       products[1].ID,
			Name:            "길이",
			Value:           "45cm",
			AdditionalPrice: 0,
			StockQuantity:   4,
			IsDefault:       true,
		},
		{
			ProductID:       products[1].ID,
			Name:            "길이",
			Value:           "50cm",
			AdditionalPrice: 50000,
			StockQuantity:   4,
		},
	}

	if err := DB.Create(&options).Error; err != nil {
		logger.Error("Failed to seed product options", err)
		return err
	}

	logger.Info("Database seeded successfully", map[string]interface{}{
		"products_count": len(products),
		"stores_count":   len(stores),
		"options_count":  len(options),
	})
	return nil
}
