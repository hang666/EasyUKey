package ws

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/hang666/EasyUKey/client/internal/confirmation"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
	"github.com/hang666/EasyUKey/shared/pkg/wsutil"

	"github.com/gorilla/websocket"
)

const (
	reconnectInterval = 5 * time.Second
	pingInterval      = 30 * time.Second
	WsPath            = "/ws"
)

var (
	conn        *websocket.Conn
	mu          sync.Mutex
	isConnected bool

	serverAddr          string
	isDeviceInitialized bool

	errNotConnected       = errors.New("websocket is not connected")
	errDeviceNotAvailable = errors.New("device information not available")
)

// Init 初始化WebSocket客户端模块
func Init(addr string, initialized bool) {
	serverAddr = addr
	isDeviceInitialized = initialized
}

// Connect 连接到WebSocket服务器
func Connect() error {
	wsURL, err := wsutil.ConvertHTTPToWS(serverAddr)
	if err != nil {
		return fmt.Errorf("转换WebSocket URL失败: %v", err)
	}
	wsURL += WsPath

	conn, _, err = websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("WebSocket连接失败: %v", err)
	}

	setConnected(true)
	logger.Logger.Info("WebSocket连接成功")

	// 根据设备初始化状态发送对应请求
	if !isDeviceInitialized {
		// 显示PIN设置页面
		if err := confirmation.ShowPINSetupPage(); err != nil {
			logger.Logger.Error("显示PIN设置页面失败", "error", err)
		} else {
			logger.Logger.Info("已打开PIN设置页面，等待用户设置PIN")
		}
		err = SendDeviceInitRequest()
	} else {
		err = SendDeviceRegistration()
	}
	if err != nil {
		logger.Logger.Error("发送设备请求失败", "error", err)
		os.Exit(1)
	}

	// 启动消息监听
	go processMessages()

	// 启动心跳
	go heartbeat()

	return nil
}

// Disconnect 关闭WebSocket连接
func Disconnect() {
	mu.Lock()
	defer mu.Unlock()

	if conn != nil {
		// 设置写入截止时间，以防连接不良时挂起
		_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		// 发送关闭消息以解除读取 goroutine 的阻塞
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "shutting down"))
		// 关闭底层连接
		_ = conn.Close()
		conn = nil
	}

	// 仅在先前已连接的情况下设置为断开连接。
	// 这样可以避免在多次关闭调用时产生垃圾日志。
	if isConnected {
		setConnected(false)
		logger.Logger.Info("WebSocket连接已断开")
	}
}

// MonitorConnection 监控连接并在连接丢失时尝试重新连接
func MonitorConnection() {
	ticker := time.NewTicker(reconnectInterval)
	defer ticker.Stop()

	for range ticker.C {
		if !IsConnected() {
			logger.Logger.Info("WebSocket已断开，尝试重新连接...")
			if err := Connect(); err != nil {
				logger.Logger.Error("重新连接WebSocket失败", "error", err)
			}
		}
	}
}

func IsConnected() bool {
	mu.Lock()
	defer mu.Unlock()
	return isConnected
}

func setConnected(state bool) {
	mu.Lock()
	defer mu.Unlock()
	isConnected = state
}

func heartbeat() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for range ticker.C {
		if !IsConnected() {
			return // 连接已关闭，退出心跳
		}

		if err := SendPingMessage(); err != nil {
			logger.Logger.Warn("发送心跳失败", "error", err)
			// 发送失败不代表连接一定断了，MonitorConnection会处理重连
		} else {
			logger.Logger.Debug("心跳发送成功")
		}
	}
}
