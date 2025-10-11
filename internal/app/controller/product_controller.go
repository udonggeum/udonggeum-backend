package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
)

type ProductController struct {
	productService service.ProductService
}

func NewProductController(productService service.ProductService) *ProductController {
	return &ProductController{
		productService: productService,
	}
}

type CreateProductRequest struct {
	Name          string                 `json:"name" binding:"required"`
	Description   string                 `json:"description"`
	Price         float64                `json:"price" binding:"required,gt=0"`
	Weight        float64                `json:"weight"`
	Purity        string                 `json:"purity"`
	Category      model.ProductCategory  `json:"category" binding:"required"`
	StockQuantity int                    `json:"stock_quantity" binding:"gte=0"`
	ImageURL      string                 `json:"image_url"`
}

// GetAllProducts returns all products
// GET /api/v1/products
func (ctrl *ProductController) GetAllProducts(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	products, err := ctrl.productService.GetAllProducts()
	if err != nil {
		log.Error("Failed to fetch products", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch products",
		})
		return
	}

	log.Info("Products fetched successfully", map[string]interface{}{
		"count": len(products),
	})

	c.JSON(http.StatusOK, gin.H{
		"products": products,
		"count":    len(products),
	})
}

// GetProductByID returns a product by ID
// GET /api/v1/products/:id
func (ctrl *ProductController) GetProductByID(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid product ID format", map[string]interface{}{
			"product_id": idStr,
			"error":      err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	product, err := ctrl.productService.GetProductByID(uint(id))
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			log.Warn("Product not found", map[string]interface{}{
				"product_id": id,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Product not found",
			})
			return
		}
		log.Error("Failed to fetch product", err, map[string]interface{}{
			"product_id": id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch product",
		})
		return
	}

	log.Info("Product fetched successfully", map[string]interface{}{
		"product_id": product.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"product": product,
	})
}

// CreateProduct creates a new product (Admin only)
// POST /api/v1/products
func (ctrl *ProductController) CreateProduct(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid product creation request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Creating product", map[string]interface{}{
		"name":     req.Name,
		"category": req.Category,
		"price":    req.Price,
	})

	product := &model.Product{
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		Weight:        req.Weight,
		Purity:        req.Purity,
		Category:      req.Category,
		StockQuantity: req.StockQuantity,
		ImageURL:      req.ImageURL,
	}

	if err := ctrl.productService.CreateProduct(product); err != nil {
		log.Error("Failed to create product", err, map[string]interface{}{
			"name":     req.Name,
			"category": req.Category,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create product",
		})
		return
	}

	log.Info("Product created successfully", map[string]interface{}{
		"product_id": product.ID,
		"name":       product.Name,
	})

	c.JSON(http.StatusCreated, gin.H{
		"message": "Product created successfully",
		"product": product,
	})
}

// UpdateProduct updates an existing product (Admin only)
// PUT /api/v1/products/:id
func (ctrl *ProductController) UpdateProduct(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid product ID format", map[string]interface{}{
			"product_id": idStr,
			"error":      err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid product update request", map[string]interface{}{
			"product_id": id,
			"error":      err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	log.Debug("Updating product", map[string]interface{}{
		"product_id": id,
		"name":       req.Name,
	})

	product := &model.Product{
		ID:            uint(id),
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		Weight:        req.Weight,
		Purity:        req.Purity,
		Category:      req.Category,
		StockQuantity: req.StockQuantity,
		ImageURL:      req.ImageURL,
	}

	if err := ctrl.productService.UpdateProduct(product); err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			log.Warn("Product not found for update", map[string]interface{}{
				"product_id": id,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Product not found",
			})
			return
		}
		log.Error("Failed to update product", err, map[string]interface{}{
			"product_id": id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update product",
		})
		return
	}

	log.Info("Product updated successfully", map[string]interface{}{
		"product_id": product.ID,
		"name":       product.Name,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Product updated successfully",
		"product": product,
	})
}

// DeleteProduct deletes a product (Admin only)
// DELETE /api/v1/products/:id
func (ctrl *ProductController) DeleteProduct(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid product ID format", map[string]interface{}{
			"product_id": idStr,
			"error":      err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	log.Debug("Deleting product", map[string]interface{}{
		"product_id": id,
	})

	if err := ctrl.productService.DeleteProduct(uint(id)); err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			log.Warn("Product not found for deletion", map[string]interface{}{
				"product_id": id,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Product not found",
			})
			return
		}
		log.Error("Failed to delete product", err, map[string]interface{}{
			"product_id": id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete product",
		})
		return
	}

	log.Info("Product deleted successfully", map[string]interface{}{
		"product_id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Product deleted successfully",
	})
}
