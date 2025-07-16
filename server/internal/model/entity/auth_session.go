package entity

import (
	"time"

	"gorm.io/gorm"
)

// AuthSession 认证会话: 记录一次认证流程，由用户发起，由特定设备响应
type AuthSession struct {
	ID                 string         `gorm:"primaryKey" json:"id"`      // UUID
	UserID             uint           `gorm:"not null" json:"user_id"`   // 发起认证的用户ID
	RespondingDeviceID *uint          `json:"responding_device_id"`      // 最终响应本次认证的设备主键 (Device.ID)
	Challenge          string         `gorm:"not null" json:"challenge"` // 挑战码
	Action             string         `json:"action"`                    // 本次认证请求的操作/权限
	Status             string         `gorm:"not null" json:"status"`    // "pending", "completed", "failed", "expired", "rejected"
	Result             string         `json:"result"`                    // "success", "failure"
	CreatedAt          time.Time      `json:"created_at"`
	ExpiresAt          time.Time      `json:"expires_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// 关联关系
	User             *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	RespondingDevice *Device `gorm:"foreignKey:RespondingDeviceID" json:"responding_device,omitempty"`
}

// TableName 指定表名
func (AuthSession) TableName() string {
	return "auth_sessions"
}

// 认证状态常量
const (
	AuthStatusPending   = "pending"
	AuthStatusCompleted = "completed"
	AuthStatusFailed    = "failed"
	AuthStatusExpired   = "expired"
	AuthStatusRejected  = "rejected"
)

// 认证结果常量
const (
	AuthResultSuccess = "success"
	AuthResultFailure = "failure"
)
