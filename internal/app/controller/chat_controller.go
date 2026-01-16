package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/service"
	"github.com/ikkim/udonggeum-backend/internal/errors"
	"github.com/ikkim/udonggeum-backend/internal/middleware"
	ws "github.com/ikkim/udonggeum-backend/internal/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		// 허용된 도메인 목록
		allowedOrigins := map[string]bool{
			"https://udg.co.kr":           true,
			"http://localhost:5173":       true,  // 개발 환경
			"http://localhost:3000":       true,  // 개발 환경
			"http://43.200.249.22:5173":   true,  // 개발 서버
		}
		return allowedOrigins[origin]
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
	Type         model.ChatRoomType `json:"type" binding:"required,oneof=STORE BUY_GOLD SELL_GOLD SALE"` // 채팅방 타입
	ProductID    *uint            `json:"product_id,omitempty"` // SELL_GOLD, BUY_GOLD 타입일 때
	StoreID      *uint            `json:"store_id,omitempty"`   // STORE 타입일 때
}

// SendMessageRequest 메시지 전송 요청
type SendMessageRequest struct {
	Content     string `json:"content" binding:"required"`
	MessageType string `json:"message_type,omitempty"` // TEXT, IMAGE, FILE 등
	FileURL     string `json:"file_url,omitempty"`     // 파일/이미지 URL
	FileName    string `json:"file_name,omitempty"`    // 원본 파일명
}

// CreateChatRoom 채팅방 생성 또는 기존 채팅방 가져오기
// POST /api/v1/chats/rooms
func (ctrl *ChatController) CreateChatRoom(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		errors.Unauthorized(c, "로그인이 필요합니다")
		return
	}

	var req CreateChatRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("Invalid request", err)
		errors.BadRequest(c, errors.ValidationInvalidInput, "입력값이 올바르지 않습니다")
		return
	}

	// 자기 자신과는 채팅 불가
	if req.TargetUserID == userID {
		errors.BadRequest(c, errors.ChatSelfRoomForbidden, "자기 자신과 채팅할 수 없습니다")
		return
	}

	// 타입에 따라 resourceID 설정
	var resourceID *uint
	if req.Type == model.ChatRoomTypeSellGold || req.Type == model.ChatRoomTypeBuyGold || req.Type == model.ChatRoomTypeSale {
		resourceID = req.ProductID
	} else if req.Type == model.ChatRoomTypeStore {
		resourceID = req.StoreID
	}

	// 채팅방 생성 또는 가져오기
	room, isNew, err := ctrl.chatService.CreateOrGetChatRoom(userID, req.TargetUserID, req.Type, resourceID)
	if err != nil {
		log.Error("Failed to create chat room", err)
		errors.InternalError(c, "채팅방 생성에 실패했습니다")
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
		errors.Unauthorized(c, "로그인이 필요합니다")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	rooms, total, err := ctrl.chatService.GetUserChatRooms(userID, page, pageSize)
	if err != nil {
		log.Error("Failed to get chat rooms", err)
		errors.InternalError(c, "채팅방 목록 조회에 실패했습니다")
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
		errors.Unauthorized(c, "로그인이 필요합니다")
		return
	}

	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.BadRequest(c, errors.ValidationInvalidID, "잘못된 채팅방 ID입니다")
		return
	}

	room, err := ctrl.chatService.GetChatRoom(uint(roomID), userID)
	if err != nil {
		if err.Error() == "unauthorized access to chat room" {
			errors.Forbidden(c, "해당 채팅방에 접근할 권한이 없습니다")
			return
		}
		log.Error("Failed to get chat room", err)
		errors.InternalError(c, "채팅방 조회에 실패했습니다")
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
		errors.Unauthorized(c, "로그인이 필요합니다")
		return
	}

	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.BadRequest(c, errors.ValidationInvalidID, "잘못된 채팅방 ID입니다")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	messages, total, err := ctrl.chatService.GetChatRoomMessages(uint(roomID), userID, page, pageSize)
	if err != nil {
		if err.Error() == "unauthorized access to chat room" {
			errors.Forbidden(c, "해당 채팅방에 접근할 권한이 없습니다")
			return
		}
		log.Error("Failed to get messages", err)
		errors.InternalError(c, "메시지 조회에 실패했습니다")
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
		errors.Unauthorized(c, "로그인이 필요합니다")
		return
	}

	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.BadRequest(c, errors.ValidationInvalidID, "잘못된 채팅방 ID입니다")
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("Invalid request", err)
		errors.BadRequest(c, errors.ValidationInvalidInput, "입력값이 올바르지 않습니다")
		return
	}

	message, err := ctrl.chatService.SendMessageWithFile(uint(roomID), userID, req.Content, req.MessageType, req.FileURL, req.FileName)
	if err != nil {
		if err.Error() == "unauthorized access to chat room" {
			errors.Forbidden(c, "해당 채팅방에 접근할 권한이 없습니다")
			return
		}
		log.Error("Failed to send message", err)
		errors.InternalError(c, "메시지 전송에 실패했습니다")
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
		errors.Unauthorized(c, "로그인이 필요합니다")
		return
	}

	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.BadRequest(c, errors.ValidationInvalidID, "잘못된 채팅방 ID입니다")
		return
	}

	if err := ctrl.chatService.MarkChatRoomAsRead(uint(roomID), userID); err != nil {
		if err.Error() == "unauthorized access to chat room" {
			errors.Forbidden(c, "해당 채팅방에 접근할 권한이 없습니다")
			return
		}
		log.Error("Failed to mark as read", err)
		errors.InternalError(c, "읽음 처리에 실패했습니다")
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
// 쿼리 파라미터로 토큰을 받지만, 로깅하지 않음 (보안)
func (ctrl *ChatController) WebSocketHandler(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)

	// 미들웨어에서 이미 인증 완료
	userID, ok := middleware.GetUserID(c)
	if !ok {
		errors.Unauthorized(c, "로그인이 필요합니다")
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("Failed to upgrade to WebSocket", err)
		return
	}

	client := &ws.Client{
		Hub:           ctrl.hub,
		Conn:          &ws.Conn{Conn: conn},
		UserID:        userID,
		Send:          make(chan []byte, 2048), // 256 → 2048 (8배 증가, 네트워크 느린 클라이언트 대응)
		ChatRooms:     make(map[uint]bool),
		LastResetTime: time.Now(),
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
		errors.Unauthorized(c, "로그인이 필요합니다")
		return
	}

	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.BadRequest(c, errors.ValidationInvalidID, "잘못된 채팅방 ID입니다")
		return
	}

	if err := ctrl.chatService.JoinChatRoom(userID, uint(roomID)); err != nil {
		if err.Error() == "unauthorized access to chat room" {
			errors.Forbidden(c, "해당 채팅방에 접근할 권한이 없습니다")
			return
		}
		log.Error("Failed to join room", err)
		errors.InternalError(c, "채팅방 참여에 실패했습니다")
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
		errors.Unauthorized(c, "로그인이 필요합니다")
		return
	}

	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.BadRequest(c, errors.ValidationInvalidID, "잘못된 채팅방 ID입니다")
		return
	}

	if err := ctrl.chatService.LeaveChatRoom(userID, uint(roomID)); err != nil {
		log.Error("Failed to leave room", err)
		errors.InternalError(c, "채팅방 나가기에 실패했습니다")
		return
	}

	log.Info("Left chat room", map[string]interface{}{
		"room_id": roomID,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// SearchMessages 메시지 검색
// GET /api/v1/chats/search?q=keyword
func (ctrl *ChatController) SearchMessages(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		errors.Unauthorized(c, "로그인이 필요합니다")
		return
	}

	keyword := c.Query("q")
	if keyword == "" {
		errors.BadRequest(c, errors.ValidationRequired, "검색어를 입력해주세요")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	messages, total, err := ctrl.chatService.SearchMessages(userID, keyword, page, pageSize)
	if err != nil {
		log.Error("Failed to search messages", err)
		errors.InternalError(c, "메시지 검색에 실패했습니다")
		return
	}

	log.Info("Messages searched", map[string]interface{}{
		"keyword": keyword,
		"count":   len(messages),
	})

	c.JSON(http.StatusOK, gin.H{
		"messages":    messages,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// UpdateMessageRequest 메시지 수정 요청
type UpdateMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// UpdateMessage 메시지 수정
// PATCH /api/v1/chats/rooms/:id/messages/:messageId
func (ctrl *ChatController) UpdateMessage(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		errors.Unauthorized(c, "로그인이 필요합니다")
		return
	}

	messageID, err := strconv.ParseUint(c.Param("messageId"), 10, 32)
	if err != nil {
		errors.BadRequest(c, errors.ValidationInvalidID, "잘못된 메시지 ID입니다")
		return
	}

	var req UpdateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("Invalid request", err)
		errors.BadRequest(c, errors.ValidationInvalidInput, "입력값이 올바르지 않습니다")
		return
	}

	message, err := ctrl.chatService.UpdateMessage(uint(messageID), userID, req.Content)
	if err != nil {
		if err.Error() == "unauthorized to update this message" {
			errors.Forbidden(c, "해당 메시지를 수정할 권한이 없습니다")
			return
		}
		if err.Error() == "cannot update deleted message" {
			errors.BadRequest(c, errors.ChatUpdateDeleted, "삭제된 메시지는 수정할 수 없습니다")
			return
		}
		log.Error("Failed to update message", err)
		errors.InternalError(c, "메시지 수정에 실패했습니다")
		return
	}

	log.Info("Message updated", map[string]interface{}{
		"message_id": messageID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
}

// DeleteMessage 메시지 삭제
// DELETE /api/v1/chats/rooms/:id/messages/:messageId
func (ctrl *ChatController) DeleteMessage(c *gin.Context) {
	log := middleware.GetLoggerFromContext(c)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		errors.Unauthorized(c, "로그인이 필요합니다")
		return
	}

	messageID, err := strconv.ParseUint(c.Param("messageId"), 10, 32)
	if err != nil {
		errors.BadRequest(c, errors.ValidationInvalidID, "잘못된 메시지 ID입니다")
		return
	}

	if err := ctrl.chatService.DeleteMessage(uint(messageID), userID); err != nil {
		if err.Error() == "unauthorized to delete this message" {
			errors.Forbidden(c, "해당 메시지를 삭제할 권한이 없습니다")
			return
		}
		if err.Error() == "message already deleted" {
			errors.BadRequest(c, errors.ChatMessageDeleted, "이미 삭제된 메시지입니다")
			return
		}
		log.Error("Failed to delete message", err)
		errors.InternalError(c, "메시지 삭제에 실패했습니다")
		return
	}

	log.Info("Message deleted", map[string]interface{}{
		"message_id": messageID,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}
