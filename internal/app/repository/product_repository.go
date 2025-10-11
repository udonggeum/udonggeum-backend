package repository

import (
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

type ProductRepository interface {
	Create(product *model.Product) error
	FindAll() ([]model.Product, error)
	FindByID(id uint) (*model.Product, error)
	FindByCategory(category model.ProductCategory) ([]model.Product, error)
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
	})

	if err := r.db.Create(product).Error; err != nil {
		logger.Error("Failed to create product in database", err, map[string]interface{}{
			"name":     product.Name,
			"category": product.Category,
		})
		return err
	}

	logger.Debug("Product created in database", map[string]interface{}{
		"product_id": product.ID,
		"name":       product.Name,
		"category":   product.Category,
	})
	return nil
}

func (r *productRepository) FindAll() ([]model.Product, error) {
	logger.Debug("Finding all products in database", map[string]interface{}{
		"operation": "find_all",
	})

	var products []model.Product
	err := r.db.Find(&products).Error
	if err != nil {
		logger.Error("Failed to find all products in database", err, map[string]interface{}{
			"operation": "find_all",
		})
		return nil, err
	}

	logger.Debug("All products found in database", map[string]interface{}{
		"count": len(products),
	})
	return products, nil
}

func (r *productRepository) FindByID(id uint) (*model.Product, error) {
	logger.Debug("Finding product by ID in database", map[string]interface{}{
		"product_id": id,
	})

	var product model.Product
	err := r.db.First(&product, id).Error
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

	var products []model.Product
	err := r.db.Where("category = ?", category).Find(&products).Error
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
