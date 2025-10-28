package repository

import (
	"fmt"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

type ProductSort string

const (
	ProductSortPrice      ProductSort = "price"
	ProductSortCreatedAt  ProductSort = "created_at"
	ProductSortPopularity ProductSort = "popularity"
)

type ProductFilter struct {
	Region         string
	District       string
	Category       *model.ProductCategory
	StoreID        *uint
	Search         string
	SortBy         ProductSort
	SortAscending  bool
	PopularOnly    bool
	Limit          int
	Offset         int
	IncludeOptions bool
}

type ProductRepository interface {
	Create(product *model.Product) error
	FindAll() ([]model.Product, error)
	FindWithFilter(filter ProductFilter) ([]model.Product, error)
	FindByID(id uint) (*model.Product, error)
	FindByCategory(category model.ProductCategory) ([]model.Product, error)
	FindPopularByCategory(category model.ProductCategory, limit int, region, district string) ([]model.Product, error)
	Update(product *model.Product) error
	Delete(id uint) error
	UpdateStock(id uint, quantity int) error
}

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) Create(product *model.Product) error {
	logger.Debug("Creating product in database", map[string]interface{}{
		"name":     product.Name,
		"category": product.Category,
		"store_id": product.StoreID,
	})

	if err := r.db.Create(product).Error; err != nil {
		logger.Error("Failed to create product in database", err, map[string]interface{}{
			"name":     product.Name,
			"category": product.Category,
			"store_id": product.StoreID,
		})
		return err
	}

	logger.Debug("Product created in database", map[string]interface{}{
		"product_id": product.ID,
		"name":       product.Name,
		"store_id":   product.StoreID,
	})
	return nil
}

func (r *productRepository) baseQuery(includeOptions bool) *gorm.DB {
	query := r.db.Model(&model.Product{}).
		Preload("Store")
	if includeOptions {
		query = query.Preload("Options")
	}
	return query
}

func (r *productRepository) FindAll() ([]model.Product, error) {
	return r.FindWithFilter(ProductFilter{})
}

func (r *productRepository) FindWithFilter(filter ProductFilter) ([]model.Product, error) {
	logger.Debug("Finding products with filter", map[string]interface{}{
		"region":    filter.Region,
		"district":  filter.District,
		"category":  filter.Category,
		"store_id":  filter.StoreID,
		"search":    filter.Search,
		"sort_by":   filter.SortBy,
		"ascending": filter.SortAscending,
		"limit":     filter.Limit,
		"offset":    filter.Offset,
	})

	query := r.baseQuery(filter.IncludeOptions)

	if filter.Region != "" || filter.District != "" {
		query = query.Joins("JOIN stores ON stores.id = products.store_id")
		if filter.Region != "" {
			query = query.Where("stores.region = ?", filter.Region)
		}
		if filter.District != "" {
			query = query.Where("stores.district = ?", filter.District)
		}
		query = query.Select("products.*")
	}

	if filter.Category != nil {
		query = query.Where("products.category = ?", *filter.Category)
	}

	if filter.StoreID != nil {
		query = query.Where("products.store_id = ?", *filter.StoreID)
	}

	if filter.Search != "" {
		like := fmt.Sprintf("%%%s%%", filter.Search)
		query = query.Where("products.name LIKE ? OR products.description LIKE ?", like, like)
	}

	if filter.PopularOnly {
		query = query.Where("products.popularity_score > 0")
	}

	switch filter.SortBy {
	case ProductSortPrice:
		direction := "DESC"
		if filter.SortAscending {
			direction = "ASC"
		}
		query = query.Order("products.price " + direction)
	case ProductSortCreatedAt:
		direction := "DESC"
		if filter.SortAscending {
			direction = "ASC"
		}
		query = query.Order("products.created_at " + direction)
	case ProductSortPopularity:
		fallthrough
	default:
		direction := "DESC"
		if filter.SortAscending {
			direction = "ASC"
		}
		query = query.Order("products.popularity_score " + direction)
		query = query.Order("products.created_at DESC")
	}

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	var products []model.Product
	if err := query.Find(&products).Error; err != nil {
		logger.Error("Failed to find products with filter", err, map[string]interface{}{
			"region":   filter.Region,
			"district": filter.District,
			"search":   filter.Search,
		})
		return nil, err
	}

	logger.Debug("Products found with filter", map[string]interface{}{
		"count": len(products),
	})
	return products, nil
}

func (r *productRepository) FindByID(id uint) (*model.Product, error) {
	logger.Debug("Finding product by ID in database", map[string]interface{}{
		"product_id": id,
	})

	var product model.Product
	err := r.baseQuery(true).First(&product, id).Error
	if err != nil {
		logger.Error("Failed to find product by ID in database", err, map[string]interface{}{
			"product_id": id,
		})
		return nil, err
	}

	logger.Debug("Product found by ID in database", map[string]interface{}{
		"product_id": product.ID,
		"name":       product.Name,
	})
	return &product, nil
}

func (r *productRepository) FindByCategory(category model.ProductCategory) ([]model.Product, error) {
	logger.Debug("Finding products by category in database", map[string]interface{}{
		"category": category,
	})

	products, err := r.FindWithFilter(ProductFilter{Category: &category})
	if err != nil {
		logger.Error("Failed to find products by category in database", err, map[string]interface{}{
			"category": category,
		})
		return nil, err
	}

	logger.Debug("Products found by category in database", map[string]interface{}{
		"category": category,
		"count":    len(products),
	})
	return products, nil
}

func (r *productRepository) FindPopularByCategory(category model.ProductCategory, limit int, region, district string) ([]model.Product, error) {
	criteria := ProductFilter{
		Category:    &category,
		Limit:       limit,
		PopularOnly: true,
		Region:      region,
		District:    district,
		SortBy:      ProductSortPopularity,
	}
	return r.FindWithFilter(criteria)
}

func (r *productRepository) Update(product *model.Product) error {
	logger.Debug("Updating product in database", map[string]interface{}{
		"product_id": product.ID,
		"name":       product.Name,
	})

	if err := r.db.Save(product).Error; err != nil {
		logger.Error("Failed to update product in database", err, map[string]interface{}{
			"product_id": product.ID,
			"name":       product.Name,
		})
		return err
	}

	logger.Debug("Product updated in database", map[string]interface{}{
		"product_id": product.ID,
		"name":       product.Name,
	})
	return nil
}

func (r *productRepository) Delete(id uint) error {
	logger.Debug("Deleting product from database", map[string]interface{}{
		"product_id": id,
	})

	if err := r.db.Delete(&model.Product{}, id).Error; err != nil {
		logger.Error("Failed to delete product from database", err, map[string]interface{}{
			"product_id": id,
		})
		return err
	}

	logger.Debug("Product deleted from database", map[string]interface{}{
		"product_id": id,
	})
	return nil
}

func (r *productRepository) UpdateStock(id uint, quantity int) error {
	logger.Debug("Updating product stock in database", map[string]interface{}{
		"product_id": id,
		"quantity":   quantity,
	})

	if err := r.db.Model(&model.Product{}).Where("id = ?", id).
		Update("stock_quantity", gorm.Expr("stock_quantity + ?", quantity)).Error; err != nil {
		logger.Error("Failed to update product stock in database", err, map[string]interface{}{
			"product_id": id,
			"quantity":   quantity,
		})
		return err
	}

	logger.Debug("Product stock updated in database", map[string]interface{}{
		"product_id": id,
		"quantity":   quantity,
	})
	return nil
}
