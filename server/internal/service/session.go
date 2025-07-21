package service

import (
	"fmt"

	"github.com/hang666/EasyUKey/server/internal/global"
	"github.com/hang666/EasyUKey/server/internal/model/entity"
)

// GetAuthSessions 获取认证会话列表
func GetAuthSessions(page, pageSize int, status, username string) ([]entity.AuthSession, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	query := global.DB.Model(&entity.AuthSession{}).Preload("User").Preload("APIKey").Preload("RespondingDevice")

	// 应用过滤条件
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if username != "" {
		query = query.Joins("LEFT JOIN users ON auth_sessions.user_id = users.id").
			Where("users.username LIKE ?", "%"+username+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取认证会话总数失败: %w", err)
	}

	var sessions []entity.AuthSession
	if err := query.Offset(offset).Limit(pageSize).
		Order("created_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, 0, fmt.Errorf("获取认证会话列表失败: %w", err)
	}

	return sessions, total, nil
}
