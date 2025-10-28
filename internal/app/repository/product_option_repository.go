package repository

import (
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	"gorm.io/gorm"
)

type ProductOptionRepository interface {
	Create(option *model.ProductOption) error
	FindByID(id uint) (*model.ProductOption, error)
	FindByProductID(productID uint) ([]model.ProductOption, error)
	Update(option *model.ProductOption) error
	UpdateStock(id uint, quantity int) error
}

type productOptionRepository struct {
	db *gorm.DB
}

func NewProductOptionRepository(db *gorm.DB) ProductOptionRepository {
	return &productOptionRepository{db: db}
}

func (r *productOptionRepository) Create(option *model.ProductOption) error {
	logger.Debug("Creating product option", map[string]interface{}{
		"product_id": option.ProductID,
		"name":       option.Name,
		"value":      option.Value,
	})

	if err := r.db.Create(option).Error; err != nil {
		logger.Error("Failed to create product option", err, map[string]interface{}{
			"product_id": option.ProductID,
			"name":       option.Name,
		})
		return err
	}

	logger.Debug("Product option created", map[string]interface{}{
		"option_id": option.ID,
	})
	return nil
}

func (r *productOptionRepository) FindByID(id uint) (*model.ProductOption, error) {
	logger.Debug("Finding product option by ID", map[string]interface{}{
		"option_id": id,
	})

	var option model.ProductOption
	if err := r.db.First(&option, id).Error; err != nil {
		logger.Error("Failed to find product option", err, map[string]interface{}{
			"option_id": id,
		})
		return nil, err
	}

	logger.Debug("Product option found", map[string]interface{}{
		"option_id":  option.ID,
		"product_id": option.ProductID,
	})
	return &option, nil
}

func (r *productOptionRepository) FindByProductID(productID uint) ([]model.ProductOption, error) {
	logger.Debug("Finding product options by product", map[string]interface{}{
		"product_id": productID,
	})

	var options []model.ProductOption
	if err := r.db.Where("product_id = ?", productID).Order("is_default DESC, additional_price ASC").Find(&options).Error; err != nil {
		logger.Error("Failed to find product options", err, map[string]interface{}{
			"product_id": productID,
		})
		return nil, err
	}

	logger.Debug("Product options found", map[string]interface{}{
		"count": len(options),
	})
	return options, nil
}

func (r *productOptionRepository) Update(option *model.ProductOption) error {
	logger.Debug("Updating product option", map[string]interface{}{
		"option_id": option.ID,
	})

	if err := r.db.Save(option).Error; err != nil {
		logger.Error("Failed to update product option", err, map[string]interface{}{
			"option_id": option.ID,
		})
		return err
	}

	logger.Debug("Product option updated", map[string]interface{}{
		"option_id": option.ID,
	})
	return nil
}

func (r *productOptionRepository) UpdateStock(id uint, quantity int) error {
	logger.Debug("Updating product option stock", map[string]interface{}{
		"option_id": id,
		"quantity":  quantity,
	})

	if err := r.db.Model(&model.ProductOption{}).Where("id = ?", id).
		Update("stock_quantity", gorm.Expr("stock_quantity + ?", quantity)).Error; err != nil {
		logger.Error("Failed to update product option stock", err, map[string]interface{}{
			"option_id": id,
			"quantity":  quantity,
		})
		return err
	}

	logger.Debug("Product option stock updated", map[string]interface{}{
		"option_id": id,
		"quantity":  quantity,
	})
	return nil
}
