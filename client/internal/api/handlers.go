package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hang666/EasyUKey/client/internal/confirmation"
	"github.com/hang666/EasyUKey/client/internal/global"
	"github.com/hang666/EasyUKey/client/internal/pin"
	"github.com/hang666/EasyUKey/shared/pkg/identity"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
)

// decodeAuthRequest 解码认证请求的通用函数
func decodeAuthRequest(encodedRequest string) (*confirmation.AuthRequest, error) {
	if encodedRequest == "" {
		return nil, fmt.Errorf("缺少认证请求参数")
	}

	// Base64 解码
	jsonData, err := base64.URLEncoding.DecodeString(encodedRequest)
	if err != nil {
		return nil, fmt.Errorf("无效的认证请求格式")
	}

	// JSON 解码
	var request confirmation.AuthRequest
	if err := json.Unmarshal(jsonData, &request); err != nil {
		return nil, fmt.Errorf("无法解析认证请求")
	}

	// 检查是否过期
	if time.Now().After(request.ExpiresAt) {
		return nil, fmt.Errorf("认证请求已过期")
	}

	return &request, nil
}

// HandleConfirmPage 处理确认页面
func HandleConfirmPage(c echo.Context) error {
	encodedRequest := c.QueryParam("request")
	request, err := decodeAuthRequest(encodedRequest)
	if err != nil {
		logger.Logger.Error("解码认证请求失败", "error", err, "request", encodedRequest)

		// 根据错误类型返回不同的错误页面
		switch err.Error() {
		case "缺少认证请求参数":
			return renderErrorPage(c, http.StatusBadRequest, "请求无效", "缺少认证请求参数。")
		case "无效的认证请求格式":
			return renderErrorPage(c, http.StatusBadRequest, "请求无效", "认证请求的格式不正确。")
		case "无法解析认证请求":
			return renderErrorPage(c, http.StatusBadRequest, "请求无效", "无法解析认证请求。")
		case "认证请求已过期":
			return renderErrorPage(c, http.StatusGone, "请求已过期", "此认证请求已过期，请重新发起。")
		default:
			return renderErrorPage(c, http.StatusBadRequest, "请求无效", "认证请求处理失败。")
		}
	}

	// 检查当前认证状态，防止重复打开页面
	currentState, currentReqID := confirmation.GetCurrentState()
	if currentReqID == request.ID {
		switch currentState {
		case confirmation.StateProcessing:
			return renderErrorPage(c, http.StatusConflict, "认证进行中", "认证正在处理中，请稍候...")
		case confirmation.StateCompleted:
			return renderErrorPage(c, http.StatusConflict, "认证已完成", "认证已完成，请勿重复提交。")
		}
	} else if currentState != confirmation.StateIdle && currentState != confirmation.StateWaiting {
		return renderErrorPage(c, http.StatusConflict, "认证冲突", "当前有其他认证请求正在处理，请稍后再试。")
	}

	data := map[string]interface{}{
		"Request":    *request,
		"RawRequest": encodedRequest,
		"Remaining":  int64(time.Until(request.ExpiresAt).Seconds()),
	}

	return c.Render(http.StatusOK, "auth.html", data)
}

// HandleConfirmAction 处理确认操作
func HandleConfirmAction(c echo.Context) error {
	var payload ConfirmActionPayload
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, ConfirmActionResponse{
			Message:       "无效的请求体",
			Status:        ConfirmActionStatusError,
			ConfirmStatus: false,
		})
	}

	request, err := decodeAuthRequest(payload.Request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ConfirmActionResponse{
			Message:       err.Error(),
			Status:        ConfirmActionStatusError,
			ConfirmStatus: false,
		})
	}

	// 检查当前认证状态，防止重复提交
	currentState, currentReqID := confirmation.GetCurrentState()
	if currentReqID == request.ID {
		switch currentState {
		case confirmation.StateProcessing:
			return c.JSON(http.StatusOK, ConfirmActionResponse{
				Message:       "认证正在处理中，请稍候...",
				Status:        ConfirmActionStatusError,
				ConfirmStatus: false,
			})
		case confirmation.StateCompleted:
			return c.JSON(http.StatusOK, ConfirmActionResponse{
				Message:       "认证已完成，请勿重复提交",
				Status:        ConfirmActionStatusError,
				ConfirmStatus: false,
			})
		}
	} else if currentState != confirmation.StateIdle && currentState != confirmation.StateWaiting {
		return c.JSON(http.StatusOK, ConfirmActionResponse{
			Message:       "当前有其他认证请求正在处理",
			Status:        ConfirmActionStatusError,
			ConfirmStatus: false,
		})
	}

	action := payload.Action
	confirmed := action == "confirm"

	// 如果用户拒绝，立即返回
	if !confirmed {
		confirmResult := confirmation.AuthConfirmation{
			RequestID: request.ID,
			Confirmed: false,
			Timestamp: time.Now(),
		}
		confirmation.SendConfirmation(confirmResult)

		return c.JSON(http.StatusOK, ConfirmActionResponse{
			Message:       "认证已拒绝",
			Status:        ConfirmActionStatusSuccess,
			ConfirmStatus: false,
		})
	}

	// 如果是确认认证且提供了PIN，需要进行PIN验证
	if payload.PIN != "" {
		if err := pin.ValidatePIN(payload.PIN); err != nil {
			return c.JSON(http.StatusBadRequest, ConfirmActionResponse{
				Message:       "PIN格式错误",
				Status:        ConfirmActionStatusError,
				ConfirmStatus: false,
			})
		}

		// 将PIN发送到PIN管理器，供认证流程使用
		if global.PinManager != nil {
			global.PinManager.SendPIN(payload.PIN)
		}
	}

	confirmResult := confirmation.AuthConfirmation{
		RequestID: request.ID,
		Confirmed: confirmed,
		Timestamp: time.Now(),
	}

	// 发送确认结果
	confirmation.SendConfirmation(confirmResult)

	// 等待WebSocket认证结果
	result, err := confirmation.WaitForResult(60 * time.Second)
	if err != nil {
		logger.Logger.Error("等待认证结果超时", "requestID", request.ID, "error", err)
		return c.JSON(http.StatusOK, ConfirmActionResponse{
			Message:       "认证超时",
			Status:        ConfirmActionStatusError,
			ConfirmStatus: false,
		})
	}

	// 返回真正的认证结果
	status := ConfirmActionStatusSuccess
	if !result.Success {
		status = ConfirmActionStatusError
	}

	return c.JSON(http.StatusOK, ConfirmActionResponse{
		Message:       result.Message,
		Status:        status,
		ConfirmStatus: result.Success,
	})
}

// renderErrorPage 渲染错误页面
func renderErrorPage(c echo.Context, statusCode int, title, message string) error {
	data := map[string]string{
		"Title":   title,
		"Message": message,
	}
	_ = c.Render(statusCode, "error.html", data)
	return nil
}

// HandlePINPage 处理PIN设置页面
func HandlePINPage(c echo.Context) error {
	// 检查设备是否已初始化
	isInitialized := identity.IsInitialized(global.SecureStoragePath)

	data := map[string]interface{}{
		"IsInitialized": isInitialized,
	}

	return c.Render(http.StatusOK, "pin.html", data)
}

// HandlePINSetup 处理PIN设置请求
func HandlePINSetup(c echo.Context) error {
	var payload PINSetupPayload
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, PINSetupResponse{
			Message: "无效的请求体",
			Status:  ConfirmActionStatusError,
		})
	}

	// 验证PIN格式
	if err := pin.ValidatePIN(payload.PIN); err != nil {
		return c.JSON(http.StatusBadRequest, PINSetupResponse{
			Message: "PIN格式错误",
			Status:  ConfirmActionStatusError,
		})
	}

	// 检查设备是否已初始化
	isInitialized := identity.IsInitialized(global.SecureStoragePath)

	if isInitialized {
		// 设备已初始化，验证PIN是否正确
		_, err := identity.GetTOTPSecret(payload.PIN, global.Config.EncryptKeyStr, global.SecureStoragePath)
		if err != nil {
			go func() {
				time.Sleep(3 * time.Second)
				os.Exit(1)
			}()
			return c.JSON(http.StatusBadRequest, PINSetupResponse{
				Message: "PIN验证失败，请检查PIN是否正确",
				Status:  ConfirmActionStatusError,
			})
		}

		// PIN验证成功，发送到PIN管理器
		if global.PinManager != nil {
			global.PinManager.SendPIN(payload.PIN)
		}

		return c.JSON(http.StatusOK, PINSetupResponse{
			Message: "PIN验证成功，正在连接服务器...",
			Status:  ConfirmActionStatusSuccess,
		})
	}

	// 设备未初始化，设置PIN用于初始化
	if global.PinManager != nil {
		global.PinManager.SendPIN(payload.PIN)
	}

	return c.JSON(http.StatusOK, PINSetupResponse{
		Message: "PIN设置成功，正在初始化设备...",
		Status:  ConfirmActionStatusSuccess,
	})
}
