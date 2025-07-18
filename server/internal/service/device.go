package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hang666/EasyUKey/sdk/request"
	"github.com/hang666/EasyUKey/server/internal/global"
	"github.com/hang666/EasyUKey/server/internal/model/entity"
	"github.com/hang666/EasyUKey/shared/pkg/errs"
	"github.com/hang666/EasyUKey/shared/pkg/identity"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
	"github.com/hang666/EasyUKey/shared/pkg/messages"
)

// InitDevice 初始化设备
func InitDevice(initReq *messages.DeviceInitRequestMessage) (string, string, error) {
	// 检查设备是否已存在
	var existingDevice entity.Device
	result := global.DB.Where("serial_number = ? AND volume_serial_number = ?",
		initReq.SerialNumber, initReq.VolumeSerialNumber).First(&existingDevice)

	if result.Error == nil {
		return "", "", errs.ErrDeviceAlreadyExists
	}

	if result.Error != gorm.ErrRecordNotFound {
		return "", "", fmt.Errorf("查询设备失败: %w", result.Error)
	}

	// 生成TOTP密钥和初始OnceKey
	totpAccount := fmt.Sprintf("%s_%s", initReq.SerialNumber, uuid.New().String()[:6])
	totpSecret, err := identity.GenerateTOTPSecretURI("EasyUKey", totpAccount)
	if err != nil {
		return "", "", fmt.Errorf("生成TOTP密钥失败: %w", err)
	}
	onceKey, err := GenerateOnceKey()
	if err != nil {
		return "", "", fmt.Errorf("生成OnceKey失败: %w", err)
	}

	// 创建设备记录
	device := entity.Device{
		Name:               fmt.Sprintf("设备_%s", initReq.SerialNumber[len(initReq.SerialNumber)-6:]),
		SerialNumber:       initReq.SerialNumber,
		VolumeSerialNumber: initReq.VolumeSerialNumber,
		TOTPSecret:         totpSecret,
		OnceKey:            onceKey,
		Permissions:        []string{},
		IsActive:           false,
		IsOnline:           false,
		HeartbeatInterval:  30,
	}

	if err := global.DB.Create(&device).Error; err != nil {
		return "", "", fmt.Errorf("创建设备记录失败: %w", err)
	}

	logger.Logger.Info("设备初始化成功",
		"device_id", device.ID,
		"serial_number", initReq.SerialNumber,
		"volume_serial_number", initReq.VolumeSerialNumber)

	return onceKey, totpSecret, nil
}

// UpdateDevice 更新设备信息
func UpdateDevice(deviceID uint, req *request.UpdateDeviceRequest) (*entity.Device, error) {
	// 查找设备
	var device entity.Device
	result := global.DB.Where("id = ?", deviceID).First(&device)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errs.ErrDeviceNotFound
		}
		return nil, fmt.Errorf("查询设备失败: %w", result.Error)
	}

	// 更新字段
	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["name"] = req.Name
	}

	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
		// 如果设备被停用，强制断开WebSocket连接
		if !*req.IsActive {
			if hub := GetWSHub(); hub != nil {
				hub.OnDeviceDisconnect(deviceID)
			}
		}
	}

	if req.Permissions != nil && device.UserID != nil {
		var user entity.User
		if err := global.DB.Where("id = ?", *device.UserID).First(&user).Error; err == nil {
			userPerms := user.Permissions
			for _, devicePerm := range req.Permissions {
				found := false
				for _, userPerm := range userPerms {
					if devicePerm == userPerm {
						found = true
						break
					}
				}
				if !found {
					return nil, fmt.Errorf("权限 '%s' 不在用户权限范围内", devicePerm)
				}
			}
		}

		jsonData, err := json.Marshal(req.Permissions)
		if err != nil {
			return nil, fmt.Errorf("序列化权限失败: %w", err)
		}
		updates["permissions"] = string(jsonData)
	}

	// 执行更新
	if len(updates) > 0 {
		if err := global.DB.Model(&device).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("更新设备失败: %w", err)
		}

		logger.Logger.Info("更新设备信息",
			"device_id", device.ID,
			"serial_number", device.SerialNumber,
			"updates", updates)
	}

	// 重新查询更新后的设备信息
	global.DB.Where("id = ?", deviceID).First(&device)

	// 从Hub获取实时在线状态
	if hub := GetWSHub(); hub != nil {
		device.IsOnline = hub.IsDeviceOnline(deviceID)
	}

	return &device, nil
}

// LinkDeviceToUser 绑定设备到用户
func LinkDeviceToUser(deviceID uint, userID uint) (*entity.Device, error) {
	// 检查设备是否存在
	var device entity.Device
	result := global.DB.Where("id = ?", deviceID).First(&device)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errs.ErrDeviceNotFound
		}
		return nil, fmt.Errorf("查询设备失败: %w", result.Error)
	}

	// 检查设备是否已绑定用户
	if device.UserID != nil && *device.UserID != 0 {
		return nil, errs.ErrDeviceAlreadyBound
	}

	// 检查用户是否存在
	var user entity.User
	result = global.DB.Where("id = ? AND is_active = ?", userID, true).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errs.ErrUserNotFound
		}
		return nil, fmt.Errorf("查询用户失败: %w", result.Error)
	}

	// 绑定设备到用户并激活
	updates := map[string]interface{}{
		"user_id":   userID,
		"is_active": true,
	}

	if err := global.DB.Model(&device).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("绑定设备失败: %w", err)
	}

	logger.Logger.Info("绑定设备到用户",
		"device_id", device.ID,
		"serial_number", device.SerialNumber,
		"user_id", userID,
		"username", user.Username)

	// 更新Hub中设备的用户归属
	if hub := GetWSHub(); hub != nil {
		if err := hub.LinkDeviceToUser(deviceID, userID); err != nil {
			logger.Logger.Warn("更新Hub中设备的用户状态失败",
				"error", err, "device_id", deviceID, "user_id", userID)
		}
	}

	// 重新查询更新后的设备信息
	global.DB.Preload("User").Where("id = ?", deviceID).First(&device)

	// 从Hub获取实时在线状态
	if hub := GetWSHub(); hub != nil {
		device.IsOnline = hub.IsDeviceOnline(deviceID)
	}

	return &device, nil
}

// UnlinkDeviceFromUser 解绑设备与用户
func UnlinkDeviceFromUser(deviceID uint) (*entity.Device, error) {
	// 检查设备是否存在
	var device entity.Device
	result := global.DB.Preload("User").Where("id = ?", deviceID).First(&device)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errs.ErrDeviceNotFound
		}
		return nil, fmt.Errorf("查询设备失败: %w", result.Error)
	}

	// 检查设备是否已绑定用户
	if device.UserID == nil || *device.UserID == 0 {
		return nil, fmt.Errorf("设备未绑定用户")
	}

	// 记录解绑前的用户信息
	var username string
	if device.User != nil {
		username = device.User.Username
	}

	// 强制断开设备的WebSocket连接
	if hub := GetWSHub(); hub != nil {
		hub.OnDeviceDisconnect(deviceID)
	}

	// 解绑设备并停用
	updates := map[string]interface{}{
		"user_id":   nil,
		"is_active": false,
	}

	if err := global.DB.Model(&device).Select("user_id", "is_active").Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("解绑设备失败: %w", err)
	}

	logger.Logger.Info("解绑设备与用户",
		"device_id", device.ID,
		"serial_number", device.SerialNumber,
		"user_id", device.UserID,
		"username", username)

	// 重新查询更新后的设备信息
	global.DB.Preload("User").Where("id = ?", deviceID).First(&device)

	// 从Hub获取实时在线状态
	if hub := GetWSHub(); hub != nil {
		device.IsOnline = hub.IsDeviceOnline(deviceID)
	}

	return &device, nil
}

// OfflineDevice 设备下线
func OfflineDevice(deviceID uint) (*entity.Device, error) {
	// 查找设备
	var device entity.Device
	result := global.DB.Where("id = ?", deviceID).First(&device)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errs.ErrDeviceNotFound
		}
		return nil, fmt.Errorf("查询设备失败: %w", result.Error)
	}

	// 通过Hub强制断开WebSocket连接
	if hub := GetWSHub(); hub != nil {
		hub.OnDeviceDisconnect(deviceID)
	}

	logger.Logger.Info("管理员手动设置设备下线",
		"device_id", device.ID,
		"serial_number", device.SerialNumber)

	// 重新查询更新后的设备信息
	global.DB.Preload("User").Where("id = ?", deviceID).First(&device)

	// 从Hub获取实时在线状态
	if hub := GetWSHub(); hub != nil {
		device.IsOnline = hub.IsDeviceOnline(deviceID)
	}

	return &device, nil
}

// GetDevices 获取设备列表
func GetDevices(page, pageSize int, filter *request.DeviceFilter) ([]entity.Device, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	query := global.DB.Model(&entity.Device{}).Preload("User")

	// 应用过滤条件
	if filter != nil {
		if filter.IsActive != nil {
			query = query.Where("is_active = ?", *filter.IsActive)
		}
		if filter.UserID != nil {
			query = query.Where("user_id = ?", *filter.UserID)
		}
		if filter.Username != "" {
			query = query.Joins("LEFT JOIN users ON devices.user_id = users.id").
				Where("users.username LIKE ?", "%"+filter.Username+"%")
		}
		if filter.Name != "" {
			query = query.Where("name LIKE ?", "%"+filter.Name+"%")
		}
	}

	// 获取所有设备
	var devices []entity.Device
	if err := query.Offset(offset).Limit(pageSize).
		Order("last_heartbeat DESC, created_at DESC").
		Find(&devices).Error; err != nil {
		return nil, 0, fmt.Errorf("获取设备列表失败: %w", err)
	}

	// 从Hub获取实时在线状态
	hub := GetWSHub()
	var filteredDevices []entity.Device

	for _, device := range devices {
		// 更新实时在线状态
		if hub != nil {
			device.IsOnline = hub.IsDeviceOnline(device.ID)
		}

		// 应用在线状态过滤
		if filter != nil {
			if filter.OnlineOnly && !device.IsOnline {
				continue
			}
			if filter.OfflineOnly && device.IsOnline {
				continue
			}
			if filter.IsOnline != nil && *filter.IsOnline != device.IsOnline {
				continue
			}
		}

		filteredDevices = append(filteredDevices, device)
	}

	return filteredDevices, int64(len(filteredDevices)), nil
}

// GetDeviceDetail 获取设备详情
func GetDeviceDetail(deviceID uint) (*entity.Device, error) {
	var device entity.Device
	result := global.DB.Preload("User").Where("id = ?", deviceID).First(&device)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errs.ErrDeviceNotFound
		}
		return nil, fmt.Errorf("查询设备失败: %w", result.Error)
	}

	// 从Hub获取实时在线状态
	if hub := GetWSHub(); hub != nil {
		device.IsOnline = hub.IsDeviceOnline(deviceID)
	}

	return &device, nil
}

// GetDeviceStatistics 获取设备统计信息
func GetDeviceStatistics() (int64, int64, int64, int64, error) {
	var totalDevices, activeDevices, boundDevices int64

	// 总设备数
	if err := global.DB.Model(&entity.Device{}).Count(&totalDevices).Error; err != nil {
		return 0, 0, 0, 0, fmt.Errorf("获取总设备数失败: %w", err)
	}

	// 激活设备数
	if err := global.DB.Model(&entity.Device{}).Where("is_active = ?", true).Count(&activeDevices).Error; err != nil {
		return 0, 0, 0, 0, fmt.Errorf("获取激活设备数失败: %w", err)
	}

	// 已绑定设备数
	if err := global.DB.Model(&entity.Device{}).Where("user_id IS NOT NULL").Count(&boundDevices).Error; err != nil {
		return 0, 0, 0, 0, fmt.Errorf("获取绑定设备数失败: %w", err)
	}

	// 从Hub获取实时在线设备数
	var onlineDevices int64 = 0
	if hub := GetWSHub(); hub != nil {
		onlineDevices = int64(hub.GetOnlineDevicesCount())
	}

	return totalDevices, onlineDevices, activeDevices, boundDevices, nil
}

// GenerateOnceKey 生成一次性密钥
func GenerateOnceKey() (string, error) {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes), nil
}

// UpdateDeviceOnceKey 更新设备的OnceKey
func UpdateDeviceOnceKey(deviceID uint, oldOnceKey string) (string, error) {
	// 查找设备
	var device entity.Device
	result := global.DB.Where("id = ?", deviceID).First(&device)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return "", errs.ErrDeviceNotFound
		}
		return "", fmt.Errorf("查询设备失败: %w", result.Error)
	}

	// 验证旧的OnceKey
	if device.OnceKey != oldOnceKey {
		return "", fmt.Errorf("旧的一次性密钥不匹配")
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

	if err := global.DB.Model(&device).Updates(updates).Error; err != nil {
		return "", fmt.Errorf("更新设备OnceKey失败: %w", err)
	}

	logger.Logger.Info("更新设备OnceKey",
		"device_id", deviceID,
		"serial_number", device.SerialNumber)

	return newOnceKey, nil
}
