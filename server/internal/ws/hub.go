package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/hang666/EasyUKey/server/internal/config"
	"github.com/hang666/EasyUKey/shared/pkg/identity"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
	"github.com/hang666/EasyUKey/shared/pkg/messages"

	"github.com/gorilla/websocket"
)

// Client WebSocket客户端连接
type Client struct {
	// 连接基本信息
	Conn     *websocket.Conn
	UserID   uint
	DeviceID uint
	Send     chan []byte

	// 设备信息
	SerialNumber       string
	VolumeSerialNumber string

	// 连接状态
	IsRegistered bool
	ConnectedAt  time.Time
	LastPongAt   time.Time

	// 加密相关
	KeyExchange     *identity.KeyExchange
	Encryptor       *identity.Encryptor
	HandshakeStatus messages.HandshakeStatus

	// 锁
	mu sync.RWMutex
}

// Hub WebSocket连接中心
type Hub struct {
	// 客户端连接映射
	clients       map[*Client]bool
	userClients   map[uint]*Client // 用户ID -> 客户端连接 (单一会话策略)
	deviceClients map[uint]*Client // 设备ID -> 客户端连接

	// 消息通道
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte

	// 锁
	mu sync.RWMutex
}

// WebSocket升级器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 检查请求来源
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源的连接
	},
	EnableCompression: false, // 禁用压缩以解决 "feature not supported" 错误
}

// InitUpgrader 使用配置初始化WebSocket升级器
func InitUpgrader() {
	upgrader.ReadBufferSize = config.GlobalConfig.WebSocket.ReadBufferSize
	upgrader.WriteBufferSize = config.GlobalConfig.WebSocket.WriteBufferSize
	upgrader.EnableCompression = config.GlobalConfig.WebSocket.EnableCompression
}

// NewHub 创建新的Hub
func NewHub() *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		userClients:   make(map[uint]*Client),
		deviceClients: make(map[uint]*Client),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		broadcast:     make(chan []byte),
	}
}

// Run 运行Hub
func (h *Hub) Run() {
	logger.Logger.Info("WebSocket Hub 开始运行")

	// 启动状态同步管理器
	GlobalStatusSync.Start()

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// registerClient 注册客户端
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 检查单一会话策略
	if client.UserID > 0 {
		if existingClient, exists := h.userClients[client.UserID]; exists {
			logger.Logger.Error("用户重复连接",
				"user_id", client.UserID,
				"existing_device_id", existingClient.DeviceID,
				"new_device_id", client.DeviceID)

			// 强制关闭现有连接
			h.forceCloseClient(existingClient)
		}

		// 注册新连接
		h.userClients[client.UserID] = client
	}

	if client.DeviceID > 0 {
		h.deviceClients[client.DeviceID] = client
		// 使用新的状态同步管理器
		GlobalStatusSync.UpdateDeviceStatus(client.DeviceID, true)
	}

	h.clients[client] = true

	logger.Logger.Info("客户端已注册",
		"user_id", client.UserID,
		"device_id", client.DeviceID,
		"serial_number", client.SerialNumber)
}

// unregisterClient 注销客户端
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)

		if client.UserID > 0 {
			// 仅当映射中的客户端是当前要注销的客户端时才删除，以避免竞态条件
			if c, ok := h.userClients[client.UserID]; ok && c == client {
				delete(h.userClients, client.UserID)
			}
		}

		if client.DeviceID > 0 {
			// 同样，检查设备映射
			if c, ok := h.deviceClients[client.DeviceID]; ok && c == client {
				delete(h.deviceClients, client.DeviceID)
				// 使用新的状态同步管理器
				GlobalStatusSync.UpdateDeviceStatus(client.DeviceID, false)
			}
		}

		close(client.Send)

		logger.Logger.Info("客户端已注销",
			"user_id", client.UserID,
			"device_id", client.DeviceID,
			"serial_number", client.SerialNumber,
			"duration", time.Since(client.ConnectedAt))
	}
}

// forceCloseClient 强制关闭客户端连接
func (h *Hub) forceCloseClient(client *Client) {
	// 先发送强制下线消息，让客户端优雅退出
	forceLogoutMsg := &messages.ForceLogoutMessage{
		Message: "设备被强制下线",
	}

	wsMsg := &messages.WSMessage{
		Type:      "force_logout",
		Data:      forceLogoutMsg,
		Timestamp: time.Now(),
	}

	msgBytes, err := json.Marshal(wsMsg)
	if err == nil {
		// 尝试发送强制下线消息，给客户端1秒时间处理
		select {
		case client.Send <- msgBytes:
			// 给客户端一点时间处理强制下线消息
			time.Sleep(1 * time.Second)
		default:
			// 如果发送通道已满，直接断开连接
		}
	}

	// 发送WebSocket关闭消息
	closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "设备被强制下线")
	client.Conn.WriteMessage(websocket.CloseMessage, closeMessage)

	// 关闭连接
	client.Conn.Close()

	// 从Hub中移除
	go func() {
		h.unregister <- client
	}()

	logger.Logger.Info("强制关闭客户端连接",
		"user_id", client.UserID,
		"device_id", client.DeviceID)
}

// broadcastMessage 广播消息
func (h *Hub) broadcastMessage(message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.Send <- message:
		default:
			close(client.Send)
			delete(h.clients, client)
		}
	}
}

// LinkDeviceToUser 将用户分配给已连接的设备，并更新Hub状态
func (h *Hub) LinkDeviceToUser(deviceID uint, userID uint) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, exists := h.deviceClients[deviceID]
	if !exists {
		return fmt.Errorf("设备 %d 未在线", deviceID)
	}

	if client.UserID == userID {
		return nil // 用户已分配，无需操作
	}

	// 检查单点登录策略：如果该用户已在其他设备上登录，则强制下线旧设备
	if existingClient, ok := h.userClients[userID]; ok {
		if existingClient.DeviceID != client.DeviceID {
			logger.Logger.Info("用户已在其他设备上连接，旧连接将被强制关闭",
				"user_id", userID,
				"old_device_id", existingClient.DeviceID,
				"new_device_id", deviceID)
			h.forceCloseClient(existingClient) // forceCloseClient内部已改为goroutine安全
		}
	}

	// 如果此设备之前已绑定其他用户，则移除旧映射关系
	if client.UserID > 0 {
		if oldClient, ok := h.userClients[client.UserID]; ok && oldClient.DeviceID == deviceID {
			delete(h.userClients, client.UserID)
		}
	}

	// 更新客户端的用户ID并建立新的映射
	client.UserID = userID
	h.userClients[userID] = client

	logger.Logger.Info("成功为在线设备关联用户",
		"user_id", client.UserID,
		"device_id", client.DeviceID)

	return nil
}

// SendToUser 向指定用户发送消息
func (h *Hub) SendToUser(userID uint, message []byte) error {
	h.mu.RLock()
	client, exists := h.userClients[userID]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("用户 %d 未在线", userID)
	}

	// 检查是否需要加密
	client.mu.RLock()
	encryptor := client.Encryptor
	handshakeStatus := client.HandshakeStatus
	client.mu.RUnlock()

	if handshakeStatus == messages.HandshakeStatusCompleted && encryptor != nil {
		// 需要加密发送
		// 先解析原始消息
		var wsMsg messages.WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			return fmt.Errorf("解析消息失败: %v", err)
		}

		// 使用加密发送
		return sendEncryptedMessage(client, wsMsg.Type, wsMsg.Data)
	}

	// 直接发送未加密消息
	select {
	case client.Send <- message:
		return nil
	default:
		return fmt.Errorf("发送消息失败，客户端发送通道已满")
	}
}

// SendToDevice 向指定设备发送消息
func (h *Hub) SendToDevice(deviceID uint, message []byte) error {
	h.mu.RLock()
	client, exists := h.deviceClients[deviceID]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("设备 %d 未在线", deviceID)
	}

	// 检查是否需要加密
	client.mu.RLock()
	encryptor := client.Encryptor
	handshakeStatus := client.HandshakeStatus
	client.mu.RUnlock()

	if handshakeStatus == messages.HandshakeStatusCompleted && encryptor != nil {
		// 需要加密发送
		// 先解析原始消息
		var wsMsg messages.WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			return fmt.Errorf("解析消息失败: %v", err)
		}

		// 使用加密发送
		return sendEncryptedMessage(client, wsMsg.Type, wsMsg.Data)
	}

	// 直接发送未加密消息
	select {
	case client.Send <- message:
		return nil
	default:
		return fmt.Errorf("发送消息失败，客户端发送通道已满")
	}
}

// GetUserClient 获取用户的客户端连接
func (h *Hub) GetUserClient(userID uint) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, exists := h.userClients[userID]
	return client, exists
}

// GetDeviceClient 获取设备的客户端连接
func (h *Hub) GetDeviceClient(deviceID uint) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, exists := h.deviceClients[deviceID]
	return client, exists
}

// GetOnlineUsersCount 获取在线用户数
func (h *Hub) GetOnlineUsersCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.userClients)
}

// IsUserOnline 检查用户是否在线
func (h *Hub) IsUserOnline(userID uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	_, exists := h.userClients[userID]
	return exists
}

// 实现WSHubInterface接口的新方法

// IsDeviceOnline 检查设备是否在线
func (h *Hub) IsDeviceOnline(deviceID uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	_, exists := h.deviceClients[deviceID]
	return exists
}

// GetOnlineDeviceIDs 获取所有在线设备ID列表
func (h *Hub) GetOnlineDeviceIDs() []uint {
	h.mu.RLock()
	defer h.mu.RUnlock()

	deviceIDs := make([]uint, 0, len(h.deviceClients))
	for deviceID := range h.deviceClients {
		deviceIDs = append(deviceIDs, deviceID)
	}
	return deviceIDs
}

// GetOnlineDevicesCount 获取在线设备数量
func (h *Hub) GetOnlineDevicesCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.deviceClients)
}

// GetOnlineDevicesList 获取在线设备详细信息列表
func (h *Hub) GetOnlineDevicesList() []map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	devices := make([]map[string]interface{}, 0, len(h.deviceClients))
	for _, client := range h.deviceClients {
		devices = append(devices, map[string]interface{}{
			"device_id":            client.DeviceID,
			"user_id":              client.UserID,
			"serial_number":        client.SerialNumber,
			"volume_serial_number": client.VolumeSerialNumber,
			"connected_at":         client.ConnectedAt,
			"last_pong_at":         client.LastPongAt,
		})
	}
	return devices
}

// OnDeviceConnect 设备连接时的回调
func (h *Hub) OnDeviceConnect(deviceID uint) error {
	// 使用新的状态同步管理器
	GlobalStatusSync.UpdateDeviceStatus(deviceID, true)
	return nil
}

// OnDeviceDisconnect 设备断开连接时的回调
func (h *Hub) OnDeviceDisconnect(deviceID uint) error {
	h.mu.RLock()
	client, exists := h.deviceClients[deviceID]
	h.mu.RUnlock()

	if !exists {
		// 设备不在线，只更新状态
		GlobalStatusSync.UpdateDeviceStatus(deviceID, false)
		logger.Logger.Info("设备不在线，仅更新状态", "device_id", deviceID)
		return nil
	}

	// 强制关闭客户端连接
	h.forceCloseClient(client)

	// 使用新的状态同步管理器
	GlobalStatusSync.UpdateDeviceStatus(deviceID, false)

	return nil
}

func (h *Hub) syncHeartbeat(deviceID uint) error {
	// 使用新的状态同步管理器
	GlobalStatusSync.UpdateHeartbeat(deviceID)
	return nil
}
