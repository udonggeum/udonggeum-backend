package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
	ws "github.com/ikkim/udonggeum-backend/internal/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// CORS 설정에 따라 조정
		return true
	},
}

type ChatController struct {
	chatService service.ChatService
	hub         *ws.Hub
}

func NewChatController(chatService service.ChatService, hub *ws.Hub) *ChatController {
	return &ChatController{
		chatService: chatService,
		hub:         hub,
	}
}

// CreateChatRoomRequest 채팅방 생성 요청
type CreateChatRoomRequest struct {
	TargetUserID uint             `json:"target_user_id" binding:"required"` // 대화 상대
	Type         model.ChatRoomType `json:"type" binding:"required,oneof=SALE STORE"`
	ProductID    *uint            `json:"product_id,omitempty"` // SALE 타입일 때
	StoreID      *uint            `json:"store_id,omitempty"`   // STORE 타입일 때
}

// SendMessageRequest 메시지 전송 요청
type SendMessageRequest struct {
	Content     string `json:"content" binding:"required"`
	MessageType string `json:"message_type,omitempty"` // TEXT, IMAGE, FILE 등
}

// CreateChatRoom 채팅방 생성 또는 기존 채팅방 가져오기
// POST /api/v1/chats/rooms
func (ctrl *ChatController) CreateChatRoom(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateChatRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("Invalid request", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	// 자기 자신과는 채팅 불가
	if req.TargetUserID == userID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot create chat room with yourself",
		})
		return
	}

	// 타입에 따라 resourceID 설정
	var resourceID *uint
	if req.Type == model.ChatRoomTypeSale {
		resourceID = req.ProductID
	} else if req.Type == model.ChatRoomTypeStore {
		resourceID = req.StoreID
	}

	// 채팅방 생성 또는 가져오기
	room, isNew, err := ctrl.chatService.CreateOrGetChatRoom(userID, req.TargetUserID, req.Type, resourceID)
	if err != nil {
		log.Error("Failed to create chat room", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create chat room",
		})
		return
	}

	log.Info("Chat room created/retrieved", map[string]interface{}{
		"room_id": room.ID,
		"is_new":  isNew,
	})

	c.JSON(http.StatusOK, gin.H{
		"room":   room,
		"is_new": isNew,
	})
}

// GetChatRooms 사용자의 채팅방 목록 조회
// GET /api/v1/chats/rooms
func (ctrl *ChatController) GetChatRooms(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	rooms, total, err := ctrl.chatService.GetUserChatRooms(userID, page, pageSize)
	if err != nil {
		log.Error("Failed to get chat rooms", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get chat rooms",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rooms":      rooms,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetChatRoom 채팅방 상세 조회
// GET /api/v1/chats/rooms/:id
func (ctrl *ChatController) GetChatRoom(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid room ID",
		})
		return
	}

	room, err := ctrl.chatService.GetChatRoom(uint(roomID), userID)
	if err != nil {
		if err.Error() == "unauthorized access to chat room" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Unauthorized access to chat room",
			})
			return
		}
		log.Error("Failed to get chat room", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get chat room",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"room": room,
	})
}

// GetMessages 채팅방의 메시지 목록 조회
// GET /api/v1/chats/rooms/:id/messages
func (ctrl *ChatController) GetMessages(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid room ID",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	messages, total, err := ctrl.chatService.GetChatRoomMessages(uint(roomID), userID, page, pageSize)
	if err != nil {
		if err.Error() == "unauthorized access to chat room" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Unauthorized access to chat room",
			})
			return
		}
		log.Error("Failed to get messages", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get messages",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages":    messages,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// SendMessage 메시지 전송
// POST /api/v1/chats/rooms/:id/messages
func (ctrl *ChatController) SendMessage(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid room ID",
		})
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("Invalid request", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	message, err := ctrl.chatService.SendMessage(uint(roomID), userID, req.Content, req.MessageType)
	if err != nil {
		if err.Error() == "unauthorized access to chat room" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Unauthorized access to chat room",
			})
			return
		}
		log.Error("Failed to send message", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to send message",
		})
		return
	}

	log.Info("Message sent", map[string]interface{}{
		"room_id":    roomID,
		"message_id": message.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
}

// MarkAsRead 채팅방을 읽음 처리
// POST /api/v1/chats/rooms/:id/read
func (ctrl *ChatController) MarkAsRead(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid room ID",
		})
		return
	}

	if err := ctrl.chatService.MarkChatRoomAsRead(uint(roomID), userID); err != nil {
		if err.Error() == "unauthorized access to chat room" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Unauthorized access to chat room",
			})
			return
		}
		log.Error("Failed to mark as read", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to mark as read",
		})
		return
	}

	log.Info("Chat room marked as read", map[string]interface{}{
		"room_id": roomID,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// WebSocketHandler WebSocket 연결 처리
// GET /api/v1/chats/ws
func (ctrl *ChatController) WebSocketHandler(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("Failed to upgrade to WebSocket", err)
		return
	}

	client := &ws.Client{
		Hub:       ctrl.hub,
		Conn:      &ws.Conn{Conn: conn},
		UserID:    userID,
		Send:      make(chan []byte, 256),
		ChatRooms: make(map[uint]bool),
	}

	ctrl.hub.Register(client)

	// goroutine으로 읽기/쓰기 시작
	go client.WritePump()
	go client.ReadPump()

	log.Info("WebSocket connection established", map[string]interface{}{
		"user_id": userID,
	})
}

// JoinRoom 채팅방 참여 (WebSocket)
// POST /api/v1/chats/rooms/:id/join
func (ctrl *ChatController) JoinRoom(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid room ID",
		})
		return
	}

	if err := ctrl.chatService.JoinChatRoom(userID, uint(roomID)); err != nil {
		if err.Error() == "unauthorized access to chat room" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Unauthorized access to chat room",
			})
			return
		}
		log.Error("Failed to join room", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to join room",
		})
		return
	}

	log.Info("Joined chat room", map[string]interface{}{
		"room_id": roomID,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// LeaveRoom 채팅방 나가기 (WebSocket)
// POST /api/v1/chats/rooms/:id/leave
func (ctrl *ChatController) LeaveRoom(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid room ID",
		})
		return
	}

	if err := ctrl.chatService.LeaveChatRoom(userID, uint(roomID)); err != nil {
		log.Error("Failed to leave room", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to leave room",
		})
		return
	}

	log.Info("Left chat room", map[string]interface{}{
		"room_id": roomID,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}
