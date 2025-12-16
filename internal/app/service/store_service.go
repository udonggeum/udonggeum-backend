package service

import (
	"errors"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
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
	BuyingGold      bool // 금 매입 가능 매장만 조회
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

	if err := s.storeRepo.Create(store); err != nil {
		logger.Error("Failed to create store", err, map[string]interface{}{
			"name":    store.Name,
			"user_id": store.UserID,
		})
		return nil, err
	}

	// Update user's nickname to store name for admin users
	user, err := s.userRepo.FindByID(store.UserID)
	if err == nil && user.Role == model.RoleAdmin {
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
				"user_id": store.UserID,
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
	existing.Address = input.Address
	existing.Latitude = input.Latitude
	existing.Longitude = input.Longitude
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
