package db

import (
	"fmt"

	"github.com/ikkim/udonggeum-backend/config"
	appLogger "github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Initialize initializes the database connection
func Initialize(cfg *config.DatabaseConfig) error {
	dsn := cfg.DSN()

	appLogger.Info("Connecting to database", map[string]interface{}{
		"host":     cfg.Host,
		"port":     cfg.Port,
		"database": cfg.DBName,
		"user":     cfg.User,
	})

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Use silent mode, we'll use our own logger
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	appLogger.Info("Database connection established successfully", map[string]interface{}{
		"max_idle_conns": 10,
		"max_open_conns": 100,
	})
	return nil
}

// Close closes the database connection
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}
