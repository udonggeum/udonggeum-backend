package repository

import (
	"fmt"

	"gorm.io/gorm"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
)

// CommunityRepository 커뮤니티 저장소 인터페이스
type CommunityRepository interface {
	// Post operations
	CreatePost(post *model.CommunityPost) error
	GetPostByID(id uint, preload bool) (*model.CommunityPost, error)
	GetPosts(query *model.PostListQuery) ([]model.CommunityPost, int64, error)
	UpdatePost(post *model.CommunityPost) error
	DeletePost(id uint) error
	IncrementViewCount(id uint) error

	// Comment operations
	CreateComment(comment *model.CommunityComment) error
	GetCommentByID(id uint) (*model.CommunityComment, error)
	GetComments(query *model.CommentListQuery) ([]model.CommunityComment, int64, error)
	UpdateComment(comment *model.CommunityComment) error
	DeleteComment(id uint) error
	GetCommentCountByPostID(postID uint) (int64, error)

	// Like operations
	LikePost(postID, userID uint) error
	UnlikePost(postID, userID uint) error
	IsPostLiked(postID, userID uint) (bool, error)
	LikeComment(commentID, userID uint) error
	UnlikeComment(commentID, userID uint) error
	IsCommentLiked(commentID, userID uint) (bool, error)

	// QnA operations
	AcceptAnswer(postID, commentID uint) error

	// Store post management
	UpdatePostPin(postID uint, isPinned bool) error
	GetPostsWithImages(storeID uint, limit, offset int) ([]model.CommunityPost, int64, error)

	// Reservation and transaction operations
	ReservePost(postID, reservedByUserID uint) error
	CancelReservation(postID uint) error
	CompleteTransaction(postID uint) error
}

type communityRepository struct {
	db *gorm.DB
}

// NewCommunityRepository 커뮤니티 저장소 생성자
func NewCommunityRepository(db *gorm.DB) CommunityRepository {
	return &communityRepository{db: db}
}

// CreatePost 게시글 생성
func (r *communityRepository) CreatePost(post *model.CommunityPost) error {
	// 게시글 생성
	if err := r.db.Create(post).Error; err != nil {
		return err
	}

	// User와 Store 정보를 Preload하여 다시 조회
	if err := r.db.Preload("User").Preload("Store").First(post, post.ID).Error; err != nil {
		return err
	}

	return nil
}

// GetPostByID 게시글 ID로 조회
func (r *communityRepository) GetPostByID(id uint, preload bool) (*model.CommunityPost, error) {
	var post model.CommunityPost
	query := r.db.Where("id = ?", id)

	if preload {
		query = query.
			Preload("User").
			Preload("Store").
			Preload("ReservedByUser").
			Preload("Comments", func(db *gorm.DB) *gorm.DB {
				return db.Where("parent_id IS NULL").Order("created_at ASC")
			}).
			Preload("Comments.User").
			Preload("Comments.Replies").
			Preload("Comments.Replies.User")
	}

	if err := query.First(&post).Error; err != nil {
		return nil, err
	}

	return &post, nil
}

// GetPosts 게시글 목록 조회
func (r *communityRepository) GetPosts(query *model.PostListQuery) ([]model.CommunityPost, int64, error) {
	var posts []model.CommunityPost
	var total int64

	// 기본 쿼리 구성
	db := r.db.Model(&model.CommunityPost{}).
		Preload("User").
		Preload("Store")

	// 필터 적용
	if query.Category != nil {
		db = db.Where("category = ?", *query.Category)
	}
	if query.Type != nil {
		db = db.Where("type = ?", *query.Type)
	}
	if query.Status != nil {
		db = db.Where("status = ?", *query.Status)
	} else {
		// 기본적으로 active 상태만 조회
		db = db.Where("status = ?", model.StatusActive)
	}
	if query.UserID != nil {
		db = db.Where("user_id = ?", *query.UserID)
	}
	if query.StoreID != nil {
		db = db.Where("store_id = ?", *query.StoreID)
	}
	if query.IsAnswered != nil {
		db = db.Where("is_answered = ?", *query.IsAnswered)
	}
	if query.Search != nil && *query.Search != "" {
		searchTerm := "%" + *query.Search + "%"
		db = db.Where("title ILIKE ? OR content ILIKE ?", searchTerm, searchTerm)
	}

	// 총 개수 조회
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 정렬
	sortBy := "created_at"
	if query.SortBy != "" {
		sortBy = query.SortBy
	}
	sortOrder := "DESC"
	if query.SortOrder != "" {
		sortOrder = query.SortOrder
	}
	db = db.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// 페이지네이션
	page := 1
	if query.Page > 0 {
		page = query.Page
	}
	pageSize := 20
	if query.PageSize > 0 {
		pageSize = query.PageSize
	}
	offset := (page - 1) * pageSize
	db = db.Offset(offset).Limit(pageSize)

	// 조회 실행
	if err := db.Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

// UpdatePost 게시글 수정
func (r *communityRepository) UpdatePost(post *model.CommunityPost) error {
	return r.db.Save(post).Error
}

// DeletePost 게시글 삭제 (소프트 삭제)
func (r *communityRepository) DeletePost(id uint) error {
	return r.db.Delete(&model.CommunityPost{}, id).Error
}

// IncrementViewCount 조회수 증가
func (r *communityRepository) IncrementViewCount(id uint) error {
	return r.db.Model(&model.CommunityPost{}).
		Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).
		Error
}

// CreateComment 댓글 생성
func (r *communityRepository) CreateComment(comment *model.CommunityComment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 댓글 생성
		if err := tx.Create(comment).Error; err != nil {
			return err
		}

		// 게시글의 댓글 수 증가
		if err := tx.Model(&model.CommunityPost{}).
			Where("id = ?", comment.PostID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).
			Error; err != nil {
			return err
		}

		// User 정보를 Preload하여 다시 조회
		if err := tx.Preload("User").First(comment, comment.ID).Error; err != nil {
			return err
		}

		return nil
	})
}

// GetCommentByID 댓글 ID로 조회
func (r *communityRepository) GetCommentByID(id uint) (*model.CommunityComment, error) {
	var comment model.CommunityComment
	if err := r.db.
		Preload("User").
		Preload("Replies").
		Preload("Replies.User").
		First(&comment, id).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

// GetComments 댓글 목록 조회
func (r *communityRepository) GetComments(query *model.CommentListQuery) ([]model.CommunityComment, int64, error) {
	var comments []model.CommunityComment
	var total int64

	db := r.db.Model(&model.CommunityComment{}).
		Preload("User").
		Preload("Replies", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Preload("Replies.User").
		Where("post_id = ?", query.PostID)

	// 최상위 댓글만 조회 또는 특정 부모 댓글의 대댓글만 조회
	if query.ParentID == nil {
		db = db.Where("parent_id IS NULL")
	} else {
		db = db.Where("parent_id = ?", *query.ParentID)
	}

	// 총 개수 조회
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 정렬
	sortBy := "created_at"
	if query.SortBy != "" {
		sortBy = query.SortBy
	}
	sortOrder := "ASC"
	if query.SortOrder != "" {
		sortOrder = query.SortOrder
	}
	db = db.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// 페이지네이션
	page := 1
	if query.Page > 0 {
		page = query.Page
	}
	pageSize := 50
	if query.PageSize > 0 {
		pageSize = query.PageSize
	}
	offset := (page - 1) * pageSize
	db = db.Offset(offset).Limit(pageSize)

	// 조회 실행
	if err := db.Find(&comments).Error; err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

// UpdateComment 댓글 수정
func (r *communityRepository) UpdateComment(comment *model.CommunityComment) error {
	return r.db.Save(comment).Error
}

// DeleteComment 댓글 삭제 (소프트 삭제)
func (r *communityRepository) DeleteComment(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var comment model.CommunityComment
		if err := tx.First(&comment, id).Error; err != nil {
			return err
		}

		// 댓글 삭제
		if err := tx.Delete(&comment).Error; err != nil {
			return err
		}

		// 게시글의 댓글 수 감소
		if err := tx.Model(&model.CommunityPost{}).
			Where("id = ?", comment.PostID).
			UpdateColumn("comment_count", gorm.Expr("comment_count - ?", 1)).
			Error; err != nil {
			return err
		}

		return nil
	})
}

// GetCommentCountByPostID 게시글의 댓글 수 조회
func (r *communityRepository) GetCommentCountByPostID(postID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.CommunityComment{}).
		Where("post_id = ?", postID).
		Count(&count).Error
	return count, err
}

// LikePost 게시글 좋아요
func (r *communityRepository) LikePost(postID, userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 중복 확인
		var count int64
		if err := tx.Model(&model.PostLike{}).
			Where("post_id = ? AND user_id = ?", postID, userID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("already liked")
		}

		// 좋아요 생성
		like := &model.PostLike{
			PostID: postID,
			UserID: userID,
		}
		if err := tx.Create(like).Error; err != nil {
			return err
		}

		// 게시글의 좋아요 수 증가
		if err := tx.Model(&model.CommunityPost{}).
			Where("id = ?", postID).
			UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).
			Error; err != nil {
			return err
		}

		return nil
	})
}

// UnlikePost 게시글 좋아요 취소
func (r *communityRepository) UnlikePost(postID, userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 좋아요 삭제
		result := tx.Where("post_id = ? AND user_id = ?", postID, userID).
			Delete(&model.PostLike{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("like not found")
		}

		// 게시글의 좋아요 수 감소
		if err := tx.Model(&model.CommunityPost{}).
			Where("id = ?", postID).
			UpdateColumn("like_count", gorm.Expr("like_count - ?", 1)).
			Error; err != nil {
			return err
		}

		return nil
	})
}

// IsPostLiked 게시글 좋아요 여부 확인
func (r *communityRepository) IsPostLiked(postID, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.PostLike{}).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Count(&count).Error
	return count > 0, err
}

// LikeComment 댓글 좋아요
func (r *communityRepository) LikeComment(commentID, userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 중복 확인
		var count int64
		if err := tx.Model(&model.CommentLike{}).
			Where("comment_id = ? AND user_id = ?", commentID, userID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("already liked")
		}

		// 좋아요 생성
		like := &model.CommentLike{
			CommentID: commentID,
			UserID:    userID,
		}
		if err := tx.Create(like).Error; err != nil {
			return err
		}

		// 댓글의 좋아요 수 증가
		if err := tx.Model(&model.CommunityComment{}).
			Where("id = ?", commentID).
			UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).
			Error; err != nil {
			return err
		}

		return nil
	})
}

// UnlikeComment 댓글 좋아요 취소
func (r *communityRepository) UnlikeComment(commentID, userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 좋아요 삭제
		result := tx.Where("comment_id = ? AND user_id = ?", commentID, userID).
			Delete(&model.CommentLike{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("like not found")
		}

		// 댓글의 좋아요 수 감소
		if err := tx.Model(&model.CommunityComment{}).
			Where("id = ?", commentID).
			UpdateColumn("like_count", gorm.Expr("like_count - ?", 1)).
			Error; err != nil {
			return err
		}

		return nil
	})
}

// IsCommentLiked 댓글 좋아요 여부 확인
func (r *communityRepository) IsCommentLiked(commentID, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.CommentLike{}).
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		Count(&count).Error
	return count > 0, err
}

// AcceptAnswer QnA 답변 채택
func (r *communityRepository) AcceptAnswer(postID, commentID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 게시글 업데이트
		if err := tx.Model(&model.CommunityPost{}).
			Where("id = ?", postID).
			Updates(map[string]interface{}{
				"is_answered":         true,
				"accepted_answer_id": commentID,
			}).Error; err != nil {
			return err
		}

		// 댓글 채택 표시
		if err := tx.Model(&model.CommunityComment{}).
			Where("id = ?", commentID).
			Update("is_accepted", true).
			Error; err != nil {
			return err
		}

		return nil
	})
}

// UpdatePostPin 게시글 고정/해제
func (r *communityRepository) UpdatePostPin(postID uint, isPinned bool) error {
	return r.db.Model(&model.CommunityPost{}).
		Where("id = ?", postID).
		Update("is_pinned", isPinned).
		Error
}

// GetPostsWithImages 이미지가 있는 매장 게시글 조회
func (r *communityRepository) GetPostsWithImages(storeID uint, limit, offset int) ([]model.CommunityPost, int64, error) {
	var posts []model.CommunityPost
	var total int64

	query := r.db.Model(&model.CommunityPost{}).
		Where("store_id = ?", storeID).
		Where("status = ?", model.StatusActive).
		Where("array_length(image_urls, 1) > 0") // 이미지가 있는 게시글만

	// 전체 개수 조회
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 게시글 조회
	if err := query.
		Preload("User").
		Preload("Store").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

// ReservePost 게시글 예약하기
func (r *communityRepository) ReservePost(postID, reservedByUserID uint) error {
	now := gorm.Expr("NOW()")
	status := "reserved"

	return r.db.Model(&model.CommunityPost{}).
		Where("id = ?", postID).
		Updates(map[string]interface{}{
			"reservation_status":  status,
			"reserved_by_user_id": reservedByUserID,
			"reserved_at":         now,
		}).Error
}

// CancelReservation 예약 취소
func (r *communityRepository) CancelReservation(postID uint) error {
	return r.db.Model(&model.CommunityPost{}).
		Where("id = ?", postID).
		Updates(map[string]interface{}{
			"reservation_status":  nil,
			"reserved_by_user_id": nil,
			"reserved_at":         nil,
		}).Error
}

// CompleteTransaction 거래 완료
func (r *communityRepository) CompleteTransaction(postID uint) error {
	now := gorm.Expr("NOW()")
	status := "completed"

	return r.db.Model(&model.CommunityPost{}).
		Where("id = ?", postID).
		Updates(map[string]interface{}{
			"reservation_status": status,
			"completed_at":       now,
		}).Error
}
