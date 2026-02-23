package memcollector

import (
	"fmt"

	"proxtop/config"
	"proxtop/models"
	"proxtop/util"
)

// formatHostMemKB formats a host memory value (all /proc/meminfo values are in KB)
func formatHostMemKB(valueKB uint64) string {
	if config.Options.HumanReadable {
		return util.FormatBytes(valueKB * 1024)
	}
	return fmt.Sprintf("%d", valueKB)
}

func hostPrint(host *models.Host) []string {
	// All /proc/meminfo values are in KB
	Total, _ := host.GetMetricUint64Raw("ram_Total", 0)
	Free, _ := host.GetMetricUint64Raw("ram_Free", 0)
	Available, _ := host.GetMetricUint64Raw("ram_Available", 0)
	Buffers, _ := host.GetMetricUint64Raw("ram_Buffers", 0)
	Cached, _ := host.GetMetricUint64Raw("ram_Cached", 0)
	SwapCached, _ := host.GetMetricUint64Raw("ram_SwapCached", 0)
	Active, _ := host.GetMetricUint64Raw("ram_Active", 0)
	Inactive, _ := host.GetMetricUint64Raw("ram_Inactive", 0)
	ActiveAanon, _ := host.GetMetricUint64Raw("ram_ActiveAanon", 0)
	InactiveAanon, _ := host.GetMetricUint64Raw("ram_InactiveAanon", 0)
	ActiveFile, _ := host.GetMetricUint64Raw("ram_ActiveFile", 0)
	InactiveFile, _ := host.GetMetricUint64Raw("ram_InactiveFile", 0)
	Unevictable, _ := host.GetMetricUint64Raw("ram_Unevictable", 0)
	Mlocked, _ := host.GetMetricUint64Raw("ram_Mlocked", 0)
	SwapTotal, _ := host.GetMetricUint64Raw("ram_SwapTotal", 0)
	SwapFree, _ := host.GetMetricUint64Raw("ram_SwapFree", 0)
	Dirty, _ := host.GetMetricUint64Raw("ram_Dirty", 0)
	Writeback, _ := host.GetMetricUint64Raw("ram_Writeback", 0)
	AnonPages, _ := host.GetMetricUint64Raw("ram_AnonPages", 0)
	Mapped, _ := host.GetMetricUint64Raw("ram_Mapped", 0)
	Shmem, _ := host.GetMetricUint64Raw("ram_Shmem", 0)
	Slab, _ := host.GetMetricUint64Raw("ram_Slab", 0)
	SReclaimable, _ := host.GetMetricUint64Raw("ram_SReclaimable", 0)
	SUnreclaim, _ := host.GetMetricUint64Raw("ram_SUnreclaim", 0)
	KernelStack, _ := host.GetMetricUint64Raw("ram_KernelStack", 0)
	PageTables, _ := host.GetMetricUint64Raw("ram_PageTables", 0)
	NFSUnstable, _ := host.GetMetricUint64Raw("ram_NFSUnstable", 0)
	Bounce, _ := host.GetMetricUint64Raw("ram_Bounce", 0)
	WritebackTmp, _ := host.GetMetricUint64Raw("ram_WritebackTmp", 0)
	CommitLimit, _ := host.GetMetricUint64Raw("ram_CommitLimit", 0)
	CommittedAS, _ := host.GetMetricUint64Raw("ram_CommittedAS", 0)
	VmallocTotal, _ := host.GetMetricUint64Raw("ram_VmallocTotal", 0)
	VmallocUsed, _ := host.GetMetricUint64Raw("ram_VmallocUsed", 0)
	VmallocChunk, _ := host.GetMetricUint64Raw("ram_VmallocChunk", 0)
	HardwareCorrupted, _ := host.GetMetricUint64Raw("ram_HardwareCorrupted", 0)
	AnonHugePages, _ := host.GetMetricUint64Raw("ram_AnonHugePages", 0)
	ShmemHugePages, _ := host.GetMetricUint64Raw("ram_ShmemHugePages", 0)
	ShmemPmdMapped, _ := host.GetMetricUint64Raw("ram_ShmemPmdMapped", 0)
	HugePagesTotal, _ := host.GetMetricUint64Raw("ram_HugePagesTotal", 0)
	HugePagesFree, _ := host.GetMetricUint64Raw("ram_HugePagesFree", 0)
	HugePagesRsvd, _ := host.GetMetricUint64Raw("ram_HugePagesRsvd", 0)
	HugePagesSurp, _ := host.GetMetricUint64Raw("ram_HugePagesSurp", 0)
	Hugepagesize, _ := host.GetMetricUint64Raw("ram_Hugepagesize", 0)
	Hugetlb, _ := host.GetMetricUint64Raw("ram_Hugetlb", 0)
	DirectMap4k, _ := host.GetMetricUint64Raw("ram_DirectMap4k", 0)
	DirectMap2M, _ := host.GetMetricUint64Raw("ram_DirectMap2M", 0)
	DirectMap1G, _ := host.GetMetricUint64Raw("ram_DirectMap1G", 0)

	// Calculate swap used
	swapUsed := SwapTotal - SwapFree

	// Format all memory values (all are in KB)
	// esxtop style: Total, Free, Available, Cached, Active, SwapUsed (default)
	result := []string{
		formatHostMemKB(Total),
		formatHostMemKB(Free),
		formatHostMemKB(Available),
		formatHostMemKB(Cached),
		formatHostMemKB(Active),
		formatHostMemKB(swapUsed),
	}
	if config.Options.Verbose {
		result = append(result,
			formatHostMemKB(Buffers),
			formatHostMemKB(SwapCached),
			formatHostMemKB(Inactive),
			formatHostMemKB(ActiveAanon),
			formatHostMemKB(InactiveAanon),
			formatHostMemKB(ActiveFile),
			formatHostMemKB(InactiveFile),
			formatHostMemKB(Unevictable),
			formatHostMemKB(Mlocked),
			formatHostMemKB(SwapTotal),
			formatHostMemKB(SwapFree),
			formatHostMemKB(Dirty),
			formatHostMemKB(Writeback),
			formatHostMemKB(AnonPages),
			formatHostMemKB(Mapped),
			formatHostMemKB(Shmem),
			formatHostMemKB(Slab),
			formatHostMemKB(SReclaimable),
			formatHostMemKB(SUnreclaim),
			formatHostMemKB(KernelStack),
			formatHostMemKB(PageTables),
			formatHostMemKB(NFSUnstable),
			formatHostMemKB(Bounce),
			formatHostMemKB(WritebackTmp),
			formatHostMemKB(CommitLimit),
			formatHostMemKB(CommittedAS),
			formatHostMemKB(VmallocTotal),
			formatHostMemKB(VmallocUsed),
			formatHostMemKB(VmallocChunk),
			formatHostMemKB(HardwareCorrupted),
			formatHostMemKB(AnonHugePages),
			formatHostMemKB(ShmemHugePages),
			formatHostMemKB(ShmemPmdMapped),
			fmt.Sprintf("%d", HugePagesTotal), // HugePages counts are not in KB
			fmt.Sprintf("%d", HugePagesFree),
			fmt.Sprintf("%d", HugePagesRsvd),
			fmt.Sprintf("%d", HugePagesSurp),
			formatHostMemKB(Hugepagesize),
			formatHostMemKB(Hugetlb),
			formatHostMemKB(DirectMap4k),
			formatHostMemKB(DirectMap2M),
			formatHostMemKB(DirectMap1G),
		)
	}
	return result
}
