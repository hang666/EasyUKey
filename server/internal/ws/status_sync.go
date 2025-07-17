package ws

import (
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/hang666/EasyUKey/server/internal/config"
	"github.com/hang666/EasyUKey/server/internal/global"
	"github.com/hang666/EasyUKey/server/internal/model/entity"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
)

// GlobalStatusSync 全局状态同步管理器实例
var GlobalStatusSync = NewStatusSyncManager()

// StatusSyncManager 状态同步管理器
type StatusSyncManager struct {
	// 缓冲区用于批量更新
	updateBuffer map[uint]*DeviceStatusUpdate
	mu           sync.RWMutex

	// 同步配置
	batchSize    int
	syncInterval time.Duration

	// 控制通道
	updates chan *DeviceStatusUpdate
	stop    chan struct{}
	done    chan struct{}
}

// DeviceStatusUpdate 设备状态更新
type DeviceStatusUpdate struct {
	DeviceID      uint
	IsOnline      bool
	LastOnlineAt  *time.Time
	LastOfflineAt *time.Time
	LastHeartbeat *time.Time
	UpdatedAt     time.Time
}

// NewStatusSyncManager 创建新的状态同步管理器
func NewStatusSyncManager() *StatusSyncManager {
	return &StatusSyncManager{
		updateBuffer: make(map[uint]*DeviceStatusUpdate),
		batchSize:    50,                                  // 默认值，将在初始化时从配置更新
		syncInterval: 5 * time.Second,                     // 默认值，将在初始化时从配置更新
		updates:      make(chan *DeviceStatusUpdate, 100), // 默认值，将在初始化时从配置更新
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
	}
}

// InitWithConfig 使用配置初始化状态同步管理器
func (sm *StatusSyncManager) InitWithConfig() {
	sm.batchSize = config.GlobalConfig.GetBatchSize()
	sm.syncInterval = config.GlobalConfig.GetSyncInterval()

	// 重新创建通道以使用新的缓冲区大小
	oldUpdates := sm.updates
	sm.updates = make(chan *DeviceStatusUpdate, config.GlobalConfig.GetUpdateChannelBuffer())
	close(oldUpdates)
}

// Start 启动状态同步管理器
func (sm *StatusSyncManager) Start() {
	go sm.run()
	logger.Logger.Info("状态同步管理器已启动")
}

// Stop 停止状态同步管理器
func (sm *StatusSyncManager) Stop() {
	close(sm.stop)
	<-sm.done
	logger.Logger.Info("状态同步管理器已停止")
}

// UpdateDeviceStatus 更新设备状态（异步）
func (sm *StatusSyncManager) UpdateDeviceStatus(deviceID uint, isOnline bool) {
	now := time.Now()
	update := &DeviceStatusUpdate{
		DeviceID:  deviceID,
		IsOnline:  isOnline,
		UpdatedAt: now,
	}

	if isOnline {
		update.LastOnlineAt = &now
		update.LastHeartbeat = &now
	} else {
		update.LastOfflineAt = &now
	}

	select {
	case sm.updates <- update:
	default:
		// 如果通道满了，直接同步更新
		sm.syncSingleUpdate(update)
		logger.Logger.Error("状态更新通道已满，执行直接同步", "device_id", deviceID)
	}
}

// UpdateHeartbeat 更新设备心跳（异步）
func (sm *StatusSyncManager) UpdateHeartbeat(deviceID uint) {
	now := time.Now()
	update := &DeviceStatusUpdate{
		DeviceID:      deviceID,
		IsOnline:      true,
		LastHeartbeat: &now,
		UpdatedAt:     now,
	}

	select {
	case sm.updates <- update:
	default:
		// 如果通道满了，直接同步更新
		sm.syncSingleUpdate(update)
		logger.Logger.Error("心跳更新通道已满，执行直接同步", "device_id", deviceID)
	}
}

// run 运行同步循环
func (sm *StatusSyncManager) run() {
	defer close(sm.done)

	ticker := time.NewTicker(sm.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case update := <-sm.updates:
			sm.addToBuffer(update)

		case <-ticker.C:
			sm.flushBuffer()

		case <-sm.stop:
			// 停止前刷新缓冲区
			sm.flushBuffer()
			return
		}
	}
}

// addToBuffer 添加更新到缓冲区
func (sm *StatusSyncManager) addToBuffer(update *DeviceStatusUpdate) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 合并同一设备的多个更新
	if existing, exists := sm.updateBuffer[update.DeviceID]; exists {
		existing.IsOnline = update.IsOnline
		if update.LastOnlineAt != nil {
			existing.LastOnlineAt = update.LastOnlineAt
		}
		if update.LastOfflineAt != nil {
			existing.LastOfflineAt = update.LastOfflineAt
		}
		if update.LastHeartbeat != nil {
			existing.LastHeartbeat = update.LastHeartbeat
		}
		existing.UpdatedAt = update.UpdatedAt
	} else {
		sm.updateBuffer[update.DeviceID] = update
	}

	// 如果缓冲区满了，立即刷新
	if len(sm.updateBuffer) >= sm.batchSize {
		sm.flushBufferLocked()
	}
}

// flushBuffer 刷新缓冲区（带锁）
func (sm *StatusSyncManager) flushBuffer() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.flushBufferLocked()
}

// flushBufferLocked 刷新缓冲区（不带锁）
func (sm *StatusSyncManager) flushBufferLocked() {
	if len(sm.updateBuffer) == 0 {
		return
	}

	updates := make([]*DeviceStatusUpdate, 0, len(sm.updateBuffer))
	for _, update := range sm.updateBuffer {
		updates = append(updates, update)
	}

	// 清空缓冲区
	sm.updateBuffer = make(map[uint]*DeviceStatusUpdate)

	// 释放锁后执行同步
	go sm.syncBatchUpdates(updates)
}

// syncBatchUpdates 批量同步更新
func (sm *StatusSyncManager) syncBatchUpdates(updates []*DeviceStatusUpdate) {
	if len(updates) == 0 {
		return
	}

	// 使用事务批量更新
	tx := global.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Logger.Error("批量状态同步发生panic", "panic", r, "count", len(updates))
		}
	}()

	for _, update := range updates {
		if err := sm.syncSingleUpdateInTx(tx, update); err != nil {
			logger.Logger.Error("同步设备状态失败", "error", err, "device_id", update.DeviceID)
		}
	}

	if err := tx.Commit().Error; err != nil {
		logger.Logger.Error("提交状态同步事务失败", "error", err, "count", len(updates))
		return
	}
}

// syncSingleUpdate 同步单个更新
func (sm *StatusSyncManager) syncSingleUpdate(update *DeviceStatusUpdate) {
	if err := sm.syncSingleUpdateInTx(global.DB, update); err != nil {
		logger.Logger.Error("同步设备状态失败", "error", err, "device_id", update.DeviceID)
	}
}

// syncSingleUpdateInTx 在事务中同步单个更新
func (sm *StatusSyncManager) syncSingleUpdateInTx(tx *gorm.DB, update *DeviceStatusUpdate) error {
	updates := map[string]interface{}{
		"is_online": update.IsOnline,
	}

	if update.LastOnlineAt != nil {
		updates["last_online_at"] = *update.LastOnlineAt
	}
	if update.LastOfflineAt != nil {
		updates["last_offline_at"] = *update.LastOfflineAt
	}
	if update.LastHeartbeat != nil {
		updates["last_heartbeat"] = *update.LastHeartbeat
	}

	result := tx.Model(&entity.Device{}).Where("id = ?", update.DeviceID).Updates(updates)
	return result.Error
}
