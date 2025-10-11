package db

import (
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
)

// Migrate runs database migrations
func Migrate() error {
	logger.Info("Running database migrations...")

	models := []interface{}{
		&model.User{},
		&model.Product{},
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

	// Check if products already exist
	var count int64
	DB.Model(&model.Product{}).Count(&count)
	if count > 0 {
		logger.Info("Database already seeded, skipping...", map[string]interface{}{
			"existing_products": count,
		})
		return nil
	}

	// Sample products
	products := []model.Product{
		{
			Name:          "24K 골드바 100g",
			Description:   "순도 99.99% 24K 골드바",
			Price:         8500000,
			Weight:        100,
			Purity:        "24K",
			Category:      model.CategoryGold,
			StockQuantity: 10,
			ImageURL:      "https://example.com/images/gold-bar-100g.jpg",
		},
		{
			Name:          "18K 골드 목걸이",
			Description:   "18K 금 목걸이 50cm",
			Price:         1200000,
			Weight:        10,
			Purity:        "18K",
			Category:      model.CategoryJewelry,
			StockQuantity: 20,
			ImageURL:      "https://example.com/images/gold-necklace.jpg",
		},
		{
			Name:          "실버 반지",
			Description:   "925 실버 반지",
			Price:         150000,
			Weight:        5,
			Purity:        "925",
			Category:      model.CategorySilver,
			StockQuantity: 50,
			ImageURL:      "https://example.com/images/silver-ring.jpg",
		},
	}

	result := DB.Create(&products)
	if result.Error != nil {
		logger.Error("Failed to seed products", result.Error)
		return result.Error
	}

	logger.Info("Database seeded successfully", map[string]interface{}{
		"products_count": len(products),
	})
	return nil
}
