package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/hang666/EasyUKey/sdk/request"
	"github.com/hang666/EasyUKey/server/internal/global"
	"github.com/hang666/EasyUKey/server/internal/model/entity"
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

// DeleteAPIKey 删除API密钥
func DeleteAPIKey(apiKeyID uint) error {
	// 查找API密钥
	var key entity.APIKey
	result := global.DB.Where("id = ?", apiKeyID).First(&key)
	if result.Error != nil {
		if result.Error.Error() == "record not found" {
			return fmt.Errorf("API密钥不存在")
		}
		return fmt.Errorf("查询API密钥失败: %w", result.Error)
	}

	// 防止删除管理员密钥时留下系统无管理员的情况
	if key.IsAdmin {
		var adminKeyCount int64
		global.DB.Model(&entity.APIKey{}).Where("is_admin = ? AND is_active = ?", true, true).Count(&adminKeyCount)
		if adminKeyCount <= 1 {
			return fmt.Errorf("无法删除最后一个管理员密钥")
		}
	}

	// 使用事务确保数据一致性
	tx := global.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if tx.Error != nil {
		return fmt.Errorf("开始事务失败: %w", tx.Error)
	}

	// 删除API密钥
	if err := tx.Delete(&key).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除API密钥失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}
