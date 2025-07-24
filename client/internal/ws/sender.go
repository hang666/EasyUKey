package ws

import (
	"encoding/json"
	"time"

	"github.com/hang666/EasyUKey/client/internal/device"
	"github.com/hang666/EasyUKey/client/internal/global"
	"github.com/hang666/EasyUKey/shared/pkg/errs"
	"github.com/hang666/EasyUKey/shared/pkg/identity"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
	"github.com/hang666/EasyUKey/shared/pkg/messages"
	"github.com/hang666/EasyUKey/shared/pkg/wsutil"
)

// sendWSMessage 是一个内部辅助函数，用于发送格式化的WebSocket消息
func sendWSMessage(msgType string, data interface{}) error {
	mu.Lock()
	defer mu.Unlock()

	if conn == nil {
		return errs.ErrWSNotConnected
	}

	// 检查是否需要加密
	if handshakeStatus == messages.HandshakeStatusCompleted && encryptor != nil {
		return sendEncryptedMessage(msgType, data)
	}

	return wsutil.SendMessage(conn, msgType, data)
}

// sendEncryptedMessage 发送加密消息
func sendEncryptedMessage(msgType string, data interface{}) error {
	if encryptor == nil {
		return errs.ErrWSNotConnected
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

	// 直接发送加密消息（不再次加密）
	return wsutil.SendMessage(conn, "encrypted", encryptedMsg)
}

// SendDeviceConnection 发送设备连接消息
func SendDeviceConnection() error {
	dev := device.DeviceInfo.GetDevice()
	if dev == nil {
		return errs.ErrDeviceNotAvailable
	}

	// 通过PIN管理器获取PIN
	pin, err := global.PinManager.WaitPIN()
	if err != nil {
		logger.Logger.Error("PIN获取失败", "error", err)
		return err
	}

	// 使用PIN从安全存储获取TOTP URI
	totpURI, err := identity.GetTOTPSecret(pin, global.Config.EncryptKeyStr, global.SecureStoragePath)
	if err != nil {
		logger.Logger.Error("获取TOTP密钥失败", "error", err)
		return err
	}

	// 解析TOTP URI并生成当前TOTP码
	totpConfig, err := identity.ParseTOTPURI(totpURI)
	if err != nil {
		logger.Logger.Error("解析TOTP URI失败", "error", err)
		return err
	}

	totpCode, err := identity.GenerateTOTPCode(totpConfig, time.Now())
	if err != nil {
		logger.Logger.Error("生成TOTP码失败", "error", err)
		return err
	}

	// 使用PIN从安全存储获取OnceKey
	onceKey, err := identity.GetOnceKey(pin, global.Config.EncryptKeyStr, global.SecureStoragePath)
	if err != nil {
		logger.Logger.Error("获取OnceKey失败", "error", err)
		return err
	}

	connection := messages.DeviceConnectionMessage{
		SerialNumber:       dev.SerialNumber,
		VolumeSerialNumber: dev.VolumeSerialNumber,
		TOTPCode:           totpCode,
		OnceKey:            onceKey,
		DevicePath:         dev.DevicePath,
		Vendor:             dev.Vendor,
		Model:              dev.Model,
	}

	return sendWSMessage("device_connection", connection)
}

// SendDeviceReconnect 发送设备重连消息
func SendDeviceReconnect() error {
	dev := device.DeviceInfo.GetDevice()
	if dev == nil {
		return errs.ErrDeviceNotAvailable
	}

	reconnect := messages.DeviceReconnectMessage{
		SerialNumber:       dev.SerialNumber,
		VolumeSerialNumber: dev.VolumeSerialNumber,
		DevicePath:         dev.DevicePath,
		Vendor:             dev.Vendor,
		Model:              dev.Model,
	}

	return sendWSMessage("device_reconnect", reconnect)
}

// SendDeviceInitRequest 发送设备初始化请求
func SendDeviceInitRequest() error {
	dev := device.DeviceInfo.GetDevice()
	if dev == nil {
		return errs.ErrDeviceNotAvailable
	}

	initRequest := messages.DeviceInitRequestMessage{
		SerialNumber:       dev.SerialNumber,
		VolumeSerialNumber: dev.VolumeSerialNumber,
		DevicePath:         dev.DevicePath,
		Vendor:             dev.Vendor,
		Model:              dev.Model,
	}

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
	return sendWSMessage("ping", nil)
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
	sendWSMessage("pong", nil)
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

// SendKeyExchangeRequest 发送密钥交换请求
func SendKeyExchangeRequest() error {
	// 创建密钥交换器
	kx, err := identity.NewKeyExchange()
	if err != nil {
		return err
	}

	// 保存密钥交换器到全局变量
	keyExchange = kx

	// 创建密钥交换请求
	keyExchReq := &messages.KeyExchangeRequestMessage{
		PublicKey: kx.GetPublicKeyBase64(),
	}

	// 密钥交换请求不能加密，必须直接发送
	mu.Lock()
	defer mu.Unlock()
	if conn == nil {
		return errs.ErrWSNotConnected
	}
	return wsutil.SendMessage(conn, "key_exchange_request", keyExchReq)
}
