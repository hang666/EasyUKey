package request

// AuthRequest 认证请求
type AuthRequest struct {
	Username    string `json:"username"`
	Challenge   string `json:"challenge"`
	Action      string `json:"action,omitempty"`
	Message     string `json:"message,omitempty"`
	Timeout     int    `json:"timeout,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"`
}

// CallbackRequest 回调请求数据结构
type CallbackRequest struct {
	SessionID string `json:"session_id"`
	Username  string `json:"username"`
	Status    string `json:"status"`
	Challenge string `json:"challenge"`
	Action    string `json:"action"`
	DeviceID  uint   `json:"device_id"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
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
	Name     string `json:"name,omitempty"`
	Remark   string `json:"remark,omitempty"`
	IsActive *bool  `json:"is_active,omitempty"`
}

// CreateAPIKeyRequest 创建API密钥请求
type CreateAPIKeyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

// DeviceFilter 设备过滤条件
type DeviceFilter struct {
	IsOnline      *bool  `json:"is_online,omitempty"`
	IsActive      *bool  `json:"is_active,omitempty"`
	Name          string `json:"name,omitempty"`
	DeviceGroupID *uint  `json:"device_group_id,omitempty"`
	OnlineOnly    bool   `json:"online_only,omitempty"`
	OfflineOnly   bool   `json:"offline_only,omitempty"`
}

// UpdateDeviceGroupRequest 更新设备组请求
type UpdateDeviceGroupRequest struct {
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	IsActive    *bool    `json:"is_active,omitempty"`
}

// LinkDeviceGroupUserRequest 关联设备组用户请求
type LinkDeviceGroupUserRequest struct {
	UserID *uint `json:"user_id"` // null表示取消关联
}
