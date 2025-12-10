package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	redisClient "github.com/ikkim/udonggeum-backend/pkg/redis"
	"github.com/ikkim/udonggeum-backend/pkg/util"
	"gorm.io/gorm"
)

var (
	ErrEmailAlreadyExists    = errors.New("email already exists")
	ErrInvalidCredentials    = errors.New("invalid email or password")
	ErrUserNotFound          = errors.New("user not found")
	ErrInvalidToken          = errors.New("invalid token")
	ErrExpiredToken          = errors.New("token has expired")
	ErrTokenRevoked          = errors.New("token has been revoked")
	ErrNicknameAlreadyExists = errors.New("nickname already exists")
)

type AuthService interface {
	Register(email, password, name, phone string) (*model.User, *util.TokenPair, error)
	Login(email, password string) (*model.User, *util.TokenPair, error)
	GetUserByID(id uint) (*model.User, error)
	UpdateProfile(userID uint, name, phone, nickname, address string) (*model.User, error)
	CheckNickname(nickname string) (bool, error)
	RefreshToken(refreshToken string) (*util.TokenPair, error)
	RevokeToken(refreshToken string) error
}

type authService struct {
	userRepo          repository.UserRepository
	jwtSecret         string
	accessExpiry      time.Duration
	refreshExpiry     time.Duration
}

func NewAuthService(
	userRepo repository.UserRepository,
	jwtSecret string,
	accessExpiry, refreshExpiry time.Duration,
) AuthService {
	return &authService{
		userRepo:      userRepo,
		jwtSecret:     jwtSecret,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (s *authService) Register(email, password, name, phone string) (*model.User, *util.TokenPair, error) {
	logger.Info("Attempting user registration", map[string]interface{}{
		"email": email,
		"name":  name,
	})

	// Check if user already exists
	existingUser, err := s.userRepo.FindByEmail(email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error("Failed to check existing user", err, map[string]interface{}{
			"email": email,
		})
		return nil, nil, err
	}
	if existingUser != nil {
		logger.Warn("Registration failed: email already exists", map[string]interface{}{
			"email": email,
		})
		return nil, nil, ErrEmailAlreadyExists
	}

	// Hash password
	hashedPassword, err := util.HashPassword(password)
	if err != nil {
		logger.Error("Failed to hash password", err, map[string]interface{}{
			"email": email,
		})
		return nil, nil, err
	}

	// Generate unique nickname
	nickname, err := s.generateUniqueNickname()
	if err != nil {
		logger.Error("Failed to generate unique nickname", err, map[string]interface{}{
			"email": email,
		})
		return nil, nil, err
	}

	// Create user
	user := &model.User{
		Email:        email,
		PasswordHash: hashedPassword,
		Name:         name,
		Nickname:     nickname,
		Phone:        phone,
		Role:         model.RoleUser,
	}

	if err := s.userRepo.Create(user); err != nil {
		logger.Error("Failed to create user in database", err, map[string]interface{}{
			"email": email,
		})
		return nil, nil, err
	}

	// Generate tokens
	tokens, err := util.GenerateTokenPair(
		user.ID,
		user.Email,
		string(user.Role),
		s.jwtSecret,
		s.accessExpiry,
		s.refreshExpiry,
	)
	if err != nil {
		logger.Error("Failed to generate tokens", err, map[string]interface{}{
			"user_id": user.ID,
			"email":   email,
		})
		return nil, nil, err
	}

	logger.Info("User registered successfully", map[string]interface{}{
		"user_id": user.ID,
		"email":   email,
		"role":    user.Role,
	})

	return user, tokens, nil
}

func (s *authService) Login(email, password string) (*model.User, *util.TokenPair, error) {
	logger.Info("Login attempt", map[string]interface{}{
		"email": email,
	})

	// Find user by email
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Login failed: user not found", map[string]interface{}{
				"email": email,
			})
			return nil, nil, ErrInvalidCredentials
		}
		logger.Error("Failed to find user", err, map[string]interface{}{
			"email": email,
		})
		return nil, nil, err
	}

	// Verify password
	if !util.VerifyPassword(user.PasswordHash, password) {
		logger.Warn("Login failed: invalid password", map[string]interface{}{
			"email":   email,
			"user_id": user.ID,
		})
		return nil, nil, ErrInvalidCredentials
	}

	// Generate tokens
	tokens, err := util.GenerateTokenPair(
		user.ID,
		user.Email,
		string(user.Role),
		s.jwtSecret,
		s.accessExpiry,
		s.refreshExpiry,
	)
	if err != nil {
		logger.Error("Failed to generate tokens", err, map[string]interface{}{
			"user_id": user.ID,
			"email":   email,
		})
		return nil, nil, err
	}

	logger.Info("User logged in successfully", map[string]interface{}{
		"user_id": user.ID,
		"email":   email,
		"role":    user.Role,
	})

	return user, tokens, nil
}

func (s *authService) GetUserByID(id uint) (*model.User, error) {
	logger.Debug("Fetching user by ID", map[string]interface{}{
		"user_id": id,
	})

	user, err := s.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("User not found", map[string]interface{}{
				"user_id": id,
			})
			return nil, ErrUserNotFound
		}
		logger.Error("Failed to fetch user", err, map[string]interface{}{
			"user_id": id,
		})
		return nil, err
	}

	logger.Debug("User fetched successfully", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	return user, nil
}

func (s *authService) UpdateProfile(userID uint, name, phone, nickname, address string) (*model.User, error) {
	logger.Info("Updating user profile", map[string]interface{}{
		"user_id": userID,
	})

	// Fetch existing user
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("User not found for profile update", map[string]interface{}{
				"user_id": userID,
			})
			return nil, ErrUserNotFound
		}
		logger.Error("Failed to fetch user for profile update", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	// Update fields if provided
	updated := false
	if name != "" && name != user.Name {
		user.Name = name
		updated = true
	}
	if phone != "" && phone != user.Phone {
		user.Phone = phone
		updated = true
	}
	if nickname != "" && nickname != user.Nickname {
		// Admin 사용자는 닉네임을 직접 수정할 수 없음 (매장 이름과 자동 동기화됨)
		if user.Role == model.RoleAdmin {
			logger.Warn("Admin users cannot update nickname directly", map[string]interface{}{
				"user_id": userID,
			})
			return nil, errors.New("admin users cannot update nickname directly - it is automatically synchronized with store name")
		}

		// Check if nickname already exists
		existingUser, err := s.userRepo.FindByNickname(nickname)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("Failed to check existing nickname", err, map[string]interface{}{
				"nickname": nickname,
			})
			return nil, err
		}
		if existingUser != nil && existingUser.ID != userID {
			logger.Warn("Nickname already exists", map[string]interface{}{
				"nickname": nickname,
			})
			return nil, ErrNicknameAlreadyExists
		}
		user.Nickname = nickname
		updated = true
	}
	// Address can be empty (to clear it), so we don't check for empty string
	if address != user.Address {
		user.Address = address
		updated = true
	}

	// Only update if there are changes
	if !updated {
		logger.Debug("No changes detected for user profile", map[string]interface{}{
			"user_id": userID,
		})
		return user, nil
	}

	// Save changes
	if err := s.userRepo.Update(user); err != nil {
		logger.Error("Failed to update user profile", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	logger.Info("User profile updated successfully", map[string]interface{}{
		"user_id":  user.ID,
		"name":     user.Name,
		"phone":    user.Phone,
		"nickname": user.Nickname,
		"address":  user.Address,
	})

	return user, nil
}

// CheckNickname checks if a nickname is available
func (s *authService) CheckNickname(nickname string) (bool, error) {
	logger.Debug("Checking nickname availability", map[string]interface{}{
		"nickname": nickname,
	})

	// Check if nickname already exists
	existingUser, err := s.userRepo.FindByNickname(nickname)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error("Failed to check nickname availability", err, map[string]interface{}{
			"nickname": nickname,
		})
		return false, err
	}

	// If user exists, nickname is not available
	isAvailable := existingUser == nil
	logger.Debug("Nickname availability checked", map[string]interface{}{
		"nickname":    nickname,
		"is_available": isAvailable,
	})

	return isAvailable, nil
}

// RefreshToken validates a refresh token and generates a new token pair
func (s *authService) RefreshToken(refreshToken string) (*util.TokenPair, error) {
	logger.Debug("Attempting to refresh token")

	// Check if token is blacklisted
	ctx := context.Background()
	isBlacklisted, err := redisClient.IsTokenBlacklisted(ctx, refreshToken)
	if err != nil {
		logger.Error("Failed to check token blacklist", err, nil)
		return nil, err
	}
	if isBlacklisted {
		logger.Warn("Attempted to use revoked refresh token", nil)
		return nil, ErrTokenRevoked
	}

	// Validate the refresh token
	claims, err := util.ValidateToken(refreshToken, s.jwtSecret)
	if err != nil {
		if errors.Is(err, util.ErrExpiredToken) {
			logger.Warn("Refresh token has expired", nil)
			return nil, ErrExpiredToken
		}
		logger.Warn("Invalid refresh token", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, ErrInvalidToken
	}

	// Verify user still exists
	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("User not found for token refresh", map[string]interface{}{
				"user_id": claims.UserID,
			})
			return nil, ErrUserNotFound
		}
		logger.Error("Failed to fetch user for token refresh", err, map[string]interface{}{
			"user_id": claims.UserID,
		})
		return nil, err
	}

	// Generate new token pair
	tokens, err := util.GenerateTokenPair(
		user.ID,
		user.Email,
		string(user.Role),
		s.jwtSecret,
		s.accessExpiry,
		s.refreshExpiry,
	)
	if err != nil {
		logger.Error("Failed to generate new token pair", err, map[string]interface{}{
			"user_id": user.ID,
		})
		return nil, err
	}

	// Blacklist the old refresh token (token rotation)
	if err := redisClient.BlacklistToken(ctx, refreshToken, s.refreshExpiry); err != nil {
		logger.Error("Failed to blacklist old refresh token", err, nil)
		// Don't fail the request, just log the error
	}

	logger.Info("Token refreshed successfully", map[string]interface{}{
		"user_id": user.ID,
	})

	return tokens, nil
}

// RevokeToken adds a refresh token to the blacklist
func (s *authService) RevokeToken(refreshToken string) error {
	logger.Debug("Attempting to revoke token")

	// Validate token to get expiry time
	claims, err := util.ValidateToken(refreshToken, s.jwtSecret)
	if err != nil {
		// Even if token is invalid/expired, we still blacklist it
		logger.Warn("Revoking invalid/expired token", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Calculate remaining TTL
	var ttl time.Duration
	if claims != nil && claims.ExpiresAt != nil {
		ttl = time.Until(claims.ExpiresAt.Time)
		if ttl < 0 {
			// Token already expired, no need to blacklist
			logger.Debug("Token already expired, skipping blacklist", nil)
			return nil
		}
	} else {
		// Default to refresh token expiry if we can't determine
		ttl = s.refreshExpiry
	}

	ctx := context.Background()
	if err := redisClient.BlacklistToken(ctx, refreshToken, ttl); err != nil {
		logger.Error("Failed to blacklist token", err, nil)
		return err
	}

	logger.Info("Token revoked successfully", nil)
	return nil
}

// generateUniqueNickname generates a random unique nickname
func (s *authService) generateUniqueNickname() (string, error) {
	const (
		maxRetries = 10
		prefix     = "사용자"
	)

	for i := 0; i < maxRetries; i++ {
		// Generate random 6-digit number
		randomNum := util.GenerateRandomNumber(100000, 999999)
		nickname := fmt.Sprintf("%s%d", prefix, randomNum)

		// Check if nickname already exists
		existingUser, err := s.userRepo.FindByNickname(nickname)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("Failed to check existing nickname", err, map[string]interface{}{
				"nickname": nickname,
			})
			return "", err
		}

		// If nickname doesn't exist, return it
		if existingUser == nil {
			logger.Debug("Generated unique nickname", map[string]interface{}{
				"nickname": nickname,
			})
			return nickname, nil
		}

		logger.Debug("Nickname already exists, retrying", map[string]interface{}{
			"nickname": nickname,
			"attempt":  i + 1,
		})
	}

	return "", errors.New("failed to generate unique nickname after maximum retries")
}
