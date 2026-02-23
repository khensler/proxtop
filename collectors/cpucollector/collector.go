package cpucollector

import (
	"proxtop/config"
	"proxtop/connector"
	"proxtop/models"
)

// Collector describes the cpu collector
type Collector struct {
	models.Collector
}

// Lookup cpu collector data
func (collector *Collector) Lookup() {
	models.Collection.Domains.Range(func(key, value interface{}) bool {
		uuid := key.(string)
		domain := value.(models.Domain)
		if connector.IsProxmox() {
			vmInfo, ok := connector.ProxmoxVMStore.Load(uuid)
			if ok {
				cpuLookupProxmox(&domain, vmInfo)
			}
		} else {
			libvirtDomain, _ := models.Collection.LibvirtDomains.Load(uuid)
			cpuLookup(&domain, libvirtDomain)
		}
		return true
	})

	// lookup details for host
	cpuLookupHost(&models.Collection.Host)
}

// Collect cpu collector data
func (collector *Collector) Collect() {
	// lookup for each domain
	models.Collection.Domains.Range(func(key, value interface{}) bool {
		// uuid := key.(string)
		domain := value.(models.Domain)
		cpuCollect(&domain)
		return true
	})

	// collect host measurements
	cpuCollectHost(&models.Collection.Host)
}

// Print returns the collectors measurements in a Printable struct
func (collector *Collector) Print() models.Printable {

	// Host fields - esxtop style naming
	hostFields := []string{
		"cpu_cores",
		"cpu_curfreq",
		"cpu_%user",
		"cpu_%sys",
		"cpu_%idle",
		"cpu_%steal",
		"cpu_%iowait",
	}
	if config.Options.Verbose {
		hostFields = append(hostFields,
			"cpu_minfreq",
			"cpu_maxfreq",
			"cpu_%nice",
			"cpu_%irq",
			"cpu_%softirq",
			"cpu_%guest",
			"cpu_%guestnice",
		)
	}
	// Domain fields - esxtop style: %USED (cpu time), %RDY (queue/steal time)
	// %sys = CPU time used by other threads (I/O, emulation) - like %SYS in esxtop
	// %othrdy = queue/wait time for other threads
	domainFields := []string{
		"cpu_cores",
		"cpu_%used",
		"cpu_%rdy",
		"cpu_%sys",
	}
	if config.Options.Verbose {
		domainFields = append(domainFields,
			"cpu_%othrdy",
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
		printable.DomainValues[uuid] = cpuPrint(&domain)
		return true
	})

	// lookup for host
	printable.HostValues = cpuPrintHost(&models.Collection.Host)

	return printable
}

// CreateCollector creates a new cpu collector
func CreateCollector() Collector {
	return Collector{}
}
