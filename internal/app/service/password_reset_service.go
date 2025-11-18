package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"github.com/ikkim/udonggeum-backend/pkg/util"
	"gorm.io/gorm"
)

var (
	ErrInvalidResetToken = errors.New("invalid or expired reset token")
	ErrResetTokenExpired = errors.New("reset token has expired")
	ErrResetTokenUsed    = errors.New("reset token has already been used")
)

const (
	// ResetTokenExpiry is the duration for which a reset token is valid
	ResetTokenExpiry = 1 * time.Hour
	// ResetTokenLength is the byte length of the reset token
	ResetTokenLength = 32
)

type PasswordResetService interface {
	RequestReset(email string) error
	ResetPassword(token, newPassword string) error
}

type passwordResetService struct {
	resetRepo repository.PasswordResetRepository
	userRepo  repository.UserRepository
}

func NewPasswordResetService(
	resetRepo repository.PasswordResetRepository,
	userRepo repository.UserRepository,
) PasswordResetService {
	return &passwordResetService{
		resetRepo: resetRepo,
		userRepo:  userRepo,
	}
}

func (s *passwordResetService) RequestReset(email string) error {
	logger.Info("Processing password reset request", map[string]interface{}{
		"email": email,
	})

	// Check if user exists (but don't reveal this information to prevent user enumeration)
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// For security, we don't reveal if email exists or not
			logger.Warn("Password reset requested for non-existent email", map[string]interface{}{
				"email": email,
			})
			// Return success to prevent user enumeration
			return nil
		}
		logger.Error("Failed to find user for password reset", err, map[string]interface{}{
			"email": email,
		})
		return err
	}

	// Generate secure random token
	token, err := generateResetToken()
	if err != nil {
		logger.Error("Failed to generate reset token", err, map[string]interface{}{
			"email": email,
		})
		return err
	}

	// Create password reset record
	reset := &model.PasswordReset{
		Email:     email,
		Token:     token,
		ExpiresAt: time.Now().Add(ResetTokenExpiry),
		Used:      false,
	}

	if err := s.resetRepo.Create(reset); err != nil {
		logger.Error("Failed to create password reset record", err, map[string]interface{}{
			"email": email,
		})
		return err
	}

	// TODO: Send email with reset link
	// For now, just log the token (this should be removed in production)
	logger.Info("Password reset token generated (EMAIL SENDING NOT IMPLEMENTED)", map[string]interface{}{
		"email":      email,
		"token":      token,
		"expires_at": reset.ExpiresAt,
		"user_id":    user.ID,
	})

	logger.Info("Password reset email would be sent (not implemented)", map[string]interface{}{
		"email": email,
	})

	return nil
}

func (s *passwordResetService) ResetPassword(token, newPassword string) error {
	logger.Info("Processing password reset with token")

	// Find reset record by token
	reset, err := s.resetRepo.FindByToken(token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Invalid reset token provided", nil)
			return ErrInvalidResetToken
		}
		logger.Error("Failed to find reset record", err, nil)
		return err
	}

	// Check if token has expired
	if time.Now().After(reset.ExpiresAt) {
		logger.Warn("Reset token has expired", map[string]interface{}{
			"email":      reset.Email,
			"expires_at": reset.ExpiresAt,
		})
		return ErrResetTokenExpired
	}

	// Check if token has already been used
	if reset.Used {
		logger.Warn("Reset token has already been used", map[string]interface{}{
			"email": reset.Email,
		})
		return ErrResetTokenUsed
	}

	// Find user by email
	user, err := s.userRepo.FindByEmail(reset.Email)
	if err != nil {
		logger.Error("Failed to find user for password reset", err, map[string]interface{}{
			"email": reset.Email,
		})
		return err
	}

	// Hash new password
	hashedPassword, err := util.HashPassword(newPassword)
	if err != nil {
		logger.Error("Failed to hash new password", err, map[string]interface{}{
			"user_id": user.ID,
		})
		return err
	}

	// Update user password
	user.PasswordHash = hashedPassword
	if err := s.userRepo.Update(user); err != nil {
		logger.Error("Failed to update user password", err, map[string]interface{}{
			"user_id": user.ID,
		})
		return err
	}

	// Mark reset token as used
	if err := s.resetRepo.MarkAsUsed(reset.ID); err != nil {
		logger.Error("Failed to mark reset token as used", err, map[string]interface{}{
			"reset_id": reset.ID,
		})
		// Don't return error as password was already updated
	}

	logger.Info("Password reset successful", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	return nil
}

// generateResetToken creates a cryptographically secure random token
func generateResetToken() (string, error) {
	bytes := make([]byte, ResetTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
