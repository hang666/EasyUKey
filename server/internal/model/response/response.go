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
	Success bool   `json:"success"`
	UserID  uint   `json:"user_id,omitempty"`
	Message string `json:"message,omitempty"`
}

// DeviceStatisticsData 设备统计数据
type DeviceStatisticsData struct {
	TotalDevices   int64 `json:"total_devices"`
	OnlineDevices  int64 `json:"online_devices"`
	OfflineDevices int64 `json:"offline_devices"`
}
