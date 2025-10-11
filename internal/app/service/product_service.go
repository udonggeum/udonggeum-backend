package service

import (
	"errors"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

var (
	ErrProductNotFound   = errors.New("product not found")
	ErrInsufficientStock = errors.New("insufficient stock")
)

type ProductService interface {
	GetAllProducts() ([]model.Product, error)
	GetProductByID(id uint) (*model.Product, error)
	GetProductsByCategory(category model.ProductCategory) ([]model.Product, error)
	CreateProduct(product *model.Product) error
	UpdateProduct(product *model.Product) error
	DeleteProduct(id uint) error
	CheckStock(productID uint, quantity int) error
}

type productService struct {
	productRepo repository.ProductRepository
}

func NewProductService(productRepo repository.ProductRepository) ProductService {
	return &productService{
		productRepo: productRepo,
	}
}

func (s *productService) GetAllProducts() ([]model.Product, error) {
	logger.Debug("Fetching all products")

	products, err := s.productRepo.FindAll()
	if err != nil {
		logger.Error("Failed to fetch products", err)
		return nil, err
	}

	logger.Info("Products fetched successfully", map[string]interface{}{
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

func (s *productService) CreateProduct(product *model.Product) error {
	logger.Info("Creating new product", map[string]interface{}{
		"name":     product.Name,
		"category": product.Category,
		"price":    product.Price,
		"stock":    product.StockQuantity,
	})

	if err := s.productRepo.Create(product); err != nil {
		logger.Error("Failed to create product", err, map[string]interface{}{
			"name":     product.Name,
			"category": product.Category,
		})
		return err
	}

	logger.Info("Product created successfully", map[string]interface{}{
		"product_id": product.ID,
		"name":       product.Name,
	})
	return nil
}

func (s *productService) UpdateProduct(product *model.Product) error {
	logger.Info("Updating product", map[string]interface{}{
		"product_id": product.ID,
		"name":       product.Name,
	})

	// Check if product exists
	_, err := s.productRepo.FindByID(product.ID)
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

	if err := s.productRepo.Update(product); err != nil {
		logger.Error("Failed to update product", err, map[string]interface{}{
			"product_id": product.ID,
		})
		return err
	}

	logger.Info("Product updated successfully", map[string]interface{}{
		"product_id": product.ID,
		"name":       product.Name,
	})
	return nil
}

func (s *productService) DeleteProduct(id uint) error {
	logger.Info("Deleting product", map[string]interface{}{
		"product_id": id,
	})

	// Check if product exists
	product, err := s.productRepo.FindByID(id)
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

	if err := s.productRepo.Delete(id); err != nil {
		logger.Error("Failed to delete product", err, map[string]interface{}{
			"product_id": id,
		})
		return err
	}

	logger.Info("Product deleted successfully", map[string]interface{}{
		"product_id": id,
		"name":       product.Name,
	})
	return nil
}

func (s *productService) CheckStock(productID uint, quantity int) error {
	logger.Debug("Checking product stock", map[string]interface{}{
		"product_id":         productID,
		"requested_quantity": quantity,
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
		logger.Warn("Insufficient stock", map[string]interface{}{
			"product_id":         productID,
			"available_stock":    product.StockQuantity,
			"requested_quantity": quantity,
		})
		return ErrInsufficientStock
	}

	logger.Debug("Stock check passed", map[string]interface{}{
		"product_id":         productID,
		"available_stock":    product.StockQuantity,
		"requested_quantity": quantity,
	})
	return nil
}
