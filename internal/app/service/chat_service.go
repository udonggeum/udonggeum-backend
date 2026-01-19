package service

import (
	"errors"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/internal/websocket"
	"gorm.io/gorm"
)

type ChatService interface {
	// ChatRoom operations
	CreateOrGetChatRoom(user1ID, user2ID uint, roomType model.ChatRoomType, resourceID *uint) (*model.ChatRoom, bool, error)
	GetChatRoom(roomID, userID uint) (*model.ChatRoom, error)
	GetUserChatRooms(userID uint, page, pageSize int) ([]model.ChatRoomWithUnread, int64, error)
	MarkChatRoomAsRead(roomID, userID uint) error

	// Message operations
	SendMessage(roomID, senderID uint, content string, messageType string) (*model.Message, error)
	SendMessageWithFile(roomID, senderID uint, content string, messageType string, fileURL string, fileName string) (*model.Message, error)
	GetChatRoomMessages(roomID, userID uint, page, pageSize int) ([]model.Message, int64, error)
	SearchMessages(userID uint, keyword string, page, pageSize int) ([]model.Message, int64, error)
	UpdateMessage(messageID, userID uint, content string) (*model.Message, error)
	DeleteMessage(messageID, userID uint) error

	// WebSocket operations
	JoinChatRoom(userID, roomID uint) error
	LeaveChatRoom(userID, roomID uint) error
}

type chatService struct {
	db   *gorm.DB
	repo repository.ChatRepository
	hub  *websocket.Hub
}

func NewChatService(db *gorm.DB, repo repository.ChatRepository, hub *websocket.Hub) ChatService {
	return &chatService{
		db:   db,
		repo: repo,
		hub:  hub,
	}
}

// CreateOrGetChatRoom 채팅방 생성 또는 기존 채팅방 가져오기
func (s *chatService) CreateOrGetChatRoom(user1ID, user2ID uint, roomType model.ChatRoomType, resourceID *uint) (*model.ChatRoom, bool, error) {
	// 기존 채팅방 찾기
	existingRoom, err := s.repo.FindExistingChatRoom(user1ID, user2ID, roomType, resourceID)
	if err != nil {
		return nil, false, err
	}

	if existingRoom != nil {
		// 재참여가 필요한지 확인
		needsRejoin := false
		user1NeedsRejoin := (existingRoom.User1ID == user1ID && existingRoom.User1LeftAt != nil) ||
			(existingRoom.User2ID == user1ID && existingRoom.User2LeftAt != nil)
		user2NeedsRejoin := (existingRoom.User1ID == user2ID && existingRoom.User1LeftAt != nil) ||
			(existingRoom.User2ID == user2ID && existingRoom.User2LeftAt != nil)

		if user1NeedsRejoin || user2NeedsRejoin {
			needsRejoin = true
		}

		// 재참여가 필요하면 트랜잭션으로 처리
		if needsRejoin {
			tx := s.db.Begin()
			if tx.Error != nil {
				return nil, false, tx.Error
			}

			defer func() {
				if r := recover(); r != nil {
					tx.Rollback()
				}
			}()

			// User1 재참여
			if user1NeedsRejoin {
				leftAtField := "user1_left_at"
				if existingRoom.User2ID == user1ID {
					leftAtField = "user2_left_at"
				}
				if err := tx.Model(&model.ChatRoom{}).
					Where("id = ?", existingRoom.ID).
					Update(leftAtField, nil).Error; err != nil {
					tx.Rollback()
					return nil, false, err
				}
			}

			// User2 재참여
			if user2NeedsRejoin {
				leftAtField := "user1_left_at"
				if existingRoom.User2ID == user2ID {
					leftAtField = "user2_left_at"
				}
				if err := tx.Model(&model.ChatRoom{}).
					Where("id = ?", existingRoom.ID).
					Update(leftAtField, nil).Error; err != nil {
					tx.Rollback()
					return nil, false, err
				}
			}

			// 트랜잭션 커밋
			if err := tx.Commit().Error; err != nil {
				tx.Rollback()
				return nil, false, err
			}
		}

		// 업데이트된 채팅방 정보를 다시 조회하여 반환
		updatedRoom, err := s.repo.GetChatRoomByIDWithUsers(existingRoom.ID)
		if err != nil {
			return nil, false, err
		}
		return updatedRoom, false, nil
	}

	// 새 채팅방 생성
	newRoom := &model.ChatRoom{
		Type:    roomType,
		User1ID: user1ID,
		User2ID: user2ID,
	}

	if roomType == model.ChatRoomTypeSellGold || roomType == model.ChatRoomTypeBuyGold || roomType == model.ChatRoomTypeSale {
		newRoom.ProductID = resourceID
	} else if roomType == model.ChatRoomTypeStore {
		newRoom.StoreID = resourceID
	}

	if err := s.repo.CreateChatRoom(newRoom); err != nil {
		return nil, false, err
	}

	// 생성된 채팅방을 사용자 정보와 함께 다시 조회
	room, err := s.repo.GetChatRoomByIDWithUsers(newRoom.ID)
	if err != nil {
		return nil, false, err
	}

	return room, true, nil
}

// GetChatRoom 채팅방 조회 (권한 검증 포함)
func (s *chatService) GetChatRoom(roomID, userID uint) (*model.ChatRoom, error) {
	room, err := s.repo.GetChatRoomByIDWithUsers(roomID)
	if err != nil {
		return nil, err
	}

	// 접근 권한 검증
	if room.User1ID != userID && room.User2ID != userID {
		return nil, errors.New("채팅방 접근 권한이 없습니다")
	}

	return room, nil
}

// GetUserChatRooms 사용자의 채팅방 목록 조회
func (s *chatService) GetUserChatRooms(userID uint, page, pageSize int) ([]model.ChatRoomWithUnread, int64, error) {
	offset := (page - 1) * pageSize
	rooms, total, err := s.repo.GetUserChatRooms(userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	// ChatRoomWithUnread로 변환
	result := make([]model.ChatRoomWithUnread, len(rooms))
	for i, room := range rooms {
		result[i] = model.ChatRoomWithUnread{
			ChatRoom: room,
		}

		// 현재 사용자의 읽지 않은 메시지 수 설정
		if room.User1ID == userID {
			result[i].UnreadCount = room.User1UnreadCount
		} else {
			result[i].UnreadCount = room.User2UnreadCount
		}
	}

	return result, total, nil
}

// MarkChatRoomAsRead 채팅방을 읽음 처리
func (s *chatService) MarkChatRoomAsRead(roomID, userID uint) error {
	// 권한 검증
	if _, err := s.GetChatRoom(roomID, userID); err != nil {
		return err
	}

	// 읽지 않은 메시지를 읽음 처리
	if err := s.repo.MarkMessagesAsRead(roomID, userID); err != nil {
		return err
	}

	// 채팅방의 읽지 않은 메시지 수 초기화
	if err := s.repo.ResetUnreadCount(roomID, userID); err != nil {
		return err
	}

	// 상대방에게 읽음 이벤트 전송 (WebSocket)
	wsMessage := map[string]interface{}{
		"type":         "read",
		"chat_room_id": roomID,
		"user_id":      userID,
	}

	// 비동기 전송 (에러는 로깅만 - 실패해도 주요 로직에 영향 없음)
	if err := s.hub.SendToRoom(roomID, wsMessage, userID); err != nil {
		// 로깅은 hub 내부에서 처리
	}

	return nil
}

// SendMessage 메시지 전송
func (s *chatService) SendMessage(roomID, senderID uint, content string, messageType string) (*model.Message, error) {
	// 채팅방 권한 검증
	room, err := s.GetChatRoom(roomID, senderID)
	if err != nil {
		return nil, err
	}

	// 메시지 타입 기본값
	if messageType == "" {
		messageType = "TEXT"
	}

	// 수신자 ID 계산
	recipientID := room.User1ID
	if senderID == room.User1ID {
		recipientID = room.User2ID
	}

	// 읽지 않은 메시지 수 필드 결정
	unreadCountField := "user1_unread_count"
	if recipientID == room.User2ID {
		unreadCountField = "user2_unread_count"
	}

	// 트랜잭션 시작
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 메시지 생성
	message := &model.Message{
		ChatRoomID:  roomID,
		SenderID:    senderID,
		Content:     content,
		MessageType: messageType,
		IsRead:      false,
	}

	if err := tx.Create(message).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 2. 채팅방의 마지막 메시지 정보 업데이트
	if err := tx.Model(&model.ChatRoom{}).
		Where("id = ?", roomID).
		Updates(map[string]interface{}{
			"last_message_id":      message.ID,
			"last_message_content": content,
			"last_message_at":      message.CreatedAt,
		}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 3. 수신자의 읽지 않은 메시지 수 증가
	if err := tx.Model(&model.ChatRoom{}).
		Where("id = ?", roomID).
		UpdateColumn(unreadCountField, gorm.Expr(unreadCountField+" + ?", 1)).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 트랜잭션 커밋
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 메시지를 다시 조회 (Sender 정보 포함)
	createdMessage, err := s.repo.GetMessageByID(message.ID)
	if err != nil {
		return nil, err
	}

	// WebSocket으로 실시간 전송 (트랜잭션 외부에서 처리)
	wsMessage := map[string]interface{}{
		"type":    "new_message",
		"message": createdMessage,
	}
	if err := s.hub.SendToRoom(roomID, wsMessage, senderID); err != nil {
		// 로깅은 hub 내부에서 처리
	}

	return createdMessage, nil
}

// GetChatRoomMessages 채팅방의 메시지 목록 조회
func (s *chatService) GetChatRoomMessages(roomID, userID uint, page, pageSize int) ([]model.Message, int64, error) {
	// 권한 검증
	if _, err := s.GetChatRoom(roomID, userID); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	return s.repo.GetChatRoomMessages(roomID, pageSize, offset)
}

// JoinChatRoom 채팅방 참여 (WebSocket)
func (s *chatService) JoinChatRoom(userID, roomID uint) error {
	// 권한 검증
	if _, err := s.GetChatRoom(roomID, userID); err != nil {
		return err
	}

	// 나간 상태였다면 재입장 처리 (user_left_at을 null로 초기화)
	if err := s.repo.RejoinChatRoom(roomID, userID); err != nil {
		return err
	}

	s.hub.JoinRoom(userID, roomID)
	return nil
}

// LeaveChatRoom 채팅방 나가기 (DB에서 나가기 + WebSocket)
func (s *chatService) LeaveChatRoom(userID, roomID uint) error {
	// 권한 검증
	if _, err := s.GetChatRoom(roomID, userID); err != nil {
		return err
	}

	// DB에서 채팅방 나가기 (soft delete)
	if err := s.repo.LeaveChatRoom(roomID, userID); err != nil {
		return err
	}

	// WebSocket 연결 끊기
	s.hub.LeaveRoom(userID, roomID)

	// 양쪽 모두 나갔으면 채팅방 삭제
	if err := s.repo.DeleteChatRoomIfBothLeft(roomID); err != nil {
		// 삭제 실패해도 무시 (중요하지 않음)
		return nil
	}

	return nil
}

// SearchMessages 메시지 검색
func (s *chatService) SearchMessages(userID uint, keyword string, page, pageSize int) ([]model.Message, int64, error) {
	offset := (page - 1) * pageSize
	return s.repo.SearchMessages(userID, keyword, pageSize, offset)
}

// SendMessageWithFile 파일이 포함된 메시지 전송
func (s *chatService) SendMessageWithFile(roomID, senderID uint, content string, messageType string, fileURL string, fileName string) (*model.Message, error) {
	// 채팅방 권한 검증
	room, err := s.GetChatRoom(roomID, senderID)
	if err != nil {
		return nil, err
	}

	// 메시지 타입 기본값
	if messageType == "" {
		messageType = "TEXT"
	}

	// 수신자 ID 계산
	recipientID := room.User1ID
	if senderID == room.User1ID {
		recipientID = room.User2ID
	}

	// 읽지 않은 메시지 수 필드 결정
	unreadCountField := "user1_unread_count"
	if recipientID == room.User2ID {
		unreadCountField = "user2_unread_count"
	}

	// 마지막 메시지 내용 결정
	lastMessageContent := content
	if messageType == "IMAGE" {
		lastMessageContent = "📷 이미지"
	} else if messageType == "FILE" {
		lastMessageContent = "📎 " + fileName
	}

	// 트랜잭션 시작
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 메시지 생성
	message := &model.Message{
		ChatRoomID:  roomID,
		SenderID:    senderID,
		Content:     content,
		MessageType: messageType,
		FileURL:     fileURL,
		FileName:    fileName,
		IsRead:      false,
	}

	if err := tx.Create(message).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 2. 채팅방의 마지막 메시지 정보 업데이트
	if err := tx.Model(&model.ChatRoom{}).
		Where("id = ?", roomID).
		Updates(map[string]interface{}{
			"last_message_id":      message.ID,
			"last_message_content": lastMessageContent,
			"last_message_at":      message.CreatedAt,
		}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 3. 수신자의 읽지 않은 메시지 수 증가
	if err := tx.Model(&model.ChatRoom{}).
		Where("id = ?", roomID).
		UpdateColumn(unreadCountField, gorm.Expr(unreadCountField+" + ?", 1)).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 트랜잭션 커밋
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 메시지를 다시 조회 (Sender 정보 포함)
	createdMessage, err := s.repo.GetMessageByID(message.ID)
	if err != nil {
		return nil, err
	}

	// WebSocket으로 실시간 전송 (트랜잭션 외부에서 처리)
	wsMessage := map[string]interface{}{
		"type":    "new_message",
		"message": createdMessage,
	}
	if err := s.hub.SendToRoom(roomID, wsMessage, senderID); err != nil {
		// 로깅은 hub 내부에서 처리
	}

	return createdMessage, nil
}

// UpdateMessage 메시지 수정
func (s *chatService) UpdateMessage(messageID, userID uint, content string) (*model.Message, error) {
	// 메시지 조회
	message, err := s.repo.GetMessageByID(messageID)
	if err != nil {
		return nil, err
	}

	// 권한 검증: 본인이 작성한 메시지인지 확인
	if message.SenderID != userID {
		return nil, errors.New("메시지 수정 권한이 없습니다")
	}

	// 삭제된 메시지는 수정 불가
	if message.IsDeleted {
		return nil, errors.New("삭제된 메시지는 수정할 수 없습니다")
	}

	// 메시지 수정
	if err := s.repo.UpdateMessage(messageID, content); err != nil {
		return nil, err
	}

	// 수정된 메시지 다시 조회
	updatedMessage, err := s.repo.GetMessageByID(messageID)
	if err != nil {
		return nil, err
	}

	// WebSocket으로 실시간 전송
	wsMessage := map[string]interface{}{
		"type":    "message_updated",
		"message": updatedMessage,
	}
	if err := s.hub.SendToRoom(updatedMessage.ChatRoomID, wsMessage, userID); err != nil {
		// 로깅은 hub 내부에서 처리
	}

	return updatedMessage, nil
}

// DeleteMessage 메시지 삭제
func (s *chatService) DeleteMessage(messageID, userID uint) error {
	// 메시지 조회
	message, err := s.repo.GetMessageByID(messageID)
	if err != nil {
		return err
	}

	// 권한 검증: 본인이 작성한 메시지인지 확인
	if message.SenderID != userID {
		return errors.New("메시지 삭제 권한이 없습니다")
	}

	// 이미 삭제된 메시지
	if message.IsDeleted {
		return errors.New("이미 삭제된 메시지입니다")
	}

	// 메시지 삭제
	if err := s.repo.DeleteMessage(messageID, userID); err != nil {
		return err
	}

	// WebSocket으로 실시간 전송
	wsMessage := map[string]interface{}{
		"type":       "message_deleted",
		"message_id": messageID,
		"room_id":    message.ChatRoomID,
	}
	if err := s.hub.SendToRoom(message.ChatRoomID, wsMessage, userID); err != nil {
		// 로깅은 hub 내부에서 처리
	}

	return nil
}
