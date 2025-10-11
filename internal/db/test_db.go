package db

import (
	"fmt"
	"log"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	// Run migrations
	err = db.AutoMigrate(
		&model.User{},
		&model.Product{},
		&model.Order{},
		&model.OrderItem{},
		&model.CartItem{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate test database: %w", err)
	}

	return db, nil
}

// CleanupTestDB cleans up the test database
func CleanupTestDB(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Failed to get DB instance: %v", err)
		return
	}
	sqlDB.Close()
}

// TruncateAllTables removes all data from tables
func TruncateAllTables(db *gorm.DB) error {
	tables := []string{"cart_items", "order_items", "orders", "products", "users"}
	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("DELETE FROM %s", table)).Error; err != nil {
			return err
		}
	}
	return nil
}
