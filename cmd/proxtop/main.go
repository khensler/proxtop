package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"proxtop/config"
	"proxtop/connector"
	"proxtop/models"
	"proxtop/runners"
)

var version = "1.1.3-1"

func main() {

	// catch panics
	defer func() {
		if r := recover(); r != nil {
			debug.PrintStack()
			shutdown(1)
		}
	}()

	// handle flags
	initializeFlags()
	if config.Options.Version {
		fmt.Println("proxtop version " + version)
		return
	}

	// Disable logging output when using ncurses printer to avoid corrupting the screen
	if config.Options.Printer == "ncurses" {
		log.SetOutput(ioutil.Discard)
	}

	// catch termination signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		shutdown(0)
	}()

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
			fmt.Println("proxtop will terminate.")
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
			fmt.Println("proxtop will terminate.")
			os.Exit(1)
		}
		connector.CurrentConnectorType = connector.ConnectorTypeLibvirt
		log.Println("Using libvirt connector")
	}

	// start runners
	runners.InitializeRunners()

	// when runners terminate, shutdown proxtop
	shutdown(0)
}

func shutdown(exitcode int) {
	// close connector
	if connector.CurrentConnector != nil {
		err := connector.CurrentConnector.Close()
		if err != nil {
			exitcode = 1
		}
	} else {
		// close libvirt connection (legacy path)
		err := connector.CloseConnection()
		if err != nil {
			exitcode = 1
		}
	}

	// close printer
	models.Collection.Printer.Close()

	// return exit code
	os.Exit(exitcode)
}
