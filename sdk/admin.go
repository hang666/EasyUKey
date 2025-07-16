package sdk

import (
	"fmt"
	"net/url"
	"strconv"
)

// AdminClient 管理员客户端
type AdminClient struct {
	*Client
}

// NewAdminClient 创建管理员客户端
func NewAdminClient(baseURL, adminKey string) *AdminClient {
	return &AdminClient{
		Client: NewClient(baseURL, adminKey),
	}
}

// VerifyAdminKey 验证管理员密钥
func (c *AdminClient) VerifyAdminKey() error {
	req := map[string]string{"admin_key": c.apiKey}
	_, err := c.request("POST", "/api/v1/admin/verify", req)
	return err
}

// CreateUser 创建用户
func (c *AdminClient) CreateUser(req *CreateUserRequest) (*User, error) {
	resp, err := c.request("POST", "/api/v1/admin/users", req)
	if err != nil {
		return nil, err
	}

	var user User
	if err := mapToStruct(resp.Data, &user); err != nil {
		return nil, fmt.Errorf("解析用户数据失败: %w", err)
	}

	return &user, nil
}

// GetUser 获取用户信息
func (c *AdminClient) GetUser(userID uint) (*User, error) {
	path := fmt.Sprintf("/api/v1/admin/users/%d", userID)
	resp, err := c.request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := mapToStruct(resp.Data, &user); err != nil {
		return nil, fmt.Errorf("解析用户数据失败: %w", err)
	}

	return &user, nil
}

// GetUsers 获取用户列表
func (c *AdminClient) GetUsers(page, pageSize int) ([]User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	params.Set("page_size", strconv.Itoa(pageSize))

	path := "/api/v1/admin/users?" + params.Encode()
	resp, err := c.request("GET", path, nil)
	if err != nil {
		return nil, 0, err
	}

	var users []User
	if err := mapToStruct(resp.Data, &users); err != nil {
		return nil, 0, fmt.Errorf("解析用户列表失败: %w", err)
	}

	total := int64(0)
	if resp.Total != nil {
		total = *resp.Total
	}

	return users, total, nil
}

// UpdateUser 更新用户
func (c *AdminClient) UpdateUser(userID uint, req *UpdateUserRequest) (*User, error) {
	path := fmt.Sprintf("/api/v1/admin/users/%d", userID)
	resp, err := c.request("PUT", path, req)
	if err != nil {
		return nil, err
	}

	var user User
	if err := mapToStruct(resp.Data, &user); err != nil {
		return nil, fmt.Errorf("解析用户数据失败: %w", err)
	}

	return &user, nil
}

// DeleteUser 删除用户
func (c *AdminClient) DeleteUser(userID uint) error {
	path := fmt.Sprintf("/api/v1/admin/users/%d", userID)
	_, err := c.request("DELETE", path, nil)
	return err
}

// GetUserDevices 获取用户设备列表
func (c *AdminClient) GetUserDevices(username string) ([]Device, error) {
	path := fmt.Sprintf("/api/v1/admin/users/%s/devices", username)
	resp, err := c.request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var devices []Device
	if err := mapToStruct(resp.Data, &devices); err != nil {
		return nil, fmt.Errorf("解析设备列表失败: %w", err)
	}

	return devices, nil
}

// GetDevices 获取设备列表
func (c *AdminClient) GetDevices(page, pageSize int, filter *DeviceFilter) ([]Device, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	params.Set("page_size", strconv.Itoa(pageSize))

	if filter != nil {
		if filter.IsOnline != nil {
			params.Set("is_online", strconv.FormatBool(*filter.IsOnline))
		}
		if filter.IsActive != nil {
			params.Set("is_active", strconv.FormatBool(*filter.IsActive))
		}
		if filter.UserID != nil {
			params.Set("user_id", strconv.FormatUint(uint64(*filter.UserID), 10))
		}
		if filter.Username != "" {
			params.Set("username", filter.Username)
		}
		if filter.Name != "" {
			params.Set("name", filter.Name)
		}
		if filter.OnlineOnly {
			params.Set("online_only", "true")
		}
		if filter.OfflineOnly {
			params.Set("offline_only", "true")
		}
	}

	path := "/api/v1/admin/devices?" + params.Encode()
	resp, err := c.request("GET", path, nil)
	if err != nil {
		return nil, 0, err
	}

	var devices []Device
	if err := mapToStruct(resp.Data, &devices); err != nil {
		return nil, 0, fmt.Errorf("解析设备列表失败: %w", err)
	}

	total := int64(0)
	if resp.Total != nil {
		total = *resp.Total
	}

	return devices, total, nil
}

// GetDevice 获取设备详情
func (c *AdminClient) GetDevice(deviceID uint) (*Device, error) {
	path := fmt.Sprintf("/api/v1/admin/devices/%d", deviceID)
	resp, err := c.request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var device Device
	if err := mapToStruct(resp.Data, &device); err != nil {
		return nil, fmt.Errorf("解析设备数据失败: %w", err)
	}

	return &device, nil
}

// UpdateDevice 更新设备
func (c *AdminClient) UpdateDevice(deviceID uint, req *UpdateDeviceRequest) (*Device, error) {
	path := fmt.Sprintf("/api/v1/admin/devices/%d", deviceID)
	resp, err := c.request("PUT", path, req)
	if err != nil {
		return nil, err
	}

	var device Device
	if err := mapToStruct(resp.Data, &device); err != nil {
		return nil, fmt.Errorf("解析设备数据失败: %w", err)
	}

	return &device, nil
}

// LinkDeviceToUser 绑定设备到用户
func (c *AdminClient) LinkDeviceToUser(deviceID uint, req *LinkDeviceToUserRequest) (*Device, error) {
	path := fmt.Sprintf("/api/v1/admin/devices/%d/user", deviceID)
	resp, err := c.request("POST", path, req)
	if err != nil {
		return nil, err
	}

	var device Device
	if err := mapToStruct(resp.Data, &device); err != nil {
		return nil, fmt.Errorf("解析设备数据失败: %w", err)
	}

	return &device, nil
}

// UnlinkDeviceFromUser 解绑设备
func (c *AdminClient) UnlinkDeviceFromUser(deviceID uint) (*Device, error) {
	path := fmt.Sprintf("/api/v1/admin/devices/%d/user", deviceID)
	resp, err := c.request("DELETE", path, nil)
	if err != nil {
		return nil, err
	}

	var device Device
	if err := mapToStruct(resp.Data, &device); err != nil {
		return nil, fmt.Errorf("解析设备数据失败: %w", err)
	}

	return &device, nil
}

// OfflineDevice 设备下线
func (c *AdminClient) OfflineDevice(deviceID uint) (*Device, error) {
	path := fmt.Sprintf("/api/v1/admin/devices/%d/offline", deviceID)
	resp, err := c.request("POST", path, nil)
	if err != nil {
		return nil, err
	}

	var device Device
	if err := mapToStruct(resp.Data, &device); err != nil {
		return nil, fmt.Errorf("解析设备数据失败: %w", err)
	}

	return &device, nil
}

// GetDeviceStatistics 获取设备统计信息
func (c *AdminClient) GetDeviceStatistics() (*DeviceStatistics, error) {
	resp, err := c.request("GET", "/api/v1/admin/devices/statistics", nil)
	if err != nil {
		return nil, err
	}

	var stats DeviceStatistics
	if err := mapToStruct(resp.Data, &stats); err != nil {
		return nil, fmt.Errorf("解析统计数据失败: %w", err)
	}

	return &stats, nil
}

// CreateAPIKey 创建API密钥
func (c *AdminClient) CreateAPIKey(req *CreateAPIKeyRequest) (*APIKey, error) {
	resp, err := c.request("POST", "/api/v1/admin/apikeys", req)
	if err != nil {
		return nil, err
	}

	var apiKey APIKey
	if err := mapToStruct(resp.Data, &apiKey); err != nil {
		return nil, fmt.Errorf("解析API密钥数据失败: %w", err)
	}

	return &apiKey, nil
}

// GetAPIKeys 获取API密钥列表
func (c *AdminClient) GetAPIKeys(page, pageSize int) ([]APIKey, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	params.Set("page_size", strconv.Itoa(pageSize))

	path := "/api/v1/admin/apikeys?" + params.Encode()
	resp, err := c.request("GET", path, nil)
	if err != nil {
		return nil, 0, err
	}

	var apiKeys []APIKey
	if err := mapToStruct(resp.Data, &apiKeys); err != nil {
		return nil, 0, fmt.Errorf("解析API密钥列表失败: %w", err)
	}

	total := int64(0)
	if resp.Total != nil {
		total = *resp.Total
	}

	return apiKeys, total, nil
}
