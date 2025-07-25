package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/hang666/EasyUKey/sdk/response"
	"github.com/hang666/EasyUKey/server/internal/global"
	"github.com/hang666/EasyUKey/server/internal/model/entity"
	"github.com/hang666/EasyUKey/shared/pkg/errs"
	"github.com/hang666/EasyUKey/shared/pkg/identity"
)

// ConvertToDeviceGroupResponse 将设备组实体转换为安全的响应结构
func ConvertToDeviceGroupResponse(group *entity.DeviceGroup) *response.DeviceGroupResponse {
	if group == nil {
		return nil
	}

	resp := &response.DeviceGroupResponse{
		ID:          group.ID,
		UserID:      group.UserID,
		Name:        group.Name,
		Description: group.Description,
		Permissions: group.Permissions,
		IsActive:    group.IsActive,
		CreatedAt:   group.CreatedAt,
		UpdatedAt:   group.UpdatedAt,
	}

	// 转换关联的用户信息
	if group.User != nil {
		resp.User = ConvertToUserResponse(group.User)
	}

	// 转换关联的设备信息
	if group.Devices != nil {
		resp.Devices = make([]response.DeviceResponse, 0, len(group.Devices))
		for _, device := range group.Devices {
			deviceResp := ConvertToDeviceResponse(&device)
			if deviceResp != nil {
				resp.Devices = append(resp.Devices, *deviceResp)
			}
		}
	}

	return resp
}

// ConvertToDeviceResponse 将设备实体转换为安全的响应结构
func ConvertToDeviceResponse(device *entity.Device) *response.DeviceResponse {
	if device == nil {
		return nil
	}

	resp := &response.DeviceResponse{
		ID:                 device.ID,
		DeviceGroupID:      device.DeviceGroupID,
		Name:               device.Name,
		SerialNumber:       device.SerialNumber,
		VolumeSerialNumber: device.VolumeSerialNumber,
		Vendor:             device.Vendor,
		Model:              device.Model,
		Remark:             device.Remark,
		IsActive:           device.IsActive,
		IsOnline:           device.IsOnline,
		LastHeartbeat:      device.LastHeartbeat,
		LastOnlineAt:       device.LastOnlineAt,
		LastOfflineAt:      device.LastOfflineAt,
		HeartbeatInterval:  device.HeartbeatInterval,
		CreatedAt:          device.CreatedAt,
		UpdatedAt:          device.UpdatedAt,
	}

	// 转换关联的设备组信息
	if device.DeviceGroup != nil {
		resp.DeviceGroup = &response.DeviceGroupResponse{
			ID:          device.DeviceGroup.ID,
			UserID:      device.DeviceGroup.UserID,
			Name:        device.DeviceGroup.Name,
			Description: device.DeviceGroup.Description,
			Permissions: device.DeviceGroup.Permissions,
			IsActive:    device.DeviceGroup.IsActive,
			CreatedAt:   device.DeviceGroup.CreatedAt,
			UpdatedAt:   device.DeviceGroup.UpdatedAt,
		}
	}

	return resp
}

// ConvertToUserResponse 将用户实体转换为安全的响应结构
func ConvertToUserResponse(user *entity.User) *response.UserResponse {
	if user == nil {
		return nil
	}

	return &response.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

// GetDeviceGroup 获取设备组详情
func GetDeviceGroup(groupID uint) (*entity.DeviceGroup, error) {
	var group entity.DeviceGroup
	result := global.DB.Preload("User").Preload("Devices").Where("id = ?", groupID).First(&group)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errs.ErrDeviceGroupNotFound
		}
		return nil, fmt.Errorf("查询设备组失败: %w", result.Error)
	}

	return &group, nil
}

// GetDeviceGroups 获取设备组列表
func GetDeviceGroups() ([]entity.DeviceGroup, error) {
	var groups []entity.DeviceGroup
	result := global.DB.Preload("User").Preload("Devices").Find(&groups)
	if result.Error != nil {
		return nil, fmt.Errorf("查询设备组失败: %w", result.Error)
	}

	return groups, nil
}

// UpdateDeviceGroup 更新设备组
func UpdateDeviceGroup(groupID uint, name, description string, permissions []string, isActive *bool) (*entity.DeviceGroup, error) {
	// 输入验证
	if groupID == 0 {
		return nil, errs.ErrInvalidDeviceID
	}
	if name == "" && description == "" && permissions == nil && isActive == nil {
		return nil, errs.ErrInvalidRequest
	}

	// 先获取更新前的设备组信息
	oldGroup, err := GetDeviceGroup(groupID)
	if err != nil {
		return nil, err
	}

	// 构建更新字段
	updates := make(map[string]interface{})

	if name != "" {
		updates["name"] = name
	}
	if description != "" {
		updates["description"] = description
	}
	if permissions != nil {
		permissionsJson, err := json.Marshal(permissions)
		if err != nil {
			return nil, errs.ErrDeviceGroupPermissions
		}
		updates["permissions"] = string(permissionsJson)
	}
	if isActive != nil {
		updates["is_active"] = *isActive
	}

	// 事务更新
	err = global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.DeviceGroup{}).Where("id = ?", groupID).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新设备组失败: %w", err)
		}
		if isActive != nil && !*isActive && oldGroup.IsActive {
			// 设备组从激活变为未激活，同时停用其下所有设备
			if err := tx.Model(&entity.Device{}).Where("device_group_id = ?", groupID).Update("is_active", false).Error; err != nil {
				return fmt.Errorf("停用设备组关联设备失败: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("更新设备组失败: %w", err)
	}

	// 返回更新后的设备组
	return GetDeviceGroup(groupID)
}

// FindDeviceGroupByAuth 通过认证密钥查找设备组
func FindDeviceGroupByAuth(totpCode, onceKey string) (*entity.DeviceGroup, error) {
	// 直接通过onceKey查找激活的设备组
	var group entity.DeviceGroup
	result := global.DB.Where("is_active = ? AND once_key = ?", true, onceKey).First(&group)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // 未找到匹配的设备组
		}
		return nil, fmt.Errorf("查询设备组失败: %w", result.Error)
	}

	// 验证TOTP - 先解析TOTP URI获取密钥
	totpConfig, err := identity.ParseTOTPURI(group.TOTPSecret)
	if err != nil {
		return nil, fmt.Errorf("解析TOTP密钥失败: %w", err)
	}

	// 验证TOTP码
	valid, err := identity.VerifyTOTPCode(totpConfig, totpCode, time.Now())
	if err != nil {
		return nil, fmt.Errorf("验证TOTP码失败: %w", err)
	}

	if !valid {
		return nil, nil // TOTP验证失败
	}

	return &group, nil
}

// GetPendingActivationDevices 获取待激活设备列表
func GetPendingActivationDevices() ([]entity.Device, error) {
	var devices []entity.Device
	if err := global.DB.Preload("DeviceGroup").Where("is_active = ? AND device_group_id IS NOT NULL", false).
		Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("查询待激活设备失败: %w", err)
	}

	return devices, nil
}

// LinkDeviceGroupUser 关联或取消关联设备组用户
func LinkDeviceGroupUser(groupID uint, userID *uint) error {
	// 先获取更新前的设备组信息，以便判断用户是否有变化
	oldGroup, err := GetDeviceGroup(groupID)
	if err != nil {
		return fmt.Errorf("获取设备组信息失败: %w", err)
	}

	// 更新设备组的用户关联
	if err := global.DB.Model(&entity.DeviceGroup{}).Where("id = ?", groupID).Update("user_id", userID).Error; err != nil {
		return fmt.Errorf("关联设备组用户失败: %w", err)
	}

	// 检查用户是否确实发生了变化
	oldUserID := uint(0)
	if oldGroup.UserID != nil {
		oldUserID = *oldGroup.UserID
	}

	newUserID := uint(0)
	if userID != nil {
		newUserID = *userID
	}

	if newUserID != oldUserID {
		// 获取该设备组下的所有在线设备，更新其用户归属
		var devices []entity.Device
		if err := global.DB.Where("device_group_id = ?", groupID).Find(&devices).Error; err != nil {
			return fmt.Errorf("查询设备组关联设备失败: %w", err)
		}

		hub := GetWSHub()
		if hub != nil {
			for _, device := range devices {
				if hub.IsDeviceOnline(device.ID) {
					if newUserID > 0 {
						// 绑定新用户
						hub.LinkDeviceToUser(device.ID, newUserID)
					} else {
						// 取消绑定，通过Hub强制断开WebSocket连接
						hub.OnDeviceDisconnect(device.ID)
					}
				}
			}
		}
	}

	return nil
}

// UpdateDeviceGroupOnceKey 更新设备组的OnceKey
func UpdateDeviceGroupOnceKey(groupID uint, oldOnceKey string) (string, error) {
	if groupID == 0 {
		return "", errs.ErrInvalidDeviceID
	}
	if oldOnceKey == "" {
		return "", errs.ErrInvalidKey
	}

	// 查找设备组
	var group entity.DeviceGroup
	result := global.DB.Where("id = ?", groupID).First(&group)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return "", errs.ErrDeviceGroupNotFound
		}
		return "", fmt.Errorf("查询设备组失败: %w", result.Error)
	}

	// 验证旧的OnceKey
	if group.OnceKey != oldOnceKey {
		return "", errs.ErrInvalidKey
	}

	// 生成新的OnceKey
	newOnceKey, err := GenerateOnceKey()
	if err != nil {
		return "", fmt.Errorf("生成新的一次性密钥失败: %w", err)
	}

	// 更新数据库
	updates := map[string]interface{}{
		"once_key":           newOnceKey,
		"last_used_once_key": oldOnceKey,
	}

	if err := global.DB.Model(&group).Updates(updates).Error; err != nil {
		return "", fmt.Errorf("更新设备组OnceKey失败: %w", err)
	}

	return newOnceKey, nil
}
