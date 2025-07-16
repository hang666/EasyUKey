package request

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username    string   `json:"username" validate:"required,min=3,max=50"`
	Permissions []string `json:"permissions"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Username    string   `json:"username,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	IsActive    *bool    `json:"is_active,omitempty"`
}

// AuthRequest 认证请求
type AuthRequest struct {
	UserID      string `json:"user_id" validate:"required"`
	Challenge   string `json:"challenge" validate:"required"`
	Action      string `json:"action"`
	Message     string `json:"message"`
	Timeout     int    `json:"timeout" validate:"min=10,max=300"` // 超时时间（秒）
	CallbackURL string `json:"callback_url"`
}

// VerifyAuthRequest 验证认证结果请求
type VerifyAuthRequest struct {
	SessionID string `json:"session_id" validate:"required"`
}

// UpdateDeviceRequest 更新设备请求
type UpdateDeviceRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	IsActive    *bool    `json:"is_active"`
}

// LinkDeviceToUserRequest 绑定设备到用户请求
type LinkDeviceToUserRequest struct {
	UserID uint `json:"user_id" validate:"required"`
}

// CreateAPIKeyRequest 创建API密钥请求
type CreateAPIKeyRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=100"`
	Description string `json:"description"`
	ExpiresAt   string `json:"expires_at"` // RFC3339格式
}

// DeviceHeartbeatRequest 设备心跳请求
type DeviceHeartbeatRequest struct {
	SerialNumber       string `json:"serial_number" validate:"required"`
	VolumeSerialNumber string `json:"volume_serial_number" validate:"required"`
	Status             string `json:"status,omitempty"` // 设备状态信息
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
