//go:build !windows && !darwin

package uid

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// LinuxBlockDevice lsblk 输出的块设备信息
type LinuxBlockDevice struct {
	Name       string             `json:"name"`
	MountPoint string             `json:"mountpoint"`
	Label      string             `json:"label"`
	UUID       string             `json:"uuid"`
	FsType     string             `json:"fstype"`
	Size       string             `json:"size"` // lsblk 返回的是字符串格式如 "1G", "512M"
	Type       string             `json:"type"`
	Removable  bool               `json:"rm"`
	Vendor     string             `json:"vendor"`
	Model      string             `json:"model"`
	Serial     string             `json:"serial"`
	Tran       string             `json:"tran"`     // 传输类型：usb, sata, etc.
	Children   []LinuxBlockDevice `json:"children"` // 子设备（分区）
}

func getUSBDevices() ([]USBDevice, error) {
	var devices []USBDevice

	// 1. 使用 lsblk 获取所有块设备信息
	blockDevices, err := getBlockDevices()
	if err != nil {
		return nil, fmt.Errorf("获取块设备信息失败: %v", err)
	}

	// 2. 过滤并转换为 USBDevice 格式
	for _, bd := range blockDevices {
		// 开发模式显示所有设备，生产模式只显示USB设备
		if DevMode != "true" && bd.Tran != "usb" {
			continue
		}

		// 检查主设备是否有挂载点的分区
		foundMountedPartition := false

		// 检查子设备（分区）是否有挂载点
		for _, child := range bd.Children {
			if child.MountPoint != "" {
				// 使用父设备的基本信息构建USB设备，但使用子设备的挂载信息
				device, err := buildUSBDeviceFromPartition(bd, child)
				if err != nil {
					// 记录错误但继续处理其他设备
					continue
				}
				devices = append(devices, device)
				foundMountedPartition = true
			}
		}

		// 如果没有找到已挂载的分区，但主设备本身有挂载点，则处理主设备
		if !foundMountedPartition && bd.MountPoint != "" {
			device, err := buildUSBDevice(bd)
			if err != nil {
				// 记录错误但继续处理其他设备
				continue
			}
			devices = append(devices, device)
		}
	}

	return devices, nil
}

// getBlockDevices 使用 lsblk 获取块设备信息
func getBlockDevices() ([]LinuxBlockDevice, error) {
	// 使用 lsblk 命令获取设备信息，输出为 JSON 格式
	cmd := exec.Command("lsblk", "-J", "-o", "NAME,MOUNTPOINT,LABEL,UUID,FSTYPE,SIZE,TYPE,RM,VENDOR,MODEL,SERIAL,TRAN")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行 lsblk 命令失败: %v", err)
	}

	// 解析 JSON 输出
	var result struct {
		BlockDevices []LinuxBlockDevice `json:"blockdevices"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("解析 lsblk 输出失败: %v", err)
	}

	// 展开嵌套的设备（包括分区）
	var allDevices []LinuxBlockDevice
	for _, device := range result.BlockDevices {
		allDevices = append(allDevices, device)
	}

	return allDevices, nil
}

// buildUSBDeviceFromPartition 从主设备和分区信息构建 USBDevice 结构
func buildUSBDeviceFromPartition(parent LinuxBlockDevice, partition LinuxBlockDevice) (USBDevice, error) {
	device := USBDevice{
		DevicePath:         partition.MountPoint,                // 使用分区的挂载点
		DeviceNode:         "/dev/" + partition.Name,            // 使用分区的设备节点
		SerialNumber:       strings.TrimSpace(parent.Serial),    // 使用主设备的序列号
		Label:              strings.TrimSpace(partition.Label),  // 使用分区的标签
		FileSystem:         strings.TrimSpace(partition.FsType), // 使用分区的文件系统
		VolumeSerialNumber: strings.TrimSpace(partition.UUID),   // 使用分区的UUID
		Vendor:             strings.TrimSpace(parent.Vendor),    // 使用主设备的厂商
		Model:              strings.TrimSpace(parent.Model),     // 使用主设备的型号
		Size:               parseSizeString(partition.Size),     // 使用分区的大小
	}

	// 如果分区没有大小信息，使用主设备的大小
	if device.Size == 0 {
		device.Size = parseSizeString(parent.Size)
	}

	// 如果没有序列号，尝试从 udev 属性获取
	if device.SerialNumber == "" {
		deviceName := "/dev/" + parent.Name
		if serialFromUdev := getDeviceSerialFromUdev(deviceName); serialFromUdev != "" {
			device.SerialNumber = serialFromUdev
		}
	}

	// 如果没有厂商信息，尝试从 Model 中提取
	if device.Vendor == "" && device.Model != "" {
		if parts := strings.Fields(device.Model); len(parts) > 0 {
			device.Vendor = parts[0]
		}
	}

	// 获取设备大小（如果都没有提供）
	if device.Size == 0 {
		if size := getDeviceSize(partition.MountPoint); size > 0 {
			device.Size = size
		}
	}

	// 如果最终还是没有序列号，设置为"null"
	if device.SerialNumber == "" {
		device.SerialNumber = "null"
	}

	return device, nil
}

// buildUSBDevice 构建 USBDevice 结构
func buildUSBDevice(bd LinuxBlockDevice) (USBDevice, error) {
	device := USBDevice{
		DevicePath:         bd.MountPoint,
		DeviceNode:         "/dev/" + bd.Name, // 添加设备节点
		SerialNumber:       strings.TrimSpace(bd.Serial),
		Label:              strings.TrimSpace(bd.Label),
		FileSystem:         strings.TrimSpace(bd.FsType),
		VolumeSerialNumber: strings.TrimSpace(bd.UUID),
		Vendor:             strings.TrimSpace(bd.Vendor),
		Model:              strings.TrimSpace(bd.Model),
		Size:               parseSizeString(bd.Size),
	}

	// 如果没有序列号，尝试从 udev 属性获取
	if device.SerialNumber == "" {
		deviceName := "/dev/" + bd.Name
		if serialFromUdev := getDeviceSerialFromUdev(deviceName); serialFromUdev != "" {
			device.SerialNumber = serialFromUdev
		}
	}

	// 如果没有厂商信息，尝试从 Model 中提取
	if device.Vendor == "" && device.Model != "" {
		if parts := strings.Fields(device.Model); len(parts) > 0 {
			device.Vendor = parts[0]
		}
	}

	// 获取设备大小（如果 lsblk 没有提供）
	if device.Size == 0 {
		if size := getDeviceSize(bd.MountPoint); size > 0 {
			device.Size = size
		}
	}

	// 如果最终还是没有序列号，设置为"null"
	if device.SerialNumber == "" {
		device.SerialNumber = "null"
	}

	return device, nil
}

// getDeviceSerialFromUdev 从 udev 属性获取设备序列号
func getDeviceSerialFromUdev(devicePath string) string {
	cmd := exec.Command("udevadm", "info", "--query=property", "--name="+devicePath)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		// 查找序列号相关的属性
		if strings.HasPrefix(line, "ID_SERIAL_SHORT=") {
			return strings.TrimPrefix(line, "ID_SERIAL_SHORT=")
		}
		if strings.HasPrefix(line, "ID_SERIAL=") {
			serial := strings.TrimPrefix(line, "ID_SERIAL=")
			// 移除厂商前缀（如果存在）
			if parts := strings.Split(serial, "_"); len(parts) > 1 {
				return parts[len(parts)-1]
			}
			return serial
		}
	}

	return ""
}

// getDeviceSize 获取挂载点对应的设备大小
func getDeviceSize(mountPoint string) uint64 {
	// 读取 /proc/mounts 找到对应的设备
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[1] == mountPoint {
			devicePath := fields[0]
			// 获取设备大小
			return getBlockDeviceSize(devicePath)
		}
	}

	return 0
}

// getBlockDeviceSize 获取块设备的大小
func getBlockDeviceSize(devicePath string) uint64 {
	// 从设备路径提取设备名
	deviceName := filepath.Base(devicePath)

	// 尝试从 /sys/block/ 读取大小
	sizePath := fmt.Sprintf("/sys/block/%s/size", deviceName)
	if !fileExists(sizePath) {
		// 如果是分区，尝试父设备
		re := regexp.MustCompile(`(\w+)\d+$`)
		if matches := re.FindStringSubmatch(deviceName); len(matches) > 1 {
			parentDevice := matches[1]
			sizePath = fmt.Sprintf("/sys/block/%s/%s/size", parentDevice, deviceName)
		}
	}

	if fileExists(sizePath) {
		if content, err := os.ReadFile(sizePath); err == nil {
			if size, err := strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64); err == nil {
				// /sys/block/*/size 的值是以512字节为单位的扇区数
				return size * 512
			}
		}
	}

	return 0
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// parseSizeString 解析lsblk返回的大小字符串（如"1G", "512M", "2048K"）
func parseSizeString(sizeStr string) uint64 {
	if sizeStr == "" {
		return 0
	}

	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" {
		return 0
	}

	// 找到最后一个数字的位置
	var numberPart string
	var unitPart string

	i := len(sizeStr) - 1
	for i >= 0 && !unicode.IsDigit(rune(sizeStr[i])) {
		i--
	}

	if i >= 0 {
		numberPart = sizeStr[:i+1]
		unitPart = strings.ToUpper(sizeStr[i+1:])
	} else {
		return 0
	}

	// 解析数字部分
	var multiplier uint64 = 1
	switch unitPart {
	case "K", "KB":
		multiplier = 1024
	case "M", "MB":
		multiplier = 1024 * 1024
	case "G", "GB":
		multiplier = 1024 * 1024 * 1024
	case "T", "TB":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "P", "PB":
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024
	case "B", "":
		multiplier = 1
	}

	// 解析可能包含小数点的数字
	if number, err := strconv.ParseFloat(numberPart, 64); err == nil {
		return uint64(number * float64(multiplier))
	}

	return 0
}
