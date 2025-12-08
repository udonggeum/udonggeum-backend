package service

import (
	"fmt"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
)

// CommunityService 커뮤니티 서비스 인터페이스
type CommunityService interface {
	// Post operations
	CreatePost(req *model.CreatePostRequest, userID uint, userRole model.UserRole) (*model.CommunityPost, error)
	GetPost(id uint, userID *uint) (*model.CommunityPost, bool, error) // post, isLiked, error
	GetPosts(query *model.PostListQuery, userID *uint) ([]model.CommunityPost, int64, error)
	UpdatePost(id uint, req *model.UpdatePostRequest, userID uint, userRole model.UserRole) (*model.CommunityPost, error)
	DeletePost(id uint, userID uint, userRole model.UserRole) error

	// Comment operations
	CreateComment(req *model.CreateCommentRequest, userID uint) (*model.CommunityComment, error)
	GetComments(query *model.CommentListQuery, userID *uint) ([]model.CommunityComment, int64, error)
	UpdateComment(id uint, req *model.UpdateCommentRequest, userID uint, userRole model.UserRole) (*model.CommunityComment, error)
	DeleteComment(id uint, userID uint, userRole model.UserRole) error

	// Like operations
	TogglePostLike(postID, userID uint) (bool, error) // returns new like status
	ToggleCommentLike(commentID, userID uint) (bool, error)

	// QnA operations
	AcceptAnswer(postID, commentID, userID uint) error
}

type communityService struct {
	repo     repository.CommunityRepository
	userRepo repository.UserRepository
}

// NewCommunityService 커뮤니티 서비스 생성자
func NewCommunityService(repo repository.CommunityRepository, userRepo repository.UserRepository) CommunityService {
	return &communityService{
		repo:     repo,
		userRepo: userRepo,
	}
}

// CreatePost 게시글 생성
func (s *communityService) CreatePost(req *model.CreatePostRequest, userID uint, userRole model.UserRole) (*model.CommunityPost, error) {
	// 권한 검증
	if err := s.validatePostCreation(req, userRole); err != nil {
		return nil, err
	}

	// buy_gold 타입일 때 사용자의 매장 ID 자동 설정
	var storeID *uint
	if req.Type == model.TypeBuyGold {
		// Admin만 buy_gold 글 작성 가능 (이미 validatePostCreation에서 검증됨)
		user, err := s.userRepo.FindByIDWithStores(userID)
		if err != nil {
			return nil, fmt.Errorf("failed to find user: %v", err)
		}

		// 매장이 없으면 에러
		if len(user.Stores) == 0 {
			return nil, fmt.Errorf("you must have at least one store to create buy_gold posts")
		}

		// 첫 번째 매장 사용 (TODO: 나중에 여러 매장이 있을 때 선택하는 UI 필요)
		storeID = &user.Stores[0].ID
	}

	post := &model.CommunityPost{
		Title:     req.Title,
		Content:   req.Content,
		Category:  req.Category,
		Type:      req.Type,
		UserID:    userID,
		Status:    model.StatusActive,
		ImageURLs: req.ImageURLs,
		GoldType:  req.GoldType,
		Weight:    req.Weight,
		Price:     req.Price,
		Location:  req.Location,
		StoreID:   storeID,
	}

	if err := s.repo.CreatePost(post); err != nil {
		return nil, err
	}

	return post, nil
}

// validatePostCreation 게시글 생성 권한 검증
func (s *communityService) validatePostCreation(req *model.CreatePostRequest, userRole model.UserRole) error {
	// FAQ는 관리자만 작성 가능
	if req.Type == model.TypeFAQ && userRole != model.RoleAdmin {
		return fmt.Errorf("only admin can create FAQ posts")
	}

	// 금 매입 글은 사장님(admin)만 작성 가능
	if req.Type == model.TypeBuyGold && userRole != model.RoleAdmin {
		return fmt.Errorf("only store owners can create buy_gold posts")
	}

	// StoreID는 사용자 입력으로 받지 않음 (자동으로 설정됨)

	return nil
}

// GetPost 게시글 조회
func (s *communityService) GetPost(id uint, userID *uint) (*model.CommunityPost, bool, error) {
	post, err := s.repo.GetPostByID(id, true)
	if err != nil {
		return nil, false, err
	}

	// 조회수 증가
	if err := s.repo.IncrementViewCount(id); err != nil {
		// 조회수 증가 실패는 무시
		fmt.Printf("failed to increment view count: %v\n", err)
	}

	// 좋아요 여부 확인
	var isLiked bool
	if userID != nil {
		isLiked, _ = s.repo.IsPostLiked(id, *userID)
	}

	return post, isLiked, nil
}

// GetPosts 게시글 목록 조회
func (s *communityService) GetPosts(query *model.PostListQuery, userID *uint) ([]model.CommunityPost, int64, error) {
	posts, total, err := s.repo.GetPosts(query)
	if err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

// UpdatePost 게시글 수정
func (s *communityService) UpdatePost(id uint, req *model.UpdatePostRequest, userID uint, userRole model.UserRole) (*model.CommunityPost, error) {
	post, err := s.repo.GetPostByID(id, false)
	if err != nil {
		return nil, err
	}

	// 권한 검증 (작성자 본인 또는 관리자만 수정 가능)
	if post.UserID != userID && userRole != model.RoleAdmin {
		return nil, fmt.Errorf("permission denied")
	}

	// 수정 가능한 필드만 업데이트
	if req.Title != nil {
		post.Title = *req.Title
	}
	if req.Content != nil {
		post.Content = *req.Content
	}
	if req.Status != nil {
		post.Status = *req.Status
	}
	if req.GoldType != nil {
		post.GoldType = req.GoldType
	}
	if req.Weight != nil {
		post.Weight = req.Weight
	}
	if req.Price != nil {
		post.Price = req.Price
	}
	if req.Location != nil {
		post.Location = req.Location
	}
	if req.ImageURLs != nil {
		post.ImageURLs = req.ImageURLs
	}

	if err := s.repo.UpdatePost(post); err != nil {
		return nil, err
	}

	return post, nil
}

// DeletePost 게시글 삭제
func (s *communityService) DeletePost(id uint, userID uint, userRole model.UserRole) error {
	post, err := s.repo.GetPostByID(id, false)
	if err != nil {
		return err
	}

	// 권한 검증 (작성자 본인 또는 관리자만 삭제 가능)
	if post.UserID != userID && userRole != model.RoleAdmin {
		return fmt.Errorf("permission denied")
	}

	return s.repo.DeletePost(id)
}

// CreateComment 댓글 생성
func (s *communityService) CreateComment(req *model.CreateCommentRequest, userID uint) (*model.CommunityComment, error) {
	// 게시글 존재 여부 확인
	post, err := s.repo.GetPostByID(req.PostID, false)
	if err != nil {
		return nil, fmt.Errorf("post not found")
	}

	// 부모 댓글 존재 여부 확인 (대댓글인 경우)
	if req.ParentID != nil {
		if _, err := s.repo.GetCommentByID(*req.ParentID); err != nil {
			return nil, fmt.Errorf("parent comment not found")
		}
	}

	comment := &model.CommunityComment{
		Content:  req.Content,
		UserID:   userID,
		PostID:   req.PostID,
		ParentID: req.ParentID,
		IsAnswer: req.IsAnswer && post.Category == model.CategoryQnA,
	}

	if err := s.repo.CreateComment(comment); err != nil {
		return nil, err
	}

	return comment, nil
}

// GetComments 댓글 목록 조회
func (s *communityService) GetComments(query *model.CommentListQuery, userID *uint) ([]model.CommunityComment, int64, error) {
	return s.repo.GetComments(query)
}

// UpdateComment 댓글 수정
func (s *communityService) UpdateComment(id uint, req *model.UpdateCommentRequest, userID uint, userRole model.UserRole) (*model.CommunityComment, error) {
	comment, err := s.repo.GetCommentByID(id)
	if err != nil {
		return nil, err
	}

	// 권한 검증 (작성자 본인 또는 관리자만 수정 가능)
	if comment.UserID != userID && userRole != model.RoleAdmin {
		return nil, fmt.Errorf("permission denied")
	}

	if req.Content != nil {
		comment.Content = *req.Content
	}

	if err := s.repo.UpdateComment(comment); err != nil {
		return nil, err
	}

	return comment, nil
}

// DeleteComment 댓글 삭제
func (s *communityService) DeleteComment(id uint, userID uint, userRole model.UserRole) error {
	comment, err := s.repo.GetCommentByID(id)
	if err != nil {
		return err
	}

	// 권한 검증 (작성자 본인 또는 관리자만 삭제 가능)
	if comment.UserID != userID && userRole != model.RoleAdmin {
		return fmt.Errorf("permission denied")
	}

	return s.repo.DeleteComment(id)
}

// TogglePostLike 게시글 좋아요 토글
func (s *communityService) TogglePostLike(postID, userID uint) (bool, error) {
	isLiked, err := s.repo.IsPostLiked(postID, userID)
	if err != nil {
		return false, err
	}

	if isLiked {
		if err := s.repo.UnlikePost(postID, userID); err != nil {
			return false, err
		}
		return false, nil
	}

	if err := s.repo.LikePost(postID, userID); err != nil {
		return false, err
	}
	return true, nil
}

// ToggleCommentLike 댓글 좋아요 토글
func (s *communityService) ToggleCommentLike(commentID, userID uint) (bool, error) {
	isLiked, err := s.repo.IsCommentLiked(commentID, userID)
	if err != nil {
		return false, err
	}

	if isLiked {
		if err := s.repo.UnlikeComment(commentID, userID); err != nil {
			return false, err
		}
		return false, nil
	}

	if err := s.repo.LikeComment(commentID, userID); err != nil {
		return false, err
	}
	return true, nil
}

// AcceptAnswer QnA 답변 채택
func (s *communityService) AcceptAnswer(postID, commentID, userID uint) error {
	// 게시글 조회
	post, err := s.repo.GetPostByID(postID, false)
	if err != nil {
		return err
	}

	// QnA 카테고리인지 확인
	if post.Category != model.CategoryQnA {
		return fmt.Errorf("only QnA posts can have accepted answers")
	}

	// 작성자 본인만 채택 가능
	if post.UserID != userID {
		return fmt.Errorf("only post author can accept answers")
	}

	// 이미 채택된 답변이 있는지 확인
	if post.IsAnswered {
		return fmt.Errorf("answer already accepted")
	}

	// 댓글이 해당 게시글에 속하는지 확인
	comment, err := s.repo.GetCommentByID(commentID)
	if err != nil {
		return err
	}
	if comment.PostID != postID {
		return fmt.Errorf("comment does not belong to this post")
	}

	return s.repo.AcceptAnswer(postID, commentID)
}
