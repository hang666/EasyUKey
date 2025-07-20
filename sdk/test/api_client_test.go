package test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hang666/EasyUKey/sdk"
	"github.com/hang666/EasyUKey/sdk/request"
)

var (
	apiKey    string
	serverURL string
	username  string
)

func init() {
	apiKey = os.Getenv("EASYUKEY_TEST_API_KEY")
	if apiKey == "" {
		apiKey = "1234567890"
	}

	serverURL = os.Getenv("EASYUKEY_TEST_ADDR")
	if serverURL == "" {
		serverURL = "http://localhost:8888"
	}

	username = os.Getenv("EASYUKEY_TEST_USERNAME")
	if username == "" {
		username = "testuser"
	}
}

// newTestClient 创建测试客户端
func newTestClient() *sdk.APIClient {
	return sdk.NewClient(serverURL, apiKey)
}

// TestClientCreation 测试客户端创建
func TestClientCreation(t *testing.T) {
	client := newTestClient()
	if client == nil {
		t.Fatal("客户端创建失败")
	}
	t.Log("客户端创建成功")
}

// TestSetTimeout 测试设置超时时间
func TestSetTimeout(t *testing.T) {
	client := newTestClient()
	client.SetTimeout(5 * time.Second)
	t.Log("设置超时时间成功")
}

// TestHealth 测试健康检查
func TestHealth(t *testing.T) {
	client := newTestClient()

	health, err := client.Health()
	if err != nil {
		t.Logf("健康检查失败: %v (这是正常的，如果服务器未运行)", err)
		return
	}

	t.Logf("健康检查成功: %v", health)
}

// TestStartAuth 测试发起认证
func TestStartAuth(t *testing.T) {
	client := newTestClient()

	authData, err := client.StartAuth(
		&request.AuthRequest{
			Username:  username,
			Challenge: fmt.Sprintf("test-challenge-%d", time.Now().Unix()),
			Timeout:   600,
			Action:    "",
			Message:   "SDK测试认证",
		},
	)
	if err != nil {
		t.Logf("StartAuth失败: %v (这是正常的，如果服务器未运行或用户不存在)", err)
		return
	}

	t.Logf("StartAuth成功: %+v", authData)

	if authData.SessionID == "" {
		t.Error("SessionID不能为空")
	}
	if authData.Status == "" {
		t.Error("Status不能为空")
	}
}

// TestVerifyAuth 测试验证认证
func TestVerifyAuth(t *testing.T) {
	client := newTestClient()

	// 使用一个测试会话ID进行验证测试
	verifyData, err := client.VerifyAuth(
		&request.VerifyAuthRequest{
			SessionID: "test-session-id",
		},
	)
	if err != nil {
		t.Logf("VerifyAuth失败: %v (这是正常的，会话ID不存在)", err)
		return
	}

	t.Logf("VerifyAuth成功: %+v", verifyData)
}

// TestAuthWorkflow 测试完整认证流程
func TestAuthWorkflow(t *testing.T) {
	client := newTestClient()

	// 第一步：发起认证
	authData, err := client.StartAuth(
		&request.AuthRequest{
			Username:  username,
			Challenge: fmt.Sprintf("workflow-test-%d", time.Now().Unix()),
			Timeout:   60,
			Action:    "login",
			Message:   "SDK工作流测试",
		},
	)
	if err != nil {
		t.Logf("StartAuth失败: %v (跳过后续测试)", err)
		return
	}

	t.Logf("StartAuth成功，SessionID: %s", authData.SessionID)

	// 第二步：立即验证认证结果（通常会是pending状态）
	verifyData, err := client.VerifyAuth(
		&request.VerifyAuthRequest{
			SessionID: authData.SessionID,
		},
	)
	if err != nil {
		t.Logf("VerifyAuth失败: %v", err)
		return
	}

	t.Logf("VerifyAuth结果: Success=%v, UserID=%d, Username=%s",
		verifyData.Success, verifyData.UserID, verifyData.Username)
}

// TestAuthWorkflowWithPolling 测试完整认证流程（带轮询验证）
func TestAuthWorkflowWithPolling(t *testing.T) {
	client := newTestClient()

	// 第一步：发起认证
	authData, err := client.StartAuth(
		&request.AuthRequest{
			Username:  username,
			Challenge: fmt.Sprintf("polling-test-%d", time.Now().Unix()),
			Timeout:   60,
			Action:    "login",
			Message:   "SDK轮询测试",
		},
	)
	if err != nil {
		t.Logf("StartAuth失败: %v (跳过后续测试)", err)
		return
	}

	t.Logf("StartAuth成功，SessionID: %s", authData.SessionID)

	// 第二步：立即验证认证结果（通常会是pending状态）
	verifyData, err := client.VerifyAuth(
		&request.VerifyAuthRequest{
			SessionID: authData.SessionID,
		},
	)
	if err != nil {
		t.Logf("VerifyAuth失败: %v", err)
		return
	}

	t.Logf("初始VerifyAuth结果: Success=%v, UserID=%d, Username=%s",
		verifyData.Success, verifyData.UserID, verifyData.Username)

	// 第三步：轮询验证结果，每分钟内每五秒检查一次
	timeout := time.After(1 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Error("轮询验证超时，认证未在1分钟内完成")
			return
		case <-ticker.C:
			verifyData, err := client.VerifyAuth(
				&request.VerifyAuthRequest{
					SessionID: authData.SessionID,
				},
			)
			if err != nil {
				t.Logf("轮询VerifyAuth失败: %v", err)
				continue
			}

			t.Logf("轮询VerifyAuth结果: Success=%v, UserID=%d, Username=%s",
				verifyData.Success, verifyData.UserID, verifyData.Username)

			if verifyData.Success {
				t.Log("认证成功完成！")
				return
			}
		}
	}
}
