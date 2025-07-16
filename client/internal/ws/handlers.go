package ws

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/hang666/EasyUKey/client/internal/confirmation"
	"github.com/hang666/EasyUKey/client/internal/device"
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

	currentOnceKey, err := identity.GetOnceKeySecure()
	if err != nil {
		logger.Logger.Error("获取当前 OnceKey 失败", "error", err)
		SendAuthResponse(authReq.RequestID, false, "", "", dev.SerialNumber, dev.VolumeSerialNumber, errors.ErrGetOnceKeyFailed.Error())
		return
	}

	fullKey, err := identity.GetFullKeySecure(dev.SerialNumber, dev.VolumeSerialNumber)
	if err != nil {
		logger.Logger.Error("获取完整密钥失败", "error", err)
		SendAuthResponse(authReq.RequestID, false, "", "", dev.SerialNumber, dev.VolumeSerialNumber, errors.ErrGetFullKeyFailed.Error())
		return
	}
	authKey := fmt.Sprintf("%s:_:%s", authReq.Challenge, fullKey)

	if !confirmResult.Confirmed {
		logger.Logger.Info("用户拒绝认证")
		SendAuthResponse(authReq.RequestID, false, authKey, currentOnceKey, dev.SerialNumber, dev.VolumeSerialNumber, errors.ErrUserRejected.Error())
		return
	}

	// 用户确认，执行认证
	logger.Logger.Info("用户确认认证")

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
		// 可能需要退出或采取其他措施
		return
	}

	// 保存OnceKey和TOTP
	if err := identity.SaveInitialKeys(resp.OnceKey, resp.TOTPURI); err != nil {
		logger.Logger.Error("保存初始密钥失败", "error", err)
		// 通知服务端保存失败
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

	// 更新OnceKey
	err = identity.SetOnceKeySecure(resp.NewOnceKey)
	if err != nil {
		logger.Logger.Error("更新OnceKey失败", "error", err)
		SendOnceKeyUpdateConfirm(resp.RequestID, false, "客户端保存新Key失败: "+err.Error())
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
