package util

import (
	"os"
	"path/filepath"
	"strings"
)

// DeviceMapperInfo contains information about a device mapper device
type DeviceMapperInfo struct {
	DMName       string // The dm-X name (e.g., "dm-0")
	FriendlyName string // The friendly name from /dev/mapper (e.g., "vg-lv" or "mpath0")
	DeviceType   string // "lvm", "mpath", or "other"
}

// GetDeviceMapperNames reads /dev/mapper/ and returns a map of dm-X -> DeviceMapperInfo
// Device type is determined by reading /sys/block/dm-X/dm/uuid:
// - UUIDs starting with "mpath-" are multipath devices
// - UUIDs starting with "LVM-" are LVM devices
func GetDeviceMapperNames() map[string]DeviceMapperInfo {
	result := make(map[string]DeviceMapperInfo)

	mapperDir := "/dev/mapper"
	entries, err := os.ReadDir(mapperDir)
	if err != nil {
		return result
	}

	for _, entry := range entries {
		name := entry.Name()
		// Skip the "control" device
		if name == "control" {
			continue
		}

		// Read the symlink to find the dm-X device
		linkPath := filepath.Join(mapperDir, name)
		target, err := os.Readlink(linkPath)
		if err != nil {
			continue
		}

		// Extract dm-X from the target (e.g., "../dm-0" -> "dm-0")
		dmName := filepath.Base(target)
		if !strings.HasPrefix(dmName, "dm-") {
			continue
		}

		// Determine device type by reading DM UUID from sysfs
		deviceType := getDMDeviceTypeFromSysfs(dmName)

		result[dmName] = DeviceMapperInfo{
			DMName:       dmName,
			FriendlyName: name,
			DeviceType:   deviceType,
		}
	}

	return result
}

// getDMDeviceTypeFromSysfs reads /sys/block/dm-X/dm/uuid to determine device type
func getDMDeviceTypeFromSysfs(dmName string) string {
	uuidPath := filepath.Join("/sys/block", dmName, "dm/uuid")
	data, err := os.ReadFile(uuidPath)
	if err != nil {
		return "other"
	}

	uuid := strings.TrimSpace(string(data))

	// Multipath UUIDs start with "mpath-"
	if strings.HasPrefix(uuid, "mpath-") {
		return "mpath"
	}

	// LVM UUIDs start with "LVM-"
	if strings.HasPrefix(uuid, "LVM-") {
		return "lvm"
	}

	return "other"
}

// GetDMFriendlyName returns the friendly name for a dm-X device, or the original name if not found
func GetDMFriendlyName(dmName string, dmMap map[string]DeviceMapperInfo) string {
	if info, ok := dmMap[dmName]; ok {
		return info.FriendlyName
	}
	return dmName
}

// GetDMDeviceType returns the device type for a dm-X device ("lvm", "mpath", or "other")
func GetDMDeviceType(dmName string, dmMap map[string]DeviceMapperInfo) string {
	if info, ok := dmMap[dmName]; ok {
		return info.DeviceType
	}
	return "other"
}

// ClassifyDiskDevice classifies a disk device into a category
// Returns: "physical", "lvm", "mpath", or "skip"
func ClassifyDiskDevice(name string, dmMap map[string]DeviceMapperInfo) string {
	// Skip loop and ram devices
	if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") {
		return "skip"
	}

	// Device mapper devices
	if strings.HasPrefix(name, "dm-") {
		return GetDMDeviceType(name, dmMap)
	}

	// Physical disks: sd*, nvme*, vd*, hd*, xvd*, etc.
	if strings.HasPrefix(name, "sd") ||
		strings.HasPrefix(name, "nvme") ||
		strings.HasPrefix(name, "vd") ||
		strings.HasPrefix(name, "hd") ||
		strings.HasPrefix(name, "xvd") {
		return "physical"
	}

	// Default to physical for unknown device types
	return "physical"
}

