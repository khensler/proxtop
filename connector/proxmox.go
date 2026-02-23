package connector

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// vmStatusCache caches qm status --verbose output per VM
type vmStatusCache struct {
	output    string
	timestamp time.Time
}

// statusCacheMu protects the status cache
var statusCacheMu sync.RWMutex

// statusCache holds cached qm status output per VMID
var statusCache = make(map[string]vmStatusCache)

// statusCacheTTL is how long cached status is valid
const statusCacheTTL = 1 * time.Second

// QMP response types
type qmpGreeting struct {
	QMP struct {
		Version struct {
			Qemu struct {
				Major int `json:"major"`
				Minor int `json:"minor"`
				Micro int `json:"micro"`
			} `json:"qemu"`
		} `json:"version"`
	} `json:"QMP"`
}

type qmpResponse struct {
	Return json.RawMessage `json:"return"`
	Error  *struct {
		Class string `json:"class"`
		Desc  string `json:"desc"`
	} `json:"error"`
}

type qmpBalloonStats struct {
	Actual          uint64 `json:"actual"`
	TotalMem        uint64 `json:"total_mem"`
	FreeMem         uint64 `json:"free_mem"`
	MaxMem          uint64 `json:"max_mem"`
	MinorPageFaults uint64 `json:"minor_page_faults"`
	MajorPageFaults uint64 `json:"major_page_faults"`
}

// ExtendedMemStats holds detailed memory statistics for a VM
type ExtendedMemStats struct {
	TotalKB      uint64 // Total memory in KB (MEMSZ)
	UsedKB       uint64 // Used memory in KB (GRANT)
	FreeKB       uint64 // Free memory in KB
	MaxKB        uint64 // Max configured memory in KB
	ActualKB     uint64 // Actual balloon size in KB (MCTL)
	SwappedIn    uint64 // Memory swapped in (bytes)
	SwappedOut   uint64 // Memory swapped out (bytes)
	ActivePct    float64 // Active memory percentage
}

type qmpBlockStats struct {
	Stats struct {
		RdBytes        int64 `json:"rd_bytes"`
		WrBytes        int64 `json:"wr_bytes"`
		RdOperations   int64 `json:"rd_operations"`
		WrOperations   int64 `json:"wr_operations"`
		FlushOps       int64 `json:"flush_operations"`
		RdTotalTimeNs  int64 `json:"rd_total_time_ns"`
		WrTotalTimeNs  int64 `json:"wr_total_time_ns"`
		FlushTotalTime int64 `json:"flush_total_time_ns"`
	} `json:"stats"`
	NodeName string `json:"node-name"`
	Qdev     string `json:"qdev"`
}

// qmpCache caches QMP query results per VM
type qmpCache struct {
	balloon    *qmpBalloonStats
	blockstats []qmpBlockStats
	timestamp  time.Time
}

// qmpCacheMu protects the QMP cache
var qmpCacheMu sync.RWMutex

// qmpCacheMap holds cached QMP results per VMID
var qmpCacheMap = make(map[string]*qmpCache)

// qmpCacheTTL is how long cached QMP data is valid
const qmpCacheTTL = 500 * time.Millisecond

// ProxmoxConnector implements Connector for Proxmox VE
type ProxmoxConnector struct {
	nodeName string
}

// ProxmoxVM represents a VM from qm list output
type ProxmoxVM struct {
	VMID   string
	Name   string
	Status string
	PID    int
}

// Initialize connects to Proxmox (verifies qm command is available)
func (p *ProxmoxConnector) Initialize() error {
	// Check if qm command exists
	_, err := exec.LookPath("qm")
	if err != nil {
		return fmt.Errorf("qm command not found - not a Proxmox host")
	}

	// Get node name
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %v", err)
	}
	p.nodeName = hostname

	log.Printf("Proxmox connector initialized on node: %s", p.nodeName)
	return nil
}

// Close closes the Proxmox connection (no-op for Proxmox)
func (p *ProxmoxConnector) Close() error {
	return nil
}

// Name returns the connector name
func (p *ProxmoxConnector) Name() string {
	return "proxmox"
}

// ListVMs returns a list of running VMs on Proxmox
func (p *ProxmoxConnector) ListVMs() ([]VMInfo, error) {
	var vms []VMInfo

	// List all .pid files in /var/run/qemu-server/
	pidFiles, err := filepath.Glob("/var/run/qemu-server/*.pid")
	if err != nil {
		return nil, fmt.Errorf("failed to list pid files: %v", err)
	}

	for _, pidFile := range pidFiles {
		// Extract VMID from filename (e.g., "105.pid" -> "105")
		base := filepath.Base(pidFile)
		vmid := strings.TrimSuffix(base, ".pid")
		if _, err := strconv.Atoi(vmid); err != nil {
			continue // Not a numeric VMID
		}

		vm, err := p.getVMInfoByVMID(vmid)
		if err != nil {
			log.Printf("Failed to get VM info for VMID %s: %v", vmid, err)
			continue
		}
		vms = append(vms, vm)
	}

	return vms, nil
}

// getVMInfoByVMID retrieves VM information for a specific VMID
func (p *ProxmoxConnector) getVMInfoByVMID(vmid string) (VMInfo, error) {
	vm := VMInfo{VMID: vmid}

	// Read PID from pid file
	pidFile := fmt.Sprintf("/var/run/qemu-server/%s.pid", vmid)
	pidData, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return vm, fmt.Errorf("failed to read pid file: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		return vm, fmt.Errorf("failed to parse pid: %v", err)
	}
	vm.PID = pid

	// Read VM config for name and UUID
	configFile := fmt.Sprintf("/etc/pve/qemu-server/%s.conf", vmid)
	config, err := p.parseVMConfig(configFile)
	if err != nil {
		// Try to get name from cmdline instead
		vm.Name = p.getVMNameFromCmdline(pid, vmid)
		vm.UUID = fmt.Sprintf("proxmox-%s", vmid) // Generate UUID
	} else {
		vm.Name = config["name"]
		if uuid, ok := config["smbios1"]; ok {
			// Parse UUID from smbios1 line: uuid=320e642e-bbd4-4513-8bef-20552bc042de
			if strings.Contains(uuid, "uuid=") {
				parts := strings.Split(uuid, "uuid=")
				if len(parts) > 1 {
					uuidPart := strings.Split(parts[1], ",")[0]
					vm.UUID = uuidPart
				}
			}
		}
		if vm.UUID == "" {
			vm.UUID = fmt.Sprintf("proxmox-%s", vmid)
		}

		// Get cores
		if cores, ok := config["cores"]; ok {
			if c, err := strconv.Atoi(cores); err == nil {
				vm.Cores = c
			}
		}
		if sockets, ok := config["sockets"]; ok {
			if s, err := strconv.Atoi(sockets); err == nil {
				vm.Cores *= s
			}
		}

		// Get memory
		if mem, ok := config["memory"]; ok {
			if m, err := strconv.ParseUint(mem, 10, 64); err == nil {
				vm.MemoryTotal = m * 1024 // Convert MB to KB
			}
		}

		// Get network interfaces
		vm.Interfaces = p.getNetworkInterfaces(vmid)
	}

	return vm, nil
}

// parseVMConfig parses a Proxmox VM config file
func (p *ProxmoxConnector) parseVMConfig(configFile string) (map[string]string, error) {
	config := make(map[string]string)

	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			config[key] = value
		}
	}

	return config, scanner.Err()
}

// getVMNameFromCmdline extracts VM name from process cmdline
func (p *ProxmoxConnector) getVMNameFromCmdline(pid int, vmid string) string {
	cmdlineFile := fmt.Sprintf("/proc/%d/cmdline", pid)
	data, err := ioutil.ReadFile(cmdlineFile)
	if err != nil {
		return fmt.Sprintf("vm-%s", vmid)
	}
	cmdline := string(data)
	// Look for -name argument
	re := regexp.MustCompile(`-name\x00([^\x00,]+)`)
	matches := re.FindStringSubmatch(cmdline)
	if len(matches) > 1 {
		return matches[1]
	}
	return fmt.Sprintf("vm-%s", vmid)
}

// getNetworkInterfaces returns tap interface names for a VM
func (p *ProxmoxConnector) getNetworkInterfaces(vmid string) []string {
	var interfaces []string
	// Proxmox uses tapXXXiY naming convention (e.g., tap105i0, tap105i1)
	pattern := fmt.Sprintf("/sys/class/net/tap%si*", vmid)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return interfaces
	}
	for _, match := range matches {
		interfaces = append(interfaces, filepath.Base(match))
	}
	return interfaces
}

// GetVMInfo returns detailed information about a specific VM
func (p *ProxmoxConnector) GetVMInfo(uuid string) (VMInfo, error) {
	// Find VM by UUID in store
	vm, ok := ProxmoxVMStore.Load(uuid)
	if !ok {
		return VMInfo{}, fmt.Errorf("VM with UUID %s not found", uuid)
	}
	return vm, nil
}

// GetCPUThreads returns the vCPU thread IDs for a VM
// Uses /proc filesystem to avoid interactive qm monitor which interferes with ncurses
func (p *ProxmoxConnector) GetCPUThreads(vm VMInfo) ([]int, error) {
	var threads []int

	if vm.PID == 0 {
		return threads, fmt.Errorf("no PID for VM %s", vm.VMID)
	}

	// Read task directory to get all thread IDs
	taskDir := fmt.Sprintf("/proc/%d/task", vm.PID)
	entries, err := ioutil.ReadDir(taskDir)
	if err != nil {
		return threads, err
	}

	// Check each thread's comm to identify vCPU threads
	// vCPU threads have names like "CPU 0/KVM", "CPU 1/KVM", etc.
	for _, entry := range entries {
		tid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		// Read thread's comm (command name)
		commPath := fmt.Sprintf("/proc/%d/task/%d/comm", vm.PID, tid)
		commData, err := ioutil.ReadFile(commPath)
		if err != nil {
			continue
		}

		comm := strings.TrimSpace(string(commData))
		// vCPU threads have names like "CPU 0/KVM"
		if strings.Contains(comm, "CPU") && strings.Contains(comm, "/KVM") {
			threads = append(threads, tid)
		}
	}

	return threads, nil
}

// getCachedVMStatus returns cached qm status output or fetches fresh if cache expired
func (p *ProxmoxConnector) getCachedVMStatus(vmid string) (string, error) {
	now := time.Now()

	// Check cache first
	statusCacheMu.RLock()
	cached, exists := statusCache[vmid]
	statusCacheMu.RUnlock()

	if exists && now.Sub(cached.timestamp) < statusCacheTTL {
		return cached.output, nil
	}

	// Cache miss or expired - fetch fresh
	cmd := exec.Command("qm", "status", vmid, "--verbose")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	outputStr := string(output)

	// Update cache
	statusCacheMu.Lock()
	statusCache[vmid] = vmStatusCache{
		output:    outputStr,
		timestamp: now,
	}
	statusCacheMu.Unlock()

	return outputStr, nil
}

// queryQMP sends commands to QMP socket and returns balloon and blockstats
func (p *ProxmoxConnector) queryQMP(vmid string) (*qmpBalloonStats, []qmpBlockStats, error) {
	now := time.Now()

	// Check cache first
	qmpCacheMu.RLock()
	cached, exists := qmpCacheMap[vmid]
	qmpCacheMu.RUnlock()

	if exists && now.Sub(cached.timestamp) < qmpCacheTTL {
		return cached.balloon, cached.blockstats, nil
	}

	socketPath := fmt.Sprintf("/var/run/qemu-server/%s.qmp", vmid)

	// Connect to QMP socket with timeout
	conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to QMP socket: %v", err)
	}
	defer conn.Close()

	// Set deadline for all operations
	conn.SetDeadline(time.Now().Add(1 * time.Second))

	reader := bufio.NewReader(conn)

	// Read greeting
	_, err = reader.ReadBytes('\n')
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read QMP greeting: %v", err)
	}

	// Send qmp_capabilities
	_, err = conn.Write([]byte(`{"execute": "qmp_capabilities"}` + "\n"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send qmp_capabilities: %v", err)
	}

	// Read response
	_, err = reader.ReadBytes('\n')
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read qmp_capabilities response: %v", err)
	}

	// Query balloon stats
	_, err = conn.Write([]byte(`{"execute": "query-balloon"}` + "\n"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send query-balloon: %v", err)
	}

	balloonLine, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read balloon response: %v", err)
	}

	var balloonResp qmpResponse
	if err := json.Unmarshal(balloonLine, &balloonResp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse balloon response: %v", err)
	}

	var balloon *qmpBalloonStats
	if balloonResp.Error == nil && balloonResp.Return != nil {
		var b qmpBalloonStats
		if err := json.Unmarshal(balloonResp.Return, &b); err == nil {
			balloon = &b
		}
	}

	// Query blockstats
	_, err = conn.Write([]byte(`{"execute": "query-blockstats"}` + "\n"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send query-blockstats: %v", err)
	}

	blockLine, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read blockstats response: %v", err)
	}

	var blockResp qmpResponse
	if err := json.Unmarshal(blockLine, &blockResp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse blockstats response: %v", err)
	}

	var blockstats []qmpBlockStats
	if blockResp.Error == nil && blockResp.Return != nil {
		json.Unmarshal(blockResp.Return, &blockstats)
	}

	// Update cache
	qmpCacheMu.Lock()
	qmpCacheMap[vmid] = &qmpCache{
		balloon:    balloon,
		blockstats: blockstats,
		timestamp:  now,
	}
	qmpCacheMu.Unlock()

	return balloon, blockstats, nil
}

// GetMemoryStats returns memory statistics for a VM via QMP
func (p *ProxmoxConnector) GetMemoryStats(vm VMInfo) (total, used uint64, err error) {
	// Try QMP first (much faster)
	balloon, _, qmpErr := p.queryQMP(vm.VMID)
	if qmpErr == nil && balloon != nil {
		// Convert from bytes to KB
		total = balloon.TotalMem / 1024
		if balloon.FreeMem > 0 && balloon.TotalMem > balloon.FreeMem {
			used = (balloon.TotalMem - balloon.FreeMem) / 1024
		}
		return total, used, nil
	}

	// Fallback to qm status if QMP fails
	output, err := p.getCachedVMStatus(vm.VMID)
	if err != nil {
		return 0, 0, err
	}

	lines := strings.Split(output, "\n")
	var maxMem, freeMem, totalMem uint64

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "max_mem:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				maxMem, _ = strconv.ParseUint(parts[1], 10, 64)
			}
		} else if strings.HasPrefix(line, "free_mem:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				freeMem, _ = strconv.ParseUint(parts[1], 10, 64)
			}
		} else if strings.HasPrefix(line, "total_mem:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				totalMem, _ = strconv.ParseUint(parts[1], 10, 64)
			}
		}
	}

	// Convert from bytes to KB
	total = totalMem / 1024
	if totalMem > 0 && freeMem > 0 {
		used = (totalMem - freeMem) / 1024
	}

	// Fallback to max_mem if total_mem not available
	if total == 0 && maxMem > 0 {
		total = maxMem / 1024
	}

	return total, used, nil
}

// GetExtendedMemoryStats returns detailed memory statistics for a VM
func (p *ProxmoxConnector) GetExtendedMemoryStats(vm VMInfo) (ExtendedMemStats, error) {
	stats := ExtendedMemStats{}

	// Try QMP first (much faster)
	balloon, _, qmpErr := p.queryQMP(vm.VMID)
	if qmpErr == nil && balloon != nil {
		stats.TotalKB = balloon.TotalMem / 1024
		stats.MaxKB = balloon.MaxMem / 1024
		stats.ActualKB = balloon.Actual / 1024
		stats.FreeKB = balloon.FreeMem / 1024
		if balloon.TotalMem > balloon.FreeMem {
			stats.UsedKB = (balloon.TotalMem - balloon.FreeMem) / 1024
		}
		if balloon.TotalMem > 0 {
			stats.ActivePct = float64(balloon.TotalMem-balloon.FreeMem) / float64(balloon.TotalMem) * 100
		}
		return stats, nil
	}

	// Fallback to qm status
	output, err := p.getCachedVMStatus(vm.VMID)
	if err != nil {
		return stats, err
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "max_mem:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				val, _ := strconv.ParseUint(parts[1], 10, 64)
				stats.MaxKB = val / 1024
			}
		} else if strings.HasPrefix(line, "free_mem:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				val, _ := strconv.ParseUint(parts[1], 10, 64)
				stats.FreeKB = val / 1024
			}
		} else if strings.HasPrefix(line, "total_mem:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				val, _ := strconv.ParseUint(parts[1], 10, 64)
				stats.TotalKB = val / 1024
			}
		} else if strings.HasPrefix(line, "mem_swapped_in:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				stats.SwappedIn, _ = strconv.ParseUint(parts[1], 10, 64)
			}
		} else if strings.HasPrefix(line, "mem_swapped_out:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				stats.SwappedOut, _ = strconv.ParseUint(parts[1], 10, 64)
			}
		}
	}

	if stats.TotalKB > stats.FreeKB {
		stats.UsedKB = stats.TotalKB - stats.FreeKB
	}
	if stats.TotalKB > 0 {
		stats.ActivePct = float64(stats.UsedKB) / float64(stats.TotalKB) * 100
	}

	return stats, nil
}

// GetDiskStats returns disk statistics for a VM via QMP or qm status fallback
func (p *ProxmoxConnector) GetDiskStats(vm VMInfo) (DiskStatsInfo, error) {
	stats := DiskStatsInfo{}

	// Try QMP first (much faster)
	_, blockstats, qmpErr := p.queryQMP(vm.VMID)
	if qmpErr == nil && len(blockstats) > 0 {
		// Sum stats from all block devices (usually just one with qdev set)
		for _, bs := range blockstats {
			// Only count devices with qdev (actual VM disks, not backing stores)
			if bs.Qdev != "" {
				stats.RdBytes += bs.Stats.RdBytes
				stats.WrBytes += bs.Stats.WrBytes
				stats.RdReq += bs.Stats.RdOperations
				stats.WrReq += bs.Stats.WrOperations
				stats.FlushReq += bs.Stats.FlushOps
				stats.RdTotalTimes += bs.Stats.RdTotalTimeNs
				stats.WrTotalTimes += bs.Stats.WrTotalTimeNs
				stats.FlushTotalTimes += bs.Stats.FlushTotalTime
			}
		}
	} else {
		// Fallback to qm status if QMP fails
		output, err := p.getCachedVMStatus(vm.VMID)
		if err != nil {
			return stats, err
		}

		// Parse blockstat section from output
		lines := strings.Split(output, "\n")
		inBlockstat := false

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "blockstat:") {
				inBlockstat = true
				continue
			}
			if inBlockstat && !strings.HasPrefix(line, " ") && line != "" && !strings.HasSuffix(line, ":") {
				break
			}
			if inBlockstat {
				if strings.HasPrefix(line, "rd_bytes:") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						stats.RdBytes, _ = strconv.ParseInt(parts[1], 10, 64)
					}
				} else if strings.HasPrefix(line, "wr_bytes:") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						stats.WrBytes, _ = strconv.ParseInt(parts[1], 10, 64)
					}
				} else if strings.HasPrefix(line, "rd_operations:") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						stats.RdReq, _ = strconv.ParseInt(parts[1], 10, 64)
					}
				} else if strings.HasPrefix(line, "wr_operations:") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						stats.WrReq, _ = strconv.ParseInt(parts[1], 10, 64)
					}
				} else if strings.HasPrefix(line, "flush_operations:") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						stats.FlushReq, _ = strconv.ParseInt(parts[1], 10, 64)
					}
				} else if strings.HasPrefix(line, "rd_total_time_ns:") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						stats.RdTotalTimes, _ = strconv.ParseInt(parts[1], 10, 64)
					}
				} else if strings.HasPrefix(line, "wr_total_time_ns:") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						stats.WrTotalTimes, _ = strconv.ParseInt(parts[1], 10, 64)
					}
				} else if strings.HasPrefix(line, "flush_total_time_ns:") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						stats.FlushTotalTimes, _ = strconv.ParseInt(parts[1], 10, 64)
					}
				}
			}
		}
	}

	// Get disk size from config
	configFile := fmt.Sprintf("/etc/pve/qemu-server/%s.conf", vm.VMID)
	config, err := p.parseVMConfig(configFile)
	if err == nil {
		// Parse scsi0, virtio0, ide0, etc. for disk size
		for key, value := range config {
			if strings.HasPrefix(key, "scsi") || strings.HasPrefix(key, "virtio") || strings.HasPrefix(key, "ide") || strings.HasPrefix(key, "sata") {
				// Parse size from value like "local:vm-105-disk-0,size=150G"
				if strings.Contains(value, "size=") {
					re := regexp.MustCompile(`size=(\d+)([GMTK]?)`)
					matches := re.FindStringSubmatch(value)
					if len(matches) >= 2 {
						size, _ := strconv.ParseUint(matches[1], 10, 64)
						unit := "G"
						if len(matches) >= 3 {
							unit = matches[2]
						}
						switch unit {
						case "T":
							size *= 1024 * 1024 * 1024 * 1024
						case "G":
							size *= 1024 * 1024 * 1024
						case "M":
							size *= 1024 * 1024
						case "K":
							size *= 1024
						default:
							size *= 1024 * 1024 * 1024 // Default to GB
						}
						stats.Capacity += size
					}
				}
			}
		}
	}

	return stats, nil
}

// GetNetworkInterfaces returns network interface names for a VM
func (p *ProxmoxConnector) GetNetworkInterfaces(vm VMInfo) ([]string, error) {
	return p.getNetworkInterfaces(vm.VMID), nil
}

// GetPerDiskStats returns per-disk statistics for a VM via QMP
// Returns a map of disk device name -> DiskStatsInfo
func (p *ProxmoxConnector) GetPerDiskStats(vm VMInfo) (map[string]DiskStatsInfo, []string, error) {
	result := make(map[string]DiskStatsInfo)
	diskNames := []string{}

	// Try QMP first (much faster)
	_, blockstats, qmpErr := p.queryQMP(vm.VMID)
	if qmpErr == nil && len(blockstats) > 0 {
		for _, bs := range blockstats {
			// Only count devices with qdev (actual VM disks, not backing stores)
			if bs.Qdev != "" {
				diskName := bs.Qdev // e.g., "scsi0", "virtio0", etc.
				diskNames = append(diskNames, diskName)
				result[diskName] = DiskStatsInfo{
					RdBytes:         bs.Stats.RdBytes,
					WrBytes:         bs.Stats.WrBytes,
					RdReq:           bs.Stats.RdOperations,
					WrReq:           bs.Stats.WrOperations,
					FlushReq:        bs.Stats.FlushOps,
					RdTotalTimes:    bs.Stats.RdTotalTimeNs,
					WrTotalTimes:    bs.Stats.WrTotalTimeNs,
					FlushTotalTimes: bs.Stats.FlushTotalTime,
				}
			}
		}
	}

	return result, diskNames, nil
}

// DetectProxmox checks if the current system is a Proxmox host
func DetectProxmox() bool {
	// Check for Proxmox-specific paths
	if _, err := os.Stat("/etc/pve"); err == nil {
		return true
	}
	if _, err := exec.LookPath("qm"); err == nil {
		return true
	}
	return false
}

// QMStatusOutput represents parsed qm status --verbose output
type QMStatusOutput struct {
	BalloonInfo struct {
		MaxMem    uint64 `json:"max_mem"`
		FreeMem   uint64 `json:"free_mem"`
		TotalMem  uint64 `json:"total_mem"`
	}
	BlockStats map[string]struct {
		RdBytes     int64 `json:"rd_bytes"`
		WrBytes     int64 `json:"wr_bytes"`
		RdOps       int64 `json:"rd_operations"`
		WrOps       int64 `json:"wr_operations"`
		FlushOps    int64 `json:"flush_operations"`
	}
}

// NewProxmoxConnector creates a new Proxmox connector
func NewProxmoxConnector() *ProxmoxConnector {
	return &ProxmoxConnector{}
}
