package ws

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hang666/EasyUKey/server/internal/global"
	"github.com/hang666/EasyUKey/server/internal/model/entity"
	"github.com/hang666/EasyUKey/server/internal/service"
	"github.com/hang666/EasyUKey/shared/pkg/identity"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
	"github.com/hang666/EasyUKey/shared/pkg/messages"
	"github.com/hang666/EasyUKey/shared/pkg/wsutil"
)

// handleDeviceConnection 处理设备连接
func handleDeviceConnection(client *Client, wsMsg *messages.WSMessage) error {
	// 解析连接消息
	connMsg, err := wsutil.ParseMessage[messages.DeviceConnectionMessage](wsMsg)
	if err != nil {
		return sendErrorToClient(client, "device_connection", "parse_error", fmt.Sprintf("解析错误: %s", err.Error()))
	}

	// 1. 尝试通过序列号查找现有设备
	var device struct {
		ID            uint
		DeviceGroupID *uint
		IsActive      bool
	}
	result := global.DB.Table("devices").
		Select("id, device_group_id, is_active").
		Where("serial_number = ? AND volume_serial_number = ? AND deleted_at IS NULL", connMsg.SerialNumber, connMsg.VolumeSerialNumber).
		First(&device)

	if result.Error == nil {
		// 找到现有设备，正常连接
		return handleExistingDeviceConnection(client, &connMsg, device.ID, device.DeviceGroupID)
	}

	// 2. 没有找到现有设备，尝试跨平台匹配
	return handleCrossPlatformDeviceConnection(client, &connMsg)
}

// handleExistingDeviceConnection 处理现有设备连接
func handleExistingDeviceConnection(client *Client, connMsg *messages.DeviceConnectionMessage, deviceID uint, deviceGroupID *uint) error {
	// 如果设备关联了设备组，获取设备组的用户信息
	if deviceGroupID != nil {
		var deviceGroup entity.DeviceGroup
		if err := global.DB.Where("id = ?", *deviceGroupID).First(&deviceGroup).Error; err == nil {
			// 如果提供了认证信息但OnceKey不匹配，记录警告
			if connMsg.OnceKey != "" && deviceGroup.OnceKey != connMsg.OnceKey {
				logger.Logger.Warn("检测到可疑设备：现有设备连接时OnceKey不匹配",
					"device_id", deviceID,
					"device_group_id", deviceGroup.ID,
					"device_group_name", deviceGroup.Name,
					"provided_once_key", connMsg.OnceKey,
					"expected_once_key", deviceGroup.OnceKey,
					"serial_number", connMsg.SerialNumber,
				)
			}

			if deviceGroup.UserID != nil {
				client.mu.Lock()
				client.UserID = *deviceGroup.UserID
				client.mu.Unlock()
			}
		}
	}

	// 更新客户端信息
	client.mu.Lock()
	client.DeviceID = deviceID
	client.SerialNumber = connMsg.SerialNumber
	client.VolumeSerialNumber = connMsg.VolumeSerialNumber
	client.IsRegistered = true
	client.mu.Unlock()

	// 注册到Hub并触发设备连接回调
	if hub := service.GetWSHub(); hub != nil {
		if h, ok := hub.(*Hub); ok {
			h.register <- client
			hub.OnDeviceConnect(deviceID)
		}
	}

	// 发送连接成功响应
	connResp := &messages.DeviceConnectionResponseMessage{
		Success: true,
		Status:  "connected",
		Message: "设备连接成功",
	}

	return sendMessageToClient(client, "device_connection_response", connResp)
}

// handleCrossPlatformDeviceConnection 处理跨平台设备连接
func handleCrossPlatformDeviceConnection(client *Client, connMsg *messages.DeviceConnectionMessage) error {
	// 使用设备提供的认证密钥匹配现有设备组
	matchedGroup, err := service.FindDeviceGroupByAuth(connMsg.TOTPCode, connMsg.OnceKey)
	if err != nil {
		logger.Logger.Error("跨平台设备匹配失败", "error", err, "serial_number", connMsg.SerialNumber)
		return sendErrorToClient(client, "device_connection", "match_error", "跨平台设备匹配失败")
	}

	if matchedGroup == nil {
		// 无法匹配到现有设备组，可能是全新设备
		return sendErrorToClient(client, "device_connection", "no_match", "无法识别的设备，请先进行设备初始化")
	}

	// 匹配成功，创建新的设备记录并关联到现有设备组
	deviceID, err := createCrossPlatformDevice(connMsg, matchedGroup)
	if err != nil {
		logger.Logger.Error("创建跨平台设备失败", "error", err, "serial_number", connMsg.SerialNumber)
		return sendErrorToClient(client, "device_connection", "create_error", fmt.Sprintf("创建跨平台设备失败: %v", err))
	}

	// 更新客户端信息
	client.mu.Lock()
	if matchedGroup.UserID != nil {
		client.UserID = *matchedGroup.UserID
	}
	client.DeviceID = deviceID
	client.SerialNumber = connMsg.SerialNumber
	client.VolumeSerialNumber = connMsg.VolumeSerialNumber
	client.IsRegistered = true
	client.mu.Unlock()

	// 注册到Hub
	if hub := service.GetWSHub(); hub != nil {
		if h, ok := hub.(*Hub); ok {
			h.register <- client
			hub.OnDeviceConnect(deviceID)
		}
	}

	// 发送连接响应
	connResp := &messages.DeviceConnectionResponseMessage{
		Success: true,
		Status:  "pending_activation",
		Message: "跨平台设备识别成功，等待管理员激活",
	}

	return sendMessageToClient(client, "device_connection_response", connResp)
}

// handleDeviceReconnect 处理设备重连
func handleDeviceReconnect(client *Client, wsMsg *messages.WSMessage) error {
	// 解析重连消息
	reconnectMsg, err := wsutil.ParseMessage[messages.DeviceReconnectMessage](wsMsg)
	if err != nil {
		logger.Logger.Error("解析设备重连消息失败", "error", err)
		return err
	}

	// 转换为标准连接消息格式
	connMsg := messages.DeviceConnectionMessage{
		SerialNumber:       reconnectMsg.SerialNumber,
		VolumeSerialNumber: reconnectMsg.VolumeSerialNumber,
		TOTPCode:           reconnectMsg.TOTPCode,
		OnceKey:            reconnectMsg.OnceKey,
		DevicePath:         reconnectMsg.DevicePath,
		Vendor:             reconnectMsg.Vendor,
		Model:              reconnectMsg.Model,
	}

	// 查找现有设备
	var device struct {
		ID            uint
		DeviceGroupID *uint
		IsActive      bool
	}
	result := global.DB.Table("devices").
		Select("id, device_group_id, is_active").
		Where("serial_number = ? AND volume_serial_number = ? AND deleted_at IS NULL", connMsg.SerialNumber, connMsg.VolumeSerialNumber).
		First(&device)

	if result.Error != nil {
		// 没有找到设备，记录日志
		logger.Logger.Warn("设备重连失败：未找到对应设备",
			"serial_number", connMsg.SerialNumber,
			"volume_serial_number", connMsg.VolumeSerialNumber)
		return nil
	}

	return handleExistingDeviceConnection(client, &connMsg, device.ID, device.DeviceGroupID)
}

// createCrossPlatformDevice 创建跨平台设备记录
func createCrossPlatformDevice(connMsg *messages.DeviceConnectionMessage, group *entity.DeviceGroup) (uint, error) {
	// 创建新的设备记录，关联到现有设备组
	device := entity.Device{
		Name:               fmt.Sprintf("设备_%s_%s", group.Name, connMsg.SerialNumber[len(connMsg.SerialNumber)-4:]),
		DeviceGroupID:      &group.ID,
		SerialNumber:       connMsg.SerialNumber,
		VolumeSerialNumber: connMsg.VolumeSerialNumber,
		Vendor:             connMsg.Vendor,
		Model:              connMsg.Model,
		Remark:             "跨平台自动识别",
		IsActive:           false, // 重要：设为非激活状态
		IsOnline:           true,
		HeartbeatInterval:  30,
	}

	if err := global.DB.Create(&device).Error; err != nil {
		return 0, fmt.Errorf("创建跨平台设备记录失败: %w", err)
	}

	logger.Logger.Info("跨平台设备自动识别成功",
		"device_id", device.ID,
		"device_group_id", group.ID,
		"serial_number", connMsg.SerialNumber,
		"status", "待管理员激活")

	return device.ID, nil
}

// handleDeviceInit 处理设备初始化
func handleDeviceInit(client *Client, wsMsg *messages.WSMessage) error {
	// 解析初始化请求
	initMsg, err := wsutil.ParseMessage[messages.DeviceInitRequestMessage](wsMsg)
	if err != nil {
		return err
	}

	// 调用设备服务处理初始化
	onceKey, totpURI, err := service.InitDevice(&initMsg)

	// 构造响应
	var initResp *messages.DeviceInitResponseMessage
	if err != nil {
		logger.Logger.Error("设备初始化失败", "error", err, "serial_number", initMsg.SerialNumber)
		initResp = &messages.DeviceInitResponseMessage{
			Success: false,
			Error:   err.Error(),
			Message: "设备初始化失败",
		}
	} else {
		initResp = &messages.DeviceInitResponseMessage{
			Success: true,
			OnceKey: onceKey,
			TOTPURI: totpURI,
			Message: "设备初始化成功，请联系管理员绑定用户",
		}
	}

	// 发送初始化响应
	if err := sendMessageToClient(client, "device_init_response", initResp); err != nil {
		return err
	}

	// 初始化成功后自动注册设备为在线
	if initResp.Success {
		var deviceID uint
		result := global.DB.Table("devices").
			Select("id").
			Where("serial_number = ? AND volume_serial_number = ? AND deleted_at IS NULL", initMsg.SerialNumber, initMsg.VolumeSerialNumber).
			Pluck("id", &deviceID)
		if result.Error == nil && deviceID > 0 {
			client.mu.Lock()
			client.DeviceID = deviceID
			client.SerialNumber = initMsg.SerialNumber
			client.VolumeSerialNumber = initMsg.VolumeSerialNumber
			client.IsRegistered = true
			client.mu.Unlock()

			if hub := service.GetWSHub(); hub != nil {
				if h, ok := hub.(*Hub); ok {
					h.register <- client
					hub.OnDeviceConnect(deviceID)
				}
			}
		}
	}

	return nil
}

// handleAuthResponse 处理认证响应
func handleAuthResponse(client *Client, wsMsg *messages.WSMessage) error {
	// 解析认证响应
	authResp, err := wsutil.ParseMessage[messages.AuthResponseMessage](wsMsg)
	if err != nil {
		return err
	}

	// 处理认证响应
	if err := service.ProcessAuthResponse(authResp.RequestID, &authResp); err != nil {
		// 服务端验证失败，发送失败响应给客户端
		failureResp := &messages.AuthSuccessResponseMessage{
			RequestID: authResp.RequestID,
			Success:   false,
		}
		sendMessageToClient(client, "auth_success_response", failureResp)
		return err
	}

	// 只有在服务端验证成功且客户端同意认证时，才生成新的OnceKey
	if authResp.Success {
		newOnceKey, err := service.UpdateDeviceOnceKey(client.DeviceID, authResp.UsedKey)
		if err != nil {
			logger.Logger.Error("OnceKey更新失败", "request_id", authResp.RequestID, "error", err)
			service.CompleteOnceKeyUpdateAuth(authResp.RequestID, false, fmt.Sprintf("OnceKey更新失败: %v", err))
			return err
		}

		// 发送新的OnceKey给客户端
		successResp := &messages.AuthSuccessResponseMessage{
			RequestID:  authResp.RequestID,
			Success:    true,
			NewOnceKey: newOnceKey,
		}

		if err := sendMessageToClient(client, "auth_success_response", successResp); err != nil {
			logger.Logger.Error("发送新OnceKey失败", "request_id", authResp.RequestID, "error", err)
			service.CompleteOnceKeyUpdateAuth(authResp.RequestID, false, fmt.Sprintf("发送新OnceKey失败: %v", err))
			return err
		}
	}

	return nil
}

// handleOnceKeyUpdate 处理OnceKey更新确认
func handleOnceKeyUpdate(client *Client, wsMsg *messages.WSMessage) error {
	// 解析确认消息
	confirmMsg, err := wsutil.ParseMessage[messages.OnceKeyUpdateConfirmMessage](wsMsg)
	if err != nil {
		logger.Logger.Error("处理once_key_update_confirm消息失败", "error", err, "user_id", client.UserID, "device_id", client.DeviceID)
		return err
	}

	// 完成OnceKey更新后的认证流程
	if err := service.CompleteOnceKeyUpdateAuth(confirmMsg.RequestID, confirmMsg.Success, confirmMsg.Error); err != nil {
		logger.Logger.Error("完成OnceKey更新认证失败", "request_id", confirmMsg.RequestID, "error", err)
		return err
	}

	if confirmMsg.Success {
		logger.Logger.Info("OnceKey更新确认成功，认证流程已完成", "request_id", confirmMsg.RequestID, "device_id", client.DeviceID)
	} else {
		logger.Logger.Error("OnceKey更新确认失败", "request_id", confirmMsg.RequestID, "error", confirmMsg.Error)
	}

	return nil
}

// handleDeviceStatus 处理设备状态响应
func handleDeviceStatus(client *Client, wsMsg *messages.WSMessage) error {
	// 解析状态响应
	_, err := wsutil.ParseMessage[messages.DeviceStatusMessage](wsMsg)
	if err != nil {
		return err
	}

	return nil
}

// handlePing 处理Ping消息
func handlePing(client *Client, wsMsg *messages.WSMessage) error {
	hub := service.GetWSHub()
	if hub != nil {
		if h, ok := hub.(*Hub); ok {
			h.syncHeartbeat(client.DeviceID)
		}
	}

	// 重置读取超时时间
	client.resetReadDeadline()
	return nil
}

// handlePong 处理Pong消息
func handlePong(client *Client, wsMsg *messages.WSMessage) error {
	client.updateLastPong()
	// 重置读取超时时间
	client.resetReadDeadline()
	return nil
}

// handleKeyExchangeRequest 处理密钥交换请求
func handleKeyExchangeRequest(client *Client, wsMsg *messages.WSMessage) error {
	// 解析密钥交换请求
	keyExchReq, err := wsutil.ParseMessage[messages.KeyExchangeRequestMessage](wsMsg)
	if err != nil {
		logger.Logger.Error("解析密钥交换请求失败", "error", err, "device_id", client.DeviceID)
		return sendErrorToClient(client, "key_exchange_response", "parse_error", "密钥交换请求解析失败")
	}

	// 创建服务端密钥交换器
	keyExchange, err := identity.NewKeyExchange()
	if err != nil {
		return sendErrorToClient(client, "key_exchange_response", "server_error", "服务端密钥交换器创建失败")
	}

	// 计算共享密钥
	if err := keyExchange.ComputeSharedKey(keyExchReq.PublicKey); err != nil {
		return sendErrorToClient(client, "key_exchange_response", "compute_error", "共享密钥计算失败")
	}

	// 创建加密器
	encryptor, err := keyExchange.CreateEncryptor()
	if err != nil {
		return sendErrorToClient(client, "key_exchange_response", "encryptor_error", "加密器创建失败")
	}

	// 更新客户端状态
	client.mu.Lock()
	client.KeyExchange = keyExchange
	client.Encryptor = encryptor
	client.HandshakeStatus = messages.HandshakeStatusCompleted
	client.mu.Unlock()

	// 发送密钥交换响应
	keyExchResp := &messages.KeyExchangeResponseMessage{
		PublicKey: keyExchange.GetPublicKeyBase64(),
		Success:   true,
	}

	if err := wsutil.SendMessageToChannel(client.Send, "key_exchange_response", keyExchResp); err != nil {
		return err
	}

	return nil
}

// handleEncryptedMessage 处理加密消息
func handleEncryptedMessage(client *Client, wsMsg *messages.WSMessage) error {
	// 检查握手状态
	client.mu.RLock()
	handshakeStatus := client.HandshakeStatus
	encryptor := client.Encryptor
	client.mu.RUnlock()

	if handshakeStatus != messages.HandshakeStatusCompleted {
		logger.Logger.Error("收到加密消息但握手未完成", "device_id", client.DeviceID)
		return sendErrorToClient(client, "encrypted", "handshake_error", "握手未完成")
	}

	if encryptor == nil {
		return sendErrorToClient(client, "encrypted", "encryptor_error", "加密器未初始化")
	}

	// 解析加密消息
	encryptedMsg, err := wsutil.ParseMessage[messages.EncryptedMessage](wsMsg)
	if err != nil {
		return sendErrorToClient(client, "encrypted", "parse_error", "加密消息解析失败")
	}

	// 解密消息
	decryptedData, err := encryptor.DecryptMessage(encryptedMsg.Payload, encryptedMsg.Nonce)
	if err != nil {
		logger.Logger.Error("解密消息失败", "error", err, "device_id", client.DeviceID)
		return sendErrorToClient(client, "encrypted", "decrypt_error", "消息解密失败")
	}

	// 解析解密后的消息
	var decryptedWSMsg messages.WSMessage
	if err := json.Unmarshal(decryptedData, &decryptedWSMsg); err != nil {
		return sendErrorToClient(client, "encrypted", "unmarshal_error", "解密后消息解析失败")
	}

	// 递归处理解密后的消息
	return dispatchMessage(client, &decryptedWSMsg)
}

// sendEncryptedMessage 发送加密消息
func sendEncryptedMessage(client *Client, msgType string, data interface{}) error {
	client.mu.RLock()
	encryptor := client.Encryptor
	handshakeStatus := client.HandshakeStatus
	client.mu.RUnlock()

	// 检查握手状态
	if handshakeStatus != messages.HandshakeStatusCompleted {
		return fmt.Errorf("握手未完成")
	}

	if encryptor == nil {
		return fmt.Errorf("加密器未初始化")
	}

	// 创建原始消息
	originalMsg := &messages.WSMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}

	// 序列化原始消息
	originalData, err := json.Marshal(originalMsg)
	if err != nil {
		return err
	}

	// 加密消息
	encryptedPayload, nonce, err := encryptor.EncryptMessage(originalData)
	if err != nil {
		return err
	}

	// 创建加密消息
	encryptedMsg := &messages.EncryptedMessage{
		Payload: encryptedPayload,
		Nonce:   nonce,
	}

	// 发送加密消息
	return wsutil.SendMessageToChannel(client.Send, "encrypted", encryptedMsg)
}

// sendMessageToClient 发送消息到客户端（支持加密）
func sendMessageToClient(client *Client, msgType string, data interface{}) error {
	client.mu.RLock()
	encryptor := client.Encryptor
	handshakeStatus := client.HandshakeStatus
	client.mu.RUnlock()

	// 检查是否需要加密
	if handshakeStatus == messages.HandshakeStatusCompleted && encryptor != nil {
		return sendEncryptedMessage(client, msgType, data)
	}

	// 直接发送未加密消息
	return wsutil.SendMessageToChannel(client.Send, msgType, data)
}

// sendErrorToClient 发送错误消息到客户端（支持加密）
func sendErrorToClient(client *Client, msgType string, errorCode string, errorMsg string) error {
	return sendMessageToClient(client, msgType, map[string]interface{}{
		"error_code": errorCode,
		"error":      errorMsg,
		"success":    false,
	})
}
