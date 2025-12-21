package service

import (
	"errors"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/internal/websocket"
)

type ChatService interface {
	// ChatRoom operations
	CreateOrGetChatRoom(user1ID, user2ID uint, roomType model.ChatRoomType, resourceID *uint) (*model.ChatRoom, bool, error)
	GetChatRoom(roomID, userID uint) (*model.ChatRoom, error)
	GetUserChatRooms(userID uint, page, pageSize int) ([]model.ChatRoomWithUnread, int64, error)
	MarkChatRoomAsRead(roomID, userID uint) error

	// Message operations
	SendMessage(roomID, senderID uint, content string, messageType string) (*model.Message, error)
	GetChatRoomMessages(roomID, userID uint, page, pageSize int) ([]model.Message, int64, error)

	// WebSocket operations
	JoinChatRoom(userID, roomID uint) error
	LeaveChatRoom(userID, roomID uint) error
}

type chatService struct {
	repo repository.ChatRepository
	hub  *websocket.Hub
}

func NewChatService(repo repository.ChatRepository, hub *websocket.Hub) ChatService {
	return &chatService{
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
		// 기존 채팅방이 있으면 반환
		return existingRoom, false, nil
	}

	// 새 채팅방 생성
	newRoom := &model.ChatRoom{
		Type:    roomType,
		User1ID: user1ID,
		User2ID: user2ID,
	}

	if roomType == model.ChatRoomTypeSale {
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
		return nil, errors.New("unauthorized access to chat room")
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
	return s.repo.ResetUnreadCount(roomID, userID)
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

	// 메시지 생성
	message := &model.Message{
		ChatRoomID:  roomID,
		SenderID:    senderID,
		Content:     content,
		MessageType: messageType,
		IsRead:      false,
	}

	if err := s.repo.CreateMessage(message); err != nil {
		return nil, err
	}

	// 메시지를 다시 조회 (Sender 정보 포함)
	createdMessage, err := s.repo.GetMessageByID(message.ID)
	if err != nil {
		return nil, err
	}

	// 채팅방의 마지막 메시지 정보 업데이트
	if err := s.repo.UpdateChatRoomLastMessage(roomID, message.ID, content, message.CreatedAt); err != nil {
		return nil, err
	}

	// 수신자의 읽지 않은 메시지 수 증가
	recipientID := room.User1ID
	if senderID == room.User1ID {
		recipientID = room.User2ID
	}
	if err := s.repo.IncrementUnreadCount(roomID, recipientID); err != nil {
		return nil, err
	}

	// WebSocket으로 실시간 전송
	go func() {
		wsMessage := map[string]interface{}{
			"type":    "new_message",
			"message": createdMessage,
		}
		s.hub.SendToRoom(roomID, wsMessage, senderID)
	}()

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

	s.hub.JoinRoom(userID, roomID)
	return nil
}

// LeaveChatRoom 채팅방 나가기 (WebSocket)
func (s *chatService) LeaveChatRoom(userID, roomID uint) error {
	s.hub.LeaveRoom(userID, roomID)
	return nil
}
