package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"proxtop/config"
	"proxtop/connector"
	"proxtop/profiler"
	"proxtop/runners"
)

func main() {

	// handle flags
	initializeFlags()

	// Determine which connector to use
	// Auto-detect by default, unless explicitly specified
	useProxmox := false
	if config.Options.Libvirt {
		// Explicitly requested libvirt
		useProxmox = false
		log.Println("Using libvirt connector (explicitly requested)")
	} else if config.Options.Proxmox {
		// Explicitly requested Proxmox
		useProxmox = true
		log.Println("Using Proxmox VE connector (explicitly requested)")
	} else {
		// Auto-detect: check if Proxmox is available
		useProxmox = connector.DetectProxmox()
		if useProxmox {
			log.Println("Auto-detected Proxmox VE environment")
		}
	}

	// Initialize connector based on selection
	if useProxmox {
		proxmoxConn := &connector.ProxmoxConnector{}
		err := proxmoxConn.Initialize()
		if err != nil {
			fmt.Printf("Failed to initialize Proxmox connector: %v\n", err)
			fmt.Println("proxprofiler will terminate.")
			os.Exit(1)
		}
		connector.CurrentConnector = proxmoxConn
		connector.CurrentConnectorType = connector.ConnectorTypeProxmox
		log.Println("Using Proxmox VE connector")
	} else {
		// connect to libvirt
		connector.Libvirt.ConnectionURI = config.Options.LibvirtURI
		err := connector.InitializeConnection()
		if err != nil {
			fmt.Println("failed to initialize connection to libvirt. proxprofiler will terminate.")
			os.Exit(1)
		}
		connector.CurrentConnectorType = connector.ConnectorTypeLibvirt
		log.Println("Using libvirt connector")
	}

	// start lookup and collect runners
	var wg sync.WaitGroup
	wg.Add(1) // terminate when first thread terminates
	go runners.InitializeLookup(&wg)
	go runners.InitializeCollect(&wg)
	go profiler.InitializeProfiler(&wg)
	wg.Wait()

}
