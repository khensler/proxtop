package util

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"proxtop/config"
)

// GetAllNetDevices reads /proc/net/dev and returns stats for all network interfaces
func GetAllNetDevices() map[string]ProcPIDNetDev {
	stats := make(map[string]ProcPIDNetDev)

	filepath := fmt.Sprint(config.Options.ProcFS, "/net/dev")
	file, err := os.Open(filepath)
	if err != nil {
		return stats
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		// Skip header lines
		if lineNum <= 2 {
			continue
		}

		row := strings.TrimSpace(scanner.Text())
		if row == "" {
			continue
		}

		// Parse interface name (before colon)
		parts := strings.SplitN(row, ":", 2)
		if len(parts) != 2 {
			continue
		}

		devName := strings.TrimSpace(parts[0])
		// Skip loopback and virtual interfaces for physical view
		if devName == "lo" {
			continue
		}

		devStats := ProcPIDNetDev{Dev: devName}
		format := "%d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d"

		_, err := fmt.Sscanf(
			strings.TrimSpace(parts[1]), format,
			&devStats.ReceivedBytes,
			&devStats.ReceivedPackets,
			&devStats.ReceivedErrs,
			&devStats.ReceivedDrop,
			&devStats.ReceivedFifo,
			&devStats.ReceivedFrame,
			&devStats.ReceivedCompressed,
			&devStats.ReceivedMulticast,
			&devStats.TransmittedBytes,
			&devStats.TransmittedPackets,
			&devStats.TransmittedErrs,
			&devStats.TransmittedDrop,
			&devStats.TransmittedFifo,
			&devStats.TransmittedColls,
			&devStats.TransmittedCarrier,
			&devStats.TransmittedCompressed,
		)
		if err == nil {
			stats[devName] = devStats
		}
	}

	return stats
}

// GetPhysicalNetDevices returns only physical network interfaces (filters out tap, veth, etc.)
func GetPhysicalNetDevices() map[string]ProcPIDNetDev {
	all := GetAllNetDevices()
	physical := make(map[string]ProcPIDNetDev)

	for name, stats := range all {
		// Skip virtual interfaces (tap, veth, virbr, docker, vnet, etc.)
		if strings.HasPrefix(name, "tap") ||
			strings.HasPrefix(name, "veth") ||
			strings.HasPrefix(name, "virbr") ||
			strings.HasPrefix(name, "docker") ||
			strings.HasPrefix(name, "vnet") ||
			strings.HasPrefix(name, "br-") ||
			strings.HasPrefix(name, "fwbr") ||
			strings.HasPrefix(name, "fwpr") ||
			strings.HasPrefix(name, "fwln") {
			continue
		}
		physical[name] = stats
	}

	return physical
}

