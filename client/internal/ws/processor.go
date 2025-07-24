package ws

import (
	"github.com/gorilla/websocket"

	"github.com/hang666/EasyUKey/shared/pkg/logger"
	"github.com/hang666/EasyUKey/shared/pkg/messages"
)

// processMessages is the main loop for reading and dispatching incoming WebSocket messages.
func processMessages() {
	defer func() {
		if r := recover(); r != nil {
			logger.Logger.Error("WebSocket消息处理异常", "error", r)
		}
		// When the loop exits, we are disconnected.
		setConnected(false)
	}()

	for {
		// Make a local copy of the connection for the read operation
		mu.Lock()
		localConn := conn
		mu.Unlock()

		if localConn == nil {
			// Connection is closed, exit the loop
			break
		}

		var message messages.WSMessage
		err := localConn.ReadJSON(&message)
		if err != nil {
			// Check if this is a clean close signal
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				logger.Logger.Info("WebSocket连接正常关闭")
			} else if IsConnected() {
				// Only log errors if we were expecting the connection to be alive
				logger.Logger.Error("读取WebSocket消息失败", "error", err)
			}
			// Any error from ReadJSON should break the loop.
			break
		}

		// Process the message
		dispatchMessage(message)
	}
}

// dispatchMessage routes a message to its corresponding handler.
func dispatchMessage(message messages.WSMessage) {
	switch message.Type {
	case "key_exchange_response":
		handleKeyExchangeResponse(message)
	case "encrypted":
		handleEncryptedMessage(message)
	case "auth_request":
		go handleAuthRequest(message) // Run in a goroutine to not block the read loop
	case "device_init_response":
		handleDeviceInitResponse(message)
	case "device_connection_response":
		handleDeviceConnectionResponse(message)
	case "auth_success_response":
		handleAuthSuccessResponse(message)
	case "ping":
		handlePing()
	case "device_status_check":
		handleDeviceStatusCheck()
	case "force_logout":
		handleForceLogout(message)
	default:
		logger.Logger.Warn("收到未知消息类型", "type", message.Type)
	}
}
