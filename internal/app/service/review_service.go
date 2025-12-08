package service

import (
	"errors"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
)

type ReviewService struct {
	reviewRepo *repository.ReviewRepository
	storeRepo  repository.StoreRepository
}

func NewReviewService(reviewRepo *repository.ReviewRepository, storeRepo repository.StoreRepository) *ReviewService {
	return &ReviewService{
		reviewRepo: reviewRepo,
		storeRepo:  storeRepo,
	}
}

// CreateReview 리뷰 생성
func (s *ReviewService) CreateReview(userID uint, input struct {
	StoreID   uint     `json:"store_id" binding:"required"`
	Rating    int      `json:"rating" binding:"required,min=1,max=5"`
	Content   string   `json:"content" binding:"required,min=10"`
	ImageURLs []string `json:"image_urls"`
	IsVisitor bool     `json:"is_visitor"`
}) (*model.StoreReview, error) {
	// 매장 존재 확인
	store, err := s.storeRepo.FindByID(input.StoreID, false)
	if err != nil {
		return nil, errors.New("매장을 찾을 수 없습니다")
	}
	if store == nil {
		return nil, errors.New("매장을 찾을 수 없습니다")
	}

	// 리뷰 생성
	review := &model.StoreReview{
		StoreID:   input.StoreID,
		UserID:    userID,
		Rating:    input.Rating,
		Content:   input.Content,
		ImageURLs: input.ImageURLs,
		IsVisitor: input.IsVisitor,
	}

	if err := s.reviewRepo.CreateReview(review); err != nil {
		return nil, err
	}

	// User 정보 로드
	loadedReview, err := s.reviewRepo.GetReviewByID(review.ID)
	if err != nil {
		return nil, err
	}

	return loadedReview, nil
}

// GetReview 리뷰 조회
func (s *ReviewService) GetReview(id uint) (*model.StoreReview, error) {
	return s.reviewRepo.GetReviewByID(id)
}

// GetStoreReviews 매장별 리뷰 목록 조회
func (s *ReviewService) GetStoreReviews(storeID uint, page, pageSize int, sortBy, sortOrder string) ([]model.StoreReview, int64, error) {
	// 매장 존재 확인
	store, err := s.storeRepo.FindByID(storeID, false)
	if err != nil {
		return nil, 0, errors.New("매장을 찾을 수 없습니다")
	}
	if store == nil {
		return nil, 0, errors.New("매장을 찾을 수 없습니다")
	}

	offset := (page - 1) * pageSize
	return s.reviewRepo.GetReviewsByStoreID(storeID, offset, pageSize, sortBy, sortOrder)
}

// GetUserReviews 사용자별 리뷰 목록 조회
func (s *ReviewService) GetUserReviews(userID uint, page, pageSize int) ([]model.StoreReview, int64, error) {
	offset := (page - 1) * pageSize
	return s.reviewRepo.GetReviewsByUserID(userID, offset, pageSize)
}

// UpdateReview 리뷰 수정
func (s *ReviewService) UpdateReview(reviewID, userID uint, input struct {
	Rating    *int     `json:"rating"`
	Content   *string  `json:"content"`
	ImageURLs []string `json:"image_urls"`
	IsVisitor *bool    `json:"is_visitor"`
}) (*model.StoreReview, error) {
	// 리뷰 조회
	review, err := s.reviewRepo.GetReviewByID(reviewID)
	if err != nil {
		return nil, errors.New("리뷰를 찾을 수 없습니다")
	}

	// 권한 확인 (작성자만 수정 가능)
	if review.UserID != userID {
		return nil, errors.New("권한이 없습니다")
	}

	// 수정
	if input.Rating != nil {
		if *input.Rating < 1 || *input.Rating > 5 {
			return nil, errors.New("평점은 1-5 사이여야 합니다")
		}
		review.Rating = *input.Rating
	}
	if input.Content != nil {
		if len(*input.Content) < 10 {
			return nil, errors.New("리뷰 내용은 최소 10자 이상이어야 합니다")
		}
		review.Content = *input.Content
	}
	if input.ImageURLs != nil {
		review.ImageURLs = input.ImageURLs
	}
	if input.IsVisitor != nil {
		review.IsVisitor = *input.IsVisitor
	}

	if err := s.reviewRepo.UpdateReview(review); err != nil {
		return nil, err
	}

	return review, nil
}

// DeleteReview 리뷰 삭제
func (s *ReviewService) DeleteReview(reviewID, userID uint, isAdmin bool) error {
	// 리뷰 조회
	review, err := s.reviewRepo.GetReviewByID(reviewID)
	if err != nil {
		return errors.New("리뷰를 찾을 수 없습니다")
	}

	// 권한 확인 (작성자 또는 관리자만 삭제 가능)
	if review.UserID != userID && !isAdmin {
		return errors.New("권한이 없습니다")
	}

	return s.reviewRepo.DeleteReview(reviewID)
}

// ToggleReviewLike 리뷰 좋아요 토글
func (s *ReviewService) ToggleReviewLike(reviewID, userID uint) (bool, error) {
	// 리뷰 존재 확인
	_, err := s.reviewRepo.GetReviewByID(reviewID)
	if err != nil {
		return false, errors.New("리뷰를 찾을 수 없습니다")
	}

	return s.reviewRepo.ToggleLike(reviewID, userID)
}

// GetStoreStatistics 매장 통계 조회
func (s *ReviewService) GetStoreStatistics(storeID uint) (map[string]interface{}, error) {
	// 매장 존재 확인
	store, err := s.storeRepo.FindByID(storeID, false)
	if err != nil {
		return nil, errors.New("매장을 찾을 수 없습니다")
	}
	if store == nil {
		return nil, errors.New("매장을 찾을 수 없습니다")
	}

	return s.reviewRepo.GetStoreStatistics(storeID)
}

// GetStoreGallery 매장 갤러리 조회
func (s *ReviewService) GetStoreGallery(storeID uint, page, pageSize int) ([]repository.GalleryImage, int64, error) {
	// 매장 존재 확인
	store, err := s.storeRepo.FindByID(storeID, false)
	if err != nil {
		return nil, 0, errors.New("매장을 찾을 수 없습니다")
	}
	if store == nil {
		return nil, 0, errors.New("매장을 찾을 수 없습니다")
	}

	offset := (page - 1) * pageSize
	return s.reviewRepo.GetStoreGallery(storeID, offset, pageSize)
}
