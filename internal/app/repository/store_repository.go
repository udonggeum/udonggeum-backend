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
	ListLocations() ([]StoreLocation, error)
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

	query := r.db.Model(&model.Store{})
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

	var stores []model.Store
	if err := query.Order("name ASC").Find(&stores).Error; err != nil {
		logger.Error("Failed to find stores", err, map[string]interface{}{
			"region":   filter.Region,
			"district": filter.District,
		})
		return nil, err
	}

	if err := r.populateCategoryCounts(&stores); err != nil {
		logger.Error("Failed to populate category counts for stores", err, nil)
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

	query := r.db.Model(&model.Store{})
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

	logger.Debug("Store found", map[string]interface{}{
		"store_id": store.ID,
		"name":     store.Name,
	})
	return &store, nil
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

func (r *storeRepository) populateCategoryCounts(stores *[]model.Store) error {
	if len(*stores) == 0 {
		return nil
	}

	storeIDs := make([]uint, len(*stores))
	storeIndex := make(map[uint]*model.Store, len(*stores))
	for i := range *stores {
		store := &(*stores)[i]
		storeIDs[i] = store.ID
		store.CategoryCounts = initializeCategoryCounts()
		storeIndex[store.ID] = store
	}

	type categoryCountRow struct {
		StoreID  uint
		Category model.ProductCategory
		Count    int64
	}

	var rows []categoryCountRow
	if err := r.db.Model(&model.Product{}).
		Select("store_id, category, COUNT(*) as count").
		Where("store_id IN ?", storeIDs).
		Group("store_id, category").
		Scan(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		if store, ok := storeIndex[row.StoreID]; ok {
			store.CategoryCounts[row.Category] = int(row.Count)
		}
	}
	return nil
}

func initializeCategoryCounts() map[model.ProductCategory]int {
	categories := productCategories()
	counts := make(map[model.ProductCategory]int, len(categories))
	for _, category := range categories {
		counts[category] = 0
	}
	return counts
}

func productCategories() []model.ProductCategory {
	return []model.ProductCategory{
		model.CategoryRing,
		model.CategoryBracelet,
		model.CategoryNecklace,
		model.CategoryEarring,
		model.CategoryOther,
	}
}
