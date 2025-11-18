package service

import (
	"errors"
	"time"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"github.com/ikkim/udonggeum-backend/pkg/util"
	"gorm.io/gorm"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
)

type AuthService interface {
	Register(email, password, name, phone string) (*model.User, *util.TokenPair, error)
	Login(email, password string) (*model.User, *util.TokenPair, error)
	GetUserByID(id uint) (*model.User, error)
	UpdateProfile(userID uint, name, phone string) (*model.User, error)
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

	// Create user
	user := &model.User{
		Email:        email,
		PasswordHash: hashedPassword,
		Name:         name,
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

func (s *authService) UpdateProfile(userID uint, name, phone string) (*model.User, error) {
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
		"user_id": user.ID,
		"name":    user.Name,
		"phone":   user.Phone,
	})

	return user, nil
}
