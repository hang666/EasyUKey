package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hang666/EasyUKey/sdk/errs"
	"github.com/hang666/EasyUKey/sdk/request"
	"github.com/hang666/EasyUKey/sdk/response"
)

// APIClient EasyUKey SDK客户端
type APIClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// NewClient 创建新的SDK客户端
func NewClient(baseURL, apiKey string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetTimeout 设置请求超时时间
func (c *APIClient) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// SetAPIKey 设置API密钥
func (c *APIClient) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

// 发送HTTP请求的通用方法
func (c *APIClient) request(method, path string, body interface{}) (*response.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errs.ErrSerializationFailed, err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errs.ErrRequestCreationFailed, err)
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
		return nil, fmt.Errorf("%w: %v", errs.ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	var result response.Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%w: %v", errs.ErrResponseParseFailed, err)
	}

	if !result.Success {
		return &result, fmt.Errorf("%w: %s", errs.ErrAPIError, result.Message)
	}

	return &result, nil
}

// StartAuth 发起用户认证
func (c *APIClient) StartAuth(req *request.AuthRequest) (*response.AuthData, error) {
	resp, err := c.request("POST", "/api/v1/auth", req)
	if err != nil {
		return nil, err
	}

	var authData response.AuthData
	if err := mapToStruct(resp.Data, &authData); err != nil {
		return nil, fmt.Errorf("%w: %v", errs.ErrDataParseFailed, err)
	}

	return &authData, nil
}

// VerifyAuth 验证认证结果
func (c *APIClient) VerifyAuth(req *request.VerifyAuthRequest) (*response.VerifyAuthData, error) {
	resp, err := c.request("POST", "/api/v1/auth/verify", req)
	if err != nil {
		return nil, err
	}

	var verifyData response.VerifyAuthData
	if err := mapToStruct(resp.Data, &verifyData); err != nil {
		return nil, fmt.Errorf("%w: %v", errs.ErrDataParseFailed, err)
	}

	return &verifyData, nil
}

// Health 健康检查
func (c *APIClient) Health() (map[string]string, error) {
	resp, err := c.request("GET", "/health", nil)
	if err != nil {
		return nil, err
	}

	data, ok := resp.Data.(map[string]string)
	if !ok {
		return nil, errs.ErrInvalidResponseFormat
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
