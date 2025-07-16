package ws

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/hang666/EasyUKey/client/internal/confirmation"
	"github.com/hang666/EasyUKey/client/internal/device"
	"github.com/hang666/EasyUKey/client/internal/global"
	"github.com/hang666/EasyUKey/shared/pkg/errors"
	"github.com/hang666/EasyUKey/shared/pkg/identity"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
	"github.com/hang666/EasyUKey/shared/pkg/messages"
)

// handleAuthRequest 处理来自服务端的认证请求
func handleAuthRequest(message messages.WSMessage) {
	logger.Logger.Info("收到认证请求")

	// 解析认证请求数据
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		logger.Logger.Error("解析认证请求数据失败", "error", err)
		return
	}

	var authReq messages.AuthRequestMessage
	if err := json.Unmarshal(dataBytes, &authReq); err != nil {
		logger.Logger.Error("反序列化认证请求失败", "error", err)
		return
	}

	// 检查设备状态
	dev := device.DeviceInfo.GetDevice()
	if dev == nil {
		SendAuthResponse(authReq.RequestID, false, "", "", "", "", errors.ErrDeviceNotFoundClient.Error())
		return
	}

	// 创建认证请求对象用于显示
	request := &confirmation.AuthRequest{
		ID:        authReq.RequestID,
		UserID:    authReq.UserID,
		Challenge: authReq.Challenge,
		Message:   authReq.Message,
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(time.Duration(authReq.Timeout) * time.Second),
	}

	// 显示确认页面，调用confirmation包
	if err := confirmation.ShowAuthRequest(request); err != nil {
		logger.Logger.Error("显示认证页面失败", "error", err)
		SendAuthResponse(authReq.RequestID, false, "", "", "", "", errors.ErrShowPageFailed.Error())
		return
	}

	// 等待用户确认，调用confirmation包
	timeout := time.Duration(authReq.Timeout) * time.Second
	confirmResult, err := confirmation.WaitForConfirmation(timeout)
	if err != nil {
		logger.Logger.Error("等待用户确认失败", "error", err)
		SendAuthResponse(authReq.RequestID, false, "", "", dev.SerialNumber, dev.VolumeSerialNumber, errors.ErrWaitConfirmFailed.Error())
		return
	}

	if !confirmResult.Confirmed {
		logger.Logger.Info("用户拒绝认证")
		SendAuthResponse(authReq.RequestID, false, "", "", dev.SerialNumber, dev.VolumeSerialNumber, errors.ErrUserRejected.Error())
		return
	}

	// 用户确认，需要等待PIN输入
	logger.Logger.Info("用户确认认证，等待PIN输入")

	// 等待PIN输入
	pin, err := global.PinManager.WaitPIN()
	if err != nil {
		SendAuthResponse(authReq.RequestID, false, "", "", dev.SerialNumber, dev.VolumeSerialNumber, "PIN输入超时")
		return
	}

	// 使用PIN获取当前OnceKey
	currentOnceKey, err := identity.GetOnceKey(pin, global.Config.EncryptKeyStr, global.SecureStoragePath)
	if err != nil {
		SendAuthResponse(authReq.RequestID, false, "", "", dev.SerialNumber, dev.VolumeSerialNumber, "PIN验证失败")
		return
	}

	// 使用PIN生成完整密钥
	fullKey, err := identity.GetFullKey(pin, global.Config.EncryptKeyStr, dev.SerialNumber, dev.VolumeSerialNumber, global.SecureStoragePath)
	if err != nil {
		SendAuthResponse(authReq.RequestID, false, "", "", dev.SerialNumber, dev.VolumeSerialNumber, "密钥生成失败")
		return
	}

	authKey := fmt.Sprintf("%s:_:%s", authReq.Challenge, fullKey)

	// 保存PIN以供后续更新OnceKey使用
	global.PinManager.SendPIN(pin)

	logger.Logger.Info("认证成功，发送认证响应")
	SendAuthResponse(authReq.RequestID, true, authKey, currentOnceKey, dev.SerialNumber, dev.VolumeSerialNumber, "")
}

// handleDeviceInitResponse 处理设备初始化响应
func handleDeviceInitResponse(message messages.WSMessage) {
	logger.Logger.Info("收到设备初始化响应")
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		logger.Logger.Error("序列化设备初始化响应失败", "error", err)
		return
	}
	var resp messages.DeviceInitResponseMessage
	err = json.Unmarshal(dataBytes, &resp)
	if err != nil {
		logger.Logger.Error("反序列化设备初始化响应失败", "error", err)
		return
	}

	if !resp.Success {
		logger.Logger.Error("设备初始化失败", "error", resp.Error, "message", resp.Message)
		return
	}

	// 等待PIN输入
	pin, err := global.PinManager.WaitPIN()
	if err != nil {
		return
	}

	// 使用PIN保存初始密钥
	if err := identity.SaveInitialKeys(pin, global.Config.EncryptKeyStr, resp.OnceKey, resp.TOTPURI, global.SecureStoragePath); err != nil {
		return
	}

	isDeviceInitialized = true
	logger.Logger.Info("设备初始化成功并已保存密钥")
}

// handleAuthSuccessResponse 处理认证成功后服务端返回的新OnceKey
func handleAuthSuccessResponse(message messages.WSMessage) {
	logger.Logger.Info("收到认证成功响应，准备更新OnceKey")
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		logger.Logger.Error("序列化认证成功响应失败", "error", err)
		return
	}
	var resp messages.AuthSuccessResponseMessage
	err = json.Unmarshal(dataBytes, &resp)
	if err != nil {
		logger.Logger.Error("反序列化认证成功响应失败", "error", err)
		return
	}

	if !resp.Success {
		logger.Logger.Error("服务端更新OnceKey失败", "error", resp.Error)
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

	logger.Logger.Info("成功更新OnceKey")
	SendOnceKeyUpdateConfirm(resp.RequestID, true, "")
}

// handlePing 处理心跳请求
func handlePing() {
	logger.Logger.Debug("收到Ping, 发送Pong")
	SendPongMessage()
}

// handleDeviceStatusCheck 处理设备状态检查请求
func handleDeviceStatusCheck() {
	logger.Logger.Info("收到设备状态检查请求")
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
	logger.Logger.Info("收到强制下线消息")

	// 解析强制下线消息数据
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		logger.Logger.Error("解析强制下线消息数据失败", "error", err)
		return
	}

	var forceLogoutMsg messages.ForceLogoutMessage
	if err := json.Unmarshal(dataBytes, &forceLogoutMsg); err != nil {
		logger.Logger.Error("反序列化强制下线消息失败", "error", err)
		return
	}

	logger.Logger.Info("设备被强制下线", "message", forceLogoutMsg.Message)
	os.Exit(0)
}

// handleKeyExchangeResponse 处理密钥交换响应
func handleKeyExchangeResponse(message messages.WSMessage) {
	logger.Logger.Info("收到密钥交换响应")

	// 解析密钥交换响应数据
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		logger.Logger.Error("解析密钥交换响应数据失败", "error", err)
		handshakeStatus = messages.HandshakeStatusFailed
		return
	}

	var keyExchResp messages.KeyExchangeResponseMessage
	if err := json.Unmarshal(dataBytes, &keyExchResp); err != nil {
		logger.Logger.Error("反序列化密钥交换响应失败", "error", err)
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

	logger.Logger.Info("客户端密钥交换完成")
}

// handleEncryptedMessage 处理加密消息
func handleEncryptedMessage(message messages.WSMessage) {
	// 检查握手状态
	if handshakeStatus != messages.HandshakeStatusCompleted {
		logger.Logger.Error("收到加密消息但握手未完成")
		return
	}

	if encryptor == nil {
		logger.Logger.Error("收到加密消息但加密器未初始化")
		return
	}

	// 解析加密消息数据
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		logger.Logger.Error("解析加密消息数据失败", "error", err)
		return
	}

	var encryptedMsg messages.EncryptedMessage
	if err := json.Unmarshal(dataBytes, &encryptedMsg); err != nil {
		logger.Logger.Error("反序列化加密消息失败", "error", err)
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
		logger.Logger.Error("解析解密后的消息失败", "error", err)
		return
	}

	// 递归处理解密后的消息
	dispatchMessage(decryptedMsg)
}
