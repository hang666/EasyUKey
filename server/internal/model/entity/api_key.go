package entity

import (
	"time"

	"gorm.io/gorm"
)

// APIKey API密钥: 用于第三方应用访问EasyUKey服务的凭证
type APIKey struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"not null" json:"name"`           // API密钥名称，便于管理
	APIKey      string         `gorm:"unique;not null" json:"api_key"` // API密钥值
	Description string         `json:"description"`                    // 描述信息
	IsActive    bool           `gorm:"default:true" json:"is_active"`  // 是否激活
	IsAdmin     bool           `gorm:"default:false" json:"is_admin"`  // 是否为管理员密钥
	ExpiresAt   *time.Time     `json:"expires_at"`                     // 过期时间，nil表示不过期
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (APIKey) TableName() string {
	return "api_keys"
}
