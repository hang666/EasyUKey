package ws

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hang666/EasyUKey/server/internal/global"
	"github.com/hang666/EasyUKey/server/internal/service"
	"github.com/hang666/EasyUKey/shared/pkg/errors"
	"github.com/hang666/EasyUKey/shared/pkg/identity"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
	"github.com/hang666/EasyUKey/shared/pkg/messages"
	"github.com/hang666/EasyUKey/shared/pkg/wsutil"
)

// handleDeviceRegister 处理设备注册
func handleDeviceRegister(client *Client, wsMsg *messages.WSMessage) error {
	// 验证消息
	if err := wsutil.ValidateMessage(wsMsg); err != nil {
		logger.Logger.Error("处理device_register消息失败", "error", err, "user_id", client.UserID, "device_id", client.DeviceID)
		return sendErrorToClient(client, "device_register", "validation_error", errors.ErrWSValidation.Error())
	}

	// 解析注册消息
	regMsg, err := wsutil.ParseMessage[messages.DeviceRegistrationMessage](wsMsg)
	if err != nil {
		logger.Logger.Error("处理device_register消息失败", "error", err, "user_id", client.UserID, "device_id", client.DeviceID)
		return sendErrorToClient(client, "device_register", "parse_error", fmt.Sprintf("解析错误: %s", err.Error()))
	}

	// 通过序列号查找设备
	var device struct {
		ID     uint
		UserID uint
	}
	result := global.DB.Table("devices").
		Select("id, user_id").
		Where("serial_number = ? AND volume_serial_number = ?", regMsg.SerialNumber, regMsg.VolumeSerialNumber).
		First(&device)

	if result.Error != nil {
		logger.Logger.Error("处理device_register消息失败", "error", result.Error, "user_id", client.UserID, "device_id", client.DeviceID, "serial_number", regMsg.SerialNumber, "volume_serial_number", regMsg.VolumeSerialNumber)
		return sendErrorToClient(client, "device_register", "device_not_found", errors.ErrDeviceNotFoundClient.Error())
	}

	// 更新客户端信息
	client.mu.Lock()
	client.UserID = device.UserID
	client.DeviceID = device.ID
	client.SerialNumber = regMsg.SerialNumber
	client.VolumeSerialNumber = regMsg.VolumeSerialNumber
	client.IsRegistered = true
	client.mu.Unlock()

	// 注册到Hub并触发设备连接回调
	if hub := service.GetWSHub(); hub != nil {
		if h, ok := hub.(*Hub); ok {
			h.register <- client
			// 触发设备连接状态同步
			hub.OnDeviceConnect(device.ID)
		}
	}

	logger.Logger.Info("device_register: 设备注册成功", "user_id", client.UserID, "device_id", client.DeviceID, "serial_number", regMsg.SerialNumber)

	return nil
}

// handleDeviceInit 处理设备初始化
func handleDeviceInit(client *Client, wsMsg *messages.WSMessage) error {
	// 解析初始化请求
	initMsg, err := wsutil.ParseMessage[messages.DeviceInitRequestMessage](wsMsg)
	if err != nil {
		logger.Logger.Error("处理device_init_request消息失败", "error", err, "user_id", client.UserID, "device_id", client.DeviceID)
		return err
	}

	// 调用设备服务处理初始化
	onceKey, totpURI, err := service.InitDevice(&initMsg)

	// 构造响应
	var initResp *messages.DeviceInitResponseMessage
	if err != nil {
		logger.Logger.Error("处理device_init_request消息失败", "error", err, "user_id", client.UserID, "device_id", client.DeviceID, "serial_number", initMsg.SerialNumber, "volume_serial_number", initMsg.VolumeSerialNumber)
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
		logger.Logger.Error("处理device_init_request消息失败", "error", err, "user_id", client.UserID, "device_id", client.DeviceID)
		return err
	}

	// 初始化成功后自动注册设备为在线
	if initResp.Success {
		var deviceID uint
		result := global.DB.Table("devices").
			Select("id").
			Where("serial_number = ? AND volume_serial_number = ?", initMsg.SerialNumber, initMsg.VolumeSerialNumber).
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

	logger.Logger.Info("device_init_request: 已处理设备初始化请求", "user_id", client.UserID, "device_id", client.DeviceID, "serial_number", initMsg.SerialNumber, "success", initResp.Success)

	return nil
}

// handleAuthResponse 处理认证响应
func handleAuthResponse(client *Client, wsMsg *messages.WSMessage) error {
	// 解析认证响应
	authResp, err := wsutil.ParseMessage[messages.AuthResponseMessage](wsMsg)
	if err != nil {
		logger.Logger.Error("处理auth_response消息失败", "error", err, "user_id", client.UserID, "device_id", client.DeviceID)
		return err
	}

	// 处理认证响应
	if err := service.ProcessAuthResponse(authResp.RequestID, &authResp); err != nil {
		logger.Logger.Error("处理auth_response消息失败", "error", err, "user_id", client.UserID, "device_id", client.DeviceID, "request_id", authResp.RequestID, "success", authResp.Success)
		return err
	}

	// 如果认证成功且客户端提供了使用过的OnceKey，需要生成新的OnceKey
	if authResp.Success && authResp.UsedKey != "" {
		newOnceKey, err := service.UpdateDeviceOnceKey(client.DeviceID, authResp.UsedKey)
		if err != nil {
			logger.Logger.Error("处理auth_response消息失败", "error", err, "user_id", client.UserID, "device_id", client.DeviceID, "request_id", authResp.RequestID, "device_id", client.DeviceID)
			// 这里不返回错误，因为认证已经成功，OnceKey更新失败不应该影响认证结果
		} else {
			// 发送新的OnceKey给客户端
			successResp := &messages.AuthSuccessResponseMessage{
				RequestID:  authResp.RequestID,
				Success:    true,
				NewOnceKey: newOnceKey,
			}

			if err := sendMessageToClient(client, "auth_success_response", successResp); err != nil {
				logger.Logger.Error("处理auth_response消息失败", "error", err, "user_id", client.UserID, "device_id", client.DeviceID, "request_id", authResp.RequestID, "device_id", client.DeviceID)
			}
		}
	}

	logger.Logger.Info("处理认证响应完成",
		"request_id", authResp.RequestID,
		"success", authResp.Success,
		"user_id", client.UserID,
		"device_id", client.DeviceID)

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

	// 记录OnceKey更新确认
	if confirmMsg.Success {
		logger.Logger.Info("OnceKey更新确认成功",
			"request_id", confirmMsg.RequestID,
			"user_id", client.UserID,
			"device_id", client.DeviceID)
	} else {
		logger.Logger.Error("OnceKey更新确认失败",
			"error", confirmMsg.Error,
			"request_id", confirmMsg.RequestID,
			"user_id", client.UserID,
			"device_id", client.DeviceID)
	}

	return nil
}

// handleDeviceStatus 处理设备状态响应
func handleDeviceStatus(client *Client, wsMsg *messages.WSMessage) error {
	// 解析状态响应
	statusMsg, err := wsutil.ParseMessage[messages.DeviceStatusMessage](wsMsg)
	if err != nil {
		logger.Logger.Error("处理device_status_response消息失败", "error", err, "user_id", client.UserID, "device_id", client.DeviceID)
		return err
	}

	logger.Logger.Info("device_status_response: 收到设备状态响应", "user_id", client.UserID, "device_id", client.DeviceID, "status", statusMsg.Status)

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
		logger.Logger.Error("创建密钥交换器失败", "error", err, "device_id", client.DeviceID)
		return sendErrorToClient(client, "key_exchange_response", "server_error", "服务端密钥交换器创建失败")
	}

	// 计算共享密钥
	if err := keyExchange.ComputeSharedKey(keyExchReq.PublicKey); err != nil {
		logger.Logger.Error("计算共享密钥失败", "error", err, "device_id", client.DeviceID)
		return sendErrorToClient(client, "key_exchange_response", "compute_error", "共享密钥计算失败")
	}

	// 创建加密器
	encryptor, err := keyExchange.CreateEncryptor()
	if err != nil {
		logger.Logger.Error("创建加密器失败", "error", err, "device_id", client.DeviceID)
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
		logger.Logger.Error("发送密钥交换响应失败", "error", err, "device_id", client.DeviceID)
		return err
	}

	logger.Logger.Info("密钥交换成功", "device_id", client.DeviceID)
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
		logger.Logger.Error("收到加密消息但加密器未初始化", "device_id", client.DeviceID)
		return sendErrorToClient(client, "encrypted", "encryptor_error", "加密器未初始化")
	}

	// 解析加密消息
	encryptedMsg, err := wsutil.ParseMessage[messages.EncryptedMessage](wsMsg)
	if err != nil {
		logger.Logger.Error("解析加密消息失败", "error", err, "device_id", client.DeviceID)
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
		logger.Logger.Error("解析解密后的消息失败", "error", err, "device_id", client.DeviceID)
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
		logger.Logger.Error("尝试发送加密消息但握手未完成", "device_id", client.DeviceID)
		return fmt.Errorf("握手未完成")
	}

	if encryptor == nil {
		logger.Logger.Error("尝试发送加密消息但加密器未初始化", "device_id", client.DeviceID)
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
		logger.Logger.Error("序列化原始消息失败", "error", err, "device_id", client.DeviceID)
		return err
	}

	// 加密消息
	encryptedPayload, nonce, err := encryptor.EncryptMessage(originalData)
	if err != nil {
		logger.Logger.Error("加密消息失败", "error", err, "device_id", client.DeviceID)
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
