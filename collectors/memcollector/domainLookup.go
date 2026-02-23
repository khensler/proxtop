package memcollector

import (
	"proxtop/connector"
	"proxtop/models"
	libvirt "github.com/libvirt/libvirt-go"
)

func domainLookup(domain *models.Domain, libvirtDomain libvirt.Domain) {
	memStats, err := libvirtDomain.MemoryStats(uint32(libvirt.DOMAIN_MEMORY_STAT_NR), 0)
	if err != nil {
		return
	}
	var total, unused, used uint64
	for _, stat := range memStats {
		if stat.Tag == int32(libvirt.DOMAIN_MEMORY_STAT_UNUSED) {
			unused = stat.Val
		}
		if stat.Tag == int32(libvirt.DOMAIN_MEMORY_STAT_AVAILABLE) {
			total = stat.Val
		}
	}
	used = total - unused
	newMeasurementTotal := models.CreateMeasurement(total)
	domain.AddMetricMeasurement("ram_total", newMeasurementTotal)
	newMeasurementUsed := models.CreateMeasurement(used)
	domain.AddMetricMeasurement("ram_used", newMeasurementUsed)

}

// domainLookupProxmox handles memory lookup for Proxmox VMs
func domainLookupProxmox(domain *models.Domain, vmInfo connector.VMInfo) {
	var total, used uint64
	var maxMem, actualMem, freeMem uint64
	var swapIn, swapOut uint64
	var activePct float64

	// Try to get extended memory stats from Proxmox
	proxmoxConn, ok := connector.CurrentConnector.(*connector.ProxmoxConnector)
	if ok {
		extStats, err := proxmoxConn.GetExtendedMemoryStats(vmInfo)
		if err == nil {
			total = extStats.TotalKB
			used = extStats.UsedKB
			maxMem = extStats.MaxKB
			actualMem = extStats.ActualKB
			freeMem = extStats.FreeKB
			swapIn = extStats.SwappedIn
			swapOut = extStats.SwappedOut
			activePct = extStats.ActivePct
		}
	}

	// Fallback to VM config memory if not available from qm status
	if total == 0 && vmInfo.MemoryTotal > 0 {
		total = vmInfo.MemoryTotal
	}

	domain.AddMetricMeasurement("ram_total", models.CreateMeasurement(total))
	domain.AddMetricMeasurement("ram_used", models.CreateMeasurement(used))
	domain.AddMetricMeasurement("ram_max", models.CreateMeasurement(maxMem))
	domain.AddMetricMeasurement("ram_actual", models.CreateMeasurement(actualMem))
	domain.AddMetricMeasurement("ram_free", models.CreateMeasurement(freeMem))
	domain.AddMetricMeasurement("ram_swapin", models.CreateMeasurement(swapIn))
	domain.AddMetricMeasurement("ram_swapout", models.CreateMeasurement(swapOut))
	domain.AddMetricMeasurement("ram_activepct", models.CreateMeasurement(activePct))
}
