package runners

import (
	"log"
	"strings"
	"sync"
	"time"

	"proxtop/config"
	"proxtop/connector"
	"proxtop/models"
	"proxtop/util"
	libvirt "github.com/libvirt/libvirt-go"
)

var processes []int


// InitializeLookup starts the periodic lookup calls
func InitializeLookup(wg *sync.WaitGroup) {

	for n := -1; config.Options.Runs == -1 || n < config.Options.Runs; n++ {
		// Skip collection when paused (overlay shown)
		if CollectionPaused {
			time.Sleep(100 * time.Millisecond)
			n-- // Don't count paused iterations
			continue
		}

		// execution, then sleep
		start := time.Now()
		Lookup()
		initialLookupDone <- true

		// Fast startup: for first two runs, use very short delay (200ms)
		// This allows differential metrics to be calculated quickly
		// After that, use normal frequency
		var sleepDuration time.Duration
		if n <= 1 {
			sleepDuration = 200 * time.Millisecond
		} else {
			freq := time.Duration(config.Options.Frequency) * time.Second
			nextRun := start.Add(freq)
			sleepDuration = nextRun.Sub(time.Now())
		}
		if sleepDuration > 0 {
			time.Sleep(sleepDuration)
		}
	}
	close(initialLookupDone)
	wg.Done()
}

// Lookup runs one lookup cycle to detect rather static metrics
func Lookup() {
	if connector.IsProxmox() {
		lookupProxmox()
	} else {
		lookupLibvirt()
	}

	// call collector lookup functions in parallel for faster startup
	var lookupWg sync.WaitGroup
	models.Collection.Collectors.Range(func(_ interface{}, collector models.Collector) bool {
		lookupWg.Add(1)
		go func(c models.Collector) {
			defer lookupWg.Done()
			c.Lookup()
		}(collector)
		return true
	})
	lookupWg.Wait()
}

// lookupProxmox discovers VMs using Proxmox connector
func lookupProxmox() {
	proxmoxConn, ok := connector.CurrentConnector.(*connector.ProxmoxConnector)
	if !ok {
		log.Printf("Proxmox connector not available")
		return
	}

	vms, err := proxmoxConn.ListVMs()
	if err != nil {
		log.Printf("Cannot get list of VMs from Proxmox: %v", err)
		return
	}

	// create list of cached domains
	domIDs := make([]string, 0, models.Collection.Domains.Length())
	models.Collection.Domains.Range(func(key, _ interface{}) bool {
		domIDs = append(domIDs, key.(string))
		return true
	})

	// Clear and rebuild Proxmox VM store
	connector.ProxmoxVMStore.Clear()

	// update domain list from Proxmox VMs
	for _, vm := range vms {
		// Store VM info for collectors to use
		connector.ProxmoxVMStore.Store(vm.UUID, vm)

		// Create or update domain
		var domain models.Domain
		var ok bool
		if domain, ok = models.Collection.Domains.Load(vm.UUID); ok {
			domain.Name = vm.Name
			domain.PID = vm.PID
		} else {
			domain = connector.DomainFromVMInfo(vm)
		}

		// write back domain
		models.Collection.Domains.Store(vm.UUID, domain)
		domIDs = util.RemoveFromArray(domIDs, vm.UUID)
	}

	// remove cached but not existent domains
	for _, id := range domIDs {
		models.Collection.Domains.Delete(id)
	}
}

// lookupLibvirt discovers VMs using libvirt connector
func lookupLibvirt() {
	// query libvirt
	doms, err := connector.Libvirt.Connection.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE)
	if err != nil {
		log.Printf("Cannot get list of domains from libvirt.")
		return
	}

	// create list of cached domains
	domIDs := make([]string, 0, models.Collection.Domains.Length())

	models.Collection.Domains.Range(func(key, _ interface{}) bool {
		domIDs = append(domIDs, key.(string))
		return true
	})

	// update process list
	processes = util.GetProcessList()

	// update domain list
	for _, dom := range doms {
		domain, err := handleDomain(dom)
		models.Collection.LibvirtDomains.Store(domain.UUID, dom)
		if err != nil {
			continue
		}
		domIDs = util.RemoveFromArray(domIDs, domain.UUID)
	}

	// remove cached but not existent domains
	for _, id := range domIDs {
		models.Collection.Domains.Delete(id)
	}
}

func handleDomain(dom libvirt.Domain) (models.Domain, error) {
	uuid, err := dom.GetUUIDString()
	if err != nil {
		return models.Domain{}, err
	}

	name, err := dom.GetName()
	if err != nil {
		return models.Domain{}, err
	}

	// lookup or create domain
	var domain models.Domain
	var ok bool
	if domain, ok = models.Collection.Domains.Load(uuid); ok {
		domain.Name = name
	} else {
		domain = models.Domain{
			Measurable: models.NewMeasurable(),
			UUID:       string(uuid),
			Name:       name,
		}
	}

	// lookup PID
	var pid int
	for _, process := range processes {
		cmdline := util.GetCmdLine(process)
		if cmdline != "" && strings.Contains(cmdline, name) {
			// fmt.Printf("Found PID %d for instance %s (cmdline: %s)", process, name, cmdline)
			pid = process
			break
		}
	}
	domain.PID = pid

	// write back domain
	models.Collection.Domains.Store(uuid, domain)

	return domain, nil
}
