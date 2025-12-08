package repository

import (
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *model.User) error
	FindByID(id uint) (*model.User, error)
	FindByIDWithStores(id uint) (*model.User, error)
	FindByEmail(email string) (*model.User, error)
	Update(user *model.User) error
	Delete(id uint) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *model.User) error {
	logger.Debug("Creating user in database", map[string]interface{}{
		"email": user.Email,
	})

	if err := r.db.Create(user).Error; err != nil {
		logger.Error("Failed to create user in database", err, map[string]interface{}{
			"email": user.Email,
		})
		return err
	}

	logger.Debug("User created in database", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})
	return nil
}

func (r *userRepository) FindByID(id uint) (*model.User, error) {
	logger.Debug("Finding user by ID in database", map[string]interface{}{
		"user_id": id,
	})

	var user model.User
	err := r.db.First(&user, id).Error
	if err != nil {
		logger.Error("Failed to find user by ID in database", err, map[string]interface{}{
			"user_id": id,
		})
		return nil, err
	}

	logger.Debug("User found by ID in database", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})
	return &user, nil
}

func (r *userRepository) FindByIDWithStores(id uint) (*model.User, error) {
	logger.Debug("Finding user by ID with stores in database", map[string]interface{}{
		"user_id": id,
	})

	var user model.User
	err := r.db.Preload("Stores").First(&user, id).Error
	if err != nil {
		logger.Error("Failed to find user by ID with stores in database", err, map[string]interface{}{
			"user_id": id,
		})
		return nil, err
	}

	logger.Debug("User with stores found by ID in database", map[string]interface{}{
		"user_id":     user.ID,
		"email":       user.Email,
		"store_count": len(user.Stores),
	})
	return &user, nil
}

func (r *userRepository) FindByEmail(email string) (*model.User, error) {
	logger.Debug("Finding user by email in database", map[string]interface{}{
		"email": email,
	})

	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		logger.Error("Failed to find user by email in database", err, map[string]interface{}{
			"email": email,
		})
		return nil, err
	}

	logger.Debug("User found by email in database", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})
	return &user, nil
}

func (r *userRepository) Update(user *model.User) error {
	logger.Debug("Updating user in database", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	if err := r.db.Save(user).Error; err != nil {
		logger.Error("Failed to update user in database", err, map[string]interface{}{
			"user_id": user.ID,
			"email":   user.Email,
		})
		return err
	}

	logger.Debug("User updated in database", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})
	return nil
}

func (r *userRepository) Delete(id uint) error {
	logger.Debug("Deleting user from database", map[string]interface{}{
		"user_id": id,
	})

	if err := r.db.Delete(&model.User{}, id).Error; err != nil {
		logger.Error("Failed to delete user from database", err, map[string]interface{}{
			"user_id": id,
		})
		return err
	}

	logger.Debug("User deleted from database", map[string]interface{}{
		"user_id": id,
	})
	return nil
}
