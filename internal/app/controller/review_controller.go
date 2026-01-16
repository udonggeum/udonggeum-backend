package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	apperrors "github.com/ikkim/udonggeum-backend/internal/errors"
)

type ReviewController struct {
	reviewService *service.ReviewService
}

func NewReviewController(reviewService *service.ReviewService) *ReviewController {
	return &ReviewController{
		reviewService: reviewService,
	}
}

// CreateReview 리뷰 작성
// @Summary 리뷰 작성
// @Tags Reviews
// @Accept json
// @Produce json
// @Param review body object true "리뷰 정보"
// @Success 201 {object} model.StoreReview
// @Router /stores/{id}/reviews [post]
func (ctrl *ReviewController) CreateReview(c *gin.Context) {
	// 사용자 ID 가져오기 (JWT 미들웨어에서 설정)
	userID, exists := c.Get("userID")
	if !exists {
		apperrors.Unauthorized(c, "로그인이 필요합니다")
		return
	}

	var input struct {
		StoreID   uint     `json:"store_id" binding:"required"`
		Rating    int      `json:"rating" binding:"required,min=1,max=5"`
		Content   string   `json:"content" binding:"required,min=10"`
		ImageURLs []string `json:"image_urls"`
		IsVisitor bool     `json:"is_visitor"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		apperrors.BadRequest(c, apperrors.ValidationInvalidInput, "입력값이 올바르지 않습니다")
		return
	}

	review, err := ctrl.reviewService.CreateReview(userID.(uint), input)
	if err != nil {
		apperrors.BadRequest(c, apperrors.InternalServerError, "리뷰 작성에 실패했습니다")
		return
	}

	c.JSON(http.StatusCreated, review)
}

// GetStoreReviews 매장 리뷰 목록 조회
// @Summary 매장 리뷰 목록
// @Tags Reviews
// @Produce json
// @Param id path int true "매장 ID"
// @Param page query int false "페이지" default(1)
// @Param page_size query int false "페이지 크기" default(20)
// @Param sort_by query string false "정렬 기준" default(created_at)
// @Param sort_order query string false "정렬 순서" default(desc)
// @Success 200 {object} object
// @Router /stores/{id}/reviews [get]
func (ctrl *ReviewController) GetStoreReviews(c *gin.Context) {
	storeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apperrors.BadRequest(c, apperrors.ValidationInvalidID, "잘못된 매장 ID입니다")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	reviews, total, err := ctrl.reviewService.GetStoreReviews(uint(storeID), page, pageSize, sortBy, sortOrder)
	if err != nil {
		apperrors.InternalError(c, "매장 리뷰 조회에 실패했습니다")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      reviews,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetUserReviews 사용자 리뷰 목록 조회
// @Summary 사용자 리뷰 목록
// @Tags Reviews
// @Produce json
// @Param page query int false "페이지" default(1)
// @Param page_size query int false "페이지 크기" default(20)
// @Success 200 {object} object
// @Router /users/me/reviews [get]
func (ctrl *ReviewController) GetUserReviews(c *gin.Context) {
	// 사용자 ID 가져오기
	userID, exists := c.Get("userID")
	if !exists {
		apperrors.Unauthorized(c, "로그인이 필요합니다")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	reviews, total, err := ctrl.reviewService.GetUserReviews(userID.(uint), page, pageSize)
	if err != nil {
		apperrors.InternalError(c, "사용자 리뷰 조회에 실패했습니다")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      reviews,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// UpdateReview 리뷰 수정
// @Summary 리뷰 수정
// @Tags Reviews
// @Accept json
// @Produce json
// @Param id path int true "리뷰 ID"
// @Param review body object true "수정할 정보"
// @Success 200 {object} model.StoreReview
// @Router /reviews/{id} [put]
func (ctrl *ReviewController) UpdateReview(c *gin.Context) {
	// 사용자 ID 가져오기
	userID, exists := c.Get("userID")
	if !exists {
		apperrors.Unauthorized(c, "로그인이 필요합니다")
		return
	}

	reviewID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apperrors.BadRequest(c, apperrors.ValidationInvalidID, "잘못된 리뷰 ID입니다")
		return
	}

	var input struct {
		Rating    *int     `json:"rating"`
		Content   *string  `json:"content"`
		ImageURLs []string `json:"image_urls"`
		IsVisitor *bool    `json:"is_visitor"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		apperrors.BadRequest(c, apperrors.ValidationInvalidInput, "입력값이 올바르지 않습니다")
		return
	}

	review, err := ctrl.reviewService.UpdateReview(uint(reviewID), userID.(uint), input)
	if err != nil {
		apperrors.InternalError(c, "리뷰 수정에 실패했습니다")
		return
	}

	c.JSON(http.StatusOK, review)
}

// DeleteReview 리뷰 삭제
// @Summary 리뷰 삭제
// @Tags Reviews
// @Param id path int true "리뷰 ID"
// @Success 204
// @Router /reviews/{id} [delete]
func (ctrl *ReviewController) DeleteReview(c *gin.Context) {
	// 사용자 ID 가져오기
	userID, exists := c.Get("userID")
	if !exists {
		apperrors.Unauthorized(c, "로그인이 필요합니다")
		return
	}

	// 관리자 여부 확인
	role, _ := c.Get("role")
	isAdmin := role == "admin"

	reviewID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apperrors.BadRequest(c, apperrors.ValidationInvalidID, "잘못된 리뷰 ID입니다")
		return
	}

	if err := ctrl.reviewService.DeleteReview(uint(reviewID), userID.(uint), isAdmin); err != nil {
		apperrors.InternalError(c, "리뷰 삭제에 실패했습니다")
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ToggleReviewLike 리뷰 좋아요 토글
// @Summary 리뷰 좋아요/좋아요 취소
// @Tags Reviews
// @Param id path int true "리뷰 ID"
// @Success 200 {object} object
// @Router /reviews/{id}/like [post]
func (ctrl *ReviewController) ToggleReviewLike(c *gin.Context) {
	// 사용자 ID 가져오기
	userID, exists := c.Get("userID")
	if !exists {
		apperrors.Unauthorized(c, "로그인이 필요합니다")
		return
	}

	reviewID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apperrors.BadRequest(c, apperrors.ValidationInvalidID, "잘못된 리뷰 ID입니다")
		return
	}

	isLiked, err := ctrl.reviewService.ToggleReviewLike(uint(reviewID), userID.(uint))
	if err != nil {
		apperrors.InternalError(c, "좋아요 처리에 실패했습니다")
		return
	}

	c.JSON(http.StatusOK, gin.H{"is_liked": isLiked})
}

// GetStoreStatistics 매장 통계 조회
// @Summary 매장 통계
// @Tags Stores
// @Produce json
// @Param id path int true "매장 ID"
// @Success 200 {object} object
// @Router /stores/{id}/stats [get]
func (ctrl *ReviewController) GetStoreStatistics(c *gin.Context) {
	storeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apperrors.BadRequest(c, apperrors.ValidationInvalidID, "잘못된 매장 ID입니다")
		return
	}

	stats, err := ctrl.reviewService.GetStoreStatistics(uint(storeID))
	if err != nil {
		apperrors.InternalError(c, "매장 통계 조회에 실패했습니다")
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetStoreGallery 매장 갤러리 조회
// @Summary 매장 갤러리 (커뮤니티 포스트 이미지)
// @Tags Stores
// @Produce json
// @Param id path int true "매장 ID"
// @Param page query int false "페이지" default(1)
// @Param page_size query int false "페이지 크기" default(20)
// @Success 200 {object} object
// @Router /stores/{id}/gallery [get]
func (ctrl *ReviewController) GetStoreGallery(c *gin.Context) {
	storeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apperrors.BadRequest(c, apperrors.ValidationInvalidID, "잘못된 매장 ID입니다")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	gallery, total, err := ctrl.reviewService.GetStoreGallery(uint(storeID), page, pageSize)
	if err != nil {
		apperrors.InternalError(c, "매장 갤러리 조회에 실패했습니다")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      gallery,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
