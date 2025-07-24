package ws

import (
	"fmt"

	"github.com/hang666/EasyUKey/shared/pkg/logger"
	"github.com/hang666/EasyUKey/shared/pkg/messages"
)

// dispatchMessage 分派消息到对应的处理函数
func dispatchMessage(client *Client, wsMsg *messages.WSMessage) error {
	switch wsMsg.Type {
	case "key_exchange_request":
		return handleKeyExchangeRequest(client, wsMsg)
	case "encrypted":
		return handleEncryptedMessage(client, wsMsg)
	case "device_connection":
		return handleDeviceConnection(client, wsMsg)
	case "device_init_request":
		return handleDeviceInit(client, wsMsg)
	case "auth_response":
		return handleAuthResponse(client, wsMsg)
	case "once_key_update_confirm":
		return handleOnceKeyUpdate(client, wsMsg)
	case "device_status_response":
		return handleDeviceStatus(client, wsMsg)
	case "ping":
		return handlePing(client, wsMsg)
	case "pong":
		return handlePong(client, wsMsg)
	default:
		logger.Logger.Warn("收到未知消息类型", "type", wsMsg.Type, "device_id", client.DeviceID)
		return sendErrorToClient(client, wsMsg.Type, "unknown_message", fmt.Sprintf("未知的消息类型: %s", wsMsg.Type))
	}
}
