package ws

import (
	"github.com/hang666/EasyUKey/client/internal/device"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
	"github.com/hang666/EasyUKey/shared/pkg/messages"
	"github.com/hang666/EasyUKey/shared/pkg/wsutil"
)

// sendWSMessage 是一个内部辅助函数，用于发送格式化的WebSocket消息
func sendWSMessage(msgType string, data interface{}) error {
	mu.Lock()
	defer mu.Unlock()

	if conn == nil {
		return errNotConnected
	}
	return wsutil.SendMessage(conn, msgType, data)
}

// SendDeviceRegistration 发送设备注册消息
func SendDeviceRegistration() error {
	dev := device.DeviceInfo.GetDevice()
	if dev == nil {
		return errDeviceNotAvailable
	}

	registration := messages.DeviceRegistrationMessage{
		SerialNumber:       dev.SerialNumber,
		VolumeSerialNumber: dev.VolumeSerialNumber,
		DevicePath:         dev.DevicePath,
		Vendor:             dev.Vendor,
		Model:              dev.Model,
	}

	return sendWSMessage("device_register", registration)
}

// SendDeviceInitRequest 发送设备初始化请求
func SendDeviceInitRequest() error {
	dev := device.DeviceInfo.GetDevice()
	if dev == nil {
		return errDeviceNotAvailable
	}

	initRequest := messages.DeviceInitRequestMessage{
		SerialNumber:       dev.SerialNumber,
		VolumeSerialNumber: dev.VolumeSerialNumber,
		DevicePath:         dev.DevicePath,
		Vendor:             dev.Vendor,
		Model:              dev.Model,
	}

	logger.Logger.Info("发送设备初始化请求", "device_id", dev.SerialNumber)
	return sendWSMessage("device_init_request", initRequest)
}

// SendAuthResponse 发送认证响应
func SendAuthResponse(requestID string, success bool, authKey string, usedOnceKey string, serialNumber string, volumeSerialNumber string, errorMsg string) {
	response := messages.AuthResponseMessage{
		RequestID:          requestID,
		Success:            success,
		AuthKey:            authKey,
		Error:              errorMsg,
		UsedKey:            usedOnceKey,
		SerialNumber:       serialNumber,
		VolumeSerialNumber: volumeSerialNumber,
	}

	if err := sendWSMessage("auth_response", response); err != nil {
		logger.Logger.Error("发送认证响应失败", "error", err)
	}
}

// SendPingMessage 发送心跳
func SendPingMessage() error {
	if err := sendWSMessage("ping", nil); err != nil {
		logger.Logger.Warn("发送心跳失败", "error", err)
		return err
	}
	return nil
}

// SendOnceKeyUpdateConfirm 发送一次性密钥更新确认
func SendOnceKeyUpdateConfirm(requestID string, success bool, errorMsg string) {
	response := messages.OnceKeyUpdateConfirmMessage{
		RequestID: requestID,
		Success:   success,
		Error:     errorMsg,
	}

	if err := sendWSMessage("once_key_update_confirm", response); err != nil {
		logger.Logger.Error("发送密钥更新确认失败", "error", err)
	}
}

// SendPongMessage 回复心跳
func SendPongMessage() {
	if err := sendWSMessage("pong", nil); err != nil {
		logger.Logger.Warn("回复心跳失败", "error", err)
	}
}

// SendDeviceStatusResponse 发送设备状态响应
func SendDeviceStatusResponse(status string, serialNumber string, volumeSerialNumber string) {
	deviceStatus := messages.DeviceStatusMessage{
		Status:             status,
		SerialNumber:       serialNumber,
		VolumeSerialNumber: volumeSerialNumber,
	}

	if err := sendWSMessage("device_status_response", deviceStatus); err != nil {
		logger.Logger.Error("发送设备状态响应失败", "error", err)
	}
}
