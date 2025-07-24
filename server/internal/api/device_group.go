package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hang666/EasyUKey/sdk/request"
	"github.com/hang666/EasyUKey/sdk/response"
	"github.com/hang666/EasyUKey/server/internal/service"
)

// GetDeviceGroup 获取设备组详情
func GetDeviceGroup(c echo.Context) error {
	groupID, err := parseUintParam(c, "id")
	if err != nil {
		return err
	}

	deviceGroup, err := service.GetDeviceGroup(groupID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &response.Response{
		Success: true,
		Data:    deviceGroup,
	})
}

// GetDeviceGroups 获取设备组列表
func GetDeviceGroups(c echo.Context) error {
	deviceGroups, err := service.GetDeviceGroups()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &response.Response{
		Success: true,
		Data:    deviceGroups,
	})
}

// UpdateDeviceGroup 更新设备组
func UpdateDeviceGroup(c echo.Context) error {
	groupID, err := parseUintParam(c, "id")
	if err != nil {
		return err
	}

	var req request.UpdateDeviceGroupRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	deviceGroup, err := service.UpdateDeviceGroup(groupID, req.Name, req.Description, req.Permissions, req.IsActive)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &response.Response{
		Success: true,
		Message: "设备组更新成功",
		Data:    deviceGroup,
	})
}

// LinkDeviceGroupUser 关联或取消关联设备组用户
func LinkDeviceGroupUser(c echo.Context) error {
	groupID, err := parseUintParam(c, "id")
	if err != nil {
		return err
	}

	var req request.LinkDeviceGroupUserRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if err := service.LinkDeviceGroupUser(groupID, req.UserID); err != nil {
		return err
	}

	message := "设备组用户关联成功"
	if req.UserID == nil {
		message = "设备组用户取消关联成功"
	}

	return c.JSON(http.StatusOK, &response.Response{
		Success: true,
		Message: message,
	})
}

// GetPendingActivationDevices 获取待激活设备列表
func GetPendingActivationDevices(c echo.Context) error {
	devices, err := service.GetPendingActivationDevices()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &response.Response{
		Success: true,
		Data:    devices,
	})
}
