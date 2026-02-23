package memcollector

import (
	"proxtop/config"
	"proxtop/connector"
	"proxtop/models"
)

// Collector describes the memory collector
type Collector struct {
	models.Collector
}

const pagesize = 4096

// Lookup memory collector data
func (collector *Collector) Lookup() {
	models.Collection.Domains.Range(func(key, value interface{}) bool {
		uuid := key.(string)
		domain := value.(models.Domain)
		if connector.IsProxmox() {
			vmInfo, ok := connector.ProxmoxVMStore.Load(uuid)
			if ok {
				domainLookupProxmox(&domain, vmInfo)
			}
		} else {
			libvirtDomain, _ := models.Collection.LibvirtDomains.Load(uuid)
			domainLookup(&domain, libvirtDomain)
		}
		return true
	})
	hostLookup(&models.Collection.Host)
}

// Collect memory collector data
func (collector *Collector) Collect() {
	// lookup for each domain
	models.Collection.Domains.Range(func(key, value interface{}) bool {
		// uuid := key.(string)
		domain := value.(models.Domain)
		domainCollect(&domain)
		return true
	})
	hostCollect(&models.Collection.Host)
}

// Print returns the collectors measurements in a Printable struct
func (collector *Collector) Print() models.Printable {
	// Host fields - esxtop style naming
	hostFields := []string{
		"mem_Total",
		"mem_Free",
		"mem_Avail",
		"mem_Cached",
		"mem_Active",
		"mem_SwapUsed",
	}
	// Domain fields - esxtop style: MEMSZ (total), GRANT (used), RSS, page faults
	domainFields := []string{
		"mem_MEMSZ",
		"mem_GRANT",
		"mem_FREE",
		"mem_%ACTV",
		"mem_RSS",
		"mem_MCTL",
		"mem_MINFLT",
		"mem_MAJFLT",
	}
	if config.Options.Verbose {
		hostFields = append(hostFields,
			"mem_Buffers",
			"mem_SwapCached",
			"mem_Inactive",
			"mem_ActiveAnon",
			"mem_InactiveAnon",
			"mem_ActiveFile",
			"mem_InactiveFile",
			"mem_Unevictable",
			"mem_Mlocked",
			"mem_SwapTotal",
			"mem_SwapFree",
			"mem_Dirty",
			"mem_Writeback",
			"mem_AnonPages",
			"mem_Mapped",
			"mem_Shmem",
			"mem_Slab",
			"mem_SReclaimable",
			"mem_SUnreclaim",
			"mem_KernelStack",
			"mem_PageTables",
			"mem_NFSUnstable",
			"mem_Bounce",
			"mem_WritebackTmp",
			"mem_CommitLimit",
			"mem_CommittedAS",
			"mem_VmallocTotal",
			"mem_VmallocUsed",
			"mem_VmallocChunk",
			"mem_HardwareCorrupted",
			"mem_AnonHugePages",
			"mem_ShmemHugePages",
			"mem_ShmemPmdMapped",
			"mem_HugePagesTotal",
			"mem_HugePagesFree",
			"mem_HugePagesRsvd",
			"mem_HugePagesSurp",
			"mem_Hugepagesize",
			"mem_Hugetlb",
			"mem_DirectMap4k",
			"mem_DirectMap2M",
			"mem_DirectMap1G",
		)
		domainFields = append(domainFields,
			"mem_MAXSZ",
			"mem_VSIZE",
			"mem_SWAPIN",
			"mem_SWAPOUT",
			"mem_CMINFLT",
			"mem_CMAJFLT",
		)
	}

	printable := models.Printable{
		HostFields:   hostFields,
		DomainFields: domainFields,
	}

	// lookup for each domain
	printable.DomainValues = make(map[string][]string)
	models.Collection.Domains.Range(func(key, value interface{}) bool {
		uuid := key.(string)
		domain := value.(models.Domain)
		printable.DomainValues[uuid] = domainPrint(&domain)
		return true
	})

	// lookup for host
	printable.HostValues = hostPrint(&models.Collection.Host)

	return printable
}

// CreateCollector creates a new memory collector
func CreateCollector() Collector {
	return Collector{}
}
