package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client EasyUKey SDK客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// NewClient 创建新的SDK客户端
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetTimeout 设置请求超时时间
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// SetAPIKey 设置API密钥
func (c *Client) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

// 发送HTTP请求的通用方法
func (c *Client) request(method, path string, body interface{}) (*Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// 添加统一的API密钥头
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if !result.Success {
		return &result, fmt.Errorf("API错误: %s", result.Message)
	}

	return &result, nil
}

// StartAuth 发起用户认证
func (c *Client) StartAuth(username string, req *AuthRequest) (*AuthData, error) {
	resp, err := c.request("POST", "/api/v1/users/"+username+"/auth", req)
	if err != nil {
		return nil, err
	}

	var authData AuthData
	if err := mapToStruct(resp.Data, &authData); err != nil {
		return nil, fmt.Errorf("解析认证数据失败: %w", err)
	}

	return &authData, nil
}

// VerifyAuth 验证认证结果
func (c *Client) VerifyAuth(req *VerifyAuthRequest) (*VerifyAuthData, error) {
	resp, err := c.request("POST", "/api/v1/auth/verify", req)
	if err != nil {
		return nil, err
	}

	var verifyData VerifyAuthData
	if err := mapToStruct(resp.Data, &verifyData); err != nil {
		return nil, fmt.Errorf("解析验证数据失败: %w", err)
	}

	return &verifyData, nil
}

// Health 健康检查
func (c *Client) Health() (map[string]string, error) {
	resp, err := c.request("GET", "/health", nil)
	if err != nil {
		return nil, err
	}

	data, ok := resp.Data.(map[string]string)
	if !ok {
		return nil, fmt.Errorf("响应数据格式错误")
	}

	return data, nil
}

// 数据类型转换辅助函数
func mapToStruct(data interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}
