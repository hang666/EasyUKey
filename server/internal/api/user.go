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

// CreateUser 创建用户
func CreateUser(c echo.Context) error {
	var req request.CreateUserRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if req.Username == "" {
		return errors.ErrMissingUsername
	}

	user, err := service.CreateUser(&req)
	if err != nil {
		logger.Logger.Error("创建用户失败", "error", err, "username", req.Username)
		return err
	}

	return c.JSON(http.StatusCreated, &response.Response{
		Success: true,
		Message: "用户创建成功",
		Data:    user,
	})
}

// GetUser 获取单个用户
func GetUser(c echo.Context) error {
	userID, err := parseUintParam(c, "id")
	if err != nil {
		return err
	}

	user, err := service.GetUser(userID)
	if err != nil {
		logger.Logger.Error("获取用户失败", "error", err, "user_id", userID)
		return err
	}

	return c.JSON(http.StatusOK, &response.Response{
		Success: true,
		Message: "获取用户成功",
		Data:    user,
	})
}

// GetUsers 获取用户列表
func GetUsers(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	users, total, err := service.GetUsers(page, pageSize)
	if err != nil {
		logger.Logger.Error("获取用户列表失败", "error", err)
		return err
	}

	return c.JSON(http.StatusOK, &response.Response{
		Success: true,
		Message: "获取用户列表成功",
		Data:    users,
		Total:   &total,
	})
}

// UpdateUser 更新用户
func UpdateUser(c echo.Context) error {
	userID, err := parseUintParam(c, "id")
	if err != nil {
		return err
	}

	var req request.UpdateUserRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	user, err := service.UpdateUser(userID, &req)
	if err != nil {
		logger.Logger.Error("更新用户失败", "error", err, "user_id", userID)
		return err
	}

	return c.JSON(http.StatusOK, &response.Response{Success: true, Message: "用户更新成功", Data: user})
}

// DeleteUser 删除用户
func DeleteUser(c echo.Context) error {
	userID, err := parseUintParam(c, "id")
	if err != nil {
		return err
	}

	if err := service.DeleteUser(userID); err != nil {
		logger.Logger.Error("删除用户失败", "error", err, "user_id", userID)
		return err
	}

	result := map[string]string{"message": "用户删除成功"}
	return c.JSON(http.StatusOK, &response.Response{Success: true, Message: "用户删除成功", Data: &result})
}

// GetUserDevices 获取用户的设备列表
func GetUserDevices(c echo.Context) error {
	username := c.Param("username")

	devices, err := service.GetUserDevices(username)
	if err != nil {
		logger.Logger.Error("获取用户设备列表失败", "error", err, "username", username)
		return err
	}

	return c.JSON(http.StatusOK, &response.Response{Success: true, Message: "获取用户设备列表成功", Data: &devices})
}
