package device

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	var device *uid.USBDevice
	var err error

	// 在Linux中，根据可执行文件路径查找对应的挂载点设备
	// 在Windows中，根据驱动器路径查找设备
	if filepath.Separator == '/' { // Linux/Unix系统
		device, err = findDeviceByExecutablePath(exeDir)
	} else { // Windows系统
		drive := filepath.VolumeName(exeDir)
		device, err = uid.FindDeviceByDrive(drive)
	}

	if err != nil || device == nil || device.SerialNumber == "" {
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

// findDeviceByExecutablePath 在Linux系统中根据可执行文件路径查找对应的USB设备
func findDeviceByExecutablePath(exePath string) (*uid.USBDevice, error) {
	// 获取可执行文件的绝对路径
	absPath, err := filepath.Abs(exePath)
	if err != nil {
		return nil, err
	}

	// 使用 df 命令获取路径对应的文件系统设备
	devicePath, err := getDevicePathByDF(absPath)
	if err != nil {
		return nil, err
	}

	// 根据设备路径查找USB设备
	return uid.FindDeviceByDevicePath(devicePath)
}

// getDevicePathByDF 使用 df 命令获取指定路径的文件系统设备路径
func getDevicePathByDF(path string) (string, error) {
	// 使用 df 命令获取路径信息，输出格式：设备 大小 已用 可用 使用率 挂载点
	cmd := exec.Command("df", path)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// 解析 df 输出
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	lineCount := 0
	for scanner.Scan() {
		lineCount++
		if lineCount == 1 {
			// 跳过标题行
			continue
		}

		line := scanner.Text()
		fields := strings.Fields(line)

		// df 输出可能因为设备名太长而换行，需要处理这种情况
		if len(fields) == 1 {
			// 设备名在单独一行，返回设备路径
			return fields[0], nil
		} else if len(fields) >= 6 {
			// 正常情况，设备路径在第1个字段
			return fields[0], nil
		}
	}

	return "", os.ErrNotExist
}
