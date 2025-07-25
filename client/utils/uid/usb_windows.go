//go:build windows

package uid

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/yusufpapurcu/wmi"
)

// Win32_LogicalDisk WMI 查询结构
type Win32_LogicalDisk struct {
	DeviceID           string
	Size               uint64
	FileSystem         string
	VolumeName         string
	VolumeSerialNumber string
	DriveType          uint32
}

// Win32_DiskDrive WMI 查询结构
type Win32_DiskDrive struct {
	DeviceID      string
	SerialNumber  string
	Model         string
	Size          uint64
	InterfaceType string
	Index         uint32
}

// Win32_DiskPartition WMI 查询结构
type Win32_DiskPartition struct {
	DeviceID       string
	DiskIndex      uint32
	Size           uint64
	StartingOffset uint64
}

// Win32_LogicalDiskToPartition WMI 关联结构
type Win32_LogicalDiskToPartition struct {
	Antecedent string
	Dependent  string
}

func getUSBDevices() ([]USBDevice, error) {
	var devices []USBDevice

	// 1. 获取磁盘驱动器 - 根据开发模式决定是否只查询USB设备
	var drives []Win32_DiskDrive
	var query string
	if DevMode == "true" {
		// 开发模式：查询所有磁盘驱动器
		query = "SELECT DeviceID, SerialNumber, Model, Size, InterfaceType, Index FROM Win32_DiskDrive"
	} else {
		// 生产模式：只查询USB磁盘驱动器
		query = "SELECT DeviceID, SerialNumber, Model, Size, InterfaceType, Index FROM Win32_DiskDrive WHERE InterfaceType = 'USB'"
	}

	err := wmi.Query(query, &drives)
	if err != nil {
		return nil, fmt.Errorf("查询磁盘驱动器失败: %v", err)
	}

	// 如果没有找到驱动器，返回空列表
	if len(drives) == 0 {
		return devices, nil
	}

	// 2. 获取所有分区信息
	var partitions []Win32_DiskPartition
	query = "SELECT DeviceID, DiskIndex, Size, StartingOffset FROM Win32_DiskPartition"
	err = wmi.Query(query, &partitions)
	if err != nil {
		return nil, fmt.Errorf("查询磁盘分区失败: %v", err)
	}

	// 3. 获取逻辑磁盘到分区的关联
	var diskToPartition []Win32_LogicalDiskToPartition
	query = "SELECT Antecedent, Dependent FROM Win32_LogicalDiskToPartition"
	err = wmi.Query(query, &diskToPartition)
	if err != nil {
		return nil, fmt.Errorf("查询磁盘分区关联失败: %v", err)
	}

	// 4. 获取所有逻辑磁盘
	var logicalDisks []Win32_LogicalDisk
	if DevMode == "true" {
		query = "SELECT DeviceID, Size, FileSystem, VolumeName, DriveType, VolumeSerialNumber FROM Win32_LogicalDisk"
	} else {
		query = "SELECT DeviceID, Size, FileSystem, VolumeName, DriveType, VolumeSerialNumber FROM Win32_LogicalDisk WHERE DriveType = 2"
	}
	err = wmi.Query(query, &logicalDisks)
	if err != nil {
		return nil, fmt.Errorf("查询逻辑磁盘失败: %v", err)
	}

	// 5. 为每个磁盘驱动器查找对应的逻辑磁盘
	for _, drive := range drives {
		// 查找该磁盘驱动器的分区
		var drivePartitions []Win32_DiskPartition
		for _, partition := range partitions {
			if partition.DiskIndex == drive.Index {
				drivePartitions = append(drivePartitions, partition)
			}
		}

		// 如果没有分区，创建一个基本设备信息
		if len(drivePartitions) == 0 {
			device := USBDevice{
				DevicePath:         drive.DeviceID,
				DeviceNode:         "", // Windows中设备节点为空
				SerialNumber:       sanitizeString(drive.SerialNumber),
				Model:              sanitizeString(drive.Model),
				Size:               drive.Size,
				FileSystem:         "",
				Label:              "",
				VolumeSerialNumber: "",
			}

			// 从型号中提取厂商信息
			if parts := strings.Fields(device.Model); len(parts) > 0 {
				device.Vendor = parts[0]
			}

			// 如果最终还是没有序列号，设置为"null"
			if device.SerialNumber == "" {
				device.SerialNumber = "null"
			}

			devices = append(devices, device)
			continue
		}

		// 为每个分区查找对应的逻辑磁盘
		for _, partition := range drivePartitions {
			logicalDisk := findLogicalDiskForPartition(partition.DeviceID, diskToPartition, logicalDisks)

			if logicalDisk != nil {
				device := USBDevice{
					DevicePath:         logicalDisk.DeviceID,
					DeviceNode:         "", // Windows中设备节点为空
					SerialNumber:       sanitizeString(drive.SerialNumber),
					Label:              sanitizeString(logicalDisk.VolumeName),
					Size:               logicalDisk.Size,
					FileSystem:         logicalDisk.FileSystem,
					Model:              sanitizeString(drive.Model),
					VolumeSerialNumber: sanitizeString(logicalDisk.VolumeSerialNumber),
				}

				// 从型号中提取厂商信息
				if parts := strings.Fields(device.Model); len(parts) > 0 {
					device.Vendor = parts[0]
				}

				// 如果最终还是没有序列号，设置为"null"
				if device.SerialNumber == "" {
					device.SerialNumber = "null"
				}

				devices = append(devices, device)
			}
		}
	}

	return devices, nil
}

// findLogicalDiskForPartition 根据分区查找对应的逻辑磁盘
func findLogicalDiskForPartition(partitionID string, diskToPartition []Win32_LogicalDiskToPartition, logicalDisks []Win32_LogicalDisk) *Win32_LogicalDisk {
	// 查找分区对应的逻辑磁盘
	for _, relation := range diskToPartition {
		if strings.Contains(relation.Antecedent, partitionID) {
			// 从 Dependent 中提取逻辑磁盘的 DeviceID
			logicalDeviceID := extractLogicalDeviceID(relation.Dependent)
			if logicalDeviceID == "" {
				continue
			}

			// 查找对应的逻辑磁盘
			for _, disk := range logicalDisks {
				if strings.EqualFold(disk.DeviceID, logicalDeviceID) {
					return &disk
				}
			}
		}
	}

	return nil
}

// extractLogicalDeviceID 从 WMI 路径中提取逻辑磁盘的设备 ID
func extractLogicalDeviceID(wmiPath string) string {
	// WMI 路径格式: \\COMPUTER\root\cimv2:Win32_LogicalDisk.DeviceID="C:"
	start := strings.Index(wmiPath, `DeviceID="`)
	if start == -1 {
		return ""
	}
	start += len(`DeviceID="`)

	end := strings.Index(wmiPath[start:], `"`)
	if end == -1 {
		return ""
	}

	return wmiPath[start : start+end]
}

// sanitizeString 移除字符串中的所有非打印字符
func sanitizeString(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, s)
}
