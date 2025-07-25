package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hang666/EasyUKey/sdk/consts"
	"github.com/hang666/EasyUKey/sdk/request"
	"github.com/hang666/EasyUKey/sdk/response"
	"github.com/hang666/EasyUKey/server/internal/model/entity"
	"github.com/hang666/EasyUKey/server/internal/service"
	"github.com/hang666/EasyUKey/shared/pkg/errs"
)

// StartAuth 发起用户认证
func StartAuth(c echo.Context) error {
	var req request.AuthRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if req.Username == "" {
		return errs.ErrMissingUsername
	}
	if req.Challenge == "" {
		return errs.ErrMissingChallenge
	}

	// 从上下文获取API密钥信息
	apiKey := c.Get("api_key").(*entity.APIKey)

	// 获取客户端IP地址
	clientIP := c.RealIP()

	session, err := service.StartAuth(&req, apiKey, clientIP)
	if err != nil {
		return err
	}

	authData := &response.AuthData{
		SessionID: session.ID,
		Status:    session.Status,
		ExpiresAt: session.ExpiresAt,
	}

	return c.JSON(http.StatusOK, &response.Response{
		Success: true,
		Message: "认证请求已发起",
		Data:    authData,
	})
}

// VerifyAuth 验证认证结果
func VerifyAuth(c echo.Context) error {
	var req request.VerifyAuthRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if req.SessionID == "" {
		return errs.ErrMissingSessionID
	}

	session, err := service.VerifyAuth(&req)
	if err != nil {
		return err
	}

	verifyData := &response.VerifyAuthData{
		Status:   session.Status,
		Result:   session.Result,
		UserID:   session.UserID,
		Username: "",
		Message:  getStatusMessage(session.Status, session.Result),
	}

	// 如果用户信息已加载，则填充Username
	if session.User != nil {
		verifyData.Username = session.User.Username
	}

	return c.JSON(http.StatusOK, &response.Response{
		Success: true,
		Message: "验证查询成功",
		Data:    verifyData,
	})
}

// getStatusMessage 根据认证状态和结果生成相应的消息
func getStatusMessage(status, result string) string {
	switch status {
	case consts.AuthStatusPending:
		return "等待用户确认认证"
	case consts.AuthStatusProcessing:
		return "认证处理中"
	case consts.AuthStatusProcessingOnceKey:
		return "正在更新密钥"
	case consts.AuthStatusCompleted:
		if result == consts.AuthResultSuccess {
			return "认证成功"
		}
		return "认证失败"
	case consts.AuthStatusFailed:
		return "认证处理失败"
	case consts.AuthStatusExpired:
		return "认证请求已过期"
	case consts.AuthStatusRejected:
		return "用户拒绝认证"
	default:
		return "未知状态"
	}
}
