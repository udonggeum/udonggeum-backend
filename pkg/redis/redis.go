package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/ikkim/udonggeum-backend/config"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"github.com/redis/go-redis/v9"
)

var client *redis.Client

// Init initializes Redis connection
func Init(cfg *config.RedisConfig) error {
	logger.Info("Initializing Redis connection", map[string]interface{}{
		"host": cfg.Host,
		"port": cfg.Port,
		"db":   cfg.DB,
	})

	client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error("Failed to connect to Redis", err, map[string]interface{}{
			"host": cfg.Host,
			"port": cfg.Port,
		})
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Redis connection established successfully", nil)
	return nil
}

// GetClient returns the Redis client instance
func GetClient() *redis.Client {
	return client
}

// Close closes the Redis connection
func Close() error {
	if client != nil {
		logger.Info("Closing Redis connection", nil)
		return client.Close()
	}
	return nil
}

// BlacklistToken adds a token to the blacklist
func BlacklistToken(ctx context.Context, token string, expiry time.Duration) error {
	logger.Debug("Adding token to blacklist", map[string]interface{}{
		"expiry": expiry.String(),
	})

	key := fmt.Sprintf("blacklist:%s", token)
	err := client.Set(ctx, key, "revoked", expiry).Err()
	if err != nil {
		logger.Error("Failed to blacklist token", err, nil)
		return err
	}

	logger.Debug("Token successfully blacklisted", nil)
	return nil
}

// IsTokenBlacklisted checks if a token is in the blacklist
func IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	key := fmt.Sprintf("blacklist:%s", token)
	val, err := client.Get(ctx, key).Result()

	if err == redis.Nil {
		// Key does not exist - token is not blacklisted
		return false, nil
	}
	if err != nil {
		logger.Error("Failed to check token blacklist", err, nil)
		return false, err
	}

	// Token is blacklisted
	return val == "revoked", nil
}
