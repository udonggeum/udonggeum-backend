package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/ikkim/udonggeum-backend/pkg/logger"
)

const (
	// Rate limiting: 최대 메시지 수 (1초당)
	maxMessagesPerSecond = 10
)

// ClientMessage 클라이언트로부터 받은 메시지
type ClientMessage struct {
	Type       string `json:"type"`        // typing_start, typing_stop
	ChatRoomID uint   `json:"chat_room_id"`
}

// Client WebSocket 클라이언트
type Client struct {
	Hub           *Hub
	Conn          *Conn
	UserID        uint
	Send          chan []byte
	ChatRooms     map[uint]bool // 현재 참여 중인 채팅방 IDs
	mu            sync.RWMutex
	MessageCount  int       // 최근 1초간 받은 메시지 수
	LastResetTime time.Time // 마지막 카운터 리셋 시간
	RateMu        sync.Mutex
}

// Hub WebSocket 연결 관리자
type Hub struct {
	// 등록된 클라이언트들 (UserID -> []*Client - 멀티 디바이스 지원)
	clients map[uint][]*Client

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
		clients:    make(map[uint][]*Client),
		rooms:      make(map[uint]map[uint]bool),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		broadcast:  make(chan *BroadcastMessage, 1024),
	}
}

// Run Hub 실행
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// 멀티 디바이스 지원: 클라이언트 리스트에 추가
			h.clients[client.UserID] = append(h.clients[client.UserID], client)
			h.mu.Unlock()
			logger.Info("WebSocket client registered", map[string]interface{}{
				"user_id":        client.UserID,
				"total_sessions": len(h.clients[client.UserID]),
			})

		case client := <-h.unregister:
			h.mu.Lock()
			if clientList, ok := h.clients[client.UserID]; ok {
				// 해당 클라이언트만 리스트에서 제거
				newList := make([]*Client, 0, len(clientList))
				for _, c := range clientList {
					if c != client {
						newList = append(newList, c)
					}
				}

				if len(newList) == 0 {
					// 마지막 세션이면 맵에서 삭제
					delete(h.clients, client.UserID)

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
				} else {
					h.clients[client.UserID] = newList
				}

				close(client.Send)
			}
			h.mu.Unlock()
			logger.Info("WebSocket client unregistered", map[string]interface{}{
				"user_id":             client.UserID,
				"remaining_sessions": len(h.clients[client.UserID]),
			})

		case message := <-h.broadcast:
			h.mu.RLock()
			if users, ok := h.rooms[message.ChatRoomID]; ok {
				for userID := range users {
					// 발신자는 제외
					if userID == message.SenderID {
						continue
					}

					// 멀티 디바이스: 모든 세션에 전송
					if clientList, ok := h.clients[userID]; ok {
						for _, client := range clientList {
							select {
							case client.Send <- message.Message:
								// 전송 성공
							default:
								// Send 채널이 막혀있음 - 비동기로 정리
								go h.Unregister(client)
								logger.Warn("Client send buffer full, disconnecting", map[string]interface{}{
									"user_id": userID,
								})
							}
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

	// 멀티 디바이스: 모든 세션을 채팅방에 추가
	if clientList, ok := h.clients[userID]; ok {
		for _, client := range clientList {
			client.mu.Lock()
			client.ChatRooms[roomID] = true
			client.mu.Unlock()
		}

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

	// 멀티 디바이스: 모든 세션에서 채팅방 제거
	if clientList, ok := h.clients[userID]; ok {
		for _, client := range clientList {
			client.mu.Lock()
			delete(client.ChatRooms, roomID)
			client.mu.Unlock()
		}
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
		logger.Error("Failed to marshal message", err, nil)
		return err
	}

	select {
	case h.broadcast <- &BroadcastMessage{
		ChatRoomID: roomID,
		Message:    data,
		SenderID:   senderID,
	}:
		return nil
	default:
		logger.Warn("Broadcast channel full, message dropped", map[string]interface{}{
			"room_id": roomID,
		})
		return nil // 메시지 손실을 허용 (주요 로직에 영향 없음)
	}
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

// HandleClientMessage 클라이언트 메시지 처리
func (h *Hub) HandleClientMessage(client *Client, message []byte) {
	// Rate limiting 체크
	client.RateMu.Lock()
	now := time.Now()
	if now.Sub(client.LastResetTime) >= time.Second {
		// 1초가 지났으면 카운터 리셋
		client.MessageCount = 0
		client.LastResetTime = now
	}
	client.MessageCount++
	count := client.MessageCount
	client.RateMu.Unlock()

	if count > maxMessagesPerSecond {
		logger.Warn("Rate limit exceeded", map[string]interface{}{
			"user_id": client.UserID,
			"count":   count,
		})
		return
	}

	var msg ClientMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		logger.Warn("Failed to parse client message", map[string]interface{}{
			"user_id": client.UserID,
			"error":   err.Error(),
		})
		return
	}

	// typing 이벤트 처리
	if msg.Type == "typing_start" || msg.Type == "typing_stop" {
		// 클라이언트가 해당 채팅방에 참여 중인지 확인
		client.mu.RLock()
		_, isInRoom := client.ChatRooms[msg.ChatRoomID]
		client.mu.RUnlock()

		if !isInRoom {
			logger.Warn("User not in chat room", map[string]interface{}{
				"user_id": client.UserID,
				"room_id": msg.ChatRoomID,
			})
			return
		}

		// 같은 채팅방의 다른 사용자에게 브로드캐스트
		response := map[string]interface{}{
			"type":        msg.Type,
			"chat_room_id": msg.ChatRoomID,
			"user_id":     client.UserID,
		}

		if err := h.SendToRoom(msg.ChatRoomID, response, client.UserID); err != nil {
			logger.Error("Failed to broadcast typing event", err, map[string]interface{}{
				"user_id": client.UserID,
				"room_id": msg.ChatRoomID,
			})
		}
	}
}
