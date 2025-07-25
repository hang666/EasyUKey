package response

import "time"

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
	Status   string `json:"status"`           // 详细状态：pending, processing, processing_oncekey, completed, failed, expired, rejected
	Result   string `json:"result,omitempty"` // 认证结果：success, failure (仅在completed状态时有值)
	UserID   uint   `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
	Message  string `json:"message,omitempty"`
}

// DeviceStatistics 设备统计数据
type DeviceStatistics struct {
	TotalDevices   int64 `json:"total_devices"`
	OnlineDevices  int64 `json:"online_devices"`
	OfflineDevices int64 `json:"offline_devices"`
	ActiveDevices  int64 `json:"active_devices"`
	BoundDevices   int64 `json:"bound_devices"`
}

// DeviceGroupResponse 设备组响应结构（排除敏感字段）
type DeviceGroupResponse struct {
	ID          uint             `json:"id"`
	UserID      *uint            `json:"user_id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Permissions []string         `json:"permissions"`
	IsActive    bool             `json:"is_active"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	User        *UserResponse    `json:"user,omitempty"`
	Devices     []DeviceResponse `json:"devices,omitempty"`
}

// DeviceResponse 设备响应结构（排除敏感字段）
type DeviceResponse struct {
	ID                 uint                 `json:"id"`
	DeviceGroupID      *uint                `json:"device_group_id"`
	Name               string               `json:"name"`
	SerialNumber       string               `json:"serial_number"`
	VolumeSerialNumber string               `json:"volume_serial_number"`
	Vendor             string               `json:"vendor"`
	Model              string               `json:"model"`
	Remark             string               `json:"remark"`
	IsActive           bool                 `json:"is_active"`
	IsOnline           bool                 `json:"is_online"`
	LastHeartbeat      *time.Time           `json:"last_heartbeat"`
	LastOnlineAt       *time.Time           `json:"last_online_at"`
	LastOfflineAt      *time.Time           `json:"last_offline_at"`
	HeartbeatInterval  int                  `json:"heartbeat_interval"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
	DeviceGroup        *DeviceGroupResponse `json:"device_group,omitempty"`
}

// UserResponse 用户响应结构（排除敏感字段）
type UserResponse struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
