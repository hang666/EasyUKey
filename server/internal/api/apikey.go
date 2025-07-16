package api

import (
	"net/http"
	"strconv"

	"github.com/hang666/EasyUKey/server/internal/model/request"
	"github.com/hang666/EasyUKey/server/internal/model/response"
	"github.com/hang666/EasyUKey/server/internal/service"
	"github.com/hang666/EasyUKey/shared/pkg/errors"
	"github.com/hang666/EasyUKey/shared/pkg/logger"

	"github.com/labstack/echo/v4"
)

// CreateAPIKey 创建API密钥
func CreateAPIKey(c echo.Context) error {
	var req request.CreateAPIKeyRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if req.Name == "" {
		return errors.ErrMissingName
	}

	apiKey, err := service.CreateAPIKey(&req)
	if err != nil {
		logger.Logger.Error("创建API密钥失败", "error", err, "name", req.Name)
		return err
	}

	return c.JSON(http.StatusCreated, &response.Response{Success: true, Message: "API密钥创建成功", Data: apiKey})
}

// GetAPIKeys 获取API密钥列表
func GetAPIKeys(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	apiKeys, total, err := service.GetAPIKeys(page, pageSize)
	if err != nil {
		logger.Logger.Error("获取API密钥列表失败", "error", err)
		return err
	}

	return c.JSON(http.StatusOK, &response.Response{Success: true, Message: "获取API密钥列表成功", Data: &apiKeys, Total: &total})
}
