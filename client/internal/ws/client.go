package ws

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/hang666/EasyUKey/client/internal/confirmation"
	"github.com/hang666/EasyUKey/shared/pkg/errs"
	"github.com/hang666/EasyUKey/shared/pkg/identity"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
	"github.com/hang666/EasyUKey/shared/pkg/messages"
	"github.com/hang666/EasyUKey/shared/pkg/wsutil"
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

	// 加密相关
	keyExchange     *identity.KeyExchange
	encryptor       *identity.Encryptor
	handshakeStatus messages.HandshakeStatus
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
		return fmt.Errorf("%w: %v", errs.ErrConvertWSURLFailed, err)
	}
	wsURL += WsPath

	conn, _, err = websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", errs.ErrWSConnectFailed, err)
	}

	setConnected(true)
	handshakeStatus = messages.HandshakeStatusPending

	// 启动消息监听
	go processMessages()

	// 启动心跳
	go heartbeat()

	// 首先进行密钥协商
	if err := SendKeyExchangeRequest(); err != nil {
		return err
	}

	// 等待密钥协商完成
	for i := 0; i < 30; i++ { // 最多等待30秒
		if handshakeStatus == messages.HandshakeStatusCompleted {
			break
		}
		if handshakeStatus == messages.HandshakeStatusFailed {
			return errs.ErrKeyExchangeFailed
		}
		time.Sleep(1 * time.Second)
	}

	if handshakeStatus != messages.HandshakeStatusCompleted {
		return errs.ErrKeyExchangeTimeout
	}

	// 根据设备初始化状态发送对应请求
	if !isDeviceInitialized {
		// 显示PIN设置页面
		if err := confirmation.ShowPINSetupPage(); err != nil {
			logger.Logger.Error("显示PIN设置页面失败", "error", err)
		}
		err = SendDeviceInitRequest()
	} else {
		err = SendDeviceRegistration()
	}
	if err != nil {
		logger.Logger.Error("发送设备请求失败", "error", err)
		os.Exit(1)
	}

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

	if isConnected {
		setConnected(false)
	}
}

// MonitorConnection 监控连接并在连接丢失时尝试重新连接
func MonitorConnection() {
	ticker := time.NewTicker(reconnectInterval)
	defer ticker.Stop()

	for range ticker.C {
		if !IsConnected() {
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
			// 发送失败不代表连接一定断了，MonitorConnection会处理重连
		}
	}
}
