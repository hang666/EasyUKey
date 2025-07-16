package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/hang666/EasyUKey/server/internal/global"
	"github.com/hang666/EasyUKey/server/internal/model/entity"
	"github.com/hang666/EasyUKey/server/internal/model/request"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
)

// CreateAPIKey 创建API密钥
func CreateAPIKey(req *request.CreateAPIKeyRequest) (*entity.APIKey, error) {
	// 生成API密钥
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("生成API密钥失败: %w", err)
	}
	apiKey := hex.EncodeToString(keyBytes)

	// 解析过期时间
	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		parsedTime, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("无效的过期时间格式: %w", err)
		}
		expiresAt = &parsedTime
	}

	// 创建API密钥记录
	key := entity.APIKey{
		Name:        req.Name,
		Description: req.Description,
		APIKey:      apiKey,
		IsActive:    true,
		IsAdmin:     false,
		ExpiresAt:   expiresAt,
	}

	if err := global.DB.Create(&key).Error; err != nil {
		return nil, fmt.Errorf("创建API密钥失败: %w", err)
	}

	logger.Logger.Info("创建API密钥",
		"api_key_id", key.ID,
		"name", key.Name)

	return &key, nil
}

// GetAPIKeys 获取API密钥列表
func GetAPIKeys(page, pageSize int) ([]entity.APIKey, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	var keys []entity.APIKey
	var total int64

	// 获取总数
	if err := global.DB.Model(&entity.APIKey{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取API密钥总数失败: %w", err)
	}

	// 获取API密钥列表
	if err := global.DB.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&keys).Error; err != nil {
		return nil, 0, fmt.Errorf("获取API密钥列表失败: %w", err)
	}

	return keys, total, nil
}

// GetAPIKey 获取API密钥
func GetAPIKey(apiKey string) (*entity.APIKey, error) {
	var key entity.APIKey
	if err := global.DB.Where("api_key = ?", apiKey).First(&key).Error; err != nil {
		return nil, fmt.Errorf("获取API密钥失败: %w", err)
	}
	return &key, nil
}
