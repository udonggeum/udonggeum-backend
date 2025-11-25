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
}

type StoreMutation struct {
	Name        string
	Region      string
	District    string
	Address     string
	PhoneNumber string
	ImageURL    string
	Description string
	OpenTime    string
	CloseTime   string
}

func NewStoreService(storeRepo repository.StoreRepository) StoreService {
	return &storeService{storeRepo: storeRepo}
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

	existing.Name = input.Name
	existing.Region = input.Region
	existing.District = input.District
	existing.Address = input.Address
	existing.PhoneNumber = input.PhoneNumber
	existing.ImageURL = input.ImageURL
	existing.Description = input.Description
	existing.OpenTime = input.OpenTime
	existing.CloseTime = input.CloseTime

	if err := s.storeRepo.Update(existing); err != nil {
		logger.Error("Failed to update store", err, map[string]interface{}{
			"store_id": storeID,
		})
		return nil, err
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
