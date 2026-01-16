package service

import (
	"fmt"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/internal/websocket"
)

// NotificationService 알림 서비스 인터페이스
type NotificationService interface {
	// Notification operations
	GetNotifications(userID uint, notifType *model.NotificationType, isRead *bool, page, pageSize int) ([]model.Notification, int64, int64, error)
	GetUnreadCount(userID uint) (int64, error)
	MarkAsRead(notificationID, userID uint) (*model.Notification, error)
	MarkAllAsRead(userID uint) error
	DeleteNotification(notificationID, userID uint) error

	// NotificationSettings operations
	GetNotificationSettings(userID uint) (*model.NotificationSettings, error)
	UpdateNotificationSettings(userID uint, req *UpdateNotificationSettingsRequest) (*model.NotificationSettings, error)

	// Notification creation helpers
	CreateNewSellPostNotification(post *model.CommunityPost) error
	CreatePostCommentNotification(comment *model.CommunityComment, post *model.CommunityPost) error
	CreateStoreLikedNotification(storeID, likedByUserID uint) error
}

type notificationService struct {
	repo repository.NotificationRepository
	hub  *websocket.Hub
}

// UpdateNotificationSettingsRequest 알림 설정 수정 요청
type UpdateNotificationSettingsRequest struct {
	SellPostNotification *bool                    `json:"sell_post_notification"`
	SellPostRange        *model.NotificationRange `json:"sell_post_range"` // Deprecated
	SelectedRegions      *[]string                `json:"selected_regions"`
	CommentNotification  *bool                    `json:"comment_notification"`
	LikeNotification     *bool                    `json:"like_notification"`
}

// NewNotificationService 알림 서비스 생성자
func NewNotificationService(repo repository.NotificationRepository, hub *websocket.Hub) NotificationService {
	return &notificationService{
		repo: repo,
		hub:  hub,
	}
}

// GetNotifications 알림 목록 조회
func (s *notificationService) GetNotifications(
	userID uint,
	notifType *model.NotificationType,
	isRead *bool,
	page, pageSize int,
) ([]model.Notification, int64, int64, error) {
	// 페이지 기본값
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	notifications, total, err := s.repo.GetNotifications(userID, notifType, isRead, pageSize, offset)
	if err != nil {
		return nil, 0, 0, err
	}

	// 안읽은 개수
	unreadCount, err := s.repo.GetUnreadCount(userID)
	if err != nil {
		return nil, 0, 0, err
	}

	return notifications, total, unreadCount, nil
}

// GetUnreadCount 안읽은 알림 개수 조회
func (s *notificationService) GetUnreadCount(userID uint) (int64, error) {
	return s.repo.GetUnreadCount(userID)
}

// MarkAsRead 알림 읽음 처리
func (s *notificationService) MarkAsRead(notificationID, userID uint) (*model.Notification, error) {
	// 알림 조회
	notification, err := s.repo.GetNotificationByID(notificationID)
	if err != nil {
		return nil, fmt.Errorf("알림을 찾을 수 없습니다")
	}

	// 권한 확인
	if notification.UserID != userID {
		return nil, fmt.Errorf("권한이 없습니다")
	}

	// 이미 읽은 알림이면 그대로 반환
	if notification.IsRead {
		return notification, nil
	}

	// 읽음 처리
	if err := s.repo.MarkAsRead(notificationID); err != nil {
		return nil, err
	}

	notification.IsRead = true
	return notification, nil
}

// MarkAllAsRead 모든 알림 읽음 처리
func (s *notificationService) MarkAllAsRead(userID uint) error {
	return s.repo.MarkAllAsRead(userID)
}

// DeleteNotification 알림 삭제
func (s *notificationService) DeleteNotification(notificationID, userID uint) error {
	// 알림 조회
	notification, err := s.repo.GetNotificationByID(notificationID)
	if err != nil {
		return fmt.Errorf("알림을 찾을 수 없습니다")
	}

	// 권한 확인
	if notification.UserID != userID {
		return fmt.Errorf("권한이 없습니다")
	}

	return s.repo.DeleteNotification(notificationID)
}

// GetNotificationSettings 알림 설정 조회
func (s *notificationService) GetNotificationSettings(userID uint) (*model.NotificationSettings, error) {
	return s.repo.GetNotificationSettings(userID)
}

// UpdateNotificationSettings 알림 설정 수정
func (s *notificationService) UpdateNotificationSettings(
	userID uint,
	req *UpdateNotificationSettingsRequest,
) (*model.NotificationSettings, error) {
	// 기존 설정 조회 (없으면 자동 생성)
	settings, err := s.repo.GetNotificationSettings(userID)
	if err != nil {
		return nil, err
	}

	// 설정 업데이트
	if req.SellPostNotification != nil {
		settings.SellPostNotification = *req.SellPostNotification
	}
	if req.SellPostRange != nil {
		settings.SellPostRange = *req.SellPostRange
	}
	if req.SelectedRegions != nil {
		settings.SelectedRegions = *req.SelectedRegions
	}
	if req.CommentNotification != nil {
		settings.CommentNotification = *req.CommentNotification
	}
	if req.LikeNotification != nil {
		settings.LikeNotification = *req.LikeNotification
	}

	if err := s.repo.UpdateNotificationSettings(settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// CreateNewSellPostNotification 금 판매글 알림 생성
func (s *notificationService) CreateNewSellPostNotification(post *model.CommunityPost) error {
	// 금 판매글이 아니면 알림 생성 안 함
	if post.Type != model.TypeSellGold {
		fmt.Printf("[DEBUG] Not a sell_gold post, skipping notification. Type: %s\n", post.Type)
		return nil
	}

	// region, district가 없으면 알림 생성 안 함
	if post.Region == nil || post.District == nil {
		fmt.Printf("[DEBUG] Region or District is nil. Region: %v, District: %v\n", post.Region, post.District)
		return nil
	}

	fmt.Printf("[DEBUG] Creating notification for new sell post in %s %s\n", *post.Region, *post.District)

	// 알림을 받을 관리자 목록 조회
	adminIDs, err := s.repo.GetAdminsForNewSellPost(*post.Region, *post.District)
	if err != nil {
		fmt.Printf("[DEBUG] Error getting admins: %v\n", err)
		return err
	}

	fmt.Printf("[DEBUG] Found %d admins to notify: %v\n", len(adminIDs), adminIDs)

	// 각 관리자에게 알림 생성
	for _, adminID := range adminIDs {
		notification := &model.Notification{
			UserID:         adminID,
			Type:           model.NotificationTypeNewSellPost,
			Title:          fmt.Sprintf("%s %s에 금 판매글이 올라왔어요", *post.Region, *post.District),
			Content:        post.Title,
			Link:           fmt.Sprintf("/community/posts/%d", post.ID),
			IsRead:         false,
			RelatedPostID:  &post.ID,
			RelatedUserID:  &post.UserID,
		}

		if err := s.repo.CreateNotification(notification); err != nil {
			// 로그만 남기고 계속 진행
			fmt.Printf("[DEBUG] Failed to create notification for admin %d: %v\n", adminID, err)
		} else {
			fmt.Printf("[DEBUG] Successfully created notification for admin %d\n", adminID)

			// WebSocket으로 실시간 알림 전송
			if s.hub != nil {
				unreadCount, _ := s.repo.GetUnreadCount(adminID)
				wsMessage := map[string]interface{}{
					"type":          "new_notification",
					"unread_count":  unreadCount,
					"notification":  notification,
				}
				if err := s.hub.SendNotificationToUser(adminID, wsMessage); err != nil {
					fmt.Printf("[DEBUG] Failed to send WebSocket notification to admin %d: %v\n", adminID, err)
				}
			}
		}
	}

	return nil
}

// CreatePostCommentNotification 게시글 댓글 알림 생성
func (s *notificationService) CreatePostCommentNotification(comment *model.CommunityComment, post *model.CommunityPost) error {
	// 본인 댓글이면 알림 생성 안 함
	if comment.UserID == post.UserID {
		return nil
	}

	// 게시글 작성자의 알림 설정 확인
	settings, err := s.repo.GetNotificationSettings(post.UserID)
	if err != nil {
		// 설정이 없으면 기본값으로 알림 생성
		fmt.Printf("Failed to get notification settings for user %d: %v\n", post.UserID, err)
	} else if !settings.CommentNotification {
		// 댓글 알림 비활성화되어 있으면 알림 생성 안 함
		return nil
	}

	notification := &model.Notification{
		UserID:         post.UserID,
		Type:           model.NotificationTypePostComment,
		Title:          "내 게시글에 댓글이 달렸어요",
		Content:        comment.Content,
		Link:           fmt.Sprintf("/community/posts/%d", post.ID),
		IsRead:         false,
		RelatedPostID:  &post.ID,
		RelatedUserID:  &comment.UserID,
	}

	if err := s.repo.CreateNotification(notification); err != nil {
		return err
	}

	// WebSocket으로 실시간 알림 전송
	if s.hub != nil {
		unreadCount, _ := s.repo.GetUnreadCount(post.UserID)
		wsMessage := map[string]interface{}{
			"type":          "new_notification",
			"unread_count":  unreadCount,
			"notification":  notification,
		}
		if err := s.hub.SendNotificationToUser(post.UserID, wsMessage); err != nil {
			fmt.Printf("Failed to send WebSocket notification: %v\n", err)
		}
	}

	return nil
}

// CreateStoreLikedNotification 매장 찜 알림 생성
func (s *notificationService) CreateStoreLikedNotification(storeID, likedByUserID uint) error {
	// TODO: storeID로 매장 조회하여 UserID 찾기
	// 지금은 StoreRepository가 없으므로 나중에 구현

	// 매장 주인의 알림 설정 확인
	// settings, err := s.repo.GetNotificationSettings(storeOwnerUserID)
	// if err != nil || !settings.LikeNotification {
	// 	return nil
	// }

	// notification := &model.Notification{
	// 	UserID:         storeOwnerUserID,
	// 	Type:           model.NotificationTypeStoreLiked,
	// 	Title:          "누군가 내 매장을 찜했어요",
	// 	Content:        "",
	// 	Link:           fmt.Sprintf("/stores/%d", storeID),
	// 	IsRead:         false,
	// 	RelatedStoreID: &storeID,
	// 	RelatedUserID:  &likedByUserID,
	// }

	// return s.repo.CreateNotification(notification)

	return nil
}
