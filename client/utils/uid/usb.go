package uid

import "fmt"

// DevMode 开发模式标志，通过构建时注入
var DevMode = "false"

// USBDevice 表示一个 USB 设备信息
type USBDevice struct {
	// DevicePath 设备路径（Windows为驱动器盘符如C:，Linux为挂载点如/media/usb）
	DevicePath string `json:"device_path"`
	// DeviceNode 设备节点（Linux下为/dev/sdb1等，Windows下为空）
	DeviceNode string `json:"device_node"`
	// SerialNumber 序列号
	SerialNumber string `json:"serial_number"`
	// Label 设备标签/名称
	Label string `json:"label"`
	// Size 设备大小（字节）
	Size uint64 `json:"size"`
	// FileSystem 文件系统类型
	FileSystem string `json:"file_system"`
	// VolumeSerialNumber 卷序列号
	VolumeSerialNumber string `json:"volume_serial_number"`
	// Vendor 厂商
	Vendor string `json:"vendor"`
	// Model 型号
	Model string `json:"model"`
}

// GetUSBDevices 获取所有 USB 存储设备列表
func GetUSBDevices() ([]USBDevice, error) {
	return getUSBDevices()
}

// GetUSBDeviceBySerial 根据序列号获取特定的 USB 设备
func GetUSBDeviceBySerial(serialNumber string) (*USBDevice, error) {
	devices, err := GetUSBDevices()
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		if device.SerialNumber == serialNumber {
			return &device, nil
		}
	}

	return nil, nil
}

// GetUSBDeviceByVolumeSerial 根据卷序列号获取特定的 USB 设备
func GetUSBDeviceByVolumeSerial(volumeSerialNumber string) (*USBDevice, error) {
	devices, err := GetUSBDevices()
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		if device.VolumeSerialNumber == volumeSerialNumber {
			return &device, nil
		}
	}

	return nil, nil
}

// FindDeviceByDrive 根据驱动器路径获取特定的 USB 设备
func FindDeviceByDrive(volume string) (*USBDevice, error) {
	devices, err := GetUSBDevices()
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		if device.DevicePath == volume {
			return &device, nil
		}
	}
	return nil, fmt.Errorf("未找到设备: %s", volume)
}

// FindDeviceByDevicePath 根据设备节点路径获取特定的 USB 设备（用于Linux）
func FindDeviceByDevicePath(devicePath string) (*USBDevice, error) {
	devices, err := GetUSBDevices()
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		// 在Linux中比较设备节点
		if device.DeviceNode == devicePath {
			return &device, nil
		}
		// 也支持通过挂载点匹配（向后兼容）
		if device.DevicePath == devicePath {
			return &device, nil
		}
	}
	return nil, fmt.Errorf("未找到设备: %s", devicePath)
}
