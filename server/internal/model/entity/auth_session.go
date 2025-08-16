package entity

import (
	"time"

	"gorm.io/gorm"
)

// AuthSession 认证会话: 记录一次认证流程，由用户发起，由特定设备响应
type AuthSession struct {
	ID                 string         `gorm:"primaryKey;type:varchar(255)" json:"id"`      // UUID
	UserID             uint           `gorm:"not null" json:"user_id"`                     // 发起认证的用户ID
	APIKeyID           uint           `gorm:"not null" json:"api_key_id"`                  // 调用认证的API密钥ID
	RespondingDeviceID *uint          `json:"responding_device_id"`                        // 最终响应本次认证的设备主键 (Device.ID)
	Challenge          string         `gorm:"not null;type:varchar(255)" json:"challenge"` // 挑战码
	Action             string         `gorm:"type:varchar(255)" json:"action"`             // 本次认证请求的操作/权限
	Status             string         `gorm:"not null;type:varchar(50)" json:"status"`     // 认证状态：pending, processing, processing_oncekey, completed, failed, expired, rejected
	Result             string         `gorm:"type:varchar(50)" json:"result"`              // 认证结果：success, failure
	CallbackURL        string         `gorm:"type:text" json:"callback_url"`               // 回调URL
	ClientIP           string         `gorm:"type:varchar(45)" json:"client_ip"`           // 客户端IP地址
	CreatedAt          time.Time      `json:"created_at"`
	ExpiresAt          time.Time      `json:"expires_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// 关联关系
	User             *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	APIKey           *APIKey `gorm:"foreignKey:APIKeyID" json:"api_key,omitempty"`
	RespondingDevice *Device `gorm:"foreignKey:RespondingDeviceID" json:"responding_device,omitempty"`
}

// TableName 指定表名
func (AuthSession) TableName() string {
	return "auth_sessions"
}
