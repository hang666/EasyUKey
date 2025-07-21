package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/hang666/EasyUKey/sdk/response"
	"github.com/hang666/EasyUKey/server/internal/service"
)

// GetAuthSessions 获取认证会话列表
func GetAuthSessions(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 获取过滤参数
	status := c.QueryParam("status")
	username := c.QueryParam("username")

	sessions, total, err := service.GetAuthSessions(page, pageSize, status, username)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &response.Response{
		Success: true,
		Message: "获取认证会话列表成功",
		Data:    &sessions,
		Total:   &total,
	})
}
