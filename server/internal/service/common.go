package service

import (
	"encoding/json"
	"time"

	"github.com/hang666/EasyUKey/shared/pkg/messages"
)

var globalWSHub WSHubInterface

// WSHubInterface WebSocket Hub接口
type WSHubInterface interface {
	IsUserOnline(userID uint) bool
	IsDeviceOnline(deviceID uint) bool
	SendToUser(userID uint, data []byte) error
	OnDeviceConnect(deviceID uint) error
	OnDeviceDisconnect(deviceID uint) error
	GetOnlineDevicesCount() int
	LinkDeviceToUser(deviceID uint, userID uint) error
}

// GetWSHub 获取WebSocket Hub实例
func GetWSHub() WSHubInterface {
	return globalWSHub
}

// SetWSHub 设置WebSocket Hub实例
func SetWSHub(hub WSHubInterface) {
	globalWSHub = hub
}

// SendWSMessage 发送WebSocket消息的辅助函数
func SendWSMessage(msgType string, data interface{}) ([]byte, error) {
	msg := messages.WSMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}

	return json.Marshal(msg)
}
