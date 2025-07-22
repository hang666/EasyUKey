//go:build darwin

package uid

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// TODO 未经测试

// DiskutilInfo diskutil info 输出的磁盘信息
type DiskutilInfo struct {
	DeviceIdentifier string
	DeviceNode       string
	VolumeName       string
	VolumeUUID       string
	FileSystem       string
	TotalSize        uint64
	ProtocolName     string
	DeviceLocation   string
	RemovableMedia   string
	MediaName        string
	VendorName       string
	ProductName      string
	SerialNumber     string
	DiskSize         uint64
	MountPoint       string
}

// USBDeviceInfo system_profiler 输出的USB设备信息
type USBDeviceInfo struct {
	ProductID    string
	VendorID     string
	SerialNumber string
	Manufacturer string
	ProductName  string
	LocationID   string
}

func getUSBDevices() ([]USBDevice, error) {
	var devices []USBDevice

	// 1. 获取所有磁盘列表
	diskList, err := getDiskList()
	if err != nil {
		return nil, fmt.Errorf("获取磁盘列表失败: %v", err)
	}

	// 2. 获取USB设备信息用于补充缺失的硬件信息
	usbDevicesMap, err := getUSBDevicesInfo()
	if err != nil {
		// USB信息获取失败不影响主要功能，只记录但继续执行
		usbDevicesMap = make(map[string]USBDeviceInfo)
	}

	// 3. 为每个磁盘获取详细信息
	for _, diskID := range diskList {
		diskInfo, err := getDiskInfo(diskID)
		if err != nil {
			continue // 跳过无法获取信息的磁盘
		}

		// 开发模式显示所有设备，生产模式只显示USB设备
		if DevMode != "true" && !isUSBDevice(diskInfo) {
			continue
		}

		// 只处理有挂载点的设备
		if diskInfo.MountPoint == "" {
			continue
		}

		// 构建 USBDevice
		device := buildUSBDeviceFromDiskInfo(diskInfo, usbDevicesMap)
		devices = append(devices, device)
	}

	return devices, nil
}

// getDiskList 获取所有磁盘标识符列表
func getDiskList() ([]string, error) {
	cmd := exec.Command("diskutil", "list", "-plist")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// 简化处理：使用正则表达式提取磁盘标识符
	// 在实际环境中可能需要更复杂的plist解析
	re := regexp.MustCompile(`/dev/(disk\d+)`)
	matches := re.FindAllStringSubmatch(string(output), -1)

	var disks []string
	for _, match := range matches {
		if len(match) > 1 {
			disks = append(disks, match[1])
		}
	}

	return disks, nil
}

// getDiskInfo 获取特定磁盘的详细信息
func getDiskInfo(diskID string) (DiskutilInfo, error) {
	var info DiskutilInfo

	cmd := exec.Command("diskutil", "info", diskID)
	output, err := cmd.Output()
	if err != nil {
		return info, err
	}

	// 解析 diskutil info 的输出
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.Contains(line, "Device Identifier:") {
			info.DeviceIdentifier = extractValue(line)
		} else if strings.Contains(line, "Device Node:") {
			info.DeviceNode = extractValue(line)
		} else if strings.Contains(line, "Volume Name:") {
			info.VolumeName = extractValue(line)
		} else if strings.Contains(line, "Volume UUID:") {
			info.VolumeUUID = extractValue(line)
		} else if strings.Contains(line, "Type (Bundle):") {
			info.FileSystem = extractValue(line)
		} else if strings.Contains(line, "Total Size:") {
			if size := extractSizeFromLine(line); size > 0 {
				info.TotalSize = size
			}
		} else if strings.Contains(line, "Protocol:") {
			info.ProtocolName = extractValue(line)
		} else if strings.Contains(line, "Device Location:") {
			info.DeviceLocation = extractValue(line)
		} else if strings.Contains(line, "Removable Media:") {
			info.RemovableMedia = extractValue(line)
		} else if strings.Contains(line, "Media Name:") {
			info.MediaName = extractValue(line)
		} else if strings.Contains(line, "Device / Media Name:") {
			info.MediaName = extractValue(line)
		} else if strings.Contains(line, "Mount Point:") {
			info.MountPoint = extractValue(line)
		}
	}

	// 如果没有获取到大小，尝试从磁盘大小获取
	if info.TotalSize == 0 {
		info.TotalSize = info.DiskSize
	}

	return info, nil
}

// getUSBDevicesInfo 获取USB设备信息
func getUSBDevicesInfo() (map[string]USBDeviceInfo, error) {
	usbDevices := make(map[string]USBDeviceInfo)

	cmd := exec.Command("system_profiler", "SPUSBDataType")
	output, err := cmd.Output()
	if err != nil {
		return usbDevices, err
	}

	// 解析 system_profiler 输出
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var currentDevice USBDeviceInfo
	var deviceName string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 检测设备开始（设备名通常不缩进或缩进较少）
		if !strings.HasPrefix(line, " ") && strings.Contains(line, ":") {
			// 保存之前的设备
			if deviceName != "" && (currentDevice.SerialNumber != "" || currentDevice.ProductName != "") {
				usbDevices[deviceName] = currentDevice
			}
			// 开始新设备
			deviceName = strings.TrimSuffix(line, ":")
			currentDevice = USBDeviceInfo{}
		} else if strings.Contains(line, "Product ID:") {
			currentDevice.ProductID = extractValue(line)
		} else if strings.Contains(line, "Vendor ID:") {
			currentDevice.VendorID = extractValue(line)
		} else if strings.Contains(line, "Serial Number:") {
			currentDevice.SerialNumber = extractValue(line)
		} else if strings.Contains(line, "Manufacturer:") {
			currentDevice.Manufacturer = extractValue(line)
		} else if strings.Contains(line, "Location ID:") {
			currentDevice.LocationID = extractValue(line)
		}
	}

	// 保存最后一个设备
	if deviceName != "" && (currentDevice.SerialNumber != "" || currentDevice.ProductName != "") {
		usbDevices[deviceName] = currentDevice
	}

	return usbDevices, nil
}

// buildUSBDeviceFromDiskInfo 从磁盘信息构建 USBDevice
func buildUSBDeviceFromDiskInfo(info DiskutilInfo, usbDevicesMap map[string]USBDeviceInfo) USBDevice {
	device := USBDevice{
		DevicePath:         info.MountPoint,
		SerialNumber:       info.SerialNumber,
		Label:              info.VolumeName,
		Size:               info.TotalSize,
		FileSystem:         info.FileSystem,
		VolumeSerialNumber: info.VolumeUUID,
		Vendor:             info.VendorName,
		Model:              info.ProductName,
	}

	// 如果磁盘信息中没有序列号，尝试从USB设备信息中获取
	if device.SerialNumber == "" {
		for deviceName, usbInfo := range usbDevicesMap {
			if strings.Contains(deviceName, info.MediaName) ||
				strings.Contains(info.MediaName, usbInfo.ProductName) {
				device.SerialNumber = usbInfo.SerialNumber
				if device.Vendor == "" {
					device.Vendor = usbInfo.Manufacturer
				}
				if device.Model == "" {
					device.Model = usbInfo.ProductName
				}
				break
			}
		}
	}

	// 如果没有厂商信息，尝试从 Model 中提取
	if device.Vendor == "" && device.Model != "" {
		if parts := strings.Fields(device.Model); len(parts) > 0 {
			device.Vendor = parts[0]
		}
	}

	// 使用媒体名称作为型号（如果没有其他信息）
	if device.Model == "" {
		device.Model = info.MediaName
	}

	return device
}

// isUSBDevice 判断是否为USB设备
func isUSBDevice(info DiskutilInfo) bool {
	// 检查协议名称
	if strings.EqualFold(info.ProtocolName, "USB") {
		return true
	}

	// 检查设备位置
	if strings.Contains(strings.ToLower(info.DeviceLocation), "usb") {
		return true
	}

	// 检查是否为可移动媒体
	if strings.EqualFold(info.RemovableMedia, "Yes") {
		return true
	}

	return false
}

// extractValue 从形如 "Key: Value" 的行中提取值
func extractValue(line string) string {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

// extractSizeFromLine 从包含大小信息的行中提取字节数
func extractSizeFromLine(line string) uint64 {
	// 查找形如 "(1,234,567,890 Bytes)" 的模式
	re := regexp.MustCompile(`\(([0-9,]+)\s+Bytes\)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		// 移除逗号并转换为数字
		sizeStr := strings.ReplaceAll(matches[1], ",", "")
		if size, err := strconv.ParseUint(sizeStr, 10, 64); err == nil {
			return size
		}
	}

	return 0
}
