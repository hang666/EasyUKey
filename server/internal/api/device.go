package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/hang666/EasyUKey/sdk/request"
	"github.com/hang666/EasyUKey/sdk/response"
	"github.com/hang666/EasyUKey/server/internal/service"
)

// UpdateDevice 更新设备信息
func UpdateDevice(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return err
	}

	var req request.UpdateDeviceRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	device, err := service.UpdateDevice(deviceID, &req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &response.Response{Success: true, Message: "设备更新成功", Data: device})
}

// DeleteDevice 删除设备
func DeleteDevice(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return err
	}

	if err := service.DeleteDevice(deviceID); err != nil {
		return err
	}

	result := map[string]string{"message": "设备删除成功"}
	return c.JSON(http.StatusOK, &response.Response{Success: true, Message: "设备删除成功", Data: &result})
}

// LinkDeviceToUser 绑定设备到用户
func LinkDeviceToUser(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return err
	}

	var req request.LinkDeviceToUserRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	device, err := service.LinkDeviceToUser(deviceID, req.UserID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &response.Response{Success: true, Message: "设备绑定成功", Data: device})
}

// UnlinkDeviceFromUser 解绑设备与用户
func UnlinkDeviceFromUser(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return err
	}

	device, err := service.UnlinkDeviceFromUser(deviceID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &response.Response{Success: true, Message: "设备解绑成功", Data: device})
}

// OfflineDevice 设备下线
func OfflineDevice(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return err
	}

	device, err := service.OfflineDevice(deviceID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &response.Response{Success: true, Message: "设备已下线", Data: device})
}

// GetDevices 获取设备列表
func GetDevices(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 解析过滤条件
	filter := &request.DeviceFilter{}

	if isOnlineStr := c.QueryParam("is_online"); isOnlineStr != "" {
		if isOnline, err := strconv.ParseBool(isOnlineStr); err == nil {
			filter.IsOnline = &isOnline
		}
	}

	if isActiveStr := c.QueryParam("is_active"); isActiveStr != "" {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			filter.IsActive = &isActive
		}
	}

	if userIDStr := c.QueryParam("user_id"); userIDStr != "" {
		if userID, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
			userIDUint := uint(userID)
			filter.UserID = &userIDUint
		}
	}

	if username := c.QueryParam("username"); username != "" {
		filter.Username = username
	}

	if name := c.QueryParam("name"); name != "" {
		filter.Name = name
	}

	if c.QueryParam("online_only") == "true" {
		filter.OnlineOnly = true
	}

	if c.QueryParam("offline_only") == "true" {
		filter.OfflineOnly = true
	}

	devices, total, err := service.GetDevices(page, pageSize, filter)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &response.Response{Success: true, Message: "获取设备列表成功", Data: &devices, Total: &total})
}

// GetDevice 获取设备详情
func GetDevice(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return err
	}

	device, err := service.GetDeviceDetail(deviceID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &response.Response{Success: true, Message: "获取设备详情成功", Data: device})
}

// GetDeviceStatistics 获取设备统计信息
func GetDeviceStatistics(c echo.Context) error {
	totalDevices, onlineDevices, activeDevices, boundDevices, err := service.GetDeviceStatistics()
	if err != nil {
		return err
	}

	statsData := &response.DeviceStatistics{
		TotalDevices:   totalDevices,
		OnlineDevices:  onlineDevices,
		OfflineDevices: totalDevices - onlineDevices,
		ActiveDevices:  activeDevices,
		BoundDevices:   boundDevices,
	}

	return c.JSON(http.StatusOK, &response.Response{Success: true, Message: "获取设备统计成功", Data: statsData})
}
