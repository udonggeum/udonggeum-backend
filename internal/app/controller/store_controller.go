package controller

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
	"github.com/ikkim/udonggeum-backend/pkg/util"
)

type StoreController struct {
	storeService  service.StoreService
	authService   service.AuthService
	reviewService *service.ReviewService
}

func NewStoreController(storeService service.StoreService, authService service.AuthService, reviewService *service.ReviewService) *StoreController {
	return &StoreController{
		storeService:  storeService,
		authService:   authService,
		reviewService: reviewService,
	}
}

type StoreRequest struct {
	Name        string   `json:"name" binding:"required"`
	Region      string   `json:"region" binding:"required"`
	District    string   `json:"district" binding:"required"`
	Address     string   `json:"address"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	PhoneNumber string   `json:"phone_number"`
	ImageURL    string   `json:"image_url"`
	Description string   `json:"description"`
	OpenTime    string   `json:"open_time"`
	CloseTime   string   `json:"close_time"`
	TagIDs      []uint   `json:"tag_ids"` // 매장 태그 ID 배열

	// 사업자 인증 정보
	BusinessNumber        string `json:"business_number" binding:"required"`          // 사업자등록번호 (필수)
	BusinessStartDate     string `json:"business_start_date" binding:"required"`      // 개업일자 (필수)
	RepresentativeName    string `json:"representative_name" binding:"required"`      // 대표자명 (필수)
}

// UpdateStoreRequest 매장 정보 수정 요청 (사업자 정보는 수정 불가)
type UpdateStoreRequest struct {
	Name        string   `json:"name" binding:"required"`
	Region      string   `json:"region"`
	District    string   `json:"district"`
	Address     string   `json:"address"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	PhoneNumber string   `json:"phone_number"`
	ImageURL    string   `json:"image_url"`
	Description string   `json:"description"`
	OpenTime    string   `json:"open_time"`
	CloseTime   string   `json:"close_time"`
	TagIDs      []uint   `json:"tag_ids"` // 매장 태그 ID 배열
}

func (ctrl *StoreController) ListStores(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	includeProducts := strings.EqualFold(c.DefaultQuery("include_products", "false"), "true")
	buyingGold := strings.EqualFold(c.DefaultQuery("buying", "false"), "true")

	// Parse user location if provided
	var userLat, userLng *float64
	if latStr := c.Query("user_lat"); latStr != "" {
		if lat, err := strconv.ParseFloat(latStr, 64); err == nil {
			userLat = &lat
		}
	}
	if lngStr := c.Query("user_lng"); lngStr != "" {
		if lng, err := strconv.ParseFloat(lngStr, 64); err == nil {
			userLng = &lng
		}
	}

	// Parse page_size if provided (for limiting results)
	var pageSize int
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	opts := service.StoreListOptions{
		Region:          c.Query("region"),
		District:        c.Query("district"),
		Search:          c.Query("search"),
		IncludeProducts: includeProducts,
		BuyingGold:      buyingGold,
		UserLat:         userLat,
		UserLng:         userLng,
	}

	stores, err := ctrl.storeService.ListStores(opts)
	if err != nil {
		log.Error("Failed to list stores", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch stores",
		})
		return
	}

	// Apply page_size limit if provided
	if pageSize > 0 && len(stores) > pageSize {
		stores = stores[:pageSize]
	}

	// 인증된 사용자의 경우 좋아요 상태 포함
	response := gin.H{
		"stores": stores,
		"count":  len(stores),
	}

	if userID, exists := middleware.GetUserID(c); exists {
		likedStoreIDs, err := ctrl.storeService.GetUserLikedStoreIDs(userID)
		if err == nil {
			// 좋아요한 매장 ID를 맵으로 변환
			likedMap := make(map[uint]bool)
			for _, id := range likedStoreIDs {
				likedMap[id] = true
			}

			// 각 매장에 is_liked 추가
			storesWithLikes := make([]map[string]interface{}, len(stores))
			for i, store := range stores {
				storeMap := map[string]interface{}{
					"id":               store.ID,
					"user_id":          store.UserID,
					"name":             store.Name,
					"region":           store.Region,
					"district":         store.District,
					"address":          store.Address,
					"latitude":         store.Latitude,
					"longitude":        store.Longitude,
					"phone_number":     store.PhoneNumber,
					"image_url":        store.ImageURL,
					"description":      store.Description,
					"open_time":        store.OpenTime,
					"close_time":       store.CloseTime,
					"tags":             store.Tags,
					"buying_gold":      store.BuyingGold,
					"buying_platinum":  store.BuyingPlatinum,
					"buying_silver":    store.BuyingSilver,
					"created_at":       store.CreatedAt,
					"updated_at":       store.UpdatedAt,
					"is_liked":         likedMap[store.ID],
				}
				storesWithLikes[i] = storeMap
			}
			response["stores"] = storesWithLikes
		}
	}

	log.Info("Stores listed", map[string]interface{}{
		"count": len(stores),
	})

	c.JSON(http.StatusOK, response)
}

func (ctrl *StoreController) GetStoreByID(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid store ID", map[string]interface{}{
			"store_id": idStr,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid store ID",
		})
		return
	}

	includeProducts := strings.EqualFold(c.DefaultQuery("include_products", "false"), "true")

	store, err := ctrl.storeService.GetStoreByID(uint(id), includeProducts)
	if err != nil {
		if err == service.ErrStoreNotFound {
			log.Warn("Store not found", map[string]interface{}{
				"store_id": id,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Store not found",
			})
			return
		}
		log.Error("Failed to fetch store", err, map[string]interface{}{
			"store_id": id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch store",
		})
		return
	}

	// 리뷰 통계 가져오기
	reviewStats, err := ctrl.reviewService.GetStoreStatistics(uint(id))
	if err != nil {
		log.Warn("Failed to fetch review statistics", map[string]interface{}{
			"store_id": id,
			"error":    err.Error(),
		})
	}

	// 좋아요 상태 확인 (인증된 사용자만)
	response := gin.H{
		"store": store,
	}

	// 리뷰 통계 추가
	if reviewStats != nil {
		response["average_rating"] = reviewStats["average_rating"]
		response["review_count"] = reviewStats["review_count"]
	}

	// 선택적으로 사용자 좋아요 상태 포함
	if userID, exists := middleware.GetUserID(c); exists {
		isLiked, err := ctrl.storeService.IsStoreLiked(uint(id), userID)
		if err == nil {
			response["is_liked"] = isLiked
		}
	}

	log.Info("Store fetched", map[string]interface{}{
		"store_id": store.ID,
	})

	c.JSON(http.StatusOK, response)
}

func (ctrl *StoreController) CreateStore(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("User ID not found in context for store creation", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	// 휴대폰 인증 확인
	user, err := ctrl.authService.GetUserByID(userID)
	if err != nil {
		log.Error("Failed to get user", err, map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify user",
		})
		return
	}

	if !user.PhoneVerified {
		log.Warn("Phone not verified for store creation", map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusForbidden, gin.H{
			"error": "휴대폰 인증이 필요합니다. 마이페이지에서 휴대폰 인증을 완료해주세요.",
		})
		return
	}

	var req StoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid store creation request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// 1. 사업자 등록번호 진위 확인
	log.Info("Verifying business number", map[string]interface{}{
		"business_number": req.BusinessNumber,
		"user_id":         userID,
	})

	verificationResult, err := util.VerifyBusinessNumber(
		req.BusinessNumber,
		req.BusinessStartDate,
		req.RepresentativeName,
	)
	if err != nil {
		log.Error("Business verification API error", err, map[string]interface{}{
			"business_number": req.BusinessNumber,
			"user_id":         userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "사업자 인증 중 오류가 발생했습니다",
			"details": err.Error(),
		})
		return
	}

	if !verificationResult.IsValid {
		log.Warn("Business verification failed", map[string]interface{}{
			"business_number": req.BusinessNumber,
			"user_id":         userID,
			"reason":          verificationResult.Message,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "사업자 인증에 실패했습니다",
			"message": verificationResult.Message,
		})
		return
	}

	log.Info("Business verification successful", map[string]interface{}{
		"business_number": req.BusinessNumber,
		"user_id":         userID,
		"status":          verificationResult.BusinessStatus,
	})

	// 2. 태그 ID로부터 Tag 객체 생성
	var tags []model.Tag
	for _, tagID := range req.TagIDs {
		tags = append(tags, model.Tag{ID: tagID})
	}

	// 3. 사업자 인증 성공 - 매장 생성
	now := time.Now()
	store := &model.Store{
		UserID:      userID,
		Name:        req.Name,
		Region:      req.Region,
		District:    req.District,
		Address:     req.Address,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		PhoneNumber: req.PhoneNumber,
		ImageURL:    req.ImageURL,
		Description: req.Description,
		OpenTime:    req.OpenTime,
		CloseTime:   req.CloseTime,
		Tags:        tags,
		// 사업자 정보는 별도 테이블로 관리
		BusinessRegistration: &model.BusinessRegistration{
			BusinessNumber:     req.BusinessNumber,
			BusinessStartDate:  req.BusinessStartDate,
			RepresentativeName: req.RepresentativeName,
			BusinessStatus:     verificationResult.BusinessStatus,
			TaxType:            verificationResult.TaxType,
			IsVerified:         true,
			VerificationDate:   &now,
		},
	}

	created, err := ctrl.storeService.CreateStore(store)
	if err != nil {
		log.Error("Failed to create store", err, map[string]interface{}{
			"user_id": userID,
			"name":    req.Name,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create store",
		})
		return
	}

	// 4. 사용자를 admin으로 승격 (UserService 필요)
	err = ctrl.storeService.PromoteUserToAdmin(userID)
	if err != nil {
		log.Error("Failed to promote user to admin", err, map[string]interface{}{
			"user_id":  userID,
			"store_id": created.ID,
		})
		// 매장은 생성됐으나 권한 승격 실패 - 경고 로그만 남기고 계속 진행
		// 관리자가 수동으로 권한을 부여할 수 있음
	}

	log.Info("Store created successfully", map[string]interface{}{
		"store_id": created.ID,
		"user_id":  userID,
		"verified": true,
	})

	c.JSON(http.StatusCreated, gin.H{
		"message": "매장이 성공적으로 등록되었습니다",
		"store":   created,
	})
}

func (ctrl *StoreController) UpdateStore(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("User ID not found in context for store update", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	idStr := c.Param("id")
	storeID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid store ID format for update", map[string]interface{}{
			"store_id": idStr,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid store ID",
		})
		return
	}

	var req StoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid store update request", map[string]interface{}{
			"store_id": storeID,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	updated, err := ctrl.storeService.UpdateStore(userID, uint(storeID), service.StoreMutation{
		Name:        req.Name,
		Region:      req.Region,
		District:    req.District,
		Address:     req.Address,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		PhoneNumber: req.PhoneNumber,
		ImageURL:    req.ImageURL,
		Description: req.Description,
		OpenTime:    req.OpenTime,
		CloseTime:   req.CloseTime,
		TagIDs:      req.TagIDs,
	})
	if err != nil {
		switch err {
		case service.ErrStoreNotFound:
			log.Warn("Cannot update store: not found", map[string]interface{}{
				"store_id": storeID,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Store not found",
			})
			return
		case service.ErrStoreAccessDenied:
			log.Warn("Store update forbidden", map[string]interface{}{
				"store_id": storeID,
				"user_id":  userID,
			})
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
			})
			return
		default:
			log.Error("Failed to update store", err, map[string]interface{}{
				"store_id": storeID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update store",
			})
			return
		}
	}

	log.Info("Store updated", map[string]interface{}{
		"store_id": storeID,
		"user_id":  userID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Store updated successfully",
		"store":   updated,
	})
}

func (ctrl *StoreController) DeleteStore(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("User ID not found in context for store deletion", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	idStr := c.Param("id")
	storeID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid store ID format for delete", map[string]interface{}{
			"store_id": idStr,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid store ID",
		})
		return
	}

	if err := ctrl.storeService.DeleteStore(userID, uint(storeID)); err != nil {
		switch err {
		case service.ErrStoreNotFound:
			log.Warn("Cannot delete store: not found", map[string]interface{}{
				"store_id": storeID,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Store not found",
			})
			return
		case service.ErrStoreAccessDenied:
			log.Warn("Store deletion forbidden", map[string]interface{}{
				"store_id": storeID,
				"user_id":  userID,
			})
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
			})
			return
		default:
			log.Error("Failed to delete store", err, map[string]interface{}{
				"store_id": storeID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to delete store",
			})
			return
		}
	}

	log.Info("Store deleted", map[string]interface{}{
		"store_id": storeID,
		"user_id":  userID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Store deleted successfully",
	})
}

func (ctrl *StoreController) ListLocations(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	locations, err := ctrl.storeService.ListLocations()
	if err != nil {
		log.Error("Failed to list store locations", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch store locations",
		})
		return
	}

	log.Info("Store locations listed", map[string]interface{}{
		"count": len(locations),
	})

	c.JSON(http.StatusOK, gin.H{
		"locations": locations,
		"count":     len(locations),
	})
}

// ToggleStoreLike 매장 좋아요 토글
func (ctrl *StoreController) ToggleStoreLike(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	// 사용자 ID 가져오기
	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("User ID not found in context for store like toggle", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "로그인이 필요합니다",
		})
		return
	}

	// 매장 ID 파싱
	idStr := c.Param("id")
	storeID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid store ID for like toggle", map[string]interface{}{
			"store_id": idStr,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "잘못된 매장 ID입니다",
		})
		return
	}

	// 좋아요 토글
	isLiked, err := ctrl.storeService.ToggleStoreLike(uint(storeID), userID)
	if err != nil {
		if err == service.ErrStoreNotFound {
			log.Warn("Store not found for like toggle", map[string]interface{}{
				"store_id": storeID,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "매장을 찾을 수 없습니다",
			})
			return
		}
		log.Error("Failed to toggle store like", err, map[string]interface{}{
			"store_id": storeID,
			"user_id":  userID,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	log.Info("Store like toggled", map[string]interface{}{
		"store_id": storeID,
		"user_id":  userID,
		"is_liked": isLiked,
	})

	c.JSON(http.StatusOK, gin.H{
		"is_liked": isLiked,
	})
}

// GetUserLikedStores 사용자가 좋아요한 매장 목록 조회
func (ctrl *StoreController) GetUserLikedStores(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("User ID not found in context", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	stores, err := ctrl.storeService.GetUserLikedStores(userID)
	if err != nil {
		log.Error("Failed to get user liked stores", err, map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get liked stores",
		})
		return
	}

	log.Info("User liked stores retrieved", map[string]interface{}{
		"user_id": userID,
		"count":   len(stores),
	})

	c.JSON(http.StatusOK, gin.H{
		"stores": stores,
		"count":  len(stores),
	})
}

// GetMyStore admin 사용자의 매장 정보 조회
func (ctrl *StoreController) GetMyStore(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("User ID not found in context for my store", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	stores, err := ctrl.storeService.GetStoresByUserID(userID)
	if err != nil {
		log.Error("Failed to get my store", err, map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get my store",
		})
		return
	}

	// admin 사용자는 하나의 매장만 가질 수 있음
	if len(stores) == 0 {
		log.Warn("No store found for admin user", map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Store not found",
		})
		return
	}

	log.Info("My store retrieved", map[string]interface{}{
		"user_id":  userID,
		"store_id": stores[0].ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"store": stores[0],
	})
}

// UpdateMyStore admin 사용자의 매장 정보 수정
func (ctrl *StoreController) UpdateMyStore(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("User ID not found in context for my store update", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	// 사용자의 매장 찾기
	stores, err := ctrl.storeService.GetStoresByUserID(userID)
	if err != nil {
		log.Error("Failed to get my store for update", err, map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get my store",
		})
		return
	}

	if len(stores) == 0 {
		log.Warn("No store found for admin user update", map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Store not found",
		})
		return
	}

	storeID := stores[0].ID

	var req UpdateStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid my store update request", map[string]interface{}{
			"store_id": storeID,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	updated, err := ctrl.storeService.UpdateStore(userID, storeID, service.StoreMutation{
		Name:        req.Name,
		Region:      req.Region,
		District:    req.District,
		Address:     req.Address,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		PhoneNumber: req.PhoneNumber,
		ImageURL:    req.ImageURL,
		Description: req.Description,
		OpenTime:    req.OpenTime,
		CloseTime:   req.CloseTime,
		TagIDs:      req.TagIDs,
	})
	if err != nil {
		switch err {
		case service.ErrStoreNotFound:
			log.Warn("Cannot update my store: not found", map[string]interface{}{
				"store_id": storeID,
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Store not found",
			})
			return
		case service.ErrStoreAccessDenied:
			log.Warn("My store update forbidden", map[string]interface{}{
				"store_id": storeID,
				"user_id":  userID,
			})
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
			})
			return
		default:
			log.Error("Failed to update my store", err, map[string]interface{}{
				"store_id": storeID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update store",
			})
			return
		}
	}

	log.Info("My store updated", map[string]interface{}{
		"store_id": storeID,
		"user_id":  userID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Store updated successfully",
		"store":   updated,
	})
}

// ClaimStoreRequest 매장 소유권 신청 요청
type ClaimStoreRequest struct {
	BusinessNumber     string `json:"business_number" binding:"required"`      // 사업자등록번호 (필수)
	BusinessStartDate  string `json:"business_start_date" binding:"required"`  // 개업일자 (필수)
	RepresentativeName string `json:"representative_name" binding:"required"`  // 대표자명 (필수)
}

// ClaimStore 기존 매장에 대한 소유권 신청 (1단계 검증)
func (ctrl *StoreController) ClaimStore(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("User ID not found in context for store claim", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	// 휴대폰 인증 확인
	user, err := ctrl.authService.GetUserByID(userID)
	if err != nil {
		log.Error("Failed to get user", err, map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify user",
		})
		return
	}

	if !user.PhoneVerified {
		log.Warn("Phone not verified for store claim", map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusForbidden, gin.H{
			"error": "휴대폰 인증이 필요합니다. 마이페이지에서 휴대폰 인증을 완료해주세요.",
		})
		return
	}

	// 매장 ID 파싱
	idStr := c.Param("id")
	storeID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid store ID format for claim", map[string]interface{}{
			"store_id": idStr,
			"error":    err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid store ID",
		})
		return
	}

	var req ClaimStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid store claim request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// 1. 매장 존재 확인 및 이미 관리 중인지 확인
	store, err := ctrl.storeService.GetStoreByID(uint(storeID), nil, nil)
	if err != nil {
		log.Warn("Store not found for claim", map[string]interface{}{
			"store_id": storeID,
		})
		c.JSON(http.StatusNotFound, gin.H{
			"error": "매장을 찾을 수 없습니다",
		})
		return
	}

	if store.IsManaged {
		log.Warn("Store already managed", map[string]interface{}{
			"store_id": storeID,
			"user_id":  store.UserID,
		})
		c.JSON(http.StatusConflict, gin.H{
			"error": "이미 관리 중인 매장입니다",
		})
		return
	}

	// 2. 사업자 등록번호 진위 확인
	log.Info("Verifying business number for claim", map[string]interface{}{
		"business_number": req.BusinessNumber,
		"store_id":        storeID,
		"user_id":         userID,
	})

	verificationResult, err := util.VerifyBusinessNumber(
		req.BusinessNumber,
		req.BusinessStartDate,
		req.RepresentativeName,
	)
	if err != nil {
		log.Error("Business verification API error", err, map[string]interface{}{
			"business_number": req.BusinessNumber,
			"user_id":         userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "사업자 인증 중 오류가 발생했습니다",
			"details": err.Error(),
		})
		return
	}

	if !verificationResult.IsValid {
		log.Warn("Business verification failed for claim", map[string]interface{}{
			"business_number": req.BusinessNumber,
			"user_id":         userID,
			"reason":          verificationResult.Message,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "사업자 인증에 실패했습니다",
			"message": verificationResult.Message,
		})
		return
	}

	log.Info("Business verification successful for claim", map[string]interface{}{
		"business_number": req.BusinessNumber,
		"store_id":        storeID,
		"user_id":         userID,
		"status":          verificationResult.BusinessStatus,
	})

	// 3. 매장 소유권 부여
	now := time.Now()
	userIDUint := userID
	store.UserID = &userIDUint
	store.IsManaged = true
	store.IsVerified = false // 아직 인증 전 (2단계 인증 필요)
	store.VerifiedAt = nil

	// 사업자 정보 추가
	store.BusinessRegistration = &model.BusinessRegistration{
		StoreID:            uint(storeID),
		BusinessNumber:     req.BusinessNumber,
		BusinessStartDate:  req.BusinessStartDate,
		RepresentativeName: req.RepresentativeName,
		BusinessStatus:     verificationResult.BusinessStatus,
		TaxType:            verificationResult.TaxType,
		IsVerified:         true,
		VerificationDate:   &now,
	}

	// DB 업데이트
	updated, err := ctrl.storeService.UpdateStoreOwnership(store)
	if err != nil {
		log.Error("Failed to claim store", err, map[string]interface{}{
			"store_id": storeID,
			"user_id":  userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "매장 소유권 등록에 실패했습니다",
		})
		return
	}

	// 4. 사용자를 admin으로 승격
	err = ctrl.storeService.PromoteUserToAdmin(userID)
	if err != nil {
		log.Error("Failed to promote user to admin after claim", err, map[string]interface{}{
			"user_id":  userID,
			"store_id": storeID,
		})
		// 매장 소유권은 부여됐으나 권한 승격 실패 - 경고만 로그
	}

	log.Info("Store claimed successfully", map[string]interface{}{
		"store_id":    storeID,
		"user_id":     userID,
		"is_managed":  true,
		"is_verified": false,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":     "매장 소유권이 등록되었습니다. 인증을 완료하면 신뢰도가 높아집니다.",
		"store":       updated,
		"is_verified": false,
	})
}

// SubmitVerificationRequest 매장 인증 신청 요청 (2단계)
type SubmitVerificationRequest struct {
	BusinessLicenseURL string `json:"business_license_url" binding:"required"` // 사업자등록증 이미지 URL (S3)
}

// SubmitVerification 매장 인증 신청 (2단계 검증)
func (ctrl *StoreController) SubmitVerification(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("User ID not found in context for verification submission", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	var req SubmitVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid verification submission request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// 1. 사용자의 매장 확인
	store, err := ctrl.storeService.GetStoreByUserID(userID)
	if err != nil {
		log.Warn("User does not have a store", map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusNotFound, gin.H{
			"error": "매장을 찾을 수 없습니다. 먼저 매장 소유권을 등록해주세요.",
		})
		return
	}

	// 2. 이미 인증된 매장인지 확인
	if store.IsVerified {
		log.Warn("Store already verified", map[string]interface{}{
			"store_id": store.ID,
			"user_id":  userID,
		})
		c.JSON(http.StatusConflict, gin.H{
			"error": "이미 인증된 매장입니다",
		})
		return
	}

	// 3. 이미 대기 중인 인증이 있는지 확인
	existingVerification, _ := ctrl.storeService.GetVerificationByStoreID(store.ID)
	if existingVerification != nil && existingVerification.Status == model.VerificationStatusPending {
		log.Warn("Verification already pending", map[string]interface{}{
			"store_id":        store.ID,
			"verification_id": existingVerification.ID,
		})
		c.JSON(http.StatusConflict, gin.H{
			"error": "이미 인증 심사가 진행 중입니다",
		})
		return
	}

	// 4. 인증 요청 생성
	now := time.Now()
	verification := &model.StoreVerification{
		StoreID:            store.ID,
		BusinessLicenseURL: req.BusinessLicenseURL,
		Status:             model.VerificationStatusPending,
		SubmittedAt:        &now,
		IPAddress:          c.ClientIP(),
		UserAgent:          c.Request.UserAgent(),
	}

	created, err := ctrl.storeService.CreateVerification(verification)
	if err != nil {
		log.Error("Failed to create verification", err, map[string]interface{}{
			"store_id": store.ID,
			"user_id":  userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "인증 신청 중 오류가 발생했습니다",
		})
		return
	}

	log.Info("Verification submitted successfully", map[string]interface{}{
		"store_id":        store.ID,
		"verification_id": created.ID,
		"user_id":         userID,
	})

	c.JSON(http.StatusCreated, gin.H{
		"message":      "인증 신청이 완료되었습니다. 검토 후 승인됩니다.",
		"verification": created,
		"status":       created.Status,
	})
}

// GetMyVerificationStatus 내 매장 인증 상태 조회
func (ctrl *StoreController) GetMyVerificationStatus(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	userID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("User ID not found in context for verification status", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	// 사용자의 매장 확인
	store, err := ctrl.storeService.GetStoreByUserID(userID)
	if err != nil {
		log.Warn("User does not have a store", map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusNotFound, gin.H{
			"error": "매장을 찾을 수 없습니다",
		})
		return
	}

	// 인증 정보 조회
	verification, err := ctrl.storeService.GetVerificationByStoreID(store.ID)
	if err != nil {
		// 인증 신청이 없는 경우
		c.JSON(http.StatusOK, gin.H{
			"is_verified":  store.IsVerified,
			"verification": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"is_verified":  store.IsVerified,
		"verification": verification,
	})
}

// ListPendingVerifications 관리자용: 대기 중인 인증 목록 조회
func (ctrl *StoreController) ListPendingVerifications(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	// 상태 필터 (기본값: pending)
	status := c.DefaultQuery("status", model.VerificationStatusPending)

	verifications, err := ctrl.storeService.ListVerificationsByStatus(status)
	if err != nil {
		log.Error("Failed to list verifications", err, map[string]interface{}{
			"status": status,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "인증 목록 조회 중 오류가 발생했습니다",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"verifications": verifications,
		"count":         len(verifications),
	})
}

// ReviewVerificationRequest 인증 승인/반려 요청
type ReviewVerificationRequest struct {
	Action string `json:"action" binding:"required,oneof=approve reject"` // approve or reject
	Reason string `json:"reason"`                                         // 반려 사유 (reject일 경우 필수)
}

// ReviewVerification 관리자용: 인증 승인/반려
func (ctrl *StoreController) ReviewVerification(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	adminID, exists := middleware.GetUserID(c)
	if !exists {
		log.Warn("Admin ID not found in context for verification review", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	// 인증 ID 파싱
	idStr := c.Param("id")
	verificationID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Warn("Invalid verification ID format", map[string]interface{}{
			"verification_id": idStr,
			"error":           err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid verification ID",
		})
		return
	}

	var req ReviewVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid verification review request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// reject일 경우 사유 필수
	if req.Action == "reject" && req.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "반려 사유를 입력해주세요",
		})
		return
	}

	// 인증 정보 조회
	verification, err := ctrl.storeService.GetVerificationByID(uint(verificationID))
	if err != nil {
		log.Warn("Verification not found", map[string]interface{}{
			"verification_id": verificationID,
		})
		c.JSON(http.StatusNotFound, gin.H{
			"error": "인증 정보를 찾을 수 없습니다",
		})
		return
	}

	// 이미 처리된 인증인지 확인
	if verification.Status != model.VerificationStatusPending {
		log.Warn("Verification already reviewed", map[string]interface{}{
			"verification_id": verificationID,
			"status":          verification.Status,
		})
		c.JSON(http.StatusConflict, gin.H{
			"error": "이미 처리된 인증 요청입니다",
		})
		return
	}

	now := time.Now()
	verification.ReviewedAt = &now
	verification.ReviewedBy = &adminID

	if req.Action == "approve" {
		// 승인 처리
		verification.Status = model.VerificationStatusApproved

		// 매장 인증 상태 업데이트
		if err := ctrl.storeService.ApproveStoreVerification(verification.StoreID, &now); err != nil {
			log.Error("Failed to approve store verification", err, map[string]interface{}{
				"store_id":        verification.StoreID,
				"verification_id": verificationID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "매장 인증 승인 중 오류가 발생했습니다",
			})
			return
		}

		log.Info("Verification approved", map[string]interface{}{
			"verification_id": verificationID,
			"store_id":        verification.StoreID,
			"admin_id":        adminID,
		})
	} else {
		// 반려 처리
		verification.Status = model.VerificationStatusRejected
		verification.RejectionReason = req.Reason

		log.Info("Verification rejected", map[string]interface{}{
			"verification_id": verificationID,
			"store_id":        verification.StoreID,
			"admin_id":        adminID,
			"reason":          req.Reason,
		})
	}

	// 인증 정보 업데이트
	if err := ctrl.storeService.UpdateVerification(verification); err != nil {
		log.Error("Failed to update verification", err, map[string]interface{}{
			"verification_id": verificationID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "인증 처리 중 오류가 발생했습니다",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "인증 처리가 완료되었습니다",
		"verification": verification,
	})
}
