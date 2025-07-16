package wsutil

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hang666/EasyUKey/shared/pkg/messages"
)

// SendMessage 发送WebSocket消息的通用函数
func SendMessage(conn interface{ WriteJSON(v interface{}) error }, msgType string, data interface{}) error {
	message := messages.WSMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}
	return conn.WriteJSON(message)
}

// SendMessageToChannel 发送消息到channel（用于server）
func SendMessageToChannel(sendChan chan<- []byte, msgType string, data interface{}) error {
	message := messages.WSMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}

	msgBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	select {
	case sendChan <- msgBytes:
		return nil
	default:
		return fmt.Errorf("发送通道已满")
	}
}

// SendErrorToChannel 发送错误消息到channel
func SendErrorToChannel(sendChan chan<- []byte, msgType string, errorCode string, errorMessage string) error {
	errorResp := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    errorCode,
			"message": errorMessage,
		},
	}
	return SendMessageToChannel(sendChan, msgType+"_error", errorResp)
}

// ParseMessage 解析消息数据
func ParseMessage[T any](wsMsg *messages.WSMessage) (T, error) {
	var result T
	msgBytes, err := json.Marshal(wsMsg.Data)
	if err != nil {
		return result, fmt.Errorf("序列化消息数据失败: %w", err)
	}

	if err := json.Unmarshal(msgBytes, &result); err != nil {
		return result, fmt.Errorf("解析消息数据失败: %w", err)
	}

	return result, nil
}

// ValidateMessage 验证消息基本格式
func ValidateMessage(wsMsg *messages.WSMessage) error {
	if wsMsg == nil {
		return fmt.Errorf("消息不能为空")
	}
	if wsMsg.Type == "" {
		return fmt.Errorf("消息类型不能为空")
	}
	if wsMsg.Data == nil {
		return fmt.Errorf("消息数据不能为空")
	}
	return nil
}
