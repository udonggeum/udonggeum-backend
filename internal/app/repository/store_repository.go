package repository

import (
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StoreFilter struct {
	Region     string
	District   string
	Search     string
	IsVerified *bool    // 인증 매장 필터
	IsManaged  *bool    // 관리 매장 필터
	Page       int      // 페이지 번호 (1부터 시작)
	PageSize   int      // 페이지당 개수
	UserLat    *float64 // 사용자 위도 (거리순 정렬용)
	UserLng    *float64 // 사용자 경도 (거리순 정렬용)
	CenterLat  *float64 // 검색 중심 위도 (지도 기반 검색용)
	CenterLng  *float64 // 검색 중심 경도 (지도 기반 검색용)
	Radius     *float64 // 검색 반경 (미터 단위)
}

type StoreLocation struct {
	Region     string
	District   string
	StoreCount int64
}

type StoreListResult struct {
	Stores     []model.Store
	TotalCount int64
}

type StoreRepository interface {
	Create(store *model.Store) error
	Update(store *model.Store) error
	Delete(id uint) error
	FindAll(filter StoreFilter) (*StoreListResult, error)
	FindByID(id uint) (*model.Store, error)
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

func (r *storeRepository) FindAll(filter StoreFilter) (*StoreListResult, error) {
	logger.Debug("Finding stores", map[string]interface{}{
		"region":      filter.Region,
		"district":    filter.District,
		"search":      filter.Search,
		"is_verified": filter.IsVerified,
		"is_managed":  filter.IsManaged,
		"page":        filter.Page,
		"page_size":   filter.PageSize,
		"user_lat":    filter.UserLat,
		"user_lng":    filter.UserLng,
	})

	query := r.db.Model(&model.Store{}).Preload("Tags")

	// 기본 필터
	if filter.Region != "" {
		query = query.Where("region = ?", filter.Region)
	}
	if filter.District != "" {
		query = query.Where("district = ?", filter.District)
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		// 매장명, 지역, 시군구, 동, 주소 모두 검색
		query = query.Where(
			"name LIKE ? OR region LIKE ? OR district LIKE ? OR dong LIKE ? OR address LIKE ?",
			like, like, like, like, like,
		)
	}

	// 인증/관리 필터
	if filter.IsVerified != nil {
		query = query.Where("is_verified = ?", *filter.IsVerified)
	}
	if filter.IsManaged != nil {
		query = query.Where("is_managed = ?", *filter.IsManaged)
	}

	// 지도 기반 반경 검색 (CenterLat, CenterLng, Radius가 모두 있을 때)
	if filter.CenterLat != nil && filter.CenterLng != nil && filter.Radius != nil {
		// Haversine 공식으로 거리 계산 (km 단위)
		distanceFormula := `(6371 * acos(
			cos(radians(?)) * cos(radians(latitude)) *
			cos(radians(longitude) - radians(?)) +
			sin(radians(?)) * sin(radians(latitude))
		))`

		// 반경 내 매장만 필터링 (Radius는 미터 단위이므로 km로 변환)
		radiusKm := *filter.Radius / 1000.0
		query = query.Where(distanceFormula+" <= ?",
			*filter.CenterLat, *filter.CenterLng, *filter.CenterLat, radiusKm)

		logger.Debug("Applying radius filter", map[string]interface{}{
			"center_lat": *filter.CenterLat,
			"center_lng": *filter.CenterLng,
			"radius_m":   *filter.Radius,
			"radius_km":  radiusKm,
		})
	}

	// 사용자 위치 기반 거리 계산 및 정렬
	// 지도 검색이 있으면 center 기준, 없으면 user 기준
	var sortByDistance bool
	var sortLat, sortLng float64

	if filter.CenterLat != nil && filter.CenterLng != nil {
		// 지도 검색: 지도 중심 기준으로 거리 계산
		sortByDistance = true
		sortLat = *filter.CenterLat
		sortLng = *filter.CenterLng
	} else if filter.UserLat != nil && filter.UserLng != nil {
		// 사용자 위치 기준 거리 계산
		sortByDistance = true
		sortLat = *filter.UserLat
		sortLng = *filter.UserLng
	}

	if sortByDistance {
		// MySQL/MariaDB의 Haversine 공식을 사용한 거리 계산
		// 결과는 km 단위
		distanceFormula := `(6371 * acos(
			cos(radians(?)) * cos(radians(latitude)) *
			cos(radians(longitude) - radians(?)) +
			sin(radians(?)) * sin(radians(latitude))
		))`

		// SELECT 절에 distance 컬럼 추가
		query = query.Select("stores.*, " + distanceFormula + " AS distance",
			sortLat, sortLng, sortLat)

		// 총 개수 조회 (SELECT distance 추가 전)
		countQuery := r.db.Model(&model.Store{})
		if filter.Region != "" {
			countQuery = countQuery.Where("region = ?", filter.Region)
		}
		if filter.District != "" {
			countQuery = countQuery.Where("district = ?", filter.District)
		}
		if filter.Search != "" {
			like := "%" + filter.Search + "%"
			// 매장명, 지역, 시군구, 동, 주소 모두 검색
			countQuery = countQuery.Where(
				"name LIKE ? OR region LIKE ? OR district LIKE ? OR dong LIKE ? OR address LIKE ?",
				like, like, like, like, like,
			)
		}
		if filter.IsVerified != nil {
			countQuery = countQuery.Where("is_verified = ?", *filter.IsVerified)
		}
		if filter.IsManaged != nil {
			countQuery = countQuery.Where("is_managed = ?", *filter.IsManaged)
		}

		// 반경 검색 필터도 count에 적용
		if filter.CenterLat != nil && filter.CenterLng != nil && filter.Radius != nil {
			distanceFormula := `(6371 * acos(
				cos(radians(?)) * cos(radians(latitude)) *
				cos(radians(longitude) - radians(?)) +
				sin(radians(?)) * sin(radians(latitude))
			))`
			radiusKm := *filter.Radius / 1000.0
			countQuery = countQuery.Where(distanceFormula+" <= ?",
				*filter.CenterLat, *filter.CenterLng, *filter.CenterLat, radiusKm)
		}

		var totalCount int64
		if err := countQuery.Count(&totalCount).Error; err != nil {
			logger.Error("Failed to count stores", err, map[string]interface{}{
				"region":   filter.Region,
				"district": filter.District,
			})
			return nil, err
		}

		// 거리순 정렬 후 페이지네이션 적용
		query = query.Order("distance ASC, name ASC")

		// 페이지네이션 적용
		if filter.Page > 0 && filter.PageSize > 0 {
			offset := (filter.Page - 1) * filter.PageSize
			query = query.Offset(offset).Limit(filter.PageSize)
		}

		var stores []model.Store
		if err := query.Find(&stores).Error; err != nil {
			logger.Error("Failed to find stores with distance sorting", err, map[string]interface{}{
				"region":   filter.Region,
				"district": filter.District,
				"sort_lat": sortLat,
				"sort_lng": sortLng,
			})
			return nil, err
		}

		if err := r.populateStoreStats(&stores); err != nil {
			logger.Error("Failed to populate store stats", err, nil)
			return nil, err
		}

		logger.Debug("Stores found and sorted by distance", map[string]interface{}{
			"count":       len(stores),
			"total_count": totalCount,
			"sort_lat":    sortLat,
			"sort_lng":    sortLng,
		})

		return &StoreListResult{
			Stores:     stores,
			TotalCount: totalCount,
		}, nil
	}

	// 사용자 위치가 없는 경우 기본 가나다순 정렬
	// 총 개수 조회
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		logger.Error("Failed to count stores", err, map[string]interface{}{
			"region":   filter.Region,
			"district": filter.District,
		})
		return nil, err
	}

	// 페이지네이션 적용
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
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
		"count":       len(stores),
		"total_count": totalCount,
	})

	return &StoreListResult{
		Stores:     stores,
		TotalCount: totalCount,
	}, nil
}

func (r *storeRepository) FindByID(id uint) (*model.Store, error) {
	logger.Debug("Finding store by ID", map[string]interface{}{
		"store_id": id,
	})

	query := r.db.Model(&model.Store{}).Preload("Tags").Preload("BusinessRegistration")

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

// BulkCreate creates or updates multiple stores in batches (UPSERT)
func (r *storeRepository) BulkCreate(stores []model.Store, batchSize int) error {
	logger.Info("Bulk creating/updating stores", map[string]interface{}{
		"total_count": len(stores),
		"batch_size":  batchSize,
	})

	// UPSERT: business_number가 중복되면 업데이트
	if err := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "business_number"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "branch_name", "slug", "region", "district", "dong",
			"address", "building_name", "floor", "unit", "postal_code",
			"longitude", "latitude", "updated_at",
		}),
	}).CreateInBatches(stores, batchSize).Error; err != nil {
		logger.Error("Failed to bulk create/update stores", err, map[string]interface{}{
			"total_count": len(stores),
			"batch_size":  batchSize,
		})
		return err
	}

	logger.Info("Bulk create/update completed", map[string]interface{}{
		"count": len(stores),
	})
	return nil
}
