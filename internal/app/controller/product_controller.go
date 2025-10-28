package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
)

type ProductController struct {
	productService service.ProductService
}

func NewProductController(productService service.ProductService) *ProductController {
	return &ProductController{productService: productService}
}

type CreateProductRequest struct {
	Name            string                `json:"name" binding:"required"`
	Description     string                `json:"description"`
	Price           float64               `json:"price" binding:"required,gt=0"`
	Weight          float64               `json:"weight"`
	Purity          string                `json:"purity"`
	Category        model.ProductCategory `json:"category" binding:"required"`
	StockQuantity   int                   `json:"stock_quantity" binding:"gte=0"`
	ImageURL        string                `json:"image_url"`
	StoreID         uint                  `json:"store_id" binding:"required"`
	PopularityScore float64               `json:"popularity_score"`
}

type productQuery struct {
	category       *model.ProductCategory
	region         string
	district       string
	storeID        *uint
	search         string
	sort           service.ProductSort
	sortAscending  bool
	includeOptions bool
	popularOnly    bool
	limit          int
	offset         int
}

func parseProductQuery(c *gin.Context) (productQuery, error) {
	var result productQuery

	if category := c.Query("category"); category != "" {
		cat := model.ProductCategory(strings.ToLower(category))
		switch cat {
		case model.CategoryGold, model.CategorySilver, model.CategoryJewelry:
			result.category = &cat
		default:
			return productQuery{}, errors.New("invalid category")
		}
	}

	if storeIDStr := c.Query("store_id"); storeIDStr != "" {
		storeIDUint, err := strconv.ParseUint(storeIDStr, 10, 32)
		if err != nil {
			return productQuery{}, errors.New("invalid store id")
		}
		storeID := uint(storeIDUint)
		result.storeID = &storeID
	}

	result.region = c.Query("region")
	result.district = c.Query("district")
	result.search = c.Query("search")

	sortKey := strings.ToLower(c.DefaultQuery("sort", "popularity"))
	switch sortKey {
	case "price_asc":
		result.sort = service.ProductSortPrice
		result.sortAscending = true
	case "price_desc":
		result.sort = service.ProductSortPrice
	case "latest", "created_at_desc":
		result.sort = service.ProductSortCreatedAt
	case "created_at_asc":
		result.sort = service.ProductSortCreatedAt
		result.sortAscending = true
	case "popularity", "popular":
		result.sort = service.ProductSortPopularity
	default:
		result.sort = service.ProductSortPopularity
	}

	if popularOnly := c.Query("popular_only"); popularOnly != "" {
		result.popularOnly = strings.EqualFold(popularOnly, "true")
	}

	result.includeOptions = strings.EqualFold(c.DefaultQuery("include_options", "false"), "true")

	pageSize := 20
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if v, err := strconv.Atoi(pageSizeStr); err == nil && v > 0 {
			pageSize = v
		}
	}

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if v, err := strconv.Atoi(pageStr); err == nil && v > 0 {
			page = v
		}
	}

	result.limit = pageSize
	result.offset = (page - 1) * pageSize

	return result, nil
}

func (ctrl *ProductController) GetAllProducts(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	query, err := parseProductQuery(c)
	if err != nil {
		log.Warn("Invalid product query", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	opts := service.ProductListOptions{
		Region:         query.region,
		District:       query.district,
		Search:         query.search,
		Sort:           query.sort,
		SortAscending:  query.sortAscending,
		PopularOnly:    query.popularOnly,
		Limit:          query.limit,
		Offset:         query.offset,
		IncludeOptions: query.includeOptions,
		StoreID:        query.storeID,
		Category:       query.category,
	}

	products, err := ctrl.productService.ListProducts(opts)
	if err != nil {
		log.Error("Failed to fetch products", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch products",
		})
		return
	}

	log.Info("Products fetched", map[string]interface{}{
		"count": len(products),
	})

	c.JSON(http.StatusOK, gin.H{
		"products":  products,
		"count":     len(products),
		"page_size": query.limit,
		"offset":    query.offset,
	})
}

func (ctrl *ProductController) GetPopularProducts(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	categoryParam := c.Query("category")
	if categoryParam == "" {
		log.Warn("Category required for popular products", nil)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "category is required",
		})
		return
	}

	category := model.ProductCategory(strings.ToLower(categoryParam))
	switch category {
	case model.CategoryGold, model.CategorySilver, model.CategoryJewelry:
	default:
		log.Warn("Invalid category for popular products", map[string]interface{}{
			"category": categoryParam,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid category",
		})
		return
	}

	limit := 6
	if limitStr := c.Query("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}

	products, err := ctrl.productService.GetPopularProducts(category, c.Query("region"), c.Query("district"), limit)
	if err != nil {
		log.Error("Failed to fetch popular products", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch popular products",
		})
		return
	}

	log.Info("Popular products fetched", map[string]interface{}{
		"category": category,
		"count":    len(products),
	})

	c.JSON(http.StatusOK, gin.H{
		"products": products,
		"count":    len(products),
	})
}

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

	log.Info("Product fetched", map[string]interface{}{
		"product_id": product.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"product": product,
	})
}

func (ctrl *ProductController) CreateProduct(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid product creation request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	product := &model.Product{
		Name:            req.Name,
		Description:     req.Description,
		Price:           req.Price,
		Weight:          req.Weight,
		Purity:          req.Purity,
		Category:        req.Category,
		StockQuantity:   req.StockQuantity,
		ImageURL:        req.ImageURL,
		StoreID:         req.StoreID,
		PopularityScore: req.PopularityScore,
	}

	if err := ctrl.productService.CreateProduct(product); err != nil {
		log.Error("Failed to create product", err, map[string]interface{}{
			"name": req.Name,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create product",
		})
		return
	}

	log.Info("Product created", map[string]interface{}{
		"product_id": product.ID,
		"name":       product.Name,
	})

	c.JSON(http.StatusCreated, gin.H{
		"message": "Product created successfully",
		"product": product,
	})
}

func (ctrl *ProductController) UpdateProduct(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("User ID not found in context for product update", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

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
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	product := &model.Product{
		ID:              uint(id),
		Name:            req.Name,
		Description:     req.Description,
		Price:           req.Price,
		Weight:          req.Weight,
		Purity:          req.Purity,
		Category:        req.Category,
		StockQuantity:   req.StockQuantity,
		ImageURL:        req.ImageURL,
		StoreID:         req.StoreID,
		PopularityScore: req.PopularityScore,
	}

	if err := ctrl.productService.UpdateProduct(userID, product); err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			log.Warn("Cannot update product: not found", map[string]interface{}{
				"product_id": id,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Product not found",
			})
			return
		}
		if errors.Is(err, service.ErrProductAccessDenied) {
			log.Warn("Product update forbidden", map[string]interface{}{
				"product_id": id,
				"user_id":    userID,
			})
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
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

	log.Info("Product updated", map[string]interface{}{
		"product_id": product.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Product updated successfully",
		"product": product,
	})
}

func (ctrl *ProductController) DeleteProduct(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("User ID not found in context for product deletion", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

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

	if err := ctrl.productService.DeleteProduct(userID, uint(id)); err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			log.Warn("Product not found", map[string]interface{}{
				"product_id": id,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Product not found",
			})
			return
		}
		if errors.Is(err, service.ErrProductAccessDenied) {
			log.Warn("Product deletion forbidden", map[string]interface{}{
				"product_id": id,
				"user_id":    userID,
			})
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
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

	log.Info("Product deleted", map[string]interface{}{
		"product_id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Product deleted successfully",
	})
}
