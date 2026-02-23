package diskcollector

import (
	"strings"

	"proxtop/config"
	"proxtop/connector"
	"proxtop/models"
)

// Collector describes the disk collector
type Collector struct {
	models.Collector
}

// Lookup disk collector data
func (collector *Collector) Lookup() {
	hostDiskSources := ""

	models.Collection.Domains.Range(func(key, value interface{}) bool {
		uuid := key.(string)
		domain := value.(models.Domain)

		if connector.IsProxmox() {
			vmInfo, ok := connector.ProxmoxVMStore.Load(uuid)
			if ok {
				diskLookupProxmox(&domain, vmInfo)
			}
		} else {
			libvirtDomain, _ := models.Collection.LibvirtDomains.Load(uuid)
			diskLookup(&domain, libvirtDomain)
		}

		// merge sourcedir metrics from domains to one metric for host
		disksources := strings.Split(domain.GetMetricString("disk_sources", 0), ",")
		for _, disksource := range disksources {
			if !strings.Contains(hostDiskSources, disksource) {
				if hostDiskSources != "" {
					hostDiskSources += ","
				}
				hostDiskSources += disksource
			}
		}

		return true
	})

	models.Collection.Host.AddMetricMeasurement("disk_sources", models.CreateMeasurement(hostDiskSources))

	diskHostLookup(&models.Collection.Host)
}

// Collect disk collector data
func (collector *Collector) Collect() {
	// lookup for each domain
	models.Collection.Domains.Range(func(key, value interface{}) bool {
		// uuid := key.(string)
		domain := value.(models.Domain)
		diskCollect(&domain, &models.Collection.Host)
		return true
	})
	diskHostCollect(&models.Collection.Host)
}

// Print returns the collectors measurements in a Printable struct
func (collector *Collector) Print() models.Printable {
	// Host fields - esxtop style with key storage metrics
	// %UTIL = device busy percentage (time spent doing I/Os)
	// QDEPTH = current I/O queue depth
	// QLEN = average queue length
	// SVCTM = average service time per I/O (ms)
	// AWAIT = average wait time per I/O (ms)
	hostFields := []string{
		"dsk_READS/s",
		"dsk_WRITES/s",
		"dsk_MBRD/s",
		"dsk_MBWR/s",
		"dsk_%UTIL",
		"dsk_QDEPTH",
		"dsk_QLEN",
		"dsk_SVCTM",
		"dsk_AWAIT",
	}
	// Domain fields - esxtop style with latency breakout
	// LAT/rd, LAT/wr, LAT/fl = average latency for read/write/flush operations (ms)
	// LAT/avg = overall average latency combining all operation types (ms)
	domainFields := []string{
		"dsk_SIZE",
		"dsk_ALLOC",
		"dsk_%UTIL",
		"dsk_READS/s",
		"dsk_WRITES/s",
		"dsk_MBRD/s",
		"dsk_MBWR/s",
		"dsk_LAT/rd",
		"dsk_LAT/wr",
		"dsk_LAT/fl",
		"dsk_LAT/avg",
	}
	if config.Options.Verbose {
		hostFields = append(hostFields,
			"dsk_rdmerged",
			"dsk_sectorsrd",
			"dsk_timerd",
			"dsk_wrmerged",
			"dsk_sectorswr",
			"dsk_timewr",
			"dsk_timeforops",
			"dsk_weightedtime",
			"dsk_count",
		)
		// Verbose adds: physical size, flush ops/s, and total time breakdown (rd/wr/flush in ms)
		domainFields = append(domainFields,
			"dsk_PHYSICAL",
			"dsk_FLUSH/s",
			"dsk_RDTM",    // Total read time (ms)
			"dsk_WRTM",    // Total write time (ms)
			"dsk_FLTM",    // Total flush time (ms)
			"dsk_BLKIO",
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
		printable.DomainValues[uuid] = diskPrint(&domain)
		return true
	})

	// lookup for host
	printable.HostValues = diskPrintHost(&models.Collection.Host)

	return printable
}

// CreateCollector creates a new disk collector
func CreateCollector() Collector {
	return Collector{}
}
