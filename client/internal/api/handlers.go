package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hang666/EasyUKey/client/internal/confirmation"
	"github.com/hang666/EasyUKey/shared/pkg/logger"

	"github.com/labstack/echo/v4"
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

	action := payload.Action
	confirmed := action == "confirm"

	confirmResult := confirmation.AuthConfirmation{
		RequestID: request.ID,
		Confirmed: confirmed,
		Timestamp: time.Now(),
	}

	// 发送确认结果
	confirmation.SendConfirmation(confirmResult)
	logger.Logger.Info("用户操作已通过Web Handler转发", "action", action)

	return c.JSON(http.StatusOK, ConfirmActionResponse{
		Message:       "操作已处理",
		Status:        ConfirmActionStatusSuccess,
		ConfirmStatus: confirmed,
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
