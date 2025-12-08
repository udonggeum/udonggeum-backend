package db

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var TestDB *gorm.DB

func SetupTestDB(t *testing.T) (*gorm.DB, error) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=udonggeum_test port=5432 sslmode=disable"
	}

	var err error
	TestDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	// Auto migrate test models
	if err := TestDB.AutoMigrate(
		&model.User{},
		&model.Store{},
		&model.PasswordReset{},
		&model.GoldPrice{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate test database: %w", err)
	}

	return TestDB, nil
}

func CleanupTestDB(t *testing.T, db *gorm.DB) {
	if db == nil {
		return
	}

	// Drop all tables in reverse order of dependencies
	tables := []interface{}{
		&model.GoldPrice{},
		&model.PasswordReset{},
		&model.Store{},
		&model.User{},
	}

	for _, table := range tables {
		if err := db.Migrator().DropTable(table); err != nil {
			log.Printf("Warning: Failed to drop table: %v", err)
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get database instance: %v", err)
	}
	sqlDB.Close()
}

func TruncateTables(db *gorm.DB) error {
	tables := []string{
		"gold_prices",
		"password_resets",
		"stores",
		"users",
	}

	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table)).Error; err != nil {
			return err
		}
	}

	return nil
}
