package api

import (
	"net/http"

	"github.com/hang666/EasyUKey/server/internal/model/request"
	"github.com/hang666/EasyUKey/server/internal/model/response"
	"github.com/hang666/EasyUKey/server/internal/service"
	"github.com/hang666/EasyUKey/shared/pkg/errors"
	"github.com/hang666/EasyUKey/shared/pkg/logger"

	"github.com/labstack/echo/v4"
)

// StartAuth 发起用户认证
func StartAuth(c echo.Context) error {
	username := c.Param("username")

	var req request.AuthRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	req.UserID = username

	if req.Challenge == "" {
		return errors.ErrMissingChallenge
	}

	session, err := service.StartAuth(&req)
	if err != nil {
		logger.Logger.Error("发起认证失败", "error", err, "username", username)
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
		return errors.ErrMissingSessionID
	}

	session, err := service.VerifyAuth(&req)
	if err != nil {
		logger.Logger.Error("验证认证失败", "error", err, "session_id", req.SessionID)
		return err
	}

	verifyData := &response.VerifyAuthData{
		Success: session.Status == "completed",
		UserID:  session.UserID,
		Message: "验证成功",
	}

	return c.JSON(http.StatusOK, &response.Response{
		Success: true,
		Message: "验证成功",
		Data:    verifyData,
	})
}
