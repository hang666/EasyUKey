package ws

import (
	"encoding/json"
	"time"

	"github.com/hang666/EasyUKey/server/internal/config"
	"github.com/hang666/EasyUKey/server/internal/service"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
	"github.com/hang666/EasyUKey/shared/pkg/messages"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

// HandleWebSocket 处理WebSocket连接
func HandleWebSocket(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		logger.Logger.Error("WebSocket升级失败", "error", err)
		return err
	}

	client := &Client{
		Conn:            conn,
		Send:            make(chan []byte, config.GlobalConfig.WebSocket.SendChannelBuffer),
		ConnectedAt:     time.Now(),
		LastPongAt:      time.Now(),
		HandshakeStatus: messages.HandshakeStatusPending,
	}

	logger.Logger.Info("新的WebSocket连接", "remote_addr", conn.RemoteAddr().String())

	// 启动客户端处理goroutines
	go client.writePump()
	go client.readPump()

	return nil
}

// writePump 处理从hub到websocket连接的消息发送
func (c *Client) writePump() {
	ticker := time.NewTicker(config.GlobalConfig.WebSocket.PingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(config.GlobalConfig.WebSocket.WriteWait))
			if !ok {
				// Hub关闭了通道
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 将队列中的其他消息一起发送
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(config.GlobalConfig.WebSocket.WriteWait))

			// 发送ping消息
			pingMsg := messages.PingMessage{
				Timestamp: time.Now(),
			}

			message, err := service.SendWSMessage("ping", pingMsg)
			if err != nil {
				logger.Logger.Error("生成ping消息失败", "error", err)
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		}
	}
}

// readPump 处理从websocket连接到hub的消息接收
func (c *Client) readPump() {
	defer func() {
		if hub := service.GetWSHub(); hub != nil {
			if h, ok := hub.(*Hub); ok {
				h.unregister <- c
			}
		}
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(config.GlobalConfig.WebSocket.MaxMessageSize)
	c.resetReadDeadline()
	c.Conn.SetPongHandler(func(string) error {
		c.updateLastPong()
		c.resetReadDeadline()
		return nil
	})

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Logger.Error("WebSocket连接异常关闭", "error", err, "device_id", c.DeviceID)
			}
			break
		}

		// 解析WebSocket消息
		var wsMsg messages.WSMessage
		if err := json.Unmarshal(messageBytes, &wsMsg); err != nil {
			logger.Logger.Error("解析WebSocket消息失败", "error", err, "device_id", c.DeviceID)
			continue
		}

		// 使用简化的消息分派机制
		if err := dispatchMessage(c, &wsMsg); err != nil {
			logger.Logger.Error("处理WebSocket消息失败", "error", err,
				"device_id", c.DeviceID,
				"message_type", wsMsg.Type)
		}
	}
}

// resetReadDeadline 重置读取超时时间
func (c *Client) resetReadDeadline() {
	c.Conn.SetReadDeadline(time.Now().Add(config.GlobalConfig.WebSocket.PongWait))
}

// updateLastPong 更新最后pong时间
func (c *Client) updateLastPong() {
	c.mu.Lock()
	c.LastPongAt = time.Now()
	c.mu.Unlock()
}
