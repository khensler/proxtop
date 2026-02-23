package iocollector

import (
	"proxtop/config"
	"proxtop/models"
)

// Collector describes the io collector
type Collector struct {
	models.Collector
}

// Lookup io collector data
func (collector *Collector) Lookup() {
	models.Collection.Domains.Range(func(key, value interface{}) bool {
		uuid := key.(string)
		domain := value.(models.Domain)
		libvirtDomain, _ := models.Collection.LibvirtDomains.Load(uuid)
		ioLookup(&domain, libvirtDomain)
		return true
	})
}

// Collect io collector data
func (collector *Collector) Collect() {
	// lookup for each domain
	models.Collection.Domains.Range(func(key, value interface{}) bool {
		// uuid := key.(string)
		domain := value.(models.Domain)
		ioCollect(&domain)
		return true
	})
}

// Print returns the collectors measurements in a Printable struct
func (collector *Collector) Print() models.Printable {
	// Domain fields - esxtop style: MBRD/s, MBWR/s, IOPS (syscalls)
	domainFields := []string{
		"io_MBRD/s",
		"io_MBWR/s",
		"io_RDOPS",
		"io_WROPS",
	}
	if config.Options.Verbose {
		domainFields = append(domainFields,
			"io_rchar",
			"io_wchar",
			"io_cancelled",
		)
	}
	printable := models.Printable{
		HostFields:   []string{},
		DomainFields: domainFields,
	}

	// lookup for each domain
	printable.DomainValues = make(map[string][]string)
	models.Collection.Domains.Range(func(key, value interface{}) bool {
		uuid := key.(string)
		domain := value.(models.Domain)
		printable.DomainValues[uuid] = ioPrint(&domain)
		return true
	})

	// lookup for host
	// printable.HostValues = cpuPrintHost(host)

	return printable
}

// CreateCollector creates a new cpu collector
func CreateCollector() Collector {
	return Collector{}
}
