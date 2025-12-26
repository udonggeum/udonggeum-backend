package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
)

// CommunityController 커뮤니티 컨트롤러
type CommunityController struct {
	service service.CommunityService
}

// NewCommunityController 커뮤니티 컨트롤러 생성자
func NewCommunityController(service service.CommunityService) *CommunityController {
	return &CommunityController{service: service}
}

// ==================== Post APIs ====================

// CreatePost godoc
// @Summary 게시글 생성
// @Description 새로운 커뮤니티 게시글을 작성합니다
// @Tags community
// @Accept json
// @Produce json
// @Param request body model.CreatePostRequest true "게시글 생성 요청"
// @Success 201 {object} model.CommunityPost
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/community/posts [post]
func (c *CommunityController) CreatePost(ctx *gin.Context) {
	var req model.CreatePostRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 인증된 사용자 정보 가져오기
	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userRole, exists := ctx.Get(middleware.UserRoleKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	post, err := c.service.CreatePost(&req, userID.(uint), userRole.(model.UserRole))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, post)
}

// GetPost godoc
// @Summary 게시글 조회
// @Description ID로 특정 게시글을 조회합니다
// @Tags community
// @Accept json
// @Produce json
// @Param id path int true "게시글 ID"
// @Success 200 {object} gin.H{data=model.CommunityPost,is_liked=bool}
// @Failure 404 {object} gin.H
// @Router /api/v1/community/posts/{id} [get]
func (c *CommunityController) GetPost(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	// 로그인한 사용자 ID 가져오기 (옵션)
	var userID *uint
	if uid, exists := ctx.Get(middleware.UserIDKey); exists {
		u := uid.(uint)
		userID = &u
	}

	post, isLiked, err := c.service.GetPost(uint(id), userID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data":     post,
		"is_liked": isLiked,
	})
}

// GetPosts godoc
// @Summary 게시글 목록 조회
// @Description 필터와 페이지네이션으로 게시글 목록을 조회합니다
// @Tags community
// @Accept json
// @Produce json
// @Param category query string false "카테고리" Enums(gold_trade, gold_news, qna)
// @Param type query string false "게시글 타입"
// @Param status query string false "상태" Enums(active, inactive)
// @Param user_id query int false "작성자 ID"
// @Param store_id query int false "매장 ID"
// @Param is_answered query bool false "답변 완료 여부 (QnA)"
// @Param search query string false "검색어 (제목+내용)"
// @Param page query int false "페이지 번호" default(1)
// @Param page_size query int false "페이지 크기" default(20)
// @Param sort_by query string false "정렬 기준" Enums(created_at, view_count, like_count, comment_count)
// @Param sort_order query string false "정렬 순서" Enums(asc, desc)
// @Success 200 {object} gin.H{data=[]model.CommunityPost,total=int,page=int,page_size=int}
// @Router /api/v1/community/posts [get]
func (c *CommunityController) GetPosts(ctx *gin.Context) {
	var query model.PostListQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 로그인한 사용자 ID 가져오기 (옵션)
	var userID *uint
	if uid, exists := ctx.Get(middleware.UserIDKey); exists {
		u := uid.(uint)
		userID = &u
	}

	posts, total, err := c.service.GetPosts(&query, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	page := query.Page
	if page == 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize == 0 {
		pageSize = 20
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data":      posts,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// UpdatePost godoc
// @Summary 게시글 수정
// @Description 게시글을 수정합니다 (작성자 본인 또는 관리자만 가능)
// @Tags community
// @Accept json
// @Produce json
// @Param id path int true "게시글 ID"
// @Param request body model.UpdatePostRequest true "게시글 수정 요청"
// @Success 200 {object} model.CommunityPost
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/community/posts/{id} [put]
func (c *CommunityController) UpdatePost(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	var req model.UpdatePostRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userRole, exists := ctx.Get(middleware.UserRoleKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	post, err := c.service.UpdatePost(uint(id), &req, userID.(uint), userRole.(model.UserRole))
	if err != nil {
		if err.Error() == "permission denied" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, post)
}

// DeletePost godoc
// @Summary 게시글 삭제
// @Description 게시글을 삭제합니다 (작성자 본인 또는 관리자만 가능)
// @Tags community
// @Accept json
// @Produce json
// @Param id path int true "게시글 ID"
// @Success 204
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/community/posts/{id} [delete]
func (c *CommunityController) DeletePost(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userRole, exists := ctx.Get(middleware.UserRoleKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := c.service.DeletePost(uint(id), userID.(uint), userRole.(model.UserRole)); err != nil {
		if err.Error() == "permission denied" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// ==================== Comment APIs ====================

// CreateComment godoc
// @Summary 댓글 생성
// @Description 게시글에 댓글을 작성합니다
// @Tags community
// @Accept json
// @Produce json
// @Param request body model.CreateCommentRequest true "댓글 생성 요청"
// @Success 201 {object} model.CommunityComment
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/community/comments [post]
func (c *CommunityController) CreateComment(ctx *gin.Context) {
	var req model.CreateCommentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	comment, err := c.service.CreateComment(&req, userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, comment)
}

// GetComments godoc
// @Summary 댓글 목록 조회
// @Description 게시글의 댓글 목록을 조회합니다
// @Tags community
// @Accept json
// @Produce json
// @Param post_id query int true "게시글 ID"
// @Param parent_id query int false "부모 댓글 ID (null이면 최상위 댓글만)"
// @Param page query int false "페이지 번호" default(1)
// @Param page_size query int false "페이지 크기" default(50)
// @Param sort_by query string false "정렬 기준" Enums(created_at, like_count)
// @Param sort_order query string false "정렬 순서" Enums(asc, desc)
// @Success 200 {object} gin.H{data=[]model.CommunityComment,total=int,page=int,page_size=int}
// @Router /api/v1/community/comments [get]
func (c *CommunityController) GetComments(ctx *gin.Context) {
	var query model.CommentListQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 로그인한 사용자 ID 가져오기 (옵션)
	var userID *uint
	if uid, exists := ctx.Get(middleware.UserIDKey); exists {
		u := uid.(uint)
		userID = &u
	}

	comments, total, err := c.service.GetComments(&query, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	page := query.Page
	if page == 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize == 0 {
		pageSize = 50
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data":      comments,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// UpdateComment godoc
// @Summary 댓글 수정
// @Description 댓글을 수정합니다 (작성자 본인 또는 관리자만 가능)
// @Tags community
// @Accept json
// @Produce json
// @Param id path int true "댓글 ID"
// @Param request body model.UpdateCommentRequest true "댓글 수정 요청"
// @Success 200 {object} model.CommunityComment
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/community/comments/{id} [put]
func (c *CommunityController) UpdateComment(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	var req model.UpdateCommentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userRole, exists := ctx.Get(middleware.UserRoleKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	comment, err := c.service.UpdateComment(uint(id), &req, userID.(uint), userRole.(model.UserRole))
	if err != nil {
		if err.Error() == "permission denied" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, comment)
}

// DeleteComment godoc
// @Summary 댓글 삭제
// @Description 댓글을 삭제합니다 (작성자 본인 또는 관리자만 가능)
// @Tags community
// @Accept json
// @Produce json
// @Param id path int true "댓글 ID"
// @Success 204
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/community/comments/{id} [delete]
func (c *CommunityController) DeleteComment(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userRole, exists := ctx.Get(middleware.UserRoleKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := c.service.DeleteComment(uint(id), userID.(uint), userRole.(model.UserRole)); err != nil {
		if err.Error() == "permission denied" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// ==================== Like APIs ====================

// TogglePostLike godoc
// @Summary 게시글 좋아요 토글
// @Description 게시글 좋아요를 추가하거나 취소합니다
// @Tags community
// @Accept json
// @Produce json
// @Param id path int true "게시글 ID"
// @Success 200 {object} gin.H{is_liked=bool}
// @Failure 401 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/community/posts/{id}/like [post]
func (c *CommunityController) TogglePostLike(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	isLiked, err := c.service.TogglePostLike(uint(id), userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"is_liked": isLiked})
}

// ToggleCommentLike godoc
// @Summary 댓글 좋아요 토글
// @Description 댓글 좋아요를 추가하거나 취소합니다
// @Tags community
// @Accept json
// @Produce json
// @Param id path int true "댓글 ID"
// @Success 200 {object} gin.H{is_liked=bool}
// @Failure 401 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/community/comments/{id}/like [post]
func (c *CommunityController) ToggleCommentLike(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	isLiked, err := c.service.ToggleCommentLike(uint(id), userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"is_liked": isLiked})
}

// ==================== QnA APIs ====================

// AcceptAnswer godoc
// @Summary QnA 답변 채택
// @Description QnA 게시글의 답변을 채택합니다 (작성자 본인만 가능)
// @Tags community
// @Accept json
// @Produce json
// @Param id path int true "게시글 ID"
// @Param comment_id path int true "댓글 ID"
// @Success 200 {object} gin.H{message=string}
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/community/posts/{id}/accept/{comment_id} [post]
func (c *CommunityController) AcceptAnswer(ctx *gin.Context) {
	postID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	commentID, err := strconv.ParseUint(ctx.Param("comment_id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := c.service.AcceptAnswer(uint(postID), uint(commentID), userID.(uint)); err != nil {
		if err.Error() == "only post author can accept answers" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "answer accepted successfully"})
}

// ==================== Store Post Management APIs ====================

// PinPost godoc
// @Summary 게시글 고정
// @Description 매장 페이지에 게시글을 상단 고정합니다 (매장 주인만 가능)
// @Tags community
// @Accept json
// @Produce json
// @Param id path int true "게시글 ID"
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/community/posts/{id}/pin [post]
func (c *CommunityController) PinPost(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := c.service.PinPost(uint(id), userID.(uint)); err != nil {
		if err.Error() == "only store owner can pin posts" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "post pinned successfully"})
}

// UnpinPost godoc
// @Summary 게시글 고정 해제
// @Description 매장 페이지의 게시글 상단 고정을 해제합니다 (매장 주인만 가능)
// @Tags community
// @Accept json
// @Produce json
// @Param id path int true "게시글 ID"
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/community/posts/{id}/unpin [post]
func (c *CommunityController) UnpinPost(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := c.service.UnpinPost(uint(id), userID.(uint)); err != nil {
		if err.Error() == "only store owner can unpin posts" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "post unpinned successfully"})
}

// GetStoreGallery godoc
// @Summary 매장 갤러리 조회
// @Description 매장의 모든 이미지를 포함한 게시글 목록을 조회합니다
// @Tags community
// @Accept json
// @Produce json
// @Param store_id query int true "매장 ID"
// @Param page query int false "페이지 번호" default(1)
// @Param page_size query int false "페이지 크기" default(20)
// @Success 200 {object} gin.H{data=[]gin.H,total=int,page=int,page_size=int}
// @Router /api/v1/community/gallery [get]
func (c *CommunityController) GetStoreGallery(ctx *gin.Context) {
	storeID, err := strconv.ParseUint(ctx.Query("store_id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid store id"})
		return
	}

	page, _ := strconv.Atoi(ctx.Query("page"))
	if page == 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(ctx.Query("page_size"))
	if pageSize == 0 {
		pageSize = 20
	}

	gallery, total, err := c.service.GetStoreGallery(uint(storeID), page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data":      gallery,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
