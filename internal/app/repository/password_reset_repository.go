package repository

import (
	"time"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

type PasswordResetRepository interface {
	Create(reset *model.PasswordReset) error
	FindByToken(token string) (*model.PasswordReset, error)
	MarkAsUsed(id uint) error
	DeleteExpired() error
}

type passwordResetRepository struct {
	db *gorm.DB
}

func NewPasswordResetRepository(db *gorm.DB) PasswordResetRepository {
	return &passwordResetRepository{db: db}
}

func (r *passwordResetRepository) Create(reset *model.PasswordReset) error {
	logger.Debug("Creating password reset in database", map[string]interface{}{
		"email": reset.Email,
	})

	if err := r.db.Create(reset).Error; err != nil {
		logger.Error("Failed to create password reset in database", err, map[string]interface{}{
			"email": reset.Email,
		})
		return err
	}

	logger.Debug("Password reset created in database", map[string]interface{}{
		"id":    reset.ID,
		"email": reset.Email,
	})
	return nil
}

func (r *passwordResetRepository) FindByToken(token string) (*model.PasswordReset, error) {
	logger.Debug("Finding password reset by token in database", nil)

	var reset model.PasswordReset
	if err := r.db.Where("token = ?", token).First(&reset).Error; err != nil {
		logger.Error("Failed to find password reset by token in database", err, nil)
		return nil, err
	}

	logger.Debug("Password reset found by token in database", map[string]interface{}{
		"id":    reset.ID,
		"email": reset.Email,
	})
	return &reset, nil
}

func (r *passwordResetRepository) MarkAsUsed(id uint) error {
	logger.Debug("Marking password reset as used in database", map[string]interface{}{
		"id": id,
	})

	if err := r.db.Model(&model.PasswordReset{}).Where("id = ?", id).
		Update("used", true).Error; err != nil {
		logger.Error("Failed to mark password reset as used in database", err, map[string]interface{}{
			"id": id,
		})
		return err
	}

	logger.Debug("Password reset marked as used in database", map[string]interface{}{
		"id": id,
	})
	return nil
}

func (r *passwordResetRepository) DeleteExpired() error {
	logger.Debug("Deleting expired password resets from database")

	result := r.db.Where("expires_at < ?", time.Now()).Delete(&model.PasswordReset{})
	if result.Error != nil {
		logger.Error("Failed to delete expired password resets from database", result.Error, nil)
		return result.Error
	}

	logger.Debug("Expired password resets deleted from database", map[string]interface{}{
		"count": result.RowsAffected,
	})
	return nil
}
