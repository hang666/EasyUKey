package sdk

import "time"

// AuthRequest 认证请求
type AuthRequest struct {
	UserID      string `json:"user_id"`
	Challenge   string `json:"challenge"`
	Action      string `json:"action,omitempty"`
	Message     string `json:"message,omitempty"`
	Timeout     int    `json:"timeout,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"`
}

// VerifyAuthRequest 验证认证请求
type VerifyAuthRequest struct {
	SessionID string `json:"session_id"`
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username    string   `json:"username"`
	Permissions []string `json:"permissions,omitempty"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Username    string   `json:"username,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	IsActive    *bool    `json:"is_active,omitempty"`
}

// UpdateDeviceRequest 更新设备请求
type UpdateDeviceRequest struct {
	Name        string   `json:"name,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	IsActive    *bool    `json:"is_active,omitempty"`
}

// LinkDeviceToUserRequest 绑定设备请求
type LinkDeviceToUserRequest struct {
	UserID uint `json:"user_id"`
}

// CreateAPIKeyRequest 创建API密钥请求
type CreateAPIKeyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

// DeviceFilter 设备过滤条件
type DeviceFilter struct {
	IsOnline    *bool  `json:"is_online,omitempty"`
	IsActive    *bool  `json:"is_active,omitempty"`
	UserID      *uint  `json:"user_id,omitempty"`
	Username    string `json:"username,omitempty"`
	Name        string `json:"name,omitempty"`
	OnlineOnly  bool   `json:"online_only,omitempty"`
	OfflineOnly bool   `json:"offline_only,omitempty"`
}

// Response 统一响应结构
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Total   *int64      `json:"total,omitempty"`
}

// AuthData 认证数据
type AuthData struct {
	SessionID string    `json:"session_id"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
}

// VerifyAuthData 验证认证数据
type VerifyAuthData struct {
	Success bool   `json:"success"`
	UserID  uint   `json:"user_id,omitempty"`
	Message string `json:"message,omitempty"`
}

// DeviceStatistics 设备统计数据
type DeviceStatistics struct {
	TotalDevices   int64 `json:"total_devices"`
	OnlineDevices  int64 `json:"online_devices"`
	OfflineDevices int64 `json:"offline_devices"`
	ActiveDevices  int64 `json:"active_devices"`
	BoundDevices   int64 `json:"bound_devices"`
}

// User 用户信息
type User struct {
	ID          uint      `json:"id"`
	Username    string    `json:"username"`
	Permissions []string  `json:"permissions"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Device 设备信息
type Device struct {
	ID                 uint      `json:"id"`
	Name               string    `json:"name"`
	SerialNumber       string    `json:"serial_number"`
	VolumeSerialNumber string    `json:"volume_serial_number"`
	UserID             *uint     `json:"user_id"`
	Username           string    `json:"username,omitempty"`
	IsOnline           bool      `json:"is_online"`
	IsActive           bool      `json:"is_active"`
	LastSeen           time.Time `json:"last_seen"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	Permissions        []string  `json:"permissions"`
}

// APIKey API密钥信息
type APIKey struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Key         string    `json:"key"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
