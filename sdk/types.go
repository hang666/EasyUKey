package sdk

import "time"

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
	DeviceGroupID      *uint     `json:"device_group_id"`
	IsOnline           bool      `json:"is_online"`
	IsActive           bool      `json:"is_active"`
	LastSeen           time.Time `json:"last_seen"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	Permissions        []string  `json:"permissions"`
	Remark             string    `json:"remark"`
}

// DeviceGroup 设备组信息
type DeviceGroup struct {
	ID          uint      `json:"id"`
	UserID      *uint     `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	User        *User     `json:"user,omitempty"`
	Devices     []Device  `json:"devices,omitempty"`
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
