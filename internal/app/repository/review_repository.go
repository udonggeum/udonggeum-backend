package repository

import (
	"github.com/ikkim/udonggeum-backend/internal/app/model"

	"gorm.io/gorm"
)

type ReviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

// CreateReview 리뷰 생성
func (r *ReviewRepository) CreateReview(review *model.StoreReview) error {
	return r.db.Create(review).Error
}

// GetReviewByID ID로 리뷰 조회
func (r *ReviewRepository) GetReviewByID(id uint) (*model.StoreReview, error) {
	var review model.StoreReview
	err := r.db.Preload("User").Preload("Store").First(&review, id).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// GetReviewsByStoreID 매장별 리뷰 목록 조회
func (r *ReviewRepository) GetReviewsByStoreID(storeID uint, offset, limit int, sortBy, sortOrder string) ([]model.StoreReview, int64, error) {
	var reviews []model.StoreReview
	var total int64

	query := r.db.Model(&model.StoreReview{}).Where("store_id = ?", storeID)

	// 전체 개수
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 정렬
	orderClause := sortBy + " " + sortOrder
	if sortBy == "" {
		orderClause = "created_at DESC"
	}

	// 데이터 조회
	err := query.Preload("User").
		Order(orderClause).
		Offset(offset).
		Limit(limit).
		Find(&reviews).Error

	if err != nil {
		return nil, 0, err
	}

	return reviews, total, nil
}

// GetReviewsByUserID 사용자별 리뷰 목록 조회
func (r *ReviewRepository) GetReviewsByUserID(userID uint, offset, limit int) ([]model.StoreReview, int64, error) {
	var reviews []model.StoreReview
	var total int64

	query := r.db.Model(&model.StoreReview{}).Where("user_id = ?", userID)

	// 전체 개수
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 데이터 조회
	err := query.Preload("Store").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&reviews).Error

	if err != nil {
		return nil, 0, err
	}

	return reviews, total, nil
}

// UpdateReview 리뷰 수정
func (r *ReviewRepository) UpdateReview(review *model.StoreReview) error {
	return r.db.Save(review).Error
}

// DeleteReview 리뷰 삭제
func (r *ReviewRepository) DeleteReview(id uint) error {
	return r.db.Delete(&model.StoreReview{}, id).Error
}

// GetStoreStatistics 매장 통계 조회
func (r *ReviewRepository) GetStoreStatistics(storeID uint) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 리뷰 개수
	var reviewCount int64
	if err := r.db.Model(&model.StoreReview{}).Where("store_id = ?", storeID).Count(&reviewCount).Error; err != nil {
		return nil, err
	}
	stats["review_count"] = reviewCount

	// 평균 평점
	var avgRating float64
	if reviewCount > 0 {
		r.db.Model(&model.StoreReview{}).
			Where("store_id = ?", storeID).
			Select("AVG(rating)").
			Scan(&avgRating)
	}
	stats["average_rating"] = avgRating

	// 방문자 리뷰 개수
	var visitorReviewCount int64
	if err := r.db.Model(&model.StoreReview{}).
		Where("store_id = ? AND is_visitor = ?", storeID, true).
		Count(&visitorReviewCount).Error; err != nil {
		return nil, err
	}
	stats["visitor_review_count"] = visitorReviewCount

	// 매장 포스트 개수
	var postCount int64
	if err := r.db.Model(&model.CommunityPost{}).
		Where("store_id = ?", storeID).
		Count(&postCount).Error; err != nil {
		return nil, err
	}
	stats["post_count"] = postCount

	// 갤러리 이미지 개수 (커뮤니티 포스트의 이미지 개수)
	var posts []model.CommunityPost
	var imageCount int64
	if err := r.db.Model(&model.CommunityPost{}).
		Where("store_id = ? AND array_length(image_urls, 1) > 0", storeID).
		Select("image_urls").
		Find(&posts).Error; err != nil {
		return nil, err
	}
	for _, post := range posts {
		imageCount += int64(len(post.ImageURLs))
	}
	stats["gallery_image_count"] = imageCount

	return stats, nil
}

// ToggleLike 리뷰 좋아요 토글
func (r *ReviewRepository) ToggleLike(reviewID, userID uint) (bool, error) {
	var like model.ReviewLike
	err := r.db.Where("review_id = ? AND user_id = ?", reviewID, userID).First(&like).Error

	if err == gorm.ErrRecordNotFound {
		// 좋아요 추가
		like = model.ReviewLike{
			ReviewID: reviewID,
			UserID:   userID,
		}
		if err := r.db.Create(&like).Error; err != nil {
			return false, err
		}

		// 좋아요 수 증가
		if err := r.db.Model(&model.StoreReview{}).
			Where("id = ?", reviewID).
			UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil {
			return false, err
		}

		return true, nil
	} else if err != nil {
		return false, err
	}

	// 좋아요 제거
	if err := r.db.Delete(&like).Error; err != nil {
		return false, err
	}

	// 좋아요 수 감소
	if err := r.db.Model(&model.StoreReview{}).
		Where("id = ?", reviewID).
		UpdateColumn("like_count", gorm.Expr("like_count - ?", 1)).Error; err != nil {
		return false, err
	}

	return false, nil
}

// IsLiked 사용자가 리뷰에 좋아요를 눌렀는지 확인
func (r *ReviewRepository) IsLiked(reviewID, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.ReviewLike{}).
		Where("review_id = ? AND user_id = ?", reviewID, userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GalleryImage 갤러리 이미지 정보
type GalleryImage struct {
	ImageURL  string `json:"image_url"`
	PostID    uint   `json:"post_id"`
	Caption   string `json:"caption"`
	CreatedAt string `json:"created_at"`
}

// GetStoreGallery 매장 갤러리 조회 (커뮤니티 포스트 이미지)
func (r *ReviewRepository) GetStoreGallery(storeID uint, offset, limit int) ([]GalleryImage, int64, error) {
	var posts []model.CommunityPost
	var gallery []GalleryImage
	var total int64

	// 이미지가 있는 포스트만 조회
	query := r.db.Model(&model.CommunityPost{}).
		Where("store_id = ? AND array_length(image_urls, 1) > 0", storeID)

	// 전체 이미지 개수 계산
	if err := query.Find(&posts).Error; err != nil {
		return nil, 0, err
	}
	for _, post := range posts {
		total += int64(len(post.ImageURLs))
	}

	// 페이지네이션을 위한 데이터 조회
	posts = []model.CommunityPost{} // 리셋
	err := r.db.Model(&model.CommunityPost{}).
		Where("store_id = ? AND array_length(image_urls, 1) > 0", storeID).
		Order("created_at DESC").
		Find(&posts).Error

	if err != nil {
		return nil, 0, err
	}

	// 이미지를 평탄화하여 갤러리 배열 생성
	for _, post := range posts {
		for _, imageURL := range post.ImageURLs {
			gallery = append(gallery, GalleryImage{
				ImageURL:  imageURL,
				PostID:    post.ID,
				Caption:   post.Title,
				CreatedAt: post.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			})
		}
	}

	// 페이지네이션 적용
	start := offset
	end := offset + limit
	if start > len(gallery) {
		return []GalleryImage{}, total, nil
	}
	if end > len(gallery) {
		end = len(gallery)
	}

	return gallery[start:end], total, nil
}
