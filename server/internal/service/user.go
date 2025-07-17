package service

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/hang666/EasyUKey/sdk/request"
	"github.com/hang666/EasyUKey/server/internal/global"
	"github.com/hang666/EasyUKey/server/internal/model/entity"
	"github.com/hang666/EasyUKey/shared/pkg/errors"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
)

// CreateUser 创建用户
func CreateUser(req *request.CreateUserRequest) (*entity.User, error) {
	// 检查用户名是否已存在
	var existingUser entity.User
	result := global.DB.Where("username = ?", req.Username).First(&existingUser)
	if result.Error == nil {
		return nil, errors.ErrUserAlreadyExists
	}

	// 创建用户
	user := entity.User{
		Username:    req.Username,
		Permissions: req.Permissions,
		IsActive:    true,
	}

	if err := global.DB.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	logger.Logger.Info("创建用户",
		"user_id", user.ID,
		"username", user.Username)

	return &user, nil
}

// GetUser 获取单个用户
func GetUser(userID uint) (*entity.User, error) {
	var user entity.User
	result := global.DB.Where("id = ?", userID).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, fmt.Errorf("查询用户失败: %w", result.Error)
	}
	return &user, nil
}

// GetUsers 获取用户列表
func GetUsers(page, pageSize int) ([]entity.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	var users []entity.User
	var total int64

	// 获取总数
	if err := global.DB.Model(&entity.User{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取用户总数失败: %w", err)
	}

	// 获取用户列表
	if err := global.DB.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("获取用户列表失败: %w", err)
	}

	return users, total, nil
}

// UpdateUser 更新用户
func UpdateUser(userID uint, req *request.UpdateUserRequest) (*entity.User, error) {
	// 查找用户
	var user entity.User
	result := global.DB.Where("id = ?", userID).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, fmt.Errorf("查询用户失败: %w", result.Error)
	}

	// 检查用户名是否已被其他用户使用
	if req.Username != "" && req.Username != user.Username {
		var existingUser entity.User
		result := global.DB.Where("username = ? AND id != ?", req.Username, userID).First(&existingUser)
		if result.Error == nil {
			return nil, errors.ErrUserAlreadyExists
		}
	}

	// 更新字段
	updates := make(map[string]interface{})

	if req.Username != "" {
		updates["username"] = req.Username
	}

	if req.Permissions != nil {
		jsonData, err := json.Marshal(req.Permissions)
		if err != nil {
			return nil, fmt.Errorf("序列化权限失败: %w", err)
		}
		updates["permissions"] = string(jsonData)
	}

	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
		// 如果用户被停用，强制断开所有设备的WebSocket连接
		if !*req.IsActive {
			if hub := GetWSHub(); hub != nil {
				// 获取用户的所有设备并断开连接
				var devices []entity.Device
				global.DB.Where("user_id = ?", userID).Find(&devices)
				for _, device := range devices {
					hub.OnDeviceDisconnect(device.ID)
				}
			}
		}
	}

	// 执行更新
	if len(updates) > 0 {
		if err := global.DB.Model(&user).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("更新用户失败: %w", err)
		}

		logger.Logger.Info("更新用户信息",
			"user_id", user.ID,
			"username", user.Username,
			"updates", updates)
	}

	// 重新查询更新后的用户信息
	global.DB.Where("id = ?", userID).First(&user)

	return &user, nil
}

// DeleteUser 删除用户
func DeleteUser(userID uint) error {
	// 查找用户
	var user entity.User
	result := global.DB.Where("id = ?", userID).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return errors.ErrUserNotFound
		}
		return fmt.Errorf("查询用户失败: %w", result.Error)
	}

	// 检查用户是否有绑定的设备
	var deviceCount int64
	global.DB.Model(&entity.Device{}).Where("user_id = ?", userID).Count(&deviceCount)
	if deviceCount > 0 {
		return fmt.Errorf("用户还有 %d 个绑定的设备，请先解绑设备", deviceCount)
	}

	// 删除用户
	if err := global.DB.Delete(&user).Error; err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}

	logger.Logger.Info("删除用户",
		"user_id", user.ID,
		"username", user.Username)

	return nil
}

// GetUserDevices 获取用户的设备列表
func GetUserDevices(username string) ([]entity.Device, error) {
	// 查找用户
	var user entity.User
	result := global.DB.Where("username = ?", username).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, fmt.Errorf("查询用户失败: %w", result.Error)
	}

	// 获取用户设备列表
	var devices []entity.Device
	if err := global.DB.Where("user_id = ?", user.ID).Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("获取用户设备列表失败: %w", err)
	}

	// 更新实时在线状态
	if hub := GetWSHub(); hub != nil {
		for i := range devices {
			devices[i].IsOnline = hub.IsDeviceOnline(devices[i].ID)
		}
	}

	return devices, nil
}
