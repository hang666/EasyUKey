package messages

import (
	"time"
)

// WSMessage WebSocket消息结构
type WSMessage struct {
	Type          string      `json:"type"`
	Data          interface{} `json:"data"`
	Timestamp     time.Time   `json:"timestamp"`
	ClientVersion string      `json:"client_version"`
}

// AuthRequestMessage 认证请求消息
type AuthRequestMessage struct {
	RequestID string `json:"request_id"`
	UserID    string `json:"user_id"`
	Challenge string `json:"challenge"`
	Action    string `json:"action,omitempty"`
	Message   string `json:"message"`
	Timeout   int    `json:"timeout"`
}

// AuthResponseMessage 认证响应消息
type AuthResponseMessage struct {
	RequestID          string `json:"request_id"`
	Success            bool   `json:"success"`
	AuthKey            string `json:"auth_key,omitempty"`
	Error              string `json:"error,omitempty"`
	UsedKey            string `json:"used_key,omitempty"`
	SerialNumber       string `json:"serial_number,omitempty"`
	VolumeSerialNumber string `json:"volume_serial_number,omitempty"`
}

// DeviceInitRequestMessage 设备初始化请求消息
type DeviceInitRequestMessage struct {
	SerialNumber       string `json:"serial_number"`
	VolumeSerialNumber string `json:"volume_serial_number"`
	DevicePath         string `json:"device_path"`
	Vendor             string `json:"vendor"`
	Model              string `json:"model"`
}

// DeviceInitResponseMessage 设备初始化响应消息
type DeviceInitResponseMessage struct {
	Success bool   `json:"success"`
	OnceKey string `json:"once_key,omitempty"`
	TOTPURI string `json:"totp_uri,omitempty"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// AuthSuccessResponseMessage 认证成功响应消息
type AuthSuccessResponseMessage struct {
	RequestID  string `json:"request_id"`
	Success    bool   `json:"success"`
	NewOnceKey string `json:"new_once_key,omitempty"`
	Error      string `json:"error,omitempty"`
}

// OnceKeyUpdateConfirmMessage 一次性密钥更新确认消息
type OnceKeyUpdateConfirmMessage struct {
	RequestID string `json:"request_id"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// DeviceRegistrationMessage 设备注册消息
type DeviceRegistrationMessage struct {
	SerialNumber       string `json:"serial_number"`
	VolumeSerialNumber string `json:"volume_serial_number"`
	DevicePath         string `json:"device_path"`
	Vendor             string `json:"vendor"`
	Model              string `json:"model"`
}

// DeviceStatusMessage 设备状态消息
type DeviceStatusMessage struct {
	Status             string `json:"status"`
	SerialNumber       string `json:"serial_number"`
	VolumeSerialNumber string `json:"volume_serial_number"`
}

// PingMessage 心跳请求消息
type PingMessage struct {
	Timestamp time.Time `json:"timestamp"`
}

// PongMessage 心跳响应消息
type PongMessage struct {
	Timestamp time.Time `json:"timestamp"`
}

// ForceLogoutMessage 强制下线消息
type ForceLogoutMessage struct {
	Message string `json:"message"`
}

// KeyExchangeRequestMessage 密钥交换请求消息
type KeyExchangeRequestMessage struct {
	PublicKey string `json:"public_key"` // Base64编码的客户端公钥
}

// KeyExchangeResponseMessage 密钥交换响应消息
type KeyExchangeResponseMessage struct {
	PublicKey string `json:"public_key"` // Base64编码的服务端公钥
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// EncryptedMessage 加密消息格式
type EncryptedMessage struct {
	Payload string `json:"payload"` // Base64编码的加密数据
	Nonce   string `json:"nonce"`   // Base64编码的nonce
}

// HandshakeStatus 握手状态
type HandshakeStatus int

const (
	// HandshakeStatusPending 等待握手
	HandshakeStatusPending HandshakeStatus = iota
	// HandshakeStatusCompleted 握手完成
	HandshakeStatusCompleted
	// HandshakeStatusFailed 握手失败
	HandshakeStatusFailed
)
