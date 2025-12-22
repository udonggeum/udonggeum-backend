package websocket

import (
	"encoding/json"
	"sync"

	"github.com/ikkim/udonggeum-backend/pkg/logger"
)

// Client WebSocket 클라이언트
type Client struct {
	Hub      *Hub
	Conn     *Conn
	UserID   uint
	Send     chan []byte
	ChatRooms map[uint]bool // 현재 참여 중인 채팅방 IDs
	mu       sync.RWMutex
}

// Hub WebSocket 연결 관리자
type Hub struct {
	// 등록된 클라이언트들 (UserID -> Client)
	clients map[uint]*Client

	// 채팅방별 클라이언트들 (ChatRoomID -> map[UserID]bool)
	rooms map[uint]map[uint]bool

	// 클라이언트 등록
	register chan *Client

	// 클라이언트 등록 해제
	unregister chan *Client

	// 메시지 브로드캐스트
	broadcast chan *BroadcastMessage

	mu sync.RWMutex
}

// BroadcastMessage 브로드캐스트 메시지
type BroadcastMessage struct {
	ChatRoomID uint
	Message    []byte
	SenderID   uint // 발신자는 제외
}

// NewHub Hub 생성
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uint]*Client),
		rooms:      make(map[uint]map[uint]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage),
	}
}

// Run Hub 실행
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.UserID] = client
			h.mu.Unlock()
			logger.Info("WebSocket client registered", map[string]interface{}{
				"user_id": client.UserID,
			})

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.UserID]; ok {
				// 모든 채팅방에서 제거
				client.mu.RLock()
				for roomID := range client.ChatRooms {
					if users, ok := h.rooms[roomID]; ok {
						delete(users, client.UserID)
						if len(users) == 0 {
							delete(h.rooms, roomID)
						}
					}
				}
				client.mu.RUnlock()

				delete(h.clients, client.UserID)
				close(client.Send)
			}
			h.mu.Unlock()
			logger.Info("WebSocket client unregistered", map[string]interface{}{
				"user_id": client.UserID,
			})

		case message := <-h.broadcast:
			h.mu.RLock()
			if users, ok := h.rooms[message.ChatRoomID]; ok {
				for userID := range users {
					// 발신자는 제외
					if userID == message.SenderID {
						continue
					}

					if client, ok := h.clients[userID]; ok {
						select {
						case client.Send <- message.Message:
						default:
							// Send 채널이 막혀있으면 클라이언트 연결 종료
							close(client.Send)
							delete(h.clients, client.UserID)
						}
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// JoinRoom 채팅방 참여
func (h *Hub) JoinRoom(userID, roomID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client, ok := h.clients[userID]; ok {
		client.mu.Lock()
		client.ChatRooms[roomID] = true
		client.mu.Unlock()

		if _, ok := h.rooms[roomID]; !ok {
			h.rooms[roomID] = make(map[uint]bool)
		}
		h.rooms[roomID][userID] = true

		logger.Info("User joined chat room", map[string]interface{}{
			"user_id": userID,
			"room_id": roomID,
		})
	}
}

// LeaveRoom 채팅방 나가기
func (h *Hub) LeaveRoom(userID, roomID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client, ok := h.clients[userID]; ok {
		client.mu.Lock()
		delete(client.ChatRooms, roomID)
		client.mu.Unlock()
	}

	if users, ok := h.rooms[roomID]; ok {
		delete(users, userID)
		if len(users) == 0 {
			delete(h.rooms, roomID)
		}

		logger.Info("User left chat room", map[string]interface{}{
			"user_id": userID,
			"room_id": roomID,
		})
	}
}

// SendToRoom 특정 채팅방에 메시지 전송
func (h *Hub) SendToRoom(roomID uint, message interface{}, senderID uint) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	h.broadcast <- &BroadcastMessage{
		ChatRoomID: roomID,
		Message:    data,
		SenderID:   senderID,
	}

	return nil
}

// Register 클라이언트 등록
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister 클라이언트 등록 해제
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// IsUserOnline 사용자 온라인 여부 확인
func (h *Hub) IsUserOnline(userID uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[userID]
	return ok
}

// GetOnlineUsersInRoom 채팅방의 온라인 사용자 목록
func (h *Hub) GetOnlineUsersInRoom(roomID uint) []uint {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var users []uint
	if roomUsers, ok := h.rooms[roomID]; ok {
		for userID := range roomUsers {
			users = append(users, userID)
		}
	}
	return users
}
