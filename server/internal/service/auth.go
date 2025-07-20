package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hang666/EasyUKey/sdk/request"
	"github.com/hang666/EasyUKey/server/internal/global"
	"github.com/hang666/EasyUKey/server/internal/model/entity"
	"github.com/hang666/EasyUKey/shared/pkg/callback"
	"github.com/hang666/EasyUKey/shared/pkg/errs"
	"github.com/hang666/EasyUKey/shared/pkg/identity"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
	"github.com/hang666/EasyUKey/shared/pkg/messages"
)

const (
	CallbackTimeout    = 10 * time.Second // 回调超时时间
	CallbackMaxRetries = 3                // 最大重试次数
)

// ValidateAPIKey 验证API密钥
func ValidateAPIKey(apiKey string) (*entity.APIKey, error) {
	var key entity.APIKey
	result := global.DB.Where("api_key = ? AND is_active = ?", apiKey, true).First(&key)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errs.ErrAPIKeyInvalid
		}
		return nil, fmt.Errorf("查询API密钥失败: %w", result.Error)
	}

	// 检查过期时间
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return nil, errs.ErrAPIKeyInvalid
	}

	return &key, nil
}

// ValidateAuthKey 验证认证密钥
func ValidateAuthKey(authKey string, deviceID uint, challenge string) (*entity.Device, error) {
	// 查找设备信息
	var device entity.Device
	result := global.DB.Where("id = ?", deviceID).First(&device)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errs.ErrDeviceNotFound
		}
		return nil, fmt.Errorf("查询设备失败: %w", result.Error)
	}

	// 检查设备是否激活
	if !device.IsActive {
		return nil, errs.ErrDeviceNotActive
	}

	// 解析认证密钥格式: {challenge}:_:{onceKey}:_:{totpCode}:_:{serialNumber}:_:{volumeSerialNumber}
	parts := strings.Split(authKey, ":_:")
	if len(parts) != 5 {
		return nil, errs.ErrAuthInvalidKey
	}

	receivedChallenge := parts[0]
	receivedOnceKey := parts[1]
	receivedTOTPCode := parts[2]
	receivedSerialNumber := parts[3]
	receivedVolumeSerial := parts[4]

	// 验证挑战码
	if receivedChallenge != challenge {
		return nil, errs.ErrAuthChallengeInvalid
	}

	// 验证设备序列号
	if receivedSerialNumber != device.SerialNumber || receivedVolumeSerial != device.VolumeSerialNumber {
		return nil, errs.ErrAuthSerialMismatch
	}

	// 验证OnceKey
	if receivedOnceKey != device.OnceKey {
		return nil, errs.ErrAuthOnceKeyMismatch
	}

	// 验证TOTP代码
	totpConfig, err := identity.ParseTOTPURI(device.TOTPSecret)
	if err != nil {
		return nil, fmt.Errorf("解析TOTP密钥失败: %w", err)
	}

	isValidTOTP, err := identity.VerifyTOTPCode(totpConfig, receivedTOTPCode, time.Now())
	if err != nil {
		return nil, fmt.Errorf("验证TOTP验证码失败: %w", err)
	}

	if !isValidTOTP {
		return nil, errs.ErrAuthTOTPInvalid
	}

	return &device, nil
}

// ProcessAuthResponse 处理认证响应
func ProcessAuthResponse(sessionID string, authResp *messages.AuthResponseMessage) error {
	// 查找认证会话
	var session entity.AuthSession
	result := global.DB.Where("id = ?", sessionID).First(&session)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return errs.ErrSessionNotFound
		}
		return fmt.Errorf("查询认证会话失败: %w", result.Error)
	}

	// 检查是否过期
	if session.ExpiresAt.Before(time.Now()) {
		// 更新会话状态为过期
		global.DB.Model(&session).Updates(map[string]interface{}{
			"status": entity.AuthStatusExpired,
		})
		return errs.ErrSessionExpired
	}

	// 原子性更新会话状态为处理中，防止重复处理
	updateResult := global.DB.Model(&session).
		Where("id = ? AND status = ?", sessionID, entity.AuthStatusPending).
		Update("status", entity.AuthStatusProcessing)
	if updateResult.Error != nil {
		return fmt.Errorf("更新会话状态失败: %w", updateResult.Error)
	}
	if updateResult.RowsAffected == 0 {
		return fmt.Errorf("认证会话已被处理或状态无效: %s", session.Status)
	}

	// 首先验证设备和密钥（无论成功还是失败都需要验证）
	var device entity.Device
	deviceResult := global.DB.Where("serial_number = ? AND volume_serial_number = ?",
		authResp.SerialNumber, authResp.VolumeSerialNumber).First(&device)

	if deviceResult.Error != nil {
		logger.Logger.Error("认证响应中的设备未找到", "session_id", sessionID, "serial_number", authResp.SerialNumber)
		return fmt.Errorf("设备未找到")
	}

	// 验证auth_key
	validDevice, err := ValidateAuthKey(authResp.AuthKey, device.ID, session.Challenge)
	if err != nil {
		logger.Logger.Error("认证密钥验证失败", "session_id", sessionID, "error", err.Error())

		// 在密钥验证失败时也记录失败状态
		updates := map[string]interface{}{
			"responding_device_id": &device.ID,
			"status":               entity.AuthStatusFailed,
			"result":               entity.AuthResultFailure,
		}
		global.DB.Model(&session).Updates(updates) // 尝试更新，忽略错误

		return fmt.Errorf("认证密钥验证失败: %w", err)
	}

	// 验证设备是否具有执行此操作的权限
	if session.Action != "" {
		canPerformAction := false
		for _, p := range validDevice.Permissions {
			if p == session.Action || p == "*" {
				canPerformAction = true
				break
			}
		}

		if !canPerformAction {
			logger.Logger.Warn("设备权限不足", "session_id", sessionID, "required_action", session.Action)

			// 更新会话状态为失败并返回错误
			updates := map[string]interface{}{
				"responding_device_id": &validDevice.ID,
				"status":               entity.AuthStatusFailed,
				"result":               entity.AuthResultFailure,
			}
			if err := global.DB.Model(&session).Updates(updates).Error; err != nil {
				return fmt.Errorf("更新认证会话失败: %w", err)
			}
			return fmt.Errorf("设备权限与请求的操作不匹配")
		}
	}

	// 初始化更新数据
	updates := map[string]interface{}{
		"responding_device_id": &validDevice.ID,
	}

	if authResp.Success {
		updates["status"] = entity.AuthStatusProcessingOnceKey
		logger.Logger.Info("用户同意认证，开始处理OnceKey更新", "session_id", sessionID, "device_id", validDevice.ID)
	} else {
		// 认证失败，区分用户拒绝和其他失败情况
		updates["result"] = entity.AuthResultFailure

		if authResp.Error == errs.ErrUserRejected.Error() {
			updates["status"] = entity.AuthStatusRejected
			logger.Logger.Info("认证拒绝", "session_id", sessionID, "device_id", validDevice.ID)
		} else {
			updates["status"] = entity.AuthStatusFailed
			logger.Logger.Info("认证失败", "session_id", sessionID, "error", authResp.Error)
		}
	}

	// 更新数据库
	if err := global.DB.Model(&session).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新认证会话失败: %w", err)
	}

	return nil
}

// StartAuth 发起用户认证
func StartAuth(req *request.AuthRequest, apiKey *entity.APIKey) (*entity.AuthSession, error) {
	// 查找用户
	var user entity.User
	result := global.DB.Where("username = ? AND is_active = ?", req.Username, true).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errs.ErrUserNotFound
		}
		return nil, fmt.Errorf("查询用户失败: %w", result.Error)
	}

	// 查找用户所有激活的在线设备
	var onlineDevices []entity.Device
	err := global.DB.Where("user_id = ? AND is_active = ? AND is_online = ?", user.ID, true, true).Find(&onlineDevices).Error
	if err != nil {
		return nil, fmt.Errorf("查询在线设备失败: %w", err)
	}

	if len(onlineDevices) == 0 {
		return nil, errs.ErrUserNotOnline
	}

	// 如果请求指定了action，检查是否有设备具备相应权限
	if req.Action != "" {
		canPerformAction := false
		for _, device := range onlineDevices {
			for _, p := range device.Permissions {
				if p == req.Action || p == "*" {
					canPerformAction = true
					break
				}
			}
			if canPerformAction {
				break
			}
		}
		if !canPerformAction {
			return nil, errs.ErrPermissionDenied
		}
	}

	// 生成会话ID
	sessionID := uuid.New().String()

	// 设置超时时间
	timeout := time.Duration(req.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute // 默认5分钟
	}
	expiresAt := time.Now().Add(timeout)

	// 创建认证会话
	session := entity.AuthSession{
		ID:          sessionID,
		UserID:      user.ID,
		APIKeyID:    apiKey.ID,
		Challenge:   req.Challenge,
		Action:      req.Action,
		Status:      entity.AuthStatusPending,
		ExpiresAt:   expiresAt,
		CallbackURL: req.CallbackURL,
	}

	if err := global.DB.Create(&session).Error; err != nil {
		return nil, fmt.Errorf("创建认证会话失败: %w", err)
	}

	// 发送WebSocket消息给用户
	authMsg := messages.AuthRequestMessage{
		RequestID: sessionID,
		Username:  req.Username,
		Challenge: req.Challenge,
		Action:    req.Action,
		Message:   req.Message,
		Timeout:   req.Timeout,
	}

	msgData, err := SendWSMessage("auth_request", authMsg)
	if err != nil {
		logger.Logger.Error("序列化认证消息失败", "error", err)
		return nil, fmt.Errorf("发送认证请求失败")
	}

	if hub := GetWSHub(); hub != nil {
		if err := hub.SendToUser(user.ID, msgData); err != nil {
			logger.Logger.Error("发送WebSocket消息失败", "error", err, "user_id", user.ID)
			return nil, fmt.Errorf("发送认证请求失败")
		}
	}

	return &session, nil
}

// sendAuthCallback 发送认证回调
func sendAuthCallback(session *entity.AuthSession, serialNumber string) {
	// 查找对应的API密钥作为签名密钥
	apiKey, err := getAPIKeyBySession(session)
	if err != nil {
		logger.Logger.Error("获取API密钥失败", "session_id", session.ID, "error", err)
		return
	}

	// 构建回调请求
	callbackReq := &messages.CallbackRequest{
		SessionID: session.ID,
		Username:  fmt.Sprintf("%d", session.UserID),
		Challenge: session.Challenge,
		Action:    session.Action,
		Timestamp: time.Now().Unix(),
	}

	// 设置状态
	if session.Status == entity.AuthStatusCompleted && session.Result == entity.AuthResultSuccess {
		callbackReq.Status = "success"
	} else {
		callbackReq.Status = "failed"
	}

	// 设置设备ID
	if session.RespondingDeviceID != nil {
		callbackReq.DeviceID = *session.RespondingDeviceID
	}

	// 生成签名
	callbackReq.Signature = callback.GenerateSignature(callbackReq, apiKey)

	// 发送回调，最多重试3次
	for i := 0; i < CallbackMaxRetries; i++ {
		if sendHTTPCallback(session.CallbackURL, callbackReq) {
			return
		}

		if i < CallbackMaxRetries-1 {
			// 递增重试间隔: 5s, 10s, 30s
			delays := []time.Duration{5 * time.Second, 10 * time.Second, 30 * time.Second}
			time.Sleep(delays[i])
		}
	}

	logger.Logger.Error("回调失败，已达到最大重试次数", "session_id", session.ID, "url", session.CallbackURL)
}

// sendHTTPCallback 发送HTTP回调请求
func sendHTTPCallback(url string, req *messages.CallbackRequest) bool {
	data, err := json.Marshal(req)
	if err != nil {
		logger.Logger.Error("序列化回调请求失败", "error", err)
		return false
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		logger.Logger.Error("创建HTTP请求失败", "url", url, "error", err)
		return false
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "EasyUKey-Callback/1.0")

	client := &http.Client{Timeout: CallbackTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		logger.Logger.Error("发送回调请求失败", "url", url, "error", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true
	}

	logger.Logger.Error("回调请求失败", "url", url, "status_code", resp.StatusCode)
	return false
}

// CompleteOnceKeyUpdateAuth 完成OnceKey更新后的认证
func CompleteOnceKeyUpdateAuth(requestID string, success bool, errorMessage string) error {
	// 查找正在处理OnceKey的认证会话
	var session entity.AuthSession
	result := global.DB.Where("id = ? AND status = ?", requestID, entity.AuthStatusProcessingOnceKey).First(&session)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			logger.Logger.Warn("未找到正在处理OnceKey的认证会话", "request_id", requestID)
			return nil // 不返回错误，避免影响其他流程
		}
		return fmt.Errorf("查询认证会话失败: %w", result.Error)
	}

	updates := map[string]interface{}{}

	if success {
		// OnceKey更新确认成功
		updates["status"] = entity.AuthStatusCompleted
		updates["result"] = entity.AuthResultSuccess
		logger.Logger.Info("OnceKey更新确认成功，认证完成", "session_id", requestID)
	} else {
		// OnceKey更新确认失败，认证失败
		updates["status"] = entity.AuthStatusFailed
		updates["result"] = entity.AuthResultFailure
		logger.Logger.Error("OnceKey更新确认失败，认证失败", "session_id", requestID, "error", errorMessage)
	}

	// 更新认证会话状态
	if err := global.DB.Model(&session).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新认证会话状态失败: %w", err)
	}

	if success && session.CallbackURL != "" {
		go sendAuthCallback(&session, "")
	}

	return nil
}

// VerifyAuth 验证认证结果
func VerifyAuth(req *request.VerifyAuthRequest) (*entity.AuthSession, error) {
	// 查找认证会话并预加载用户信息
	var session entity.AuthSession
	result := global.DB.Preload("User").Where("id = ?", req.SessionID).First(&session)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errs.ErrSessionNotFound
		}
		return nil, fmt.Errorf("查询认证会话失败: %w", result.Error)
	}

	// 检查会话状态
	if session.ExpiresAt.Before(time.Now()) {
		return nil, errs.ErrSessionExpired
	}

	return &session, nil
}

// getAPIKeyBySession 通过会话获取API密钥
func getAPIKeyBySession(session *entity.AuthSession) (string, error) {
	var apiKey entity.APIKey
	err := global.DB.Where("id = ?", session.APIKeyID).First(&apiKey).Error
	if err != nil {
		return "", fmt.Errorf("查找API密钥失败: %w", err)
	}
	return apiKey.APIKey, nil
}
