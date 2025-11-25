package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	CORS     CORSConfig
	Payment  PaymentConfig
	S3       S3Config
}

type ServerConfig struct {
	Port        string
	GinMode     string
	Environment string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type JWTConfig struct {
	Secret             string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

type CORSConfig struct {
	AllowedOrigins []string
}

type PaymentConfig struct {
	KakaoPay KakaoPayConfig
}

type KakaoPayConfig struct {
	AdminKey    string
	CID         string
	BaseURL     string
	ApprovalURL string
	FailURL     string
	CancelURL   string
}

type S3Config struct {
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	BaseURL         string // CloudFront or S3 direct URL
}

func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	config := &Config{
		Server: ServerConfig{
			Port:        getEnv("SERVER_PORT", "8080"),
			GinMode:     getEnv("GIN_MODE", "debug"),
			Environment: getEnv("ENVIRONMENT", "development"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "admin"),
			Password: getEnv("DB_PASSWORD", "1234"),
			DBName:   getEnv("DB_NAME", "udonggeum"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:             getEnv("JWT_SECRET", "your-secret-key"),
			AccessTokenExpiry:  parseDuration(getEnv("JWT_ACCESS_TOKEN_EXPIRY", "15m")),
			RefreshTokenExpiry: parseDuration(getEnv("JWT_REFRESH_TOKEN_EXPIRY", "168h")),
		},
		CORS: CORSConfig{
			AllowedOrigins: parseSlice(getEnv("ALLOWED_ORIGINS", "http://localhost:3000")),
		},
		Payment: PaymentConfig{
			KakaoPay: KakaoPayConfig{
				AdminKey:    getEnv("KAKAOPAY_ADMIN_KEY", ""),
				CID:         getEnv("KAKAOPAY_CID", "TC0ONETIME"),
				BaseURL:     getEnv("KAKAOPAY_BASE_URL", "https://open-api.kakaopay.com/online/v1/payment"),
				ApprovalURL: getEnv("KAKAOPAY_APPROVAL_URL", "http://localhost:8080/api/v1/payments/kakao/success"),
				FailURL:     getEnv("KAKAOPAY_FAIL_URL", "http://localhost:8080/api/v1/payments/kakao/fail"),
				CancelURL:   getEnv("KAKAOPAY_CANCEL_URL", "http://localhost:8080/api/v1/payments/kakao/cancel"),
			},
		},
		S3: S3Config{
			Region:          getEnv("AWS_REGION", "ap-northeast-2"),
			Bucket:          getEnv("AWS_S3_BUCKET", "udonggeum-uploads"),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
			BaseURL:         getEnv("AWS_S3_BASE_URL", ""),
		},
	}

	return config, nil
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	duration, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("Invalid duration %s, using default 15m", s)
		return 15 * time.Minute
	}
	return duration
}

func parseSlice(s string) []string {
	if s == "" {
		return []string{}
	}
	var result []string
	for i := 0; i < len(s); {
		end := i
		for end < len(s) && s[end] != ',' {
			end++
		}
		result = append(result, s[i:end])
		i = end + 1
	}
	return result
}
