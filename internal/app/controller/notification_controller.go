package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	apperrors "github.com/ikkim/udonggeum-backend/internal/errors"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
)

// NotificationController 알림 컨트롤러
type NotificationController struct {
	service service.NotificationService
}

// NewNotificationController 알림 컨트롤러 생성자
func NewNotificationController(service service.NotificationService) *NotificationController {
	return &NotificationController{
		service: service,
	}
}

// GetNotifications godoc
// @Summary 알림 목록 조회
// @Description 사용자의 알림 목록을 조회합니다
// @Tags notifications
// @Accept json
// @Produce json
// @Param page query int false "페이지 번호" default(1)
// @Param page_size query int false "페이지 크기" default(20)
// @Param type query string false "알림 타입 (new_sell_post, post_comment, store_liked)"
// @Param is_read query bool false "읽음 상태"
// @Success 200 {object} gin.H{data=[]model.Notification,total=int,page=int,page_size=int,unread_count=int}
// @Failure 401 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/notifications [get]
func (c *NotificationController) GetNotifications(ctx *gin.Context) {
	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		apperrors.Unauthorized(ctx, "로그인이 필요합니다")
		return
	}

	// 쿼리 파라미터 파싱
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))

	var notifType *model.NotificationType
	if typeStr := ctx.Query("type"); typeStr != "" {
		t := model.NotificationType(typeStr)
		notifType = &t
	}

	var isRead *bool
	if isReadStr := ctx.Query("is_read"); isReadStr != "" {
		if isReadStr == "true" {
			t := true
			isRead = &t
		} else if isReadStr == "false" {
			f := false
			isRead = &f
		}
	}

	notifications, total, unreadCount, err := c.service.GetNotifications(
		userID.(uint),
		notifType,
		isRead,
		page,
		pageSize,
	)
	if err != nil {
		apperrors.InternalError(ctx, "알림 목록을 조회하는 중 오류가 발생했습니다")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data":          notifications,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
		"unread_count":  unreadCount,
	})
}

// GetUnreadCount godoc
// @Summary 안읽은 알림 개수 조회
// @Description 사용자의 안읽은 알림 개수를 조회합니다
// @Tags notifications
// @Accept json
// @Produce json
// @Success 200 {object} gin.H{unread_count=int}
// @Failure 401 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/notifications/unread-count [get]
func (c *NotificationController) GetUnreadCount(ctx *gin.Context) {
	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		apperrors.Unauthorized(ctx, "로그인이 필요합니다")
		return
	}

	count, err := c.service.GetUnreadCount(userID.(uint))
	if err != nil {
		apperrors.InternalError(ctx, "안읽은 알림 개수를 조회하는 중 오류가 발생했습니다")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"unread_count": count,
	})
}

// MarkAsRead godoc
// @Summary 알림 읽음 처리
// @Description 특정 알림을 읽음 처리합니다
// @Tags notifications
// @Accept json
// @Produce json
// @Param id path int true "알림 ID"
// @Success 200 {object} gin.H{notification=model.Notification}
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 404 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/notifications/{id}/read [patch]
func (c *NotificationController) MarkAsRead(ctx *gin.Context) {
	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		apperrors.Unauthorized(ctx, "로그인이 필요합니다")
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		apperrors.BadRequest(ctx, apperrors.ValidationInvalidID, "잘못된 알림 ID입니다")
		return
	}

	notification, err := c.service.MarkAsRead(uint(id), userID.(uint))
	if err != nil {
		if err.Error() == "unauthorized" {
			apperrors.Forbidden(ctx, "해당 알림에 대한 권한이 없습니다")
			return
		}
		apperrors.NotFound(ctx, apperrors.NotificationNotFound, "알림을 찾을 수 없습니다")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"notification": notification,
	})
}

// MarkAllAsRead godoc
// @Summary 모든 알림 읽음 처리
// @Description 사용자의 모든 알림을 읽음 처리합니다
// @Tags notifications
// @Accept json
// @Produce json
// @Success 200 {object} gin.H{message=string}
// @Failure 401 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/notifications/read-all [patch]
func (c *NotificationController) MarkAllAsRead(ctx *gin.Context) {
	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		apperrors.Unauthorized(ctx, "로그인이 필요합니다")
		return
	}

	if err := c.service.MarkAllAsRead(userID.(uint)); err != nil {
		apperrors.InternalError(ctx, "알림을 읽음 처리하는 중 오류가 발생했습니다")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "모든 알림을 읽음 처리했습니다",
	})
}

// DeleteNotification godoc
// @Summary 알림 삭제
// @Description 특정 알림을 삭제합니다
// @Tags notifications
// @Accept json
// @Produce json
// @Param id path int true "알림 ID"
// @Success 200 {object} gin.H{message=string}
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 404 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/notifications/{id} [delete]
func (c *NotificationController) DeleteNotification(ctx *gin.Context) {
	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		apperrors.Unauthorized(ctx, "로그인이 필요합니다")
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		apperrors.BadRequest(ctx, apperrors.ValidationInvalidID, "잘못된 알림 ID입니다")
		return
	}

	if err := c.service.DeleteNotification(uint(id), userID.(uint)); err != nil {
		if err.Error() == "unauthorized" {
			apperrors.Forbidden(ctx, "해당 알림에 대한 권한이 없습니다")
			return
		}
		apperrors.NotFound(ctx, apperrors.NotificationNotFound, "알림을 찾을 수 없습니다")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "알림이 삭제되었습니다",
	})
}

// GetNotificationSettings godoc
// @Summary 알림 설정 조회
// @Description 사용자의 알림 설정을 조회합니다
// @Tags notifications
// @Accept json
// @Produce json
// @Success 200 {object} gin.H{settings=model.NotificationSettings}
// @Failure 401 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/users/notification-settings [get]
func (c *NotificationController) GetNotificationSettings(ctx *gin.Context) {
	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		apperrors.Unauthorized(ctx, "로그인이 필요합니다")
		return
	}

	settings, err := c.service.GetNotificationSettings(userID.(uint))
	if err != nil {
		apperrors.InternalError(ctx, "알림 설정을 조회하는 중 오류가 발생했습니다")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"settings": settings,
	})
}

// UpdateNotificationSettings godoc
// @Summary 알림 설정 수정
// @Description 사용자의 알림 설정을 수정합니다
// @Tags notifications
// @Accept json
// @Produce json
// @Param request body service.UpdateNotificationSettingsRequest true "알림 설정 수정 요청"
// @Success 200 {object} gin.H{settings=model.NotificationSettings}
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/users/notification-settings [put]
func (c *NotificationController) UpdateNotificationSettings(ctx *gin.Context) {
	userID, exists := ctx.Get(middleware.UserIDKey)
	if !exists {
		apperrors.Unauthorized(ctx, "로그인이 필요합니다")
		return
	}

	var req service.UpdateNotificationSettingsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		apperrors.BadRequest(ctx, apperrors.ValidationInvalidInput, "입력값이 올바르지 않습니다")
		return
	}

	settings, err := c.service.UpdateNotificationSettings(userID.(uint), &req)
	if err != nil {
		apperrors.InternalError(ctx, "알림 설정을 수정하는 중 오류가 발생했습니다")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"settings": settings,
	})
}
