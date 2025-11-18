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
	ProductSortWishlist   ProductSort = "wishlist"
	ProductSortViewCount  ProductSort = "view_count"
)

type ProductFilter struct {
	Region         string
	District       string
	Category       *model.ProductCategory
	Material       *model.ProductMaterial
	StoreID        *uint
	Search         string
	SortBy         ProductSort
	SortAscending  bool
	Limit          int
	Offset         int
	IncludeOptions bool
}

type ProductAttributes struct {
	Categories []model.ProductCategory
	Materials  []model.ProductMaterial
}

type ProductRepository interface {
	Create(product *model.Product) error
	FindAll() ([]model.Product, error)
	FindWithFilter(filter ProductFilter) ([]model.Product, error)
	FindByID(id uint) (*model.Product, error)
	FindByCategory(category model.ProductCategory) ([]model.Product, error)
	FindPopularByCategory(category *model.ProductCategory, limit int, region, district string) ([]model.Product, error)
	ListAttributes() (ProductAttributes, error)
	Update(product *model.Product) error
	Delete(id uint) error
	UpdateStock(id uint, quantity int) error
	IncrementViewCount(id uint) error
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
		"material": product.Material,
		"store_id": product.StoreID,
	})

	if err := r.db.Create(product).Error; err != nil {
		logger.Error("Failed to create product in database", err, map[string]interface{}{
			"name":     product.Name,
			"category": product.Category,
			"material": product.Material,
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
		"material":  filter.Material,
		"store_id":  filter.StoreID,
		"search":    filter.Search,
		"sort_by":   filter.SortBy,
		"ascending": filter.SortAscending,
		"limit":     filter.Limit,
		"offset":    filter.Offset,
	})

	query := r.baseQuery(filter.IncludeOptions)

	wishlistCountsSubquery := r.db.Table("wishlist_items").
		Select("wishlist_items.product_id, COUNT(*) AS count").
		Where("wishlist_items.deleted_at IS NULL").
		Group("wishlist_items.product_id")

	query = query.Joins("LEFT JOIN (?) AS wishlist_counts ON wishlist_counts.product_id = products.id", wishlistCountsSubquery)

	if filter.Region != "" || filter.District != "" {
		query = query.Joins("JOIN stores ON stores.id = products.store_id")
		if filter.Region != "" {
			query = query.Where("stores.region = ?", filter.Region)
		}
		if filter.District != "" {
			query = query.Where("stores.district = ?", filter.District)
		}
	}

	query = query.Select("products.*, COALESCE(wishlist_counts.count, 0) AS wishlist_count")

	if filter.Category != nil {
		query = query.Where("products.category = ?", *filter.Category)
	}

	if filter.Material != nil {
		query = query.Where("products.material = ?", *filter.Material)
	}

	if filter.StoreID != nil {
		query = query.Where("products.store_id = ?", *filter.StoreID)
	}

	if filter.Search != "" {
		like := fmt.Sprintf("%%%s%%", filter.Search)
		query = query.Where("products.name LIKE ? OR products.description LIKE ?", like, like)
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
	case ProductSortWishlist:
		direction := "DESC"
		if filter.SortAscending {
			direction = "ASC"
		}
		query = query.Order("COALESCE(wishlist_counts.count, 0) " + direction)
		query = query.Order("products.created_at DESC")
	case ProductSortViewCount:
		direction := "DESC"
		if filter.SortAscending {
			direction = "ASC"
		}
		query = query.Order("products.view_count " + direction)
		query = query.Order("products.created_at DESC")
	case ProductSortPopularity:
		fallthrough
	default:
		direction := "DESC"
		if filter.SortAscending {
			direction = "ASC"
		}
		popularityExpr := "COALESCE(wishlist_counts.count, 0) * CASE WHEN products.view_count > 0 THEN products.view_count ELSE 0 END"
		query = query.Order(popularityExpr + " " + direction)
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

func (r *productRepository) ListAttributes() (ProductAttributes, error) {
	logger.Debug("Listing product attributes", nil)

	result := ProductAttributes{}

	var categoryValues []string
	if err := r.db.Model(&model.Product{}).
		Where("category IS NOT NULL AND category <> ''").
		Distinct().
		Order("category ASC").
		Pluck("category", &categoryValues).Error; err != nil {
		logger.Error("Failed to fetch distinct categories", err, nil)
		return result, err
	}

	for _, category := range categoryValues {
		result.Categories = append(result.Categories, model.ProductCategory(category))
	}

	var materialValues []string
	if err := r.db.Model(&model.Product{}).
		Where("material IS NOT NULL AND material <> ''").
		Distinct().
		Order("material ASC").
		Pluck("material", &materialValues).Error; err != nil {
		logger.Error("Failed to fetch distinct materials", err, nil)
		return result, err
	}

	for _, material := range materialValues {
		result.Materials = append(result.Materials, model.ProductMaterial(material))
	}

	logger.Debug("Product attributes listed", map[string]interface{}{
		"category_count": len(result.Categories),
		"material_count": len(result.Materials),
	})
	return result, nil
}

func (r *productRepository) FindPopularByCategory(category *model.ProductCategory, limit int, region, district string) ([]model.Product, error) {
	criteria := ProductFilter{
		Limit:    limit,
		Region:   region,
		District: district,
		SortBy:   ProductSortPopularity,
	}
	if category != nil {
		criteria.Category = category
	}
	return r.FindWithFilter(criteria)
}

func (r *productRepository) Update(product *model.Product) error {
	logger.Debug("Updating product in database", map[string]interface{}{
		"product_id": product.ID,
		"name":       product.Name,
		"category":   product.Category,
		"material":   product.Material,
	})

	if err := r.db.Save(product).Error; err != nil {
		logger.Error("Failed to update product in database", err, map[string]interface{}{
			"product_id": product.ID,
			"name":       product.Name,
			"category":   product.Category,
			"material":   product.Material,
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

func (r *productRepository) IncrementViewCount(id uint) error {
	logger.Debug("Incrementing product view count in database", map[string]interface{}{
		"product_id": id,
	})

	if err := r.db.Model(&model.Product{}).Where("id = ?", id).
		Update("view_count", gorm.Expr("view_count + ?", 1)).Error; err != nil {
		logger.Error("Failed to increment product view count in database", err, map[string]interface{}{
			"product_id": id,
		})
		return err
	}

	logger.Debug("Product view count incremented in database", map[string]interface{}{
		"product_id": id,
	})
	return nil
}
