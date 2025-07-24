package entity

import (
	"time"

	"gorm.io/gorm"
)

// DeviceGroup 设备组: 管理跨平台设备的统一认证密钥
type DeviceGroup struct {
	ID          uint     `gorm:"primaryKey" json:"id"`
	UserID      *uint    `gorm:"index" json:"user_id"`                         // 外键，关联到User模型
	Name        string   `gorm:"not null" json:"name"`                         // 设备组名称
	Description string   `gorm:"type:text" json:"description"`                 // 设备组描述
	Permissions []string `gorm:"type:json;serializer:json" json:"permissions"` // JSON存储权限列表

	// 认证密钥统一管理
	TOTPSecret      string `gorm:"not null;index" json:"-"` // TOTP密钥
	OnceKey         string `gorm:"not null;index" json:"-"` // 当前有效的一次性密钥
	LastUsedOnceKey string `gorm:"index" json:"-"`          // 上次使用的一次性密钥

	IsActive  bool           `gorm:"default:true;index" json:"is_active"` // 设备组是否激活
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// 关联关系
	User    *User    `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL" json:"user,omitempty"`
	Devices []Device `gorm:"foreignKey:DeviceGroupID" json:"devices,omitempty"`
}

// TableName 指定表名
func (DeviceGroup) TableName() string {
	return "device_groups"
}
