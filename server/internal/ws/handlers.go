package ws

import (
	"fmt"

	"github.com/hang666/EasyUKey/server/internal/global"
	"github.com/hang666/EasyUKey/server/internal/service"
	"github.com/hang666/EasyUKey/shared/pkg/errors"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
	"github.com/hang666/EasyUKey/shared/pkg/messages"
	"github.com/hang666/EasyUKey/shared/pkg/wsutil"
)

// handleDeviceRegister 处理设备注册
func handleDeviceRegister(client *Client, wsMsg *messages.WSMessage) error {
	// 验证消息
	if err := wsutil.ValidateMessage(wsMsg); err != nil {
		logger.Logger.Error("处理device_register消息失败", "error", err, "user_id", client.UserID, "device_id", client.DeviceID)
		return wsutil.SendErrorToChannel(client.Send, "device_register", "validation_error", errors.ErrWSValidation.Error())
	}

	// 解析注册消息
	regMsg, err := wsutil.ParseMessage[messages.DeviceRegistrationMessage](wsMsg)
	if err != nil {
		logger.Logger.Error("处理device_register消息失败", "error", err, "user_id", client.UserID, "device_id", client.DeviceID)
		return wsutil.SendErrorToChannel(client.Send, "device_register", "parse_error", fmt.Sprintf("解析错误: %s", err.Error()))
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
		return wsutil.SendErrorToChannel(client.Send, "device_register", "device_not_found", errors.ErrDeviceNotFoundClient.Error())
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
			hub.OnDeviceConnect(device.ID, device.UserID, regMsg.SerialNumber, regMsg.VolumeSerialNumber)
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
	if err := wsutil.SendMessageToChannel(client.Send, "device_init_response", initResp); err != nil {
		logger.Logger.Error("处理device_init_request消息失败", "error", err, "user_id", client.UserID, "device_id", client.DeviceID)
		return err
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

			if err := wsutil.SendMessageToChannel(client.Send, "auth_success_response", successResp); err != nil {
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
