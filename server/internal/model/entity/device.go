package entity

import (
	"time"

	"gorm.io/gorm"
)

// Device 设备
type Device struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`                        // 内部主键，在系统中被称为 DeviceID
	DeviceGroupID      *uint          `gorm:"index" json:"device_group_id"`                // 外键，关联到DeviceGroup模型
	Name               string         `gorm:"not null" json:"name"`                        // 用户为设备设置的别名, 如 "我的主力UKey"
	SerialNumber       string         `gorm:"unique;not null" json:"serial_number"`        // 硬件序列号 (来自客户端)
	VolumeSerialNumber string         `gorm:"unique;not null" json:"volume_serial_number"` // 卷序列号 (来自客户端)
	Remark             string         `json:"remark"`                                      // 设备备注，如"跨平台自动识别"
	IsActive           bool           `gorm:"default:true" json:"is_active"`               // 设备是否激活（管理状态）
	IsOnline           bool           `gorm:"default:false" json:"is_online"`              // 设备是否在线（实时状态）
	LastHeartbeat      *time.Time     `gorm:"index" json:"last_heartbeat"`                 // 最后心跳时间
	LastOnlineAt       *time.Time     `json:"last_online_at"`                              // 最后上线时间
	LastOfflineAt      *time.Time     `json:"last_offline_at"`                             // 最后离线时间
	HeartbeatInterval  int            `gorm:"default:30" json:"heartbeat_interval"`        // 心跳间隔（秒）
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// 关联关系
	DeviceGroup  *DeviceGroup  `gorm:"foreignKey:DeviceGroupID;constraint:OnDelete:SET NULL" json:"device_group,omitempty"`
	AuthSessions []AuthSession `gorm:"foreignKey:RespondingDeviceID" json:"auth_sessions,omitempty"`
}

// TableName 指定表名
func (Device) TableName() string {
	return "devices"
}
