package device

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hang666/EasyUKey/client/utils/uid"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
)

const (
	deviceCheckInterval = 10 * time.Second
	maxUnfindTime       = 10
)

var (
	unfindTime int
	DeviceInfo DeviceInfoType
	doneChan   chan struct{}
)

// DeviceInfoType 设备信息类型，提供线程安全的设备信息访问
type DeviceInfoType struct {
	mu     sync.Mutex     // 互斥锁
	Device *uid.USBDevice // USB设备信息
}

func (d *DeviceInfoType) GetDevice() *uid.USBDevice {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.Device
}

func (d *DeviceInfoType) SetDevice(device *uid.USBDevice) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Device = device
}

func init() {
	doneChan = make(chan struct{})
}

// StartTimer 启动设备检测定时器
func StartTimer(exeDir string) {
	ticker := time.NewTicker(deviceCheckInterval)
	defer ticker.Stop()

	CheckDevice(exeDir)

	for {
		select {
		case <-ticker.C:
			CheckDevice(exeDir)
		case <-doneChan:
			logger.Logger.Info("设备检查定时器已停止")
			return
		}
	}
}

// StopTimer 停止设备检测定时器
func StopTimer() {
	close(doneChan)
}

// CheckDevice 检查设备是否存在
func CheckDevice(exeDir string) {
	drive := filepath.VolumeName(exeDir)
	device, err := uid.FindDeviceByDrive(drive)

	if err != nil || device.SerialNumber == "" {
		unfindTime++
		if unfindTime > maxUnfindTime {
			logger.Logger.Error("未找到设备，程序退出", "unfind_time", unfindTime, "max_time", maxUnfindTime)
			os.Exit(1)
		}
		return
	}

	if DeviceInfo.GetDevice() == nil {
		DeviceInfo.SetDevice(device)
	}
	unfindTime = 0
}
