package connector

import (
	"proxtop/models"
)

// VMInfo represents common VM information from any hypervisor
type VMInfo struct {
	UUID        string
	Name        string
	VMID        string // Proxmox VMID (e.g., "105")
	PID         int
	Cores       int
	MemoryTotal uint64 // in KB
	MemoryUsed  uint64 // in KB
	Interfaces  []string
	DiskStats   DiskStatsInfo
	CPUThreads  []int // vCPU thread IDs
}

// DiskStatsInfo contains disk statistics
type DiskStatsInfo struct {
	Capacity       uint64
	Allocation     uint64
	Physical       uint64
	RdBytes        int64
	RdReq          int64
	WrBytes        int64
	WrReq          int64
	FlushReq       int64
	RdTotalTimes   int64
	WrTotalTimes   int64
	FlushTotalTimes int64
}

// Connector is the interface for hypervisor connectors
type Connector interface {
	// Initialize connects to the hypervisor
	Initialize() error
	// Close closes the connection
	Close() error
	// ListVMs returns a list of running VMs
	ListVMs() ([]VMInfo, error)
	// GetVMInfo returns detailed information about a specific VM
	GetVMInfo(uuid string) (VMInfo, error)
	// GetCPUThreads returns the vCPU thread IDs for a VM
	GetCPUThreads(vm VMInfo) ([]int, error)
	// GetMemoryStats returns memory statistics for a VM
	GetMemoryStats(vm VMInfo) (total, used uint64, err error)
	// GetDiskStats returns disk statistics for a VM
	GetDiskStats(vm VMInfo) (DiskStatsInfo, error)
	// GetNetworkInterfaces returns network interface names for a VM
	GetNetworkInterfaces(vm VMInfo) ([]string, error)
	// Name returns the connector name
	Name() string
}

// CurrentConnector holds the active connector
var CurrentConnector Connector

// ConnectorType represents the type of hypervisor
type ConnectorType int

const (
	ConnectorTypeLibvirt ConnectorType = iota
	ConnectorTypeProxmox
)

// CurrentConnectorType holds the current connector type
var CurrentConnectorType ConnectorType

// IsProxmox returns true if using Proxmox connector
func IsProxmox() bool {
	return CurrentConnectorType == ConnectorTypeProxmox
}

// IsLibvirt returns true if using Libvirt connector
func IsLibvirt() bool {
	return CurrentConnectorType == ConnectorTypeLibvirt
}

// ProxmoxVMs stores VM info for Proxmox (similar to LibvirtDomains)
type ProxmoxVMs struct {
	vms map[string]VMInfo
}

// NewProxmoxVMs creates a new ProxmoxVMs store
func NewProxmoxVMs() *ProxmoxVMs {
	return &ProxmoxVMs{
		vms: make(map[string]VMInfo),
	}
}

// Store adds a VM to the store
func (p *ProxmoxVMs) Store(uuid string, vm VMInfo) {
	p.vms[uuid] = vm
}

// Load retrieves a VM from the store
func (p *ProxmoxVMs) Load(uuid string) (VMInfo, bool) {
	vm, ok := p.vms[uuid]
	return vm, ok
}

// Range iterates over all VMs
func (p *ProxmoxVMs) Range(f func(uuid string, vm VMInfo) bool) {
	for uuid, vm := range p.vms {
		if !f(uuid, vm) {
			break
		}
	}
}

// Clear removes all VMs from the store
func (p *ProxmoxVMs) Clear() {
	p.vms = make(map[string]VMInfo)
}

// ProxmoxVMStore is the global store for Proxmox VM info
var ProxmoxVMStore = NewProxmoxVMs()

// DomainFromVMInfo creates a models.Domain from VMInfo
func DomainFromVMInfo(vm VMInfo) models.Domain {
	return models.Domain{
		Measurable: models.NewMeasurable(),
		UUID:       vm.UUID,
		Name:       vm.Name,
		PID:        vm.PID,
	}
}

