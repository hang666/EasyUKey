package entity

import (
	"time"

	"gorm.io/gorm"
)

// Device 设备: 从属于一个用户，是认证的具体执行者
type Device struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`                         // 内部主键，在系统中被称为 DeviceID
	UserID             *uint          `gorm:"index" json:"user_id"`                         // 外键，关联到User模型
	Name               string         `gorm:"not null" json:"name"`                         // 用户为设备设置的别名, 如 "我的主力UKey"
	SerialNumber       string         `gorm:"unique;not null" json:"serial_number"`         // 硬件序列号 (来自客户端)
	VolumeSerialNumber string         `gorm:"unique;not null" json:"volume_serial_number"`  // 卷序列号 (来自客户端)
	TOTPSecret         string         `gorm:"not null" json:"-"`                            // TOTP密钥 (加密存储，不在JSON中显示)
	OnceKey            string         `gorm:"not null" json:"-"`                            // 当前有效的一次性密钥 (加密存储，不在JSON中显示)
	LastUsedOnceKey    string         `json:"-"`                                            // 上次使用的一次性密钥
	Permissions        []string       `gorm:"type:json;serializer:json" json:"permissions"` // JSON存储权限列表，只能设置用户有的权限
	IsActive           bool           `gorm:"default:true" json:"is_active"`                // 设备是否激活（管理状态）
	IsOnline           bool           `gorm:"default:false" json:"is_online"`               // 设备是否在线（实时状态）
	LastHeartbeat      *time.Time     `gorm:"index" json:"last_heartbeat"`                  // 最后心跳时间
	LastOnlineAt       *time.Time     `json:"last_online_at"`                               // 最后上线时间
	LastOfflineAt      *time.Time     `json:"last_offline_at"`                              // 最后离线时间
	HeartbeatInterval  int            `gorm:"default:30" json:"heartbeat_interval"`         // 心跳间隔（秒）
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// 关联关系
	User         *User         `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL" json:"user,omitempty"`
	AuthSessions []AuthSession `gorm:"foreignKey:RespondingDeviceID" json:"auth_sessions,omitempty"`
}

// TableName 指定表名
func (Device) TableName() string {
	return "devices"
}
