package websocket

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 100 * 1024 // 100KB (기존 512KB에서 축소)

	// Rate limiting: 최대 메시지 수 (1초당)
	maxMessagesPerSecond = 10
)

// Conn WebSocket 연결 래퍼
type Conn struct {
	*websocket.Conn
}

// ReadPump 클라이언트로부터 메시지 읽기
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket read error", err, map[string]interface{}{
					"user_id": c.UserID,
				})
			}
			break
		}

		// Handle typing events
		c.Hub.HandleClientMessage(c, message)

		logger.Debug("WebSocket message received", map[string]interface{}{
			"user_id": c.UserID,
			"message": string(message),
		})
	}
}

// WritePump 클라이언트로 메시지 쓰기
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub가 채널을 닫음
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 메시지를 개별적으로 JSON으로 전송
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Error("Failed to write message", err, map[string]interface{}{
					"user_id": c.UserID,
				})
				return
			}

			// 대기 중인 메시지도 개별적으로 전송 (배치 처리)
			n := len(c.Send)
			for i := 0; i < n; i++ {
				msg := <-c.Send
				if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					logger.Error("Failed to write queued message", err, map[string]interface{}{
						"user_id": c.UserID,
					})
					return
				}
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
