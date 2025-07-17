package api

import (
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/hang666/EasyUKey/shared/pkg/errors"
)

// bindAndValidate 绑定并验证请求参数
func bindAndValidate(c echo.Context, req interface{}) error {
	if err := c.Bind(req); err != nil {
		return errors.ErrInvalidRequest
	}
	return nil
}

// parseUintParam 解析uint参数
func parseUintParam(c echo.Context, paramName string) (uint, error) {
	param := c.Param(paramName)
	if param == "" {
		return 0, errors.ErrInvalidDeviceID
	}

	value, err := strconv.ParseUint(param, 10, 32)
	if err != nil {
		return 0, errors.ErrInvalidDeviceID
	}

	return uint(value), nil
}
