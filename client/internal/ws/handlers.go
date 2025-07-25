package ws

import (
	"encoding/json"
	"os"
	"time"

	"github.com/hang666/EasyUKey/client/internal/confirmation"
	"github.com/hang666/EasyUKey/client/internal/device"
	"github.com/hang666/EasyUKey/client/internal/global"
	"github.com/hang666/EasyUKey/shared/pkg/auth"
	"github.com/hang666/EasyUKey/shared/pkg/errs"
	"github.com/hang666/EasyUKey/shared/pkg/identity"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
	"github.com/hang666/EasyUKey/shared/pkg/messages"
)

// handleAuthRequest 处理来自服务端的认证请求
func handleAuthRequest(message messages.WSMessage) {
	// 解析认证请求数据
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return
	}

	var authReq messages.AuthRequestMessage
	if err := json.Unmarshal(dataBytes, &authReq); err != nil {
		return
	}

	// 检查设备状态
	dev := device.DeviceInfo.GetDevice()
	if dev == nil {
		SendAuthResponse(authReq.RequestID, false, "", "", "", "", errs.ErrDeviceNotFoundClient.Error())
		return
	}

	// 创建认证请求对象用于显示
	request := &confirmation.AuthRequest{
		ID:        authReq.RequestID,
		UserID:    authReq.Username,
		Challenge: authReq.Challenge,
		Message:   authReq.Message,
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(time.Duration(authReq.Timeout) * time.Second),
	}

	// 显示确认页面，调用confirmation包
	if err := confirmation.ShowAuthRequest(request); err != nil {
		SendAuthResponse(authReq.RequestID, false, "", "", "", "", errs.ErrShowPageFailed.Error())
		return
	}

	// 等待用户确认，调用confirmation包
	timeout := time.Duration(authReq.Timeout) * time.Second
	confirmResult, err := confirmation.WaitForConfirmation(timeout)
	if err != nil {
		confirmation.SendResult(false, "认证超时")
		SendAuthResponse(authReq.RequestID, false, "", "", dev.SerialNumber, dev.VolumeSerialNumber, errs.ErrWaitConfirmFailed.Error())
		return
	}

	if !confirmResult.Confirmed {
		confirmation.SendResult(false, "用户拒绝认证")
		SendAuthResponse(authReq.RequestID, false, "", "", dev.SerialNumber, dev.VolumeSerialNumber, errs.ErrUserRejected.Error())
		return
	}

	// 等待PIN输入
	pin, err := global.PinManager.WaitPIN()
	if err != nil {
		confirmation.SendResult(false, "PIN验证失败")
		SendAuthResponse(authReq.RequestID, false, "", "", dev.SerialNumber, dev.VolumeSerialNumber, "PIN输入超时")
		return
	}

	// 使用PIN获取当前OnceKey
	currentOnceKey, err := identity.GetOnceKey(pin, global.Config.EncryptKeyStr, global.SecureStoragePath)
	if err != nil {
		confirmation.SendResult(false, "PIN验证失败")
		SendAuthResponse(authReq.RequestID, false, "", "", dev.SerialNumber, dev.VolumeSerialNumber, "PIN验证失败")
		return
	}

	// 使用新格式生成认证token
	authKey, err := auth.GenerateAuthToken(
		authReq.Challenge,
		pin,
		global.Config.EncryptKeyStr,
		dev.SerialNumber,
		dev.VolumeSerialNumber,
		global.SecureStoragePath,
	)
	if err != nil {
		confirmation.SendResult(false, "认证token生成失败")
		SendAuthResponse(authReq.RequestID, false, "", "", dev.SerialNumber, dev.VolumeSerialNumber, "认证token生成失败")
		return
	}

	// 保存PIN以供后续更新OnceKey使用
	global.PinManager.SendPIN(pin)

	SendAuthResponse(authReq.RequestID, true, authKey, currentOnceKey, dev.SerialNumber, dev.VolumeSerialNumber, "")

	// 认证响应已发送，等待服务端的 auth_success_response 消息来确定最终结果
}

// handleDeviceInitResponse 处理设备初始化响应
func handleDeviceInitResponse(message messages.WSMessage) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return
	}
	var resp messages.DeviceInitResponseMessage
	err = json.Unmarshal(dataBytes, &resp)
	if err != nil {
		return
	}

	if !resp.Success {
		logger.Logger.Error("设备初始化失败", "error", resp.Error, "message", resp.Message)
		return
	}

	// 等待PIN输入
	pin, err := global.PinManager.WaitPIN()
	if err != nil {
		logger.Logger.Error("PIN获取失败")
		os.Exit(1)
		return
	}

	// 使用PIN保存初始密钥
	if err := identity.SaveInitialKeys(pin, global.Config.EncryptKeyStr, resp.OnceKey, resp.TOTPURI, global.SecureStoragePath); err != nil {
		logger.Logger.Error("保存初始密钥失败")
		os.Exit(1)
		return
	}

	isDeviceInitialized = true
}

// handleAuthSuccessResponse 处理认证成功后服务端返回的新OnceKey
func handleAuthSuccessResponse(message messages.WSMessage) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return
	}
	var resp messages.AuthSuccessResponseMessage
	err = json.Unmarshal(dataBytes, &resp)
	if err != nil {
		return
	}

	if !resp.Success {
		// 服务端认证失败，通知页面
		confirmation.SendResult(false, "服务端认证验证失败")
		SendOnceKeyUpdateConfirm(resp.RequestID, false, "客户端收到错误响应")
		return
	}

	// 等待PIN输入（复用之前保存的PIN）
	pin, err := global.PinManager.WaitPIN()
	if err != nil {
		SendOnceKeyUpdateConfirm(resp.RequestID, false, "PIN获取失败")
		return
	}

	// 使用PIN更新OnceKey
	err = identity.SetOnceKey(pin, global.Config.EncryptKeyStr, resp.NewOnceKey, global.SecureStoragePath)
	if err != nil {
		SendOnceKeyUpdateConfirm(resp.RequestID, false, "保存新Key失败")
		return
	}

	SendOnceKeyUpdateConfirm(resp.RequestID, true, "")

	// 在确认收到服务端成功响应并完成OnceKey更新后，通知页面认证成功
	confirmation.SendResult(true, "认证成功")
}

// handleDeviceConnectionResponse 处理设备连接响应
func handleDeviceConnectionResponse(message messages.WSMessage) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return
	}
	var resp messages.DeviceConnectionResponseMessage
	err = json.Unmarshal(dataBytes, &resp)
	if err != nil {
		return
	}

	if !resp.Success {
		logger.Logger.Error("设备连接失败", "error", resp.Error, "message", resp.Message)
		return
	}

	if resp.Status == "pending_activation" {
		logger.Logger.Info("跨平台设备识别成功，等待管理员激活")
	}
}

// handlePing 处理心跳请求
func handlePing() {
	SendPongMessage()
}

// handleDeviceStatusCheck 处理设备状态检查请求
func handleDeviceStatusCheck() {
	var status string
	var serialNumber string
	var volumeSerialNumber string

	if dev := device.DeviceInfo.GetDevice(); dev != nil && dev.SerialNumber != "" {
		status = "online"
		serialNumber = dev.SerialNumber
		volumeSerialNumber = dev.VolumeSerialNumber
	} else {
		status = "offline"
	}
	SendDeviceStatusResponse(status, serialNumber, volumeSerialNumber)
}

// handleForceLogout 处理强制下线消息
func handleForceLogout(message messages.WSMessage) {
	// 解析强制下线消息数据
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return
	}

	var forceLogoutMsg messages.ForceLogoutMessage
	if err := json.Unmarshal(dataBytes, &forceLogoutMsg); err != nil {
		return
	}

	logger.Logger.Info("设备被强制下线", "message", forceLogoutMsg.Message)
	os.Exit(0)
}

// handleKeyExchangeResponse 处理密钥交换响应
func handleKeyExchangeResponse(message messages.WSMessage) {
	// 解析密钥交换响应数据
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		handshakeStatus = messages.HandshakeStatusFailed
		return
	}

	var keyExchResp messages.KeyExchangeResponseMessage
	if err := json.Unmarshal(dataBytes, &keyExchResp); err != nil {
		handshakeStatus = messages.HandshakeStatusFailed
		return
	}

	if !keyExchResp.Success {
		logger.Logger.Error("服务端密钥交换失败", "error", keyExchResp.Error)
		handshakeStatus = messages.HandshakeStatusFailed
		return
	}

	// 计算共享密钥
	if err := keyExchange.ComputeSharedKey(keyExchResp.PublicKey); err != nil {
		logger.Logger.Error("计算共享密钥失败", "error", err)
		handshakeStatus = messages.HandshakeStatusFailed
		return
	}

	// 创建加密器
	enc, err := keyExchange.CreateEncryptor()
	if err != nil {
		logger.Logger.Error("创建加密器失败", "error", err)
		handshakeStatus = messages.HandshakeStatusFailed
		return
	}

	// 更新全局状态
	encryptor = enc
	handshakeStatus = messages.HandshakeStatusCompleted
}

// handleEncryptedMessage 处理加密消息
func handleEncryptedMessage(message messages.WSMessage) {
	// 检查握手状态
	if handshakeStatus != messages.HandshakeStatusCompleted {
		return
	}

	if encryptor == nil {
		return
	}

	// 解析加密消息数据
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return
	}

	var encryptedMsg messages.EncryptedMessage
	if err := json.Unmarshal(dataBytes, &encryptedMsg); err != nil {
		return
	}

	// 解密消息
	decryptedData, err := encryptor.DecryptMessage(encryptedMsg.Payload, encryptedMsg.Nonce)
	if err != nil {
		logger.Logger.Error("解密消息失败", "error", err)
		return
	}

	// 解析解密后的消息
	var decryptedMsg messages.WSMessage
	if err := json.Unmarshal(decryptedData, &decryptedMsg); err != nil {
		return
	}

	// 递归处理解密后的消息
	dispatchMessage(decryptedMsg)
}
