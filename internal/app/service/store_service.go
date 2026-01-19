package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"github.com/ikkim/udonggeum-backend/pkg/util"
	"gorm.io/gorm"
)

var (
	ErrStoreNotFound     = errors.New("매장을 찾을 수 없습니다")
	ErrStoreAccessDenied = errors.New("매장 접근 권한이 없습니다")
)

type StoreListOptions struct {
	Region     string
	District   string
	Search     string
	UserLat    *float64 // 사용자 위도 (거리순 정렬용)
	UserLng    *float64 // 사용자 경도 (거리순 정렬용)
	IsVerified *bool    // 인증 매장 필터
	IsManaged  *bool    // 관리 매장 필터
	Page       int      // 페이지 번호 (1부터 시작)
	PageSize   int      // 페이지당 개수
}

type StoreLocationSummary struct {
	Region     string `json:"region"`
	District   string `json:"district"`
	StoreCount int64  `json:"store_count"`
}

type StoreService interface {
	ListStores(opts StoreListOptions) (*repository.StoreListResult, error)
	GetStoreByID(id uint) (*model.Store, error)
	GetStoresByUserID(userID uint) ([]model.Store, error)
	GetStoreByUserID(userID uint) (*model.Store, error)
	GetStoreByBusinessNumber(businessNumber string) (*model.Store, error)
	ListLocations() ([]StoreLocationSummary, error)
	CreateStore(store *model.Store) (*model.Store, error)
	UpdateStore(userID uint, storeID uint, input StoreMutation) (*model.Store, error)
	UpdateStoreOwnership(store *model.Store) (*model.Store, error)
	ClaimStoreTransaction(store *model.Store, userID uint) (*model.Store, error)
	DeleteStore(userID uint, storeID uint) error
	ToggleStoreLike(storeID, userID uint) (bool, error)
	IsStoreLiked(storeID, userID uint) (bool, error)
	GetUserLikedStores(userID uint) ([]model.Store, error)
	GetUserLikedStoreIDs(userID uint) ([]uint, error)
	PromoteUserToAdmin(userID uint) error
	CreateVerification(verification *model.StoreVerification) (*model.StoreVerification, error)
	GetVerificationByStoreID(storeID uint) (*model.StoreVerification, error)
	GetVerificationByID(verificationID uint) (*model.StoreVerification, error)
	ListVerificationsByStatus(status string) ([]*model.StoreVerification, error)
	ApproveStoreVerification(storeID uint, verifiedAt *time.Time) error
	UpdateVerification(verification *model.StoreVerification) error
}

type storeService struct {
	db        *gorm.DB
	storeRepo repository.StoreRepository
	userRepo  repository.UserRepository
}

type StoreMutation struct {
	Name        *string
	Region      *string
	District    *string
	Address     *string
	Latitude    *float64
	Longitude   *float64
	PhoneNumber *string
	ImageURL    *string
	Description *string
	OpenTime    *string
	CloseTime   *string
	TagIDs      []uint                 // 태그 ID 배열
	Background  *model.StoreBackground // 배경 설정
}

func NewStoreService(db *gorm.DB, storeRepo repository.StoreRepository, userRepo repository.UserRepository) StoreService {
	return &storeService{
		db:        db,
		storeRepo: storeRepo,
		userRepo:  userRepo,
	}
}

func (s *storeService) ListStores(opts StoreListOptions) (*repository.StoreListResult, error) {
	logger.Debug("Listing stores", map[string]interface{}{
		"region":      opts.Region,
		"district":    opts.District,
		"is_verified": opts.IsVerified,
		"is_managed":  opts.IsManaged,
		"page":        opts.Page,
		"page_size":   opts.PageSize,
		"user_lat":    opts.UserLat,
		"user_lng":    opts.UserLng,
	})

	// Repository에서 거리 계산 및 정렬 처리
	result, err := s.storeRepo.FindAll(repository.StoreFilter{
		Region:     opts.Region,
		District:   opts.District,
		Search:     opts.Search,
		IsVerified: opts.IsVerified,
		IsManaged:  opts.IsManaged,
		Page:       opts.Page,
		PageSize:   opts.PageSize,
		UserLat:    opts.UserLat,
		UserLng:    opts.UserLng,
	})
	if err != nil {
		logger.Error("Failed to list stores", err)
		return nil, err
	}

	logger.Info("Stores fetched", map[string]interface{}{
		"count":       len(result.Stores),
		"total_count": result.TotalCount,
		"user_lat":    opts.UserLat,
		"user_lng":    opts.UserLng,
	})
	return result, nil
}

func (s *storeService) GetStoreByID(id uint) (*model.Store, error) {
	logger.Debug("Fetching store by ID", map[string]interface{}{
		"store_id": id,
	})

	store, err := s.storeRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Store not found", map[string]interface{}{
				"store_id": id,
			})
			return nil, ErrStoreNotFound
		}
		logger.Error("Failed to fetch store", err, map[string]interface{}{
			"store_id": id,
		})
		return nil, err
	}

	return store, nil
}

func (s *storeService) GetStoresByUserID(userID uint) ([]model.Store, error) {
	logger.Debug("Fetching stores by user ID", map[string]interface{}{
		"user_id": userID,
	})

	stores, err := s.storeRepo.FindByUserID(userID)
	if err != nil {
		logger.Error("Failed to fetch stores by user ID", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	logger.Info("Stores fetched by user ID", map[string]interface{}{
		"user_id": userID,
		"count":   len(stores),
	})
	return stores, nil
}

func (s *storeService) GetStoreByBusinessNumber(businessNumber string) (*model.Store, error) {
	logger.Debug("Fetching store by business number", map[string]interface{}{
		"business_number": businessNumber,
	})

	store, err := s.storeRepo.FindByBusinessNumber(businessNumber)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Debug("Store not found with business number", map[string]interface{}{
				"business_number": businessNumber,
			})
			return nil, nil // 찾지 못한 경우 nil 반환 (에러 아님)
		}
		logger.Error("Failed to fetch store by business number", err, map[string]interface{}{
			"business_number": businessNumber,
		})
		return nil, err
	}

	return store, nil
}

func (s *storeService) CreateStore(store *model.Store) (*model.Store, error) {
	logger.Info("Creating store", map[string]interface{}{
		"name":    store.Name,
		"user_id": store.UserID,
	})

	// Geocode address to get coordinates if address is provided
	if store.Address != "" && (store.Latitude == nil || store.Longitude == nil) {
		lat, lng, err := util.GeocodeAddress(store.Address)
		if err != nil {
			logger.Warn("Failed to geocode store address during creation", map[string]interface{}{
				"address": store.Address,
				"error":   err.Error(),
			})
			// Continue without coordinates if geocoding fails
		} else {
			store.Latitude = lat
			store.Longitude = lng
			logger.Info("Successfully geocoded store address during creation", map[string]interface{}{
				"address":   store.Address,
				"latitude":  lat,
				"longitude": lng,
			})
		}
	}

	// Begin transaction to ensure atomic store creation + user nickname update
	tx := s.db.Begin()
	if tx.Error != nil {
		logger.Error("Failed to begin transaction for CreateStore", tx.Error)
		return nil, tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error("Panic in CreateStore, transaction rolled back", fmt.Errorf("%v", r))
			panic(r)
		}
	}()

	// Create store
	if err := tx.Create(store).Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to create store", err, map[string]interface{}{
			"name":    store.Name,
			"user_id": store.UserID,
		})
		return nil, err
	}

	// Update user's nickname to store name (매장 등록 시 무조건 닉네임 변경)
	if store.UserID != nil {
		if err := tx.Model(&model.User{}).Where("id = ?", *store.UserID).Update("nickname", store.Name).Error; err != nil {
			tx.Rollback()
			logger.Error("Failed to update user nickname after store creation", err, map[string]interface{}{
				"user_id":    *store.UserID,
				"store_name": store.Name,
			})
			return nil, fmt.Errorf("failed to update user nickname: %w", err)
		}
		logger.Info("User nickname updated to store name", map[string]interface{}{
			"user_id":  *store.UserID,
			"nickname": store.Name,
		})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		logger.Error("Failed to commit CreateStore transaction", err)
		return nil, err
	}

	logger.Info("Store created", map[string]interface{}{
		"store_id": store.ID,
		"name":     store.Name,
	})
	return store, nil
}

func (s *storeService) UpdateStore(userID uint, storeID uint, input StoreMutation) (*model.Store, error) {
	logger.Info("Updating store", map[string]interface{}{
		"store_id": storeID,
		"user_id":  userID,
	})

	existing, err := s.storeRepo.FindByID(storeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Store not found for update", map[string]interface{}{
				"store_id": storeID,
			})
			return nil, ErrStoreNotFound
		}
		logger.Error("Failed to find store for update", err, map[string]interface{}{
			"store_id": storeID,
		})
		return nil, err
	}

	if existing.UserID == nil || *existing.UserID != userID {
		logger.Warn("Store update forbidden", map[string]interface{}{
			"store_id": storeID,
			"user_id":  userID,
		})
		return nil, ErrStoreAccessDenied
	}

	// Check if store name changed
	storeNameChanged := false
	if input.Name != nil && existing.Name != *input.Name {
		storeNameChanged = true
		existing.Name = *input.Name
	}

	// Update region if provided
	if input.Region != nil {
		existing.Region = *input.Region
	}

	// Update district if provided
	if input.District != nil {
		existing.District = *input.District
	}

	// Address handling with geocoding
	addressChanged := false
	if input.Address != nil && existing.Address != *input.Address {
		addressChanged = true
		existing.Address = *input.Address

		// If address changed and not empty, geocode it to get new coordinates
		if *input.Address != "" {
			lat, lng, err := util.GeocodeAddress(*input.Address)
			if err != nil {
				logger.Warn("Failed to geocode store address, using provided coordinates", map[string]interface{}{
					"store_id": storeID,
					"address":  *input.Address,
					"error":    err.Error(),
				})
				// Fall back to provided coordinates if geocoding fails
				if input.Latitude != nil {
					existing.Latitude = input.Latitude
				}
				if input.Longitude != nil {
					existing.Longitude = input.Longitude
				}
			} else {
				existing.Latitude = lat
				existing.Longitude = lng
				logger.Info("Successfully geocoded store address", map[string]interface{}{
					"store_id":  storeID,
					"address":   *input.Address,
					"latitude":  lat,
					"longitude": lng,
				})
			}
		} else {
			// Address cleared
			existing.Latitude = nil
			existing.Longitude = nil
		}
	} else if !addressChanged {
		// If address didn't change, update coordinates only if provided
		if input.Latitude != nil {
			existing.Latitude = input.Latitude
		}
		if input.Longitude != nil {
			existing.Longitude = input.Longitude
		}
	}

	// Update other fields if provided
	if input.PhoneNumber != nil {
		existing.PhoneNumber = *input.PhoneNumber
	}
	if input.ImageURL != nil {
		existing.ImageURL = *input.ImageURL
	}
	if input.Description != nil {
		existing.Description = *input.Description
	}
	if input.OpenTime != nil {
		existing.OpenTime = *input.OpenTime
	}
	if input.CloseTime != nil {
		existing.CloseTime = *input.CloseTime
	}

	// 배경 설정 업데이트
	if input.Background != nil {
		existing.Background = input.Background
	}

	// 태그 업데이트 (Many-to-Many 관계)
	if input.TagIDs != nil {
		var tags []model.Tag
		for _, tagID := range input.TagIDs {
			tags = append(tags, model.Tag{ID: tagID})
		}
		existing.Tags = tags
	}

	if err := s.storeRepo.Update(existing); err != nil {
		logger.Error("Failed to update store", err, map[string]interface{}{
			"store_id": storeID,
		})
		return nil, err
	}

	// Update user's nickname to new store name if it changed and user is admin
	if storeNameChanged && input.Name != nil {
		user, err := s.userRepo.FindByID(userID)
		if err == nil && user.Role == model.RoleAdmin {
			user.Nickname = *input.Name
			if err := s.userRepo.Update(user); err != nil {
				logger.Warn("Failed to update user nickname after store name change", map[string]interface{}{
					"user_id":    userID,
					"store_name": *input.Name,
					"error":      err.Error(),
				})
				// Don't fail the entire operation if nickname update fails
			} else {
				logger.Info("User nickname updated to new store name", map[string]interface{}{
					"user_id":  userID,
					"nickname": *input.Name,
				})
			}
		}
	}

	logger.Info("Store updated", map[string]interface{}{
		"store_id": storeID,
	})
	return existing, nil
}

func (s *storeService) DeleteStore(userID uint, storeID uint) error {
	logger.Info("Deleting store", map[string]interface{}{
		"store_id": storeID,
		"user_id":  userID,
	})

	existing, err := s.storeRepo.FindByID(storeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Store not found for delete", map[string]interface{}{
				"store_id": storeID,
			})
			return ErrStoreNotFound
		}
		logger.Error("Failed to find store for delete", err, map[string]interface{}{
			"store_id": storeID,
		})
		return err
	}

	if existing.UserID == nil || *existing.UserID != userID {
		logger.Warn("Store delete forbidden", map[string]interface{}{
			"store_id": storeID,
			"user_id":  userID,
		})
		return ErrStoreAccessDenied
	}

	// 트랜잭션으로 Store와 연관 데이터를 함께 soft delete
	tx := s.db.Begin()
	if tx.Error != nil {
		logger.Error("Failed to begin transaction for store deletion", tx.Error, map[string]interface{}{
			"store_id": storeID,
		})
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error("Transaction rolled back due to panic during store deletion", nil, map[string]interface{}{
				"store_id": storeID,
				"panic":    r,
			})
		}
	}()

	// Store soft delete
	if err := tx.Delete(&model.Store{}, storeID).Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to delete store", err, map[string]interface{}{
			"store_id": storeID,
		})
		return err
	}

	// BusinessRegistration soft delete (if exists)
	if err := tx.Where("store_id = ?", storeID).Delete(&model.BusinessRegistration{}).Error; err != nil {
		// BusinessRegistration이 없을 수도 있으므로 RecordNotFound는 무시
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			logger.Error("Failed to delete business registration", err, map[string]interface{}{
				"store_id": storeID,
			})
			return err
		}
	}

	// 트랜잭션 커밋
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to commit store deletion transaction", err, map[string]interface{}{
			"store_id": storeID,
		})
		return err
	}

	logger.Info("Store and related data deleted successfully", map[string]interface{}{
		"store_id": storeID,
	})
	return nil
}

func (s *storeService) ListLocations() ([]StoreLocationSummary, error) {
	logger.Debug("Listing store locations")

	locations, err := s.storeRepo.ListLocations()
	if err != nil {
		logger.Error("Failed to list store locations", err)
		return nil, err
	}

	summaries := make([]StoreLocationSummary, 0, len(locations))
	for _, loc := range locations {
		summaries = append(summaries, StoreLocationSummary{
			Region:     loc.Region,
			District:   loc.District,
			StoreCount: loc.StoreCount,
		})
	}

	logger.Info("Store locations fetched", map[string]interface{}{
		"count": len(summaries),
	})
	return summaries, nil
}

// ToggleStoreLike 매장 좋아요 토글
func (s *storeService) ToggleStoreLike(storeID, userID uint) (bool, error) {
	logger.Debug("Toggling store like", map[string]interface{}{
		"store_id": storeID,
		"user_id":  userID,
	})

	// 매장 존재 확인
	_, err := s.storeRepo.FindByID(storeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Store not found for like toggle", map[string]interface{}{
				"store_id": storeID,
			})
			return false, ErrStoreNotFound
		}
		logger.Error("Failed to find store for like toggle", err, map[string]interface{}{
			"store_id": storeID,
		})
		return false, err
	}

	isLiked, err := s.storeRepo.ToggleLike(storeID, userID)
	if err != nil {
		logger.Error("Failed to toggle store like", err, map[string]interface{}{
			"store_id": storeID,
			"user_id":  userID,
		})
		return false, err
	}

	logger.Info("Store like toggled", map[string]interface{}{
		"store_id": storeID,
		"user_id":  userID,
		"is_liked": isLiked,
	})
	return isLiked, nil
}

// IsStoreLiked 사용자가 매장에 좋아요를 눌렀는지 확인
func (s *storeService) IsStoreLiked(storeID, userID uint) (bool, error) {
	logger.Debug("Checking if store is liked", map[string]interface{}{
		"store_id": storeID,
		"user_id":  userID,
	})

	isLiked, err := s.storeRepo.IsLiked(storeID, userID)
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
		"is_liked": isLiked,
	})
	return isLiked, nil
}

// GetUserLikedStores retrieves all stores liked by the user
func (s *storeService) GetUserLikedStores(userID uint) ([]model.Store, error) {
	logger.Debug("Getting user liked stores", map[string]interface{}{
		"user_id": userID,
	})

	stores, err := s.storeRepo.GetUserLikedStores(userID)
	if err != nil {
		logger.Error("Failed to get user liked stores", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	logger.Debug("User liked stores retrieved", map[string]interface{}{
		"user_id": userID,
		"count":   len(stores),
	})
	return stores, nil
}

// GetUserLikedStoreIDs retrieves IDs of all stores liked by the user
func (s *storeService) GetUserLikedStoreIDs(userID uint) ([]uint, error) {
	logger.Debug("Getting user liked store IDs", map[string]interface{}{
		"user_id": userID,
	})

	storeIDs, err := s.storeRepo.GetUserLikedStoreIDs(userID)
	if err != nil {
		logger.Error("Failed to get user liked store IDs", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	logger.Debug("User liked store IDs retrieved", map[string]interface{}{
		"user_id": userID,
		"count":   len(storeIDs),
	})
	return storeIDs, nil
}

// PromoteUserToAdmin promotes a user to admin role
func (s *storeService) PromoteUserToAdmin(userID uint) error {
	logger.Info("Promoting user to admin", map[string]interface{}{
		"user_id": userID,
	})

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		logger.Error("Failed to find user for promotion", err, map[string]interface{}{
			"user_id": userID,
		})
		return err
	}

	if user.Role == model.RoleAdmin {
		logger.Info("User is already admin", map[string]interface{}{
			"user_id": userID,
		})
		return nil // Already admin, no need to update
	}

	user.Role = model.RoleAdmin
	if err := s.userRepo.Update(user); err != nil {
		logger.Error("Failed to promote user to admin", err, map[string]interface{}{
			"user_id": userID,
		})
		return err
	}

	logger.Info("User promoted to admin successfully", map[string]interface{}{
		"user_id": userID,
	})
	return nil
}

// UpdateStoreOwnership updates store ownership information (for claiming stores)
func (s *storeService) UpdateStoreOwnership(store *model.Store) (*model.Store, error) {
	logger.Info("Updating store ownership", map[string]interface{}{
		"store_id": store.ID,
		"user_id":  store.UserID,
	})

	// 트랜잭션으로 처리하여 일부만 성공하는 문제 방지
	tx := s.db.Begin()
	if tx.Error != nil {
		logger.Error("Failed to begin transaction", tx.Error, map[string]interface{}{
			"store_id": store.ID,
		})
		return nil, tx.Error
	}

	// 트랜잭션 완료 시 커밋 또는 롤백
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error("Transaction rolled back due to panic", nil, map[string]interface{}{
				"store_id": store.ID,
				"panic":    r,
			})
		}
	}()

	// Update store with new ownership information (including business registration via association)
	if err := tx.Save(store).Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to update store ownership", err, map[string]interface{}{
			"store_id": store.ID,
		})
		return nil, err
	}

	// 트랜잭션 커밋
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to commit transaction", err, map[string]interface{}{
			"store_id": store.ID,
		})
		return nil, err
	}

	logger.Info("Store ownership updated successfully", map[string]interface{}{
		"store_id":   store.ID,
		"user_id":    store.UserID,
		"is_managed": store.IsManaged,
	})

	// Reload store with all associations
	updated, err := s.storeRepo.FindByID(store.ID)
	if err != nil {
		logger.Error("Failed to reload claimed store", err, map[string]interface{}{
			"store_id": store.ID,
		})
		return nil, err
	}

	return updated, nil
}

// ClaimStoreTransaction handles the entire store claim process in a single transaction
// This ensures atomicity: either all operations succeed (store update + user promotion) or all fail
func (s *storeService) ClaimStoreTransaction(store *model.Store, userID uint) (*model.Store, error) {
	logger.Info("Starting store claim transaction", map[string]interface{}{
		"store_id": store.ID,
		"user_id":  userID,
	})

	// 트랜잭션 시작
	tx := s.db.Begin()
	if tx.Error != nil {
		logger.Error("Failed to begin transaction for store claim", tx.Error, map[string]interface{}{
			"store_id": store.ID,
			"user_id":  userID,
		})
		return nil, tx.Error
	}

	// 트랜잭션 완료 시 커밋 또는 롤백
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error("Transaction rolled back due to panic during store claim", nil, map[string]interface{}{
				"store_id": store.ID,
				"user_id":  userID,
				"panic":    r,
			})
		}
	}()

	// 1. Store 업데이트 (BusinessRegistration은 GORM association으로 자동 저장)
	if err := tx.Save(store).Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to update store in claim transaction", err, map[string]interface{}{
			"store_id": store.ID,
			"user_id":  userID,
		})
		return nil, err
	}

	// 2. User role을 admin으로 승격
	if err := tx.Model(&model.User{}).
		Where("id = ?", userID).
		Update("role", model.RoleAdmin).Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to promote user to admin in claim transaction", err, map[string]interface{}{
			"user_id":  userID,
			"store_id": store.ID,
		})
		return nil, err
	}

	// 3. User nickname을 store name으로 업데이트
	if err := tx.Model(&model.User{}).
		Where("id = ?", userID).
		Update("nickname", store.Name).Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to update user nickname in claim transaction", err, map[string]interface{}{
			"user_id":  userID,
			"store_id": store.ID,
			"nickname": store.Name,
		})
		return nil, err
	}

	// 4. 트랜잭션 커밋
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to commit store claim transaction", err, map[string]interface{}{
			"store_id": store.ID,
			"user_id":  userID,
		})
		return nil, err
	}

	logger.Info("Store claim transaction completed successfully", map[string]interface{}{
		"store_id":   store.ID,
		"user_id":    userID,
		"is_managed": store.IsManaged,
	})

	// Reload store with all associations
	updated, err := s.storeRepo.FindByID(store.ID)
	if err != nil {
		logger.Error("Failed to reload claimed store", err, map[string]interface{}{
			"store_id": store.ID,
		})
		return nil, err
	}

	return updated, nil
}

// GetStoreByUserID gets a store by user ID
func (s *storeService) GetStoreByUserID(userID uint) (*model.Store, error) {
	logger.Info("Getting store by user ID", map[string]interface{}{
		"user_id": userID,
	})

	store, err := s.storeRepo.FindSingleByUserID(userID)
	if err != nil {
		logger.Error("Failed to find store by user ID", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	return store, nil
}

// CreateVerification creates a new store verification request
func (s *storeService) CreateVerification(verification *model.StoreVerification) (*model.StoreVerification, error) {
	logger.Info("Creating verification", map[string]interface{}{
		"store_id": verification.StoreID,
	})

	if err := s.storeRepo.CreateVerification(verification); err != nil {
		logger.Error("Failed to create verification", err, map[string]interface{}{
			"store_id": verification.StoreID,
		})
		return nil, err
	}

	logger.Info("Verification created successfully", map[string]interface{}{
		"verification_id": verification.ID,
		"store_id":        verification.StoreID,
	})

	return verification, nil
}

// GetVerificationByStoreID gets verification by store ID
func (s *storeService) GetVerificationByStoreID(storeID uint) (*model.StoreVerification, error) {
	logger.Debug("Getting verification by store ID", map[string]interface{}{
		"store_id": storeID,
	})

	verification, err := s.storeRepo.FindVerificationByStoreID(storeID)
	if err != nil {
		logger.Debug("Verification not found for store", map[string]interface{}{
			"store_id": storeID,
		})
		return nil, err
	}

	return verification, nil
}

// GetVerificationByID gets verification by ID
func (s *storeService) GetVerificationByID(verificationID uint) (*model.StoreVerification, error) {
	logger.Debug("Getting verification by ID", map[string]interface{}{
		"verification_id": verificationID,
	})

	verification, err := s.storeRepo.FindVerificationByID(verificationID)
	if err != nil {
		logger.Error("Verification not found", err, map[string]interface{}{
			"verification_id": verificationID,
		})
		return nil, err
	}

	return verification, nil
}

// ListVerificationsByStatus lists verifications by status
func (s *storeService) ListVerificationsByStatus(status string) ([]*model.StoreVerification, error) {
	logger.Info("Listing verifications by status", map[string]interface{}{
		"status": status,
	})

	verifications, err := s.storeRepo.FindVerificationsByStatus(status)
	if err != nil {
		logger.Error("Failed to list verifications", err, map[string]interface{}{
			"status": status,
		})
		return nil, err
	}

	logger.Info("Verifications listed", map[string]interface{}{
		"status": status,
		"count":  len(verifications),
	})

	return verifications, nil
}

// ApproveStoreVerification approves a store verification (sets is_verified to true)
func (s *storeService) ApproveStoreVerification(storeID uint, verifiedAt *time.Time) error {
	logger.Info("Approving store verification", map[string]interface{}{
		"store_id": storeID,
	})

	store, err := s.storeRepo.FindByID(storeID)
	if err != nil {
		logger.Error("Store not found for verification approval", err, map[string]interface{}{
			"store_id": storeID,
		})
		return err
	}

	store.IsVerified = true
	store.VerifiedAt = verifiedAt

	if err := s.storeRepo.Update(store); err != nil {
		logger.Error("Failed to update store verification status", err, map[string]interface{}{
			"store_id": storeID,
		})
		return err
	}

	logger.Info("Store verification approved", map[string]interface{}{
		"store_id": storeID,
	})

	return nil
}

// UpdateVerification updates a verification record
func (s *storeService) UpdateVerification(verification *model.StoreVerification) error {
	logger.Info("Updating verification", map[string]interface{}{
		"verification_id": verification.ID,
		"status":          verification.Status,
	})

	if err := s.storeRepo.UpdateVerification(verification); err != nil {
		logger.Error("Failed to update verification", err, map[string]interface{}{
			"verification_id": verification.ID,
		})
		return err
	}

	logger.Info("Verification updated successfully", map[string]interface{}{
		"verification_id": verification.ID,
		"status":          verification.Status,
	})

	return nil
}
