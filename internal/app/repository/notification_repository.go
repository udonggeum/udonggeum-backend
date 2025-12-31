package repository

import (
	"fmt"

	"gorm.io/gorm"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
)

// NotificationRepository 알림 저장소 인터페이스
type NotificationRepository interface {
	// Notification operations
	CreateNotification(notification *model.Notification) error
	GetNotificationByID(id uint) (*model.Notification, error)
	GetNotifications(userID uint, notifType *model.NotificationType, isRead *bool, limit, offset int) ([]model.Notification, int64, error)
	GetUnreadCount(userID uint) (int64, error)
	MarkAsRead(id uint) error
	MarkAllAsRead(userID uint) error
	DeleteNotification(id uint) error

	// NotificationSettings operations
	GetNotificationSettings(userID uint) (*model.NotificationSettings, error)
	CreateNotificationSettings(settings *model.NotificationSettings) error
	UpdateNotificationSettings(settings *model.NotificationSettings) error

	// Utility operations
	GetAdminsForNewSellPost(region, district string) ([]uint, error)
}

type notificationRepository struct {
	db *gorm.DB
}

// NewNotificationRepository 알림 저장소 생성자
func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

// CreateNotification 알림 생성
func (r *notificationRepository) CreateNotification(notification *model.Notification) error {
	return r.db.Create(notification).Error
}

// GetNotificationByID 알림 ID로 조회
func (r *notificationRepository) GetNotificationByID(id uint) (*model.Notification, error) {
	var notification model.Notification
	if err := r.db.First(&notification, id).Error; err != nil {
		return nil, err
	}
	return &notification, nil
}

// GetNotifications 알림 목록 조회
func (r *notificationRepository) GetNotifications(
	userID uint,
	notifType *model.NotificationType,
	isRead *bool,
	limit, offset int,
) ([]model.Notification, int64, error) {
	var notifications []model.Notification
	var total int64

	query := r.db.Model(&model.Notification{}).Where("user_id = ?", userID)

	// 타입 필터
	if notifType != nil {
		query = query.Where("type = ?", *notifType)
	}

	// 읽음 상태 필터
	if isRead != nil {
		query = query.Where("is_read = ?", *isRead)
	}

	// 총 개수 조회
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 페이지네이션
	query = query.Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&notifications).Error; err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

// GetUnreadCount 안읽은 알림 개수 조회
func (r *notificationRepository) GetUnreadCount(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// MarkAsRead 알림 읽음 처리
func (r *notificationRepository) MarkAsRead(id uint) error {
	return r.db.Model(&model.Notification{}).
		Where("id = ?", id).
		Update("is_read", true).Error
}

// MarkAllAsRead 모든 알림 읽음 처리
func (r *notificationRepository) MarkAllAsRead(userID uint) error {
	return r.db.Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}

// DeleteNotification 알림 삭제
func (r *notificationRepository) DeleteNotification(id uint) error {
	return r.db.Delete(&model.Notification{}, id).Error
}

// GetNotificationSettings 알림 설정 조회
func (r *notificationRepository) GetNotificationSettings(userID uint) (*model.NotificationSettings, error) {
	var settings model.NotificationSettings
	err := r.db.Where("user_id = ?", userID).First(&settings).Error
	if err == gorm.ErrRecordNotFound {
		// 설정이 없으면 기본값으로 생성
		settings = model.NotificationSettings{
			UserID:               userID,
			SellPostNotification: true,
			SellPostRange:        model.NotificationRangeDistrict,
			SelectedRegions:      []string{}, // 빈 배열로 초기화
			CommentNotification:  true,
			LikeNotification:     true,
		}
		if err := r.CreateNotificationSettings(&settings); err != nil {
			return nil, err
		}
		return &settings, nil
	}
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// CreateNotificationSettings 알림 설정 생성
func (r *notificationRepository) CreateNotificationSettings(settings *model.NotificationSettings) error {
	return r.db.Create(settings).Error
}

// UpdateNotificationSettings 알림 설정 수정
func (r *notificationRepository) UpdateNotificationSettings(settings *model.NotificationSettings) error {
	return r.db.Save(settings).Error
}

// GetAdminsForNewSellPost 금 판매글에 대한 알림을 받을 관리자 목록 조회
func (r *notificationRepository) GetAdminsForNewSellPost(region, district string) ([]uint, error) {
	var userIDs []uint

	// 1. 알림을 활성화한 관리자 조회
	var settings []model.NotificationSettings
	err := r.db.Joins("JOIN users ON users.id = notification_settings.user_id").
		Where("users.role = ? AND notification_settings.sell_post_notification = ?",
			model.RoleAdmin, true).
		Find(&settings).Error

	if err != nil {
		fmt.Printf("[DEBUG] Error querying notification settings: %v\n", err)
		return nil, err
	}

	fmt.Printf("[DEBUG] Found %d admin settings with notification enabled\n", len(settings))

	// 게시글 위치 문자열 생성 (예: "서울 강남구")
	postLocation := region + " " + district
	fmt.Printf("[DEBUG] Post location: %s\n", postLocation)

	// 2. 각 관리자의 알림 설정 확인
	for _, setting := range settings {
		shouldNotify := false

		fmt.Printf("[DEBUG] Checking admin user_id=%d, selected_regions=%v\n", setting.UserID, setting.SelectedRegions)

		// 새로운 방식: 선택한 지역 목록 기반 (우선)
		if len(setting.SelectedRegions) > 0 {
			// selected_regions 배열에 게시글 위치가 포함되어 있는지 확인
			for _, selectedRegion := range setting.SelectedRegions {
				fmt.Printf("[DEBUG]   Comparing selectedRegion='%s' with postLocation='%s'\n", selectedRegion, postLocation)
				if selectedRegion == postLocation {
					shouldNotify = true
					fmt.Printf("[DEBUG]   Match found! Will notify admin %d\n", setting.UserID)
					break
				}
			}
		} else {
			fmt.Printf("[DEBUG]   Using legacy sell_post_range=%s\n", setting.SellPostRange)
			// 기존 방식: sell_post_range 기반 (하위 호환성)
			// 관리자의 매장 조회
			var store model.Store
			err := r.db.Where("user_id = ?", setting.UserID).First(&store).Error
			if err != nil {
				fmt.Printf("[DEBUG]   No store found for admin %d, skipping\n", setting.UserID)
				continue // 매장이 없으면 스킵
			}

			// 알림 범위에 따라 필터링
			switch setting.SellPostRange {
			case model.NotificationRangeDistrict:
				// 같은 구
				if store.Region == region && store.District == district {
					shouldNotify = true
				}
			case model.NotificationRangeRegion:
				// 같은 시
				if store.Region == region {
					shouldNotify = true
				}
			case model.NotificationRangeNationwide:
				// 전국
				shouldNotify = true
			}
		}

		if shouldNotify {
			userIDs = append(userIDs, setting.UserID)
		}
	}

	fmt.Printf("[DEBUG] Final list of admins to notify: %v\n", userIDs)
	return userIDs, nil
}
