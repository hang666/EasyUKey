package entity

import (
	"time"

	"gorm.io/gorm"
)

// User 用户: 系统的核心主体，可以拥有一个或多个设备
type User struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Username    string         `gorm:"unique;not null;type:varchar(255)" json:"username"` // 如 "john.doe"
	Permissions []string       `gorm:"type:json;serializer:json" json:"permissions"`      // JSON存储权限列表
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// 关联关系
	DeviceGroups []DeviceGroup `gorm:"foreignKey:UserID" json:"device_groups,omitempty"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
