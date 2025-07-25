package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hang666/EasyUKey/sdk/request"
	"github.com/hang666/EasyUKey/server/internal/global"
	"github.com/hang666/EasyUKey/server/internal/model/entity"
	"github.com/hang666/EasyUKey/shared/pkg/errs"
	"github.com/hang666/EasyUKey/shared/pkg/identity"
	"github.com/hang666/EasyUKey/shared/pkg/messages"
)

// InitDevice 初始化设备 - 简化版，创建设备和设备组
func InitDevice(initReq *messages.DeviceInitRequestMessage) (string, string, error) {
	// 检查设备是否已存在（同平台重复注册）
	var existingDevice entity.Device
	result := global.DB.Where("serial_number = ? AND volume_serial_number = ?",
		initReq.SerialNumber, initReq.VolumeSerialNumber).First(&existingDevice)

	if result.Error == nil {
		return "", "", errs.ErrDeviceAlreadyExists
	}

	if result.Error != gorm.ErrRecordNotFound {
		return "", "", fmt.Errorf("查询设备失败: %w", result.Error)
	}

	// 创建新设备和设备组
	return createNewDeviceWithGroup(initReq)
}

// createNewDeviceWithGroup 创建新设备和对应的设备组
func createNewDeviceWithGroup(initReq *messages.DeviceInitRequestMessage) (string, string, error) {
	// 生成认证密钥
	totpAccount := fmt.Sprintf("%s_%s", initReq.SerialNumber, uuid.New().String()[:6])
	totpSecret, err := identity.GenerateTOTPSecretURI("EasyUKey", totpAccount)
	if err != nil {
		return "", "", fmt.Errorf("生成TOTP密钥失败: %w", err)
	}

	onceKey, err := GenerateOnceKey()
	if err != nil {
		return "", "", fmt.Errorf("生成OnceKey失败: %w", err)
	}

	tx := global.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if tx.Error != nil {
		return "", "", fmt.Errorf("开始事务失败: %w", tx.Error)
	}

	// 创建设备组
	deviceGroup := entity.DeviceGroup{
		Name:        fmt.Sprintf("设备组_%s", initReq.SerialNumber[len(initReq.SerialNumber)-6:]),
		Description: "设备初始化时自动创建",
		TOTPSecret:  totpSecret,
		OnceKey:     onceKey,
		Permissions: []string{},
		IsActive:    false, // 等待管理员激活
	}

	if err := tx.Create(&deviceGroup).Error; err != nil {
		tx.Rollback()
		return "", "", fmt.Errorf("创建设备组失败: %w", err)
	}

	// 创建设备记录
	device := entity.Device{
		Name:               fmt.Sprintf("设备_%s", initReq.SerialNumber[len(initReq.SerialNumber)-6:]),
		DeviceGroupID:      &deviceGroup.ID,
		SerialNumber:       initReq.SerialNumber,
		VolumeSerialNumber: initReq.VolumeSerialNumber,
		Vendor:             initReq.Vendor,
		Model:              initReq.Model,
		Remark:             "设备初始化",
		IsActive:           false,
		IsOnline:           false,
		HeartbeatInterval:  30,
	}

	if err := tx.Create(&device).Error; err != nil {
		tx.Rollback()
		return "", "", fmt.Errorf("创建设备记录失败: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return "", "", fmt.Errorf("提交事务失败: %w", err)
	}

	return onceKey, totpSecret, nil
}

// UpdateDevice 更新设备信息
func UpdateDevice(deviceID uint, req *request.UpdateDeviceRequest) (*entity.Device, error) {
	// 输入验证
	if deviceID == 0 {
		return nil, errs.ErrInvalidDeviceID
	}
	if req == nil {
		return nil, errs.ErrInvalidRequest
	}
	if req.Name == "" && req.Remark == "" && req.IsActive == nil {
		return nil, errs.ErrInvalidRequest
	}

	// 查找设备和设备组信息
	var device entity.Device
	result := global.DB.Preload("DeviceGroup").Where("id = ?", deviceID).First(&device)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errs.ErrDeviceNotFound
		}
		return nil, fmt.Errorf("查询设备失败: %w", result.Error)
	}

	// 记录原始激活状态，用于后续判断
	oldIsActive := device.IsActive

	// 更新字段
	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["name"] = req.Name
	}

	if req.Remark != "" {
		updates["remark"] = req.Remark
	}

	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	// 执行更新
	if len(updates) > 0 {
		if err := global.DB.Model(&device).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("更新设备失败: %w", err)
		}
	}

	// 处理设备激活状态变化的Hub更新
	if req.IsActive != nil && *req.IsActive != oldIsActive {
		if hub := GetWSHub(); hub != nil && hub.IsDeviceOnline(deviceID) {
			if !*req.IsActive {
				// 设备被停用，断开WebSocket连接
				hub.OnDeviceDisconnect(deviceID)
			} else if *req.IsActive && device.DeviceGroup != nil &&
				device.DeviceGroup.UserID != nil && *device.DeviceGroup.UserID > 0 {
				// 设备被激活且设备组有绑定用户，建立用户关联
				hub.LinkDeviceToUser(deviceID, *device.DeviceGroup.UserID)
			}
		}
	}

	// 重新查询更新后的设备信息
	global.DB.Preload("DeviceGroup").Where("id = ?", deviceID).First(&device)

	// 从Hub获取实时在线状态
	if hub := GetWSHub(); hub != nil {
		device.IsOnline = hub.IsDeviceOnline(deviceID)
	}

	return &device, nil
}

// DeleteDevice 删除设备
func DeleteDevice(deviceID uint) error {
	// 查找设备
	var device entity.Device
	result := global.DB.Where("id = ?", deviceID).First(&device)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return errs.ErrDeviceNotFound
		}
		return fmt.Errorf("查询设备失败: %w", result.Error)
	}

	// 强制断开设备的 WebSocket 连接
	if hub := GetWSHub(); hub != nil {
		hub.OnDeviceDisconnect(deviceID)
	}

	// 删除设备
	if err := global.DB.Delete(&device).Error; err != nil {
		return fmt.Errorf("删除设备失败: %w", err)
	}

	return nil
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

	// 重新查询更新后的设备信息
	global.DB.Preload("DeviceGroup").Where("id = ?", deviceID).First(&device)

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
	query := global.DB.Model(&entity.Device{}).Preload("DeviceGroup")

	// 应用过滤条件
	if filter != nil {
		if filter.IsActive != nil {
			query = query.Where("is_active = ?", *filter.IsActive)
		}
		if filter.DeviceGroupID != nil {
			query = query.Where("device_group_id = ?", *filter.DeviceGroupID)
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
	result := global.DB.Preload("DeviceGroup").Where("id = ?", deviceID).First(&device)

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

	// 已关联设备组的设备数
	if err := global.DB.Model(&entity.Device{}).Where("device_group_id IS NOT NULL").Count(&boundDevices).Error; err != nil {
		return 0, 0, 0, 0, fmt.Errorf("获取关联设备数失败: %w", err)
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

// UpdateDeviceOnceKey 更新设备组的OnceKey（通过设备ID）
func UpdateDeviceOnceKey(deviceID uint, oldOnceKey string) (string, error) {
	// 查找设备及其设备组
	var device entity.Device
	result := global.DB.Preload("DeviceGroup").Where("id = ?", deviceID).First(&device)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return "", errs.ErrDeviceNotFound
		}
		return "", fmt.Errorf("查询设备失败: %w", result.Error)
	}

	// 检查设备是否关联设备组
	if device.DeviceGroup == nil {
		return "", fmt.Errorf("设备未关联设备组")
	}

	// 使用设备组服务更新OnceKey
	return UpdateDeviceGroupOnceKey(device.DeviceGroup.ID, oldOnceKey)
}
