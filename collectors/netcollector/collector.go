package netcollector

import (
	"proxtop/config"
	"proxtop/connector"
	"proxtop/models"
)

// Collector describes the network collector
type Collector struct {
	models.Collector
}

// Lookup network collector data
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

// Collect network collector data
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
	// Host fields - esxtop style
	hostFields := []string{
		"net_MbRX/s",
		"net_MbTX/s",
		"net_PKTRX/s",
		"net_PKTTX/s",
		"net_speed",
	}
	// Domain fields - esxtop style: MbRX/s, MbTX/s, PKTRX/s, PKTTX/s, %DRPRX, %DRPTX
	domainFields := []string{
		"net_MbRX/s",
		"net_MbTX/s",
		"net_PKTRX/s",
		"net_PKTTX/s",
		"net_%DRPRX",
		"net_%DRPTX",
	}
	if config.Options.Verbose {
		hostFields = append(hostFields,
			"net_host_errsRX",
			"net_host_dropRX",
			"net_host_fifoRX",
			"net_host_frameRX",
			"net_host_compRX",
			"net_host_mcastRX",
			"net_host_errsTX",
			"net_host_dropTX",
			"net_host_fifoTX",
			"net_host_collsTX",
			"net_host_carrierTX",
			"net_host_compTX",
		)
		domainFields = append(domainFields,
			"net_errsRX",
			"net_fifoRX",
			"net_frameRX",
			"net_compRX",
			"net_mcastRX",
			"net_errsTX",
			"net_fifoTX",
			"net_collsTX",
			"net_carrierTX",
			"net_compTX",
			"net_interfaces",
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

// CreateCollector creates a new network collector
func CreateCollector() Collector {
	return Collector{}
}
