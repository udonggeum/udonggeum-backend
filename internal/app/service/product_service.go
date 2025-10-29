package service

import (
	"errors"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

var (
	ErrProductNotFound      = errors.New("product not found")
	ErrInsufficientStock    = errors.New("insufficient stock")
	ErrInvalidProductOption = errors.New("invalid product option")
	ErrProductAccessDenied  = errors.New("product access denied")
)

type ProductSort string

const (
	ProductSortPrice      ProductSort = "price"
	ProductSortCreatedAt  ProductSort = "created_at"
	ProductSortPopularity ProductSort = "popularity"
)

type ProductListOptions struct {
	Region         string
	District       string
	Category       *model.ProductCategory
	Material       *model.ProductMaterial
	StoreID        *uint
	Search         string
	Sort           ProductSort
	SortAscending  bool
	PopularOnly    bool
	Limit          int
	Offset         int
	IncludeOptions bool
}

type ProductFilterSummary struct {
	Categories []model.ProductCategory
	Materials  []model.ProductMaterial
}

type ProductService interface {
	ListProducts(opts ProductListOptions) ([]model.Product, error)
	GetProductByID(id uint) (*model.Product, error)
	GetProductsByCategory(category model.ProductCategory) ([]model.Product, error)
	GetPopularProducts(category model.ProductCategory, region, district string, limit int) ([]model.Product, error)
	GetAvailableFilters() (ProductFilterSummary, error)
	CreateProduct(product *model.Product) error
	UpdateProduct(userID uint, product *model.Product) error
	DeleteProduct(userID uint, id uint) error
	CheckStock(productID uint, productOptionID *uint, quantity int) error
}

type productService struct {
	productRepo       repository.ProductRepository
	productOptionRepo repository.ProductOptionRepository
}

func NewProductService(productRepo repository.ProductRepository, productOptionRepo ...repository.ProductOptionRepository) ProductService {
	var optionRepo repository.ProductOptionRepository
	if len(productOptionRepo) > 0 {
		optionRepo = productOptionRepo[0]
	}
	return &productService{
		productRepo:       productRepo,
		productOptionRepo: optionRepo,
	}
}

func (s *productService) ListProducts(opts ProductListOptions) ([]model.Product, error) {
	logger.Debug("Listing products", map[string]interface{}{
		"region":   opts.Region,
		"district": opts.District,
		"category": opts.Category,
		"material": opts.Material,
		"search":   opts.Search,
		"sort":     opts.Sort,
		"limit":    opts.Limit,
		"offset":   opts.Offset,
	})

	filter := repository.ProductFilter{
		Region:         opts.Region,
		District:       opts.District,
		StoreID:        opts.StoreID,
		Search:         opts.Search,
		Material:       opts.Material,
		SortAscending:  opts.SortAscending,
		PopularOnly:    opts.PopularOnly,
		Limit:          opts.Limit,
		Offset:         opts.Offset,
		IncludeOptions: opts.IncludeOptions,
	}

	switch opts.Sort {
	case ProductSortPrice:
		filter.SortBy = repository.ProductSortPrice
	case ProductSortCreatedAt:
		filter.SortBy = repository.ProductSortCreatedAt
	case ProductSortPopularity:
		fallthrough
	default:
		filter.SortBy = repository.ProductSortPopularity
	}

	if opts.Category != nil {
		filter.Category = opts.Category
	}

	products, err := s.productRepo.FindWithFilter(filter)
	if err != nil {
		logger.Error("Failed to list products", err)
		return nil, err
	}

	logger.Info("Products listed", map[string]interface{}{
		"count": len(products),
	})
	return products, nil
}

func (s *productService) GetProductByID(id uint) (*model.Product, error) {
	logger.Debug("Fetching product by ID", map[string]interface{}{
		"product_id": id,
	})

	product, err := s.productRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Product not found", map[string]interface{}{
				"product_id": id,
			})
			return nil, ErrProductNotFound
		}
		logger.Error("Failed to fetch product", err, map[string]interface{}{
			"product_id": id,
		})
		return nil, err
	}
	return product, nil
}

func (s *productService) GetProductsByCategory(category model.ProductCategory) ([]model.Product, error) {
	logger.Debug("Fetching products by category", map[string]interface{}{
		"category": category,
	})

	products, err := s.productRepo.FindByCategory(category)
	if err != nil {
		logger.Error("Failed to fetch products by category", err, map[string]interface{}{
			"category": category,
		})
		return nil, err
	}

	logger.Info("Products fetched by category", map[string]interface{}{
		"category": category,
		"count":    len(products),
	})
	return products, nil
}

func (s *productService) GetPopularProducts(category model.ProductCategory, region, district string, limit int) ([]model.Product, error) {
	logger.Debug("Fetching popular products", map[string]interface{}{
		"category": category,
		"region":   region,
		"district": district,
		"limit":    limit,
	})

	products, err := s.productRepo.FindPopularByCategory(category, limit, region, district)
	if err != nil {
		logger.Error("Failed to fetch popular products", err, map[string]interface{}{
			"category": category,
		})
		return nil, err
	}

	logger.Info("Popular products fetched", map[string]interface{}{
		"category": category,
		"count":    len(products),
	})
	return products, nil
}

func (s *productService) GetAvailableFilters() (ProductFilterSummary, error) {
	logger.Debug("Fetching product filter metadata", nil)

	attrs, err := s.productRepo.ListAttributes()
	if err != nil {
		logger.Error("Failed to fetch product filter metadata", err, nil)
		return ProductFilterSummary{}, err
	}

	summary := ProductFilterSummary{
		Categories: attrs.Categories,
		Materials:  attrs.Materials,
	}

	logger.Info("Product filter metadata fetched", map[string]interface{}{
		"category_count": len(summary.Categories),
		"material_count": len(summary.Materials),
	})

	return summary, nil
}

func (s *productService) CreateProduct(product *model.Product) error {
	if product.Category == "" {
		product.Category = model.CategoryOther
	}
	if product.Material == "" {
		product.Material = model.MaterialOther
	}

	logger.Info("Creating new product", map[string]interface{}{
		"name":     product.Name,
		"category": product.Category,
		"material": product.Material,
		"store_id": product.StoreID,
	})

	if err := s.productRepo.Create(product); err != nil {
		logger.Error("Failed to create product", err, map[string]interface{}{
			"name":     product.Name,
			"category": product.Category,
			"material": product.Material,
		})
		return err
	}

	logger.Info("Product created successfully", map[string]interface{}{
		"product_id": product.ID,
		"name":       product.Name,
	})
	return nil
}

func (s *productService) UpdateProduct(userID uint, product *model.Product) error {
	logger.Info("Updating product", map[string]interface{}{
		"product_id": product.ID,
		"name":       product.Name,
		"category":   product.Category,
		"material":   product.Material,
		"user_id":    userID,
	})

	existing, err := s.productRepo.FindByID(product.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Cannot update: product not found", map[string]interface{}{
				"product_id": product.ID,
			})
			return ErrProductNotFound
		}
		logger.Error("Failed to check product existence", err, map[string]interface{}{
			"product_id": product.ID,
		})
		return err
	}

	if existing.Store.UserID != userID {
		logger.Warn("Product update forbidden", map[string]interface{}{
			"product_id": product.ID,
			"user_id":    userID,
			"store_id":   existing.StoreID,
		})
		return ErrProductAccessDenied
	}

	if product.StoreID != 0 && product.StoreID != existing.StoreID {
		logger.Warn("Attempt to change product store rejected", map[string]interface{}{
			"product_id":      product.ID,
			"user_id":         userID,
			"existing_store":  existing.StoreID,
			"requested_store": product.StoreID,
		})
		return ErrProductAccessDenied
	}

	product.StoreID = existing.StoreID
	if product.Category == "" {
		product.Category = existing.Category
	}
	if product.Material == "" {
		product.Material = existing.Material
	}

	if err := s.productRepo.Update(product); err != nil {
		logger.Error("Failed to update product", err, map[string]interface{}{
			"product_id": product.ID,
			"category":   product.Category,
			"material":   product.Material,
		})
		return err
	}

	logger.Info("Product updated successfully", map[string]interface{}{
		"product_id": product.ID,
		"name":       product.Name,
		"category":   product.Category,
		"material":   product.Material,
	})
	return nil
}

func (s *productService) DeleteProduct(userID uint, id uint) error {
	logger.Info("Deleting product", map[string]interface{}{
		"product_id": id,
		"user_id":    userID,
	})

	existing, err := s.productRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Cannot delete: product not found", map[string]interface{}{
				"product_id": id,
			})
			return ErrProductNotFound
		}
		logger.Error("Failed to check product existence", err, map[string]interface{}{
			"product_id": id,
		})
		return err
	}

	if existing.Store.UserID != userID {
		logger.Warn("Product delete forbidden", map[string]interface{}{
			"product_id": id,
			"user_id":    userID,
			"store_id":   existing.StoreID,
		})
		return ErrProductAccessDenied
	}

	if err := s.productRepo.Delete(id); err != nil {
		logger.Error("Failed to delete product", err, map[string]interface{}{
			"product_id": id,
		})
		return err
	}

	logger.Info("Product deleted successfully", map[string]interface{}{
		"product_id": id,
	})
	return nil
}

func (s *productService) CheckStock(productID uint, productOptionID *uint, quantity int) error {
	logger.Debug("Checking product stock", map[string]interface{}{
		"product_id":        productID,
		"product_option_id": productOptionID,
		"quantity":          quantity,
	})

	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Product not found for stock check", map[string]interface{}{
				"product_id": productID,
			})
			return ErrProductNotFound
		}
		logger.Error("Failed to fetch product for stock check", err, map[string]interface{}{
			"product_id": productID,
		})
		return err
	}

	if product.StockQuantity < quantity {
		logger.Warn("Insufficient product stock", map[string]interface{}{
			"product_id":      productID,
			"requested":       quantity,
			"available_stock": product.StockQuantity,
		})
		return ErrInsufficientStock
	}

	if productOptionID != nil {
		if s.productOptionRepo == nil {
			logger.Warn("Product option repository unavailable for stock check", map[string]interface{}{
				"product_id":        productID,
				"product_option_id": *productOptionID,
			})
			return ErrInvalidProductOption
		}
		option, err := s.productOptionRepo.FindByID(*productOptionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Warn("Product option not found for stock check", map[string]interface{}{
					"product_option_id": *productOptionID,
				})
				return ErrInvalidProductOption
			}
			logger.Error("Failed to fetch product option", err, map[string]interface{}{
				"product_option_id": *productOptionID,
			})
			return err
		}

		if option.ProductID != productID {
			logger.Warn("Product option does not belong to product", map[string]interface{}{
				"product_id":        productID,
				"product_option_id": *productOptionID,
			})
			return ErrInvalidProductOption
		}

		if option.StockQuantity < quantity {
			logger.Warn("Insufficient product option stock", map[string]interface{}{
				"product_option_id": *productOptionID,
				"requested":         quantity,
				"available_stock":   option.StockQuantity,
			})
			return ErrInsufficientStock
		}
	}

	logger.Debug("Product stock sufficient", map[string]interface{}{
		"product_id": productID,
	})
	return nil
}
