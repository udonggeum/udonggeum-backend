package service

import (
	"errors"
	"sort"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"github.com/ikkim/udonggeum-backend/pkg/util"
	"gorm.io/gorm"
)

var (
	ErrStoreNotFound     = errors.New("store not found")
	ErrStoreAccessDenied = errors.New("store access denied")
)

type StoreListOptions struct {
	Region          string
	District        string
	Search          string
	IncludeProducts bool
	BuyingGold      bool     // 금 매입 가능 매장만 조회
	UserLat         *float64 // 사용자 위도 (거리순 정렬용)
	UserLng         *float64 // 사용자 경도 (거리순 정렬용)
}

type StoreLocationSummary struct {
	Region     string `json:"region"`
	District   string `json:"district"`
	StoreCount int64  `json:"store_count"`
}

type StoreService interface {
	ListStores(opts StoreListOptions) ([]model.Store, error)
	GetStoreByID(id uint, includeProducts bool) (*model.Store, error)
	GetStoresByUserID(userID uint) ([]model.Store, error)
	ListLocations() ([]StoreLocationSummary, error)
	CreateStore(store *model.Store) (*model.Store, error)
	UpdateStore(userID uint, storeID uint, input StoreMutation) (*model.Store, error)
	DeleteStore(userID uint, storeID uint) error
	ToggleStoreLike(storeID, userID uint) (bool, error)
	IsStoreLiked(storeID, userID uint) (bool, error)
	GetUserLikedStores(userID uint) ([]model.Store, error)
	GetUserLikedStoreIDs(userID uint) ([]uint, error)
	PromoteUserToAdmin(userID uint) error
}

type storeService struct {
	storeRepo repository.StoreRepository
	userRepo  repository.UserRepository
}

type StoreMutation struct {
	Name        string
	Region      string
	District    string
	Address     string
	Latitude    *float64
	Longitude   *float64
	PhoneNumber string
	ImageURL    string
	Description string
	OpenTime    string
	CloseTime   string
	TagIDs      []uint // 태그 ID 배열
}

func NewStoreService(storeRepo repository.StoreRepository, userRepo repository.UserRepository) StoreService {
	return &storeService{
		storeRepo: storeRepo,
		userRepo:  userRepo,
	}
}

func (s *storeService) ListStores(opts StoreListOptions) ([]model.Store, error) {
	logger.Debug("Listing stores", map[string]interface{}{
		"region":   opts.Region,
		"district": opts.District,
		"user_lat": opts.UserLat,
		"user_lng": opts.UserLng,
	})

	stores, err := s.storeRepo.FindAll(repository.StoreFilter{
		Region:          opts.Region,
		District:        opts.District,
		Search:          opts.Search,
		IncludeProducts: opts.IncludeProducts,
		BuyingGold:      opts.BuyingGold,
	})
	if err != nil {
		logger.Error("Failed to list stores", err)
		return nil, err
	}

	// If user location provided, sort by distance
	if opts.UserLat != nil && opts.UserLng != nil {
		type storeWithDistance struct {
			store    model.Store
			distance float64
		}

		storesWithDistance := make([]storeWithDistance, 0, len(stores))

		for _, store := range stores {
			// Calculate distance if store has coordinates
			distance := 999999.0 // Default large distance for stores without coordinates
			if store.Latitude != nil && store.Longitude != nil {
				distance = util.CalculateDistance(
					*opts.UserLat, *opts.UserLng,
					*store.Latitude, *store.Longitude,
				)
			}

			storesWithDistance = append(storesWithDistance, storeWithDistance{
				store:    store,
				distance: distance,
			})
		}

		// Sort by distance
		sort.Slice(storesWithDistance, func(i, j int) bool {
			return storesWithDistance[i].distance < storesWithDistance[j].distance
		})

		// Extract sorted stores
		sortedStores := make([]model.Store, len(storesWithDistance))
		for i, swd := range storesWithDistance {
			sortedStores[i] = swd.store
		}

		logger.Info("Stores fetched and sorted by distance", map[string]interface{}{
			"count":    len(sortedStores),
			"user_lat": *opts.UserLat,
			"user_lng": *opts.UserLng,
		})

		return sortedStores, nil
	}

	logger.Info("Stores fetched", map[string]interface{}{
		"count": len(stores),
	})
	return stores, nil
}

func (s *storeService) GetStoreByID(id uint, includeProducts bool) (*model.Store, error) {
	logger.Debug("Fetching store by ID", map[string]interface{}{
		"store_id": id,
	})

	store, err := s.storeRepo.FindByID(id, includeProducts)
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

	if err := s.storeRepo.Create(store); err != nil {
		logger.Error("Failed to create store", err, map[string]interface{}{
			"name":    store.Name,
			"user_id": store.UserID,
		})
		return nil, err
	}

	// Update user's nickname to store name (매장 등록 시 무조건 닉네임 변경)
	user, err := s.userRepo.FindByID(store.UserID)
	if err == nil {
		user.Nickname = store.Name
		if err := s.userRepo.Update(user); err != nil {
			logger.Warn("Failed to update user nickname after store creation", map[string]interface{}{
				"user_id":    store.UserID,
				"store_name": store.Name,
				"error":      err.Error(),
			})
			// Don't fail the entire operation if nickname update fails
		} else {
			logger.Info("User nickname updated to store name", map[string]interface{}{
				"user_id":  store.UserID,
				"nickname": store.Name,
			})
		}
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

	existing, err := s.storeRepo.FindByID(storeID, false)
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

	if existing.UserID != userID {
		logger.Warn("Store update forbidden", map[string]interface{}{
			"store_id": storeID,
			"user_id":  userID,
		})
		return nil, ErrStoreAccessDenied
	}

	// Check if store name changed
	storeNameChanged := existing.Name != input.Name

	existing.Name = input.Name
	existing.Region = input.Region
	existing.District = input.District

	// Address handling with geocoding
	addressChanged := existing.Address != input.Address
	existing.Address = input.Address

	// If address changed, geocode it to get new coordinates
	if addressChanged && input.Address != "" {
		lat, lng, err := util.GeocodeAddress(input.Address)
		if err != nil {
			logger.Warn("Failed to geocode store address, using provided coordinates", map[string]interface{}{
				"store_id": storeID,
				"address":  input.Address,
				"error":    err.Error(),
			})
			// Fall back to provided coordinates if geocoding fails
			existing.Latitude = input.Latitude
			existing.Longitude = input.Longitude
		} else {
			existing.Latitude = lat
			existing.Longitude = lng
			logger.Info("Successfully geocoded store address", map[string]interface{}{
				"store_id":  storeID,
				"address":   input.Address,
				"latitude":  lat,
				"longitude": lng,
			})
		}
	} else if !addressChanged {
		// If address didn't change, keep existing coordinates or use provided ones
		if input.Latitude != nil {
			existing.Latitude = input.Latitude
		}
		if input.Longitude != nil {
			existing.Longitude = input.Longitude
		}
	} else {
		// Address cleared
		existing.Latitude = nil
		existing.Longitude = nil
	}

	existing.PhoneNumber = input.PhoneNumber
	existing.ImageURL = input.ImageURL
	existing.Description = input.Description
	existing.OpenTime = input.OpenTime
	existing.CloseTime = input.CloseTime

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
	if storeNameChanged {
		user, err := s.userRepo.FindByID(userID)
		if err == nil && user.Role == model.RoleAdmin {
			user.Nickname = input.Name
			if err := s.userRepo.Update(user); err != nil {
				logger.Warn("Failed to update user nickname after store name change", map[string]interface{}{
					"user_id":    userID,
					"store_name": input.Name,
					"error":      err.Error(),
				})
				// Don't fail the entire operation if nickname update fails
			} else {
				logger.Info("User nickname updated to new store name", map[string]interface{}{
					"user_id":  userID,
					"nickname": input.Name,
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

	existing, err := s.storeRepo.FindByID(storeID, false)
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

	if existing.UserID != userID {
		logger.Warn("Store delete forbidden", map[string]interface{}{
			"store_id": storeID,
			"user_id":  userID,
		})
		return ErrStoreAccessDenied
	}

	if err := s.storeRepo.Delete(storeID); err != nil {
		logger.Error("Failed to delete store", err, map[string]interface{}{
			"store_id": storeID,
		})
		return err
	}

	logger.Info("Store deleted", map[string]interface{}{
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
	_, err := s.storeRepo.FindByID(storeID, false)
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
