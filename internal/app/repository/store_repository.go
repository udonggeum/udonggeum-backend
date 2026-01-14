package repository

import (
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

type StoreFilter struct {
	Region          string
	District        string
	Search          string
	IncludeProducts bool
	BuyingGold      bool // 금 매입 가능 매장만 조회
}

type StoreLocation struct {
	Region     string
	District   string
	StoreCount int64
}

type StoreRepository interface {
	Create(store *model.Store) error
	Update(store *model.Store) error
	Delete(id uint) error
	FindAll(filter StoreFilter) ([]model.Store, error)
	FindByID(id uint, includeProducts bool) (*model.Store, error)
	FindByUserID(userID uint) ([]model.Store, error)
	FindSingleByUserID(userID uint) (*model.Store, error)
	FindByBusinessNumber(businessNumber string) (*model.Store, error)
	ListLocations() ([]StoreLocation, error)
	ToggleLike(storeID, userID uint) (bool, error)
	IsLiked(storeID, userID uint) (bool, error)
	GetUserLikedStores(userID uint) ([]model.Store, error)
	GetUserLikedStoreIDs(userID uint) ([]uint, error)
	BulkCreate(stores []model.Store, batchSize int) error
	CreateBusinessRegistration(businessReg *model.BusinessRegistration) error
	CreateVerification(verification *model.StoreVerification) error
	FindVerificationByStoreID(storeID uint) (*model.StoreVerification, error)
	FindVerificationByID(verificationID uint) (*model.StoreVerification, error)
	FindVerificationsByStatus(status string) ([]*model.StoreVerification, error)
	UpdateVerification(verification *model.StoreVerification) error
}

type storeRepository struct {
	db *gorm.DB
}

func NewStoreRepository(db *gorm.DB) StoreRepository {
	return &storeRepository{db: db}
}

func (r *storeRepository) Create(store *model.Store) error {
	logger.Debug("Creating store in database", map[string]interface{}{
		"name":   store.Name,
		"region": store.Region,
		"userID": store.UserID,
	})

	if err := r.db.Create(store).Error; err != nil {
		logger.Error("Failed to create store in database", err, map[string]interface{}{
			"name":   store.Name,
			"region": store.Region,
			"userID": store.UserID,
		})
		return err
	}

	logger.Debug("Store created in database", map[string]interface{}{
		"store_id": store.ID,
		"name":     store.Name,
		"userID":   store.UserID,
	})
	return nil
}

func (r *storeRepository) Update(store *model.Store) error {
	logger.Debug("Updating store in database", map[string]interface{}{
		"store_id": store.ID,
		"name":     store.Name,
		"userID":   store.UserID,
	})

	if err := r.db.Save(store).Error; err != nil {
		logger.Error("Failed to update store in database", err, map[string]interface{}{
			"store_id": store.ID,
			"name":     store.Name,
			"userID":   store.UserID,
		})
		return err
	}

	logger.Debug("Store updated in database", map[string]interface{}{
		"store_id": store.ID,
		"name":     store.Name,
	})
	return nil
}

func (r *storeRepository) Delete(id uint) error {
	logger.Debug("Deleting store from database", map[string]interface{}{
		"store_id": id,
	})

	if err := r.db.Delete(&model.Store{}, id).Error; err != nil {
		logger.Error("Failed to delete store from database", err, map[string]interface{}{
			"store_id": id,
		})
		return err
	}

	logger.Debug("Store deleted from database", map[string]interface{}{
		"store_id": id,
	})
	return nil
}

func (r *storeRepository) FindAll(filter StoreFilter) ([]model.Store, error) {
	logger.Debug("Finding stores", map[string]interface{}{
		"region":   filter.Region,
		"district": filter.District,
		"search":   filter.Search,
	})

	query := r.db.Model(&model.Store{}).Preload("Tags")
	if filter.IncludeProducts {
		query = query.Preload("Products", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Options")
		})
	}

	if filter.Region != "" {
		query = query.Where("region = ?", filter.Region)
	}
	if filter.District != "" {
		query = query.Where("district = ?", filter.District)
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		query = query.Where("name LIKE ?", like)
	}
	if filter.BuyingGold {
		query = query.Where("buying_gold = ?", true)
	}

	var stores []model.Store
	if err := query.Order("name ASC").Find(&stores).Error; err != nil {
		logger.Error("Failed to find stores", err, map[string]interface{}{
			"region":   filter.Region,
			"district": filter.District,
		})
		return nil, err
	}

	if err := r.populateStoreStats(&stores); err != nil {
		logger.Error("Failed to populate store stats", err, nil)
		return nil, err
	}

	logger.Debug("Stores found", map[string]interface{}{
		"count": len(stores),
	})
	return stores, nil
}

func (r *storeRepository) FindByID(id uint, includeProducts bool) (*model.Store, error) {
	logger.Debug("Finding store by ID", map[string]interface{}{
		"store_id": id,
	})

	query := r.db.Model(&model.Store{}).Preload("Tags").Preload("BusinessRegistration")
	if includeProducts {
		query = query.Preload("Products", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Options")
		})
	}

	var store model.Store
	if err := query.First(&store, id).Error; err != nil {
		logger.Error("Failed to find store", err, map[string]interface{}{
			"store_id": id,
		})
		return nil, err
	}

	// Populate category counts and total products
	stores := []model.Store{store}
	if err := r.populateStoreStats(&stores); err != nil {
		logger.Error("Failed to populate store stats", err, map[string]interface{}{
			"store_id": id,
		})
		return nil, err
	}
	store = stores[0]

	logger.Debug("Store found", map[string]interface{}{
		"store_id": store.ID,
		"name":     store.Name,
	})
	return &store, nil
}

func (r *storeRepository) FindByUserID(userID uint) ([]model.Store, error) {
	logger.Debug("Finding stores by user ID in database", map[string]interface{}{
		"user_id": userID,
	})

	var stores []model.Store
	if err := r.db.Preload("Tags").Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&stores).Error; err != nil {
		logger.Error("Failed to find stores by user ID in database", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	if err := r.populateStoreStats(&stores); err != nil {
		logger.Error("Failed to populate store stats for user stores", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	logger.Debug("Stores found by user ID in database", map[string]interface{}{
		"user_id": userID,
		"count":   len(stores),
	})
	return stores, nil
}

func (r *storeRepository) ListLocations() ([]StoreLocation, error) {
	logger.Debug("Listing unique store locations")

	var locations []StoreLocation
	if err := r.db.Model(&model.Store{}).
		Select("region, district, COUNT(*) as store_count").
		Group("region, district").
		Order("region ASC, district ASC").
		Scan(&locations).Error; err != nil {
		logger.Error("Failed to list store locations", err)
		return nil, err
	}

	logger.Debug("Store locations listed", map[string]interface{}{
		"count": len(locations),
	})
	return locations, nil
}

func (r *storeRepository) populateStoreStats(stores *[]model.Store) error {
	// Product 관련 기능 제거됨 - 홍보 사이트로 전환
	return nil
}

// ToggleLike 매장 좋아요 토글
func (r *storeRepository) ToggleLike(storeID, userID uint) (bool, error) {
	logger.Debug("Toggling store like", map[string]interface{}{
		"store_id": storeID,
		"user_id":  userID,
	})

	var like model.StoreLike
	err := r.db.Where("store_id = ? AND user_id = ?", storeID, userID).First(&like).Error

	if err == gorm.ErrRecordNotFound {
		// 좋아요 추가
		like = model.StoreLike{
			StoreID: storeID,
			UserID:  userID,
		}
		if err := r.db.Create(&like).Error; err != nil {
			logger.Error("Failed to create store like", err, map[string]interface{}{
				"store_id": storeID,
				"user_id":  userID,
			})
			return false, err
		}

		logger.Debug("Store like added", map[string]interface{}{
			"store_id": storeID,
			"user_id":  userID,
		})
		return true, nil
	} else if err != nil {
		logger.Error("Failed to check store like", err, map[string]interface{}{
			"store_id": storeID,
			"user_id":  userID,
		})
		return false, err
	}

	// 좋아요 제거
	if err := r.db.Delete(&like).Error; err != nil {
		logger.Error("Failed to delete store like", err, map[string]interface{}{
			"store_id": storeID,
			"user_id":  userID,
		})
		return false, err
	}

	logger.Debug("Store like removed", map[string]interface{}{
		"store_id": storeID,
		"user_id":  userID,
	})
	return false, nil
}

// IsLiked 사용자가 매장에 좋아요를 눌렀는지 확인
func (r *storeRepository) IsLiked(storeID, userID uint) (bool, error) {
	logger.Debug("Checking if store is liked", map[string]interface{}{
		"store_id": storeID,
		"user_id":  userID,
	})

	var count int64
	err := r.db.Model(&model.StoreLike{}).
		Where("store_id = ? AND user_id = ?", storeID, userID).
		Count(&count).Error
	if err != nil {
		logger.Error("Failed to check if store is liked", err, map[string]interface{}{
			"store_id": storeID,
			"user_id":  userID,
		})
		return false, err
	}

	logger.Debug("Store like status checked", map[string]interface{}{
		"store_id": storeID,
		"user_id":  userID,
		"is_liked": count > 0,
	})
	return count > 0, nil
}

// GetUserLikedStores retrieves all stores liked by the user
func (r *storeRepository) GetUserLikedStores(userID uint) ([]model.Store, error) {
	logger.Debug("Getting user liked stores from repository", map[string]interface{}{
		"user_id": userID,
	})

	var stores []model.Store
	err := r.db.
		Joins("JOIN store_likes ON store_likes.store_id = stores.id").
		Where("store_likes.user_id = ?", userID).
		Preload("Tags").
		Find(&stores).Error

	if err != nil {
		logger.Error("Failed to query user liked stores", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	logger.Debug("User liked stores queried", map[string]interface{}{
		"user_id": userID,
		"count":   len(stores),
	})
	return stores, nil
}

// GetUserLikedStoreIDs retrieves IDs of all stores liked by the user
func (r *storeRepository) GetUserLikedStoreIDs(userID uint) ([]uint, error) {
	logger.Debug("Getting user liked store IDs from repository", map[string]interface{}{
		"user_id": userID,
	})

	var storeIDs []uint
	err := r.db.Model(&model.StoreLike{}).
		Where("user_id = ?", userID).
		Pluck("store_id", &storeIDs).Error

	if err != nil {
		logger.Error("Failed to query user liked store IDs", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	logger.Debug("User liked store IDs queried", map[string]interface{}{
		"user_id": userID,
		"count":   len(storeIDs),
	})
	return storeIDs, nil
}

// CreateBusinessRegistration creates a new business registration
func (r *storeRepository) CreateBusinessRegistration(businessReg *model.BusinessRegistration) error {
	logger.Info("Creating business registration", map[string]interface{}{
		"store_id":        businessReg.StoreID,
		"business_number": businessReg.BusinessNumber,
	})

	if err := r.db.Create(businessReg).Error; err != nil {
		logger.Error("Failed to create business registration", err, map[string]interface{}{
			"store_id": businessReg.StoreID,
		})
		return err
	}

	logger.Info("Business registration created successfully", map[string]interface{}{
		"store_id": businessReg.StoreID,
	})
	return nil
}

// FindSingleByUserID finds a single store by user ID
func (r *storeRepository) FindSingleByUserID(userID uint) (*model.Store, error) {
	var store model.Store
	if err := r.db.
		Preload("BusinessRegistration").
		Preload("Tags").
		Preload("Verification").
		Where("user_id = ?", userID).
		First(&store).Error; err != nil {
		return nil, err
	}
	return &store, nil
}

// FindByBusinessNumber finds a store by business number
func (r *storeRepository) FindByBusinessNumber(businessNumber string) (*model.Store, error) {
	logger.Debug("Finding store by business number", map[string]interface{}{
		"business_number": businessNumber,
	})

	var businessReg model.BusinessRegistration
	if err := r.db.
		Preload("Store").
		Where("business_number = ?", businessNumber).
		First(&businessReg).Error; err != nil {
		logger.Debug("No store found with business number", map[string]interface{}{
			"business_number": businessNumber,
		})
		return nil, err
	}

	logger.Debug("Store found with business number", map[string]interface{}{
		"business_number": businessNumber,
		"store_id":        businessReg.StoreID,
	})
	return &businessReg.Store, nil
}

// CreateVerification creates a new store verification request
func (r *storeRepository) CreateVerification(verification *model.StoreVerification) error {
	logger.Info("Creating verification", map[string]interface{}{
		"store_id": verification.StoreID,
	})

	if err := r.db.Create(verification).Error; err != nil {
		logger.Error("Failed to create verification", err, map[string]interface{}{
			"store_id": verification.StoreID,
		})
		return err
	}

	logger.Info("Verification created successfully", map[string]interface{}{
		"verification_id": verification.ID,
		"store_id":        verification.StoreID,
	})
	return nil
}

// FindVerificationByStoreID finds verification by store ID (latest one)
func (r *storeRepository) FindVerificationByStoreID(storeID uint) (*model.StoreVerification, error) {
	var verification model.StoreVerification
	if err := r.db.
		Where("store_id = ?", storeID).
		Order("created_at DESC").
		First(&verification).Error; err != nil {
		return nil, err
	}
	return &verification, nil
}

// FindVerificationByID finds verification by ID
func (r *storeRepository) FindVerificationByID(verificationID uint) (*model.StoreVerification, error) {
	var verification model.StoreVerification
	if err := r.db.
		Preload("Store").
		First(&verification, verificationID).Error; err != nil {
		return nil, err
	}
	return &verification, nil
}

// FindVerificationsByStatus finds verifications by status
func (r *storeRepository) FindVerificationsByStatus(status string) ([]*model.StoreVerification, error) {
	var verifications []*model.StoreVerification
	query := r.db.Preload("Store").Where("status = ?", status).Order("created_at DESC")

	if err := query.Find(&verifications).Error; err != nil {
		logger.Error("Failed to find verifications by status", err, map[string]interface{}{
			"status": status,
		})
		return nil, err
	}

	logger.Debug("Verifications found by status", map[string]interface{}{
		"status": status,
		"count":  len(verifications),
	})
	return verifications, nil
}

// UpdateVerification updates a verification record
func (r *storeRepository) UpdateVerification(verification *model.StoreVerification) error {
	logger.Info("Updating verification", map[string]interface{}{
		"verification_id": verification.ID,
	})

	if err := r.db.Save(verification).Error; err != nil {
		logger.Error("Failed to update verification", err, map[string]interface{}{
			"verification_id": verification.ID,
		})
		return err
	}

	logger.Info("Verification updated successfully", map[string]interface{}{
		"verification_id": verification.ID,
	})
	return nil
}

// BulkCreate creates multiple stores in batches
func (r *storeRepository) BulkCreate(stores []model.Store, batchSize int) error {
	logger.Info("Bulk creating stores", map[string]interface{}{
		"total_count": len(stores),
		"batch_size":  batchSize,
	})

	if err := r.db.CreateInBatches(stores, batchSize).Error; err != nil {
		logger.Error("Failed to bulk create stores", err, map[string]interface{}{
			"total_count": len(stores),
			"batch_size":  batchSize,
		})
		return err
	}

	logger.Info("Bulk create completed", map[string]interface{}{
		"count": len(stores),
	})
	return nil
}
