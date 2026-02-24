package printers

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/cha87de/goncurses"
	"proxtop/collectors/diskcollector"
	"proxtop/collectors/netcollector"
	"proxtop/config"
	"proxtop/models"
	"proxtop/runners"
)

var screen *goncurses.Window

// ViewMode represents which metrics to display
type ViewMode int

const (
	ViewAll ViewMode = iota
	ViewCPU
	ViewMem
	ViewDisk
	ViewNet
	ViewIO
	ViewPhysNet  // Physical network interfaces
	ViewPhysDisk // Physical disk devices
	ViewLVM      // LVM logical volumes
	ViewMpath    // Multipath devices
	ViewHelp
)

// NcursesPrinter describes the ncurses printer
type NcursesPrinter struct {
	models.Printer
	currentView  ViewMode
	sortColumn   int
	showHelp     bool
}

const HOSTWINHEIGHT = 5
const DOMAINMINFIELDWIDTH = 6  // Minimum column width
const DOMAINMAXFIELDWIDTH = 10 // Maximum column width to prevent overflow
const HOSTFIELDWIDTH = 10       // Width for host field names
const HOSTVALUEWIDTH = 12       // Width for host values

type KeyValue struct {
	Key   string
	Value string
}

var domainColumnWidths []int
var currentViewMode ViewMode = ViewCPU
var currentSortColumn int = 3
var sortAscending bool = false // false = descending (default), true = ascending
var showHelpOverlay bool = false
var helpDrawn bool = false
var quitRequested bool = false
var currentInterval int = 1  // Current refresh interval in seconds

// Field selection state
var showFieldSelector bool = false
var fieldSelectorDrawn bool = false
var fieldSelectorCursor int = 0
var availableFields []string        // Fields available in current view
var hiddenFields map[string]bool    // Map of field name -> hidden status (global for now)

// Open opens the printer
func (printer *NcursesPrinter) Open() {
	// Init goncurses
	var err error
	screen, err = goncurses.Init()
	if err != nil {
		log.Fatal("init", err)
	}
	goncurses.Echo(false)      // turn echoing of typed characters off
	goncurses.Cursor(0)        // hide cursor
	goncurses.Raw(true)        // enable raw mode for immediate key input
	screen.Keypad(true)        // enable keypad for special keys
	screen.Timeout(100)        // non-blocking input with 100ms timeout

	// Always enable verbose mode so all fields are collected
	// Users can show/hide individual fields via the field selector ('f' key)
	config.Options.Verbose = true

	// Initialize field selection state with verbose-only fields hidden by default
	hiddenFields = make(map[string]bool)

	// Verbose-only fields (hidden by default, can be enabled via field selector)
	// CPU collector
	hiddenFields["cpu_%othrdy"] = true
	hiddenFields["cpu_minfreq"] = true
	hiddenFields["cpu_maxfreq"] = true
	hiddenFields["cpu_%nice"] = true
	hiddenFields["cpu_%irq"] = true
	hiddenFields["cpu_%softirq"] = true
	hiddenFields["cpu_%guest"] = true
	hiddenFields["cpu_%guestnice"] = true

	// Memory collector (domain)
	hiddenFields["mem_MAXSZ"] = true
	hiddenFields["mem_VSIZE"] = true
	hiddenFields["mem_SWAPIN"] = true
	hiddenFields["mem_SWAPOUT"] = true
	hiddenFields["mem_CMINFLT"] = true
	hiddenFields["mem_CMAJFLT"] = true

	// Memory collector (host) - detailed breakdown fields
	hiddenFields["mem_buffers"] = true
	hiddenFields["mem_cached"] = true
	hiddenFields["mem_swapcached"] = true
	hiddenFields["mem_active"] = true
	hiddenFields["mem_inactive"] = true
	hiddenFields["mem_activeanon"] = true
	hiddenFields["mem_inactiveanon"] = true
	hiddenFields["mem_activefile"] = true
	hiddenFields["mem_inactivefile"] = true
	hiddenFields["mem_unevictable"] = true
	hiddenFields["mem_mlocked"] = true
	hiddenFields["mem_dirty"] = true
	hiddenFields["mem_writeback"] = true
	hiddenFields["mem_anonpages"] = true
	hiddenFields["mem_mapped"] = true
	hiddenFields["mem_shmem"] = true
	hiddenFields["mem_kreclaimable"] = true
	hiddenFields["mem_slab"] = true
	hiddenFields["mem_sreclaimable"] = true
	hiddenFields["mem_sunreclaim"] = true
	hiddenFields["mem_kernelstack"] = true
	hiddenFields["mem_pagetables"] = true
	hiddenFields["mem_nfs_unstable"] = true
	hiddenFields["mem_bounce"] = true
	hiddenFields["mem_writebacktmp"] = true
	hiddenFields["mem_commitlimit"] = true
	hiddenFields["mem_committed_as"] = true
	hiddenFields["mem_vmalloctotal"] = true
	hiddenFields["mem_vmallocused"] = true
	hiddenFields["mem_vmallocchunk"] = true
	hiddenFields["mem_percpu"] = true
	hiddenFields["mem_hardwarecorrupted"] = true
	hiddenFields["mem_anonhugepages"] = true
	hiddenFields["mem_shmemhugepages"] = true
	hiddenFields["mem_shmempmdmapped"] = true
	hiddenFields["mem_cmatotal"] = true
	hiddenFields["mem_cmafree"] = true
	hiddenFields["mem_hugepages_total"] = true
	hiddenFields["mem_hugepages_free"] = true
	hiddenFields["mem_hugepages_rsvd"] = true
	hiddenFields["mem_hugepages_surp"] = true
	hiddenFields["mem_hugepagesize"] = true
	hiddenFields["mem_hugetlb"] = true
	hiddenFields["mem_directmap4k"] = true
	hiddenFields["mem_directmap2m"] = true
	hiddenFields["mem_directmap1g"] = true

	// Disk collector (domain)
	hiddenFields["dsk_PHYSICAL"] = true
	hiddenFields["dsk_FLUSH/s"] = true
	hiddenFields["dsk_RDTM"] = true
	hiddenFields["dsk_WRTM"] = true
	hiddenFields["dsk_FLTM"] = true
	hiddenFields["dsk_BLKIO"] = true

	// Disk collector (host)
	hiddenFields["dsk_rdmerged"] = true
	hiddenFields["dsk_sectorsrd"] = true
	hiddenFields["dsk_timerd"] = true
	hiddenFields["dsk_wrmerged"] = true
	hiddenFields["dsk_sectorswr"] = true
	hiddenFields["dsk_timewr"] = true
	hiddenFields["dsk_timeforops"] = true
	hiddenFields["dsk_weightedtime"] = true
	hiddenFields["dsk_count"] = true

	// I/O collector
	hiddenFields["io_rchar"] = true
	hiddenFields["io_wchar"] = true
	hiddenFields["io_cancelled"] = true

	// PSI collector (all PSI fields are verbose-only)
	hiddenFields["psi_some_cpu_avg10"] = true
	hiddenFields["psi_some_cpu_avg300"] = true
	hiddenFields["psi_some_cpu_total"] = true
	hiddenFields["psi_some_io_avg10"] = true
	hiddenFields["psi_some_io_avg300"] = true
	hiddenFields["psi_some_io_total"] = true
	hiddenFields["psi_full_io_avg10"] = true
	hiddenFields["psi_full_io_avg300"] = true
	hiddenFields["psi_full_io_total"] = true
	hiddenFields["psi_some_mem_avg10"] = true
	hiddenFields["psi_some_mem_avg300"] = true
	hiddenFields["psi_some_mem_total"] = true
	hiddenFields["psi_full_mem_avg10"] = true
	hiddenFields["psi_full_mem_avg300"] = true
	hiddenFields["psi_full_mem_total"] = true

	// Host collector
	hiddenFields["host_uuid"] = true

	// Network collector (domain) - verbose fields hidden by default
	hiddenFields["net_errsRX"] = true
	hiddenFields["net_fifoRX"] = true
	hiddenFields["net_frameRX"] = true
	hiddenFields["net_compRX"] = true
	hiddenFields["net_mcastRX"] = true
	hiddenFields["net_errsTX"] = true
	hiddenFields["net_fifoTX"] = true
	hiddenFields["net_collsTX"] = true
	hiddenFields["net_carrierTX"] = true
	hiddenFields["net_compTX"] = true
	hiddenFields["net_interfaces"] = true // Redundant in per-interface view (DEVICE column shows this)

	// Physical network verbose fields (hidden by default)
	hiddenFields["net_RX-Fifo"] = true
	hiddenFields["net_RX-Frame"] = true
	hiddenFields["net_RX-Compressed"] = true
	hiddenFields["net_RX-Multicast"] = true
	hiddenFields["net_TX-Fifo"] = true
	hiddenFields["net_TX-Colls"] = true
	hiddenFields["net_TX-Carrier"] = true
	hiddenFields["net_TX-Compressed"] = true

	// Physical network raw counters (hidden by default - rates are more useful)
	hiddenFields["net_RX-Bytes"] = true
	hiddenFields["net_RX-Pkts"] = true
	hiddenFields["net_RX-Errs"] = true
	hiddenFields["net_RX-Drop"] = true
	hiddenFields["net_TX-Bytes"] = true
	hiddenFields["net_TX-Pkts"] = true
	hiddenFields["net_TX-Errs"] = true
	hiddenFields["net_TX-Drop"] = true
}

// handleInput processes keyboard input and returns true if we should quit
func handleInput() bool {
	ch := screen.GetChar()
	if ch == 0 {
		return false
	}

	// Handle field selector input separately
	if showFieldSelector {
		switch ch {
		case 'q', 'Q', 27, 'f', 'F': // q, Escape, or f to close
			showFieldSelector = false
			fieldSelectorDrawn = false
		case goncurses.KEY_UP, 'k', 'K':
			if fieldSelectorCursor > 0 {
				fieldSelectorCursor--
			}
			fieldSelectorDrawn = false
		case goncurses.KEY_DOWN, 'j', 'J':
			if fieldSelectorCursor < len(availableFields)-1 {
				fieldSelectorCursor++
			}
			fieldSelectorDrawn = false
		case ' ', 13, goncurses.KEY_ENTER: // Space or Enter to toggle
			if fieldSelectorCursor < len(availableFields) {
				field := availableFields[fieldSelectorCursor]
				hiddenFields[field] = !hiddenFields[field]
			}
			fieldSelectorDrawn = false
		}
		return false
	}

	switch ch {
	case 'q', 'Q', 3: // 3 is Ctrl+C
		quitRequested = true
		return true // quit
	case 27: // Escape key - close help overlay if open
		if showHelpOverlay {
			showHelpOverlay = false
			helpDrawn = false
		}
	case 'h', 'H', '?':
		showHelpOverlay = !showHelpOverlay
		helpDrawn = false // force redraw of help
	case 'f', 'F': // Field selector
		showFieldSelector = !showFieldSelector
		fieldSelectorDrawn = false
		fieldSelectorCursor = 0
	case 'a', 'A':
		currentViewMode = ViewAll
		showHelpOverlay = false
		helpDrawn = false
	case 'c', 'C':
		currentViewMode = ViewCPU
		showHelpOverlay = false
		helpDrawn = false
	case 'm', 'M':
		currentViewMode = ViewMem
		showHelpOverlay = false
		helpDrawn = false
	case 'd', 'D':
		currentViewMode = ViewDisk
		showHelpOverlay = false
		helpDrawn = false
	case 'n', 'N':
		currentViewMode = ViewNet
		showHelpOverlay = false
		helpDrawn = false
	case 'i', 'I':
		currentViewMode = ViewIO
		showHelpOverlay = false
		helpDrawn = false
	case 'p', 'P':
		currentViewMode = ViewPhysNet
		showHelpOverlay = false
		helpDrawn = false
	case 's', 'S':
		currentViewMode = ViewPhysDisk
		showHelpOverlay = false
		helpDrawn = false
	case 'l', 'L':
		currentViewMode = ViewLVM
		showHelpOverlay = false
		helpDrawn = false
	case 'x', 'X':
		currentViewMode = ViewMpath
		showHelpOverlay = false
		helpDrawn = false
	case '<':
		if currentSortColumn > 0 {
			currentSortColumn--
		}
	case '>':
		currentSortColumn++
	case 'r', 'R': // Toggle sort direction (reverse)
		sortAscending = !sortAscending
		runners.ForceRefresh = true
	case '+', '=': // Increase refresh interval
		if config.Options.Frequency < 60 {
			config.Options.Frequency++
		}
	case '-', '_': // Decrease refresh interval
		if config.Options.Frequency > 1 {
			config.Options.Frequency--
		}
	case 'u', 'U': // Toggle human-readable units
		config.Options.HumanReadable = !config.Options.HumanReadable
		runners.ForceRefresh = true
	}
	return false
}

// getViewModeName returns the display name for the current view mode
func getViewModeName() string {
	switch currentViewMode {
	case ViewCPU:
		return "CPU"
	case ViewMem:
		return "MEMORY"
	case ViewDisk:
		return "DISK"
	case ViewNet:
		return "NETWORK"
	case ViewIO:
		return "I/O"
	case ViewPhysNet:
		return "PHYS-NET"
	case ViewPhysDisk:
		return "PHYS-DISK"
	case ViewLVM:
		return "LVM"
	case ViewMpath:
		return "MULTIPATH"
	default:
		return "ALL"
	}
}

// filterFieldsByView filters fields and values based on current view mode and hidden fields
func filterFieldsByView(fields []string, values map[string][]string) ([]string, map[string][]string) {
	// Always include UUID and name (first two columns)
	filteredFields := []string{}
	includeIndices := []int{}

	for i, field := range fields {
		// Skip hidden fields (but always include first two columns)
		if i >= 2 && hiddenFields[field] {
			continue
		}

		// Check if field matches current view mode
		include := false
		if i < 2 {
			include = true // Always include UUID and name
		} else {
			fieldLower := strings.ToLower(field)
			switch currentViewMode {
			case ViewCPU:
				include = strings.HasPrefix(fieldLower, "cpu_")
			case ViewMem:
				include = strings.HasPrefix(fieldLower, "mem_")
			case ViewDisk:
				include = strings.HasPrefix(fieldLower, "dsk_")
			case ViewNet:
				include = strings.HasPrefix(fieldLower, "net_")
			case ViewIO:
				include = strings.HasPrefix(fieldLower, "io_") || strings.HasPrefix(fieldLower, "psi_")
			default:
				include = true // ViewAll - include all fields
			}
		}

		if include {
			filteredFields = append(filteredFields, field)
			includeIndices = append(includeIndices, i)
		}
	}

	filteredValues := make(map[string][]string)
	for uuid, vals := range values {
		filtered := []string{}
		for _, idx := range includeIndices {
			if idx < len(vals) {
				filtered = append(filtered, vals[idx])
			}
		}
		filteredValues[uuid] = filtered
	}

	return filteredFields, filteredValues
}

// applyHiddenFields filters out hidden fields from already-filtered data
// Used after expandPerDeviceView which may add fields that should be hidden
func applyHiddenFields(fields []string, values map[string][]string) ([]string, map[string][]string) {
	// Build list of indices to keep (always keep first 3: UUID, name, DEVICE)
	keepIndices := []int{}
	filteredFields := []string{}

	for i, field := range fields {
		// Always keep UUID, name, DEVICE (first 3 columns in expanded view)
		if i < 3 || !hiddenFields[field] {
			keepIndices = append(keepIndices, i)
			filteredFields = append(filteredFields, field)
		}
	}

	// Filter values to only include kept indices
	filteredValues := make(map[string][]string)
	for uuid, vals := range values {
		filtered := make([]string, 0, len(keepIndices))
		for _, idx := range keepIndices {
			if idx < len(vals) {
				filtered = append(filtered, vals[idx])
			}
		}
		filteredValues[uuid] = filtered
	}

	return filteredFields, filteredValues
}

// Screen prints the measurements on the screen
func (printer *NcursesPrinter) Screen(printable models.Printable) {
	// Handle keyboard input
	if handleInput() || quitRequested {
		// User pressed quit - clean up and exit
		goncurses.End()
		os.Exit(0)
	}

	maxy, maxx := screen.MaxYX()

	// Pause collection when overlays are shown for responsive UI
	runners.CollectionPaused = showHelpOverlay || showFieldSelector

	// Show help overlay if enabled
	if showHelpOverlay {
		// Only draw help once, then just handle input
		if !helpDrawn {
			screen.Clear()
			screen.Refresh()
			printHelpOverlay(maxy, maxx)
			helpDrawn = true
		}
		return
	}
	helpDrawn = false

	// Show field selector overlay if enabled
	if showFieldSelector {
		if !fieldSelectorDrawn {
			screen.Clear()
			screen.Refresh()
			printFieldSelectorOverlay(maxy, maxx, printable.DomainFields)
			fieldSelectorDrawn = true
		}
		return
	}
	fieldSelectorDrawn = false

	screen.Clear()

	// Define status bar at top (row 0)
	screen.Move(0, 0)
	screen.AttrOn(goncurses.A_REVERSE)
	// Show actual measured refresh time and target interval
	actualInterval := runners.LastRefreshDuration.Seconds()
	if actualInterval < 0.1 {
		actualInterval = float64(config.Options.Frequency) // First run, use target
	}
	modeIndicators := ""
	if config.Options.HumanReadable {
		modeIndicators += " [H]"
	}
	statusLine := fmt.Sprintf(" proxtop | View: %s%s | Refresh: %.1fs (+/-) | 'h' help, 'u' units, 'f' fields, 'q' quit ",
		getViewModeName(), modeIndicators, actualInterval)
	for len(statusLine) < maxx {
		statusLine += " "
	}
	if len(statusLine) > maxx {
		statusLine = statusLine[:maxx]
	}
	screen.Printf("%s", statusLine)
	screen.AttrOff(goncurses.A_REVERSE)

	// Handle physical device views differently
	if currentViewMode == ViewPhysNet || currentViewMode == ViewPhysDisk ||
		currentViewMode == ViewLVM || currentViewMode == ViewMpath {
		// Use full screen for device list (no host panel)
		deviceWin, _ := goncurses.NewWindow(maxy-1, maxx, 1, 0)
		goncurses.UpdatePanels()
		goncurses.Update()
		goncurses.NewPanel(deviceWin)

		switch currentViewMode {
		case ViewPhysNet:
			printPhysicalNetDevices(deviceWin)
		case ViewPhysDisk:
			printPhysicalDiskDevices(deviceWin)
		case ViewLVM:
			printLVMDevices(deviceWin)
		case ViewMpath:
			printMpathDevices(deviceWin)
		}

		screen.NoutRefresh()
		goncurses.Update()
		return
	}

	// Filter fields based on current view mode
	filteredFields, filteredValues := filterFieldsByView(printable.DomainFields, printable.DomainValues)

	// Expand per-device data for Net/Disk views
	if currentViewMode == ViewNet || currentViewMode == ViewDisk {
		filteredFields, filteredValues = expandPerDeviceView(filteredFields, filteredValues, currentViewMode)
		// Re-apply hidden field filtering after expansion (expansion adds raw collector data)
		filteredFields, filteredValues = applyHiddenFields(filteredFields, filteredValues)
	}

	// Define host panel (starts at row 1)
	hostWin, _ := goncurses.NewWindow(HOSTWINHEIGHT, maxx, 1, 0)
	goncurses.UpdatePanels()
	goncurses.Update()
	goncurses.NewPanel(hostWin)

	// Filter host fields based on view mode
	filteredHostFields, filteredHostValues := filterHostFieldsByView(printable.HostFields, printable.HostValues)
	printHost(hostWin, filteredHostFields, filteredHostValues)

	// Define domain panel
	domainWin, _ := goncurses.NewWindow(maxy-HOSTWINHEIGHT-1, maxx, HOSTWINHEIGHT+1, 0)
	goncurses.UpdatePanels()
	goncurses.Update()
	goncurses.NewPanel(domainWin)

	// Adjust sort column if out of bounds
	if currentSortColumn >= len(filteredFields) {
		currentSortColumn = len(filteredFields) - 1
	}
	if currentSortColumn < 0 {
		currentSortColumn = 0
	}

	printDomain(domainWin, filteredFields, filteredValues, currentSortColumn)
	screen.Refresh()
}

// expandPerDeviceView expands VM rows to show per-device stats for Net/Disk views
func expandPerDeviceView(fields []string, values map[string][]string, viewMode ViewMode) ([]string, map[string][]string) {
	expandedValues := make(map[string][]string)

	// Add DEVICE column after UUID and name
	expandedFields := make([]string, 0, len(fields)+1)
	if len(fields) >= 2 {
		expandedFields = append(expandedFields, fields[0], fields[1], "DEVICE")
		expandedFields = append(expandedFields, fields[2:]...)
	} else {
		expandedFields = append([]string{"DEVICE"}, fields...)
	}

	// Iterate through domains and expand per-device
	models.Collection.Domains.Range(func(key, value interface{}) bool {
		uuid := key.(string)
		domain := value.(models.Domain)
		baseValues, ok := values[uuid]
		if !ok {
			return true
		}

		if viewMode == ViewNet {
			// Get per-interface stats
			perIfStats := netcollector.DomainPrintPerInterface(&domain)
			if len(perIfStats) > 1 {
				// Multiple interfaces - create a row for each
				for ifName, ifValues := range perIfStats {
					// Create unique key for this interface row
					rowKey := fmt.Sprintf("%s:%s", uuid, ifName)
					// Build row: UUID, name, device, then per-device values
					row := make([]string, 0, len(expandedFields))
					if len(baseValues) >= 2 {
						row = append(row, baseValues[0], baseValues[1], ifName)
						row = append(row, ifValues...)
					}
					expandedValues[rowKey] = row
				}
			} else if len(perIfStats) == 1 {
				// Single interface - show device name but use original key
				for ifName, ifValues := range perIfStats {
					row := make([]string, 0, len(expandedFields))
					if len(baseValues) >= 2 {
						row = append(row, baseValues[0], baseValues[1], ifName)
						row = append(row, ifValues...)
					}
					expandedValues[uuid] = row
				}
			} else {
				// No per-interface data - use totals with "-" as device
				row := make([]string, 0, len(expandedFields))
				if len(baseValues) >= 2 {
					row = append(row, baseValues[0], baseValues[1], "-")
					row = append(row, baseValues[2:]...)
				}
				expandedValues[uuid] = row
			}
		} else if viewMode == ViewDisk {
			// Get per-disk stats
			perDiskStats := diskcollector.DiskPrintPerDevice(&domain)
			if len(perDiskStats) > 1 {
				// Multiple disks - create a row for each
				for diskName, diskValues := range perDiskStats {
					rowKey := fmt.Sprintf("%s:%s", uuid, diskName)
					row := make([]string, 0, len(expandedFields))
					if len(baseValues) >= 2 {
						// For disk: UUID, name, device, then SIZE, ALLOC, %UTIL from base, then per-device IO
						row = append(row, baseValues[0], baseValues[1], diskName)
						// Add SIZE, ALLOC, %UTIL from base values
						if len(baseValues) >= 5 {
							row = append(row, baseValues[2], baseValues[3], baseValues[4])
						}
						row = append(row, diskValues...)
					}
					expandedValues[rowKey] = row
				}
			} else if len(perDiskStats) == 1 {
				// Single disk - show device name
				for diskName, diskValues := range perDiskStats {
					row := make([]string, 0, len(expandedFields))
					if len(baseValues) >= 2 {
						row = append(row, baseValues[0], baseValues[1], diskName)
						if len(baseValues) >= 5 {
							row = append(row, baseValues[2], baseValues[3], baseValues[4])
						}
						row = append(row, diskValues...)
					}
					expandedValues[uuid] = row
				}
			} else {
				// No per-disk data - use totals with "-" as device
				row := make([]string, 0, len(expandedFields))
				if len(baseValues) >= 2 {
					row = append(row, baseValues[0], baseValues[1], "-")
					row = append(row, baseValues[2:]...)
				}
				expandedValues[uuid] = row
			}
		}
		return true
	})

	return expandedFields, expandedValues
}

// filterHostFieldsByView filters host fields based on current view mode
func filterHostFieldsByView(fields []string, values []string) ([]string, []string) {
	if currentViewMode == ViewAll {
		return fields, values
	}

	filteredFields := []string{}
	filteredValues := []string{}

	for i, field := range fields {
		fieldLower := strings.ToLower(field)
		include := false
		switch currentViewMode {
		case ViewCPU:
			include = strings.HasPrefix(fieldLower, "cpu_")
		case ViewMem:
			include = strings.HasPrefix(fieldLower, "mem_")
		case ViewDisk:
			include = strings.HasPrefix(fieldLower, "dsk_")
		case ViewNet:
			include = strings.HasPrefix(fieldLower, "net_")
		case ViewIO:
			include = strings.HasPrefix(fieldLower, "io_") || strings.HasPrefix(fieldLower, "psi_")
		default:
			include = true
		}
		if include {
			filteredFields = append(filteredFields, field)
			if i < len(values) {
				filteredValues = append(filteredValues, values[i])
			}
		}
	}

	return filteredFields, filteredValues
}

// printHelpOverlay displays the help screen
func printHelpOverlay(maxy, maxx int) {
	// Center the help box
	helpWidth := 50
	helpHeight := 33
	startY := (maxy - helpHeight) / 2
	startX := (maxx - helpWidth) / 2

	if startY < 0 {
		startY = 0
	}
	if startX < 0 {
		startX = 0
	}

	// Draw help box
	helpWin, err := goncurses.NewWindow(helpHeight, helpWidth, startY, startX)
	if err != nil || helpWin == nil {
		// Fallback: draw directly on screen
		screen.Move(1, 2)
		screen.AttrOn(goncurses.A_BOLD)
		screen.Printf("proxtop - Keybindings (press 'h' to close)")
		screen.AttrOff(goncurses.A_BOLD)
		screen.Move(3, 2)
		screen.Printf("a/c/m/d/n/i - Views | u - Units | f - Fields | q - Quit")
		screen.Refresh()
		return
	}

	helpWin.Box(0, 0)

	helpWin.Move(1, 2)
	helpWin.AttrOn(goncurses.A_BOLD)
	helpWin.Printf("proxtop - Keybindings")
	helpWin.AttrOff(goncurses.A_BOLD)

	helpWin.Move(3, 2)
	helpWin.Printf("VM Views:")
	helpWin.Move(4, 4)
	helpWin.Printf("a - Show ALL metrics")
	helpWin.Move(5, 4)
	helpWin.Printf("c - Show CPU metrics")
	helpWin.Move(6, 4)
	helpWin.Printf("m - Show MEMORY metrics")
	helpWin.Move(7, 4)
	helpWin.Printf("d - Show DISK metrics")
	helpWin.Move(8, 4)
	helpWin.Printf("n - Show NETWORK metrics")
	helpWin.Move(9, 4)
	helpWin.Printf("i - Show I/O metrics")

	helpWin.Move(11, 2)
	helpWin.Printf("Host Device Views:")
	helpWin.Move(12, 4)
	helpWin.Printf("p - PHYSICAL NETWORK interfaces")
	helpWin.Move(13, 4)
	helpWin.Printf("s - PHYSICAL DISK devices (sd*, nvme*, vd*)")
	helpWin.Move(14, 4)
	helpWin.Printf("l - LVM logical volumes")
	helpWin.Move(15, 4)
	helpWin.Printf("x - MULTIPATH devices")

	helpWin.Move(17, 2)
	helpWin.Printf("Sorting:")
	helpWin.Move(18, 4)
	helpWin.Printf("< - Sort by previous column")
	helpWin.Move(19, 4)
	helpWin.Printf("> - Sort by next column")
	helpWin.Move(20, 4)
	helpWin.Printf("r - Reverse sort direction (asc/desc)")

	helpWin.Move(22, 2)
	helpWin.Printf("Display:")
	helpWin.Move(23, 4)
	helpWin.Printf("u - Toggle human-readable units (KB/MB/GB)")
	helpWin.Move(24, 4)
	helpWin.Printf("+ - Increase refresh interval (slower)")
	helpWin.Move(25, 4)
	helpWin.Printf("- - Decrease refresh interval (faster)")

	helpWin.Move(27, 2)
	helpWin.Printf("Other:")
	helpWin.Move(28, 4)
	helpWin.Printf("f - Field selector (show/hide columns)")
	helpWin.Move(29, 4)
	helpWin.Printf("h/? - Toggle this help")
	helpWin.Move(30, 4)
	helpWin.Printf("q   - Quit (also Ctrl+C)")

	helpWin.NoutRefresh()
	goncurses.Update()
}

// printFieldSelectorOverlay displays the field selection screen
func printFieldSelectorOverlay(maxy, maxx int, domainFields []string) {
	// Filter fields for current view and store in availableFields
	availableFields = filterFieldsForSelector(domainFields)

	// Ensure cursor is in bounds
	if fieldSelectorCursor >= len(availableFields) {
		fieldSelectorCursor = len(availableFields) - 1
	}
	if fieldSelectorCursor < 0 {
		fieldSelectorCursor = 0
	}

	// Calculate box dimensions
	boxWidth := 55
	boxHeight := len(availableFields) + 8 // Header + footer + fields
	if boxHeight > maxy-2 {
		boxHeight = maxy - 2
	}
	startY := (maxy - boxHeight) / 2
	startX := (maxx - boxWidth) / 2
	if startY < 0 {
		startY = 0
	}
	if startX < 0 {
		startX = 0
	}

	// Create field selector window
	fieldWin, err := goncurses.NewWindow(boxHeight, boxWidth, startY, startX)
	if err != nil || fieldWin == nil {
		// Fallback: draw on screen
		screen.Move(1, 2)
		screen.Printf("Field Selector (press 'f' to close)")
		screen.Refresh()
		return
	}

	fieldWin.Box(0, 0)
	fieldWin.Move(1, 2)
	fieldWin.AttrOn(goncurses.A_BOLD)
	fieldWin.Printf("Field Selection - %s View", getViewModeName())
	fieldWin.AttrOff(goncurses.A_BOLD)

	fieldWin.Move(2, 2)
	fieldWin.Printf("Up/Down: navigate, Space/Enter: toggle, f/q: close")

	// Calculate visible range for scrolling
	visibleFields := boxHeight - 6
	startIdx := 0
	if fieldSelectorCursor >= visibleFields {
		startIdx = fieldSelectorCursor - visibleFields + 1
	}

	// Display fields with checkboxes
	for i := 0; i < visibleFields && startIdx+i < len(availableFields); i++ {
		fieldIdx := startIdx + i
		field := availableFields[fieldIdx]

		row := 4 + i
		fieldWin.Move(row, 2)

		// Highlight current selection
		if fieldIdx == fieldSelectorCursor {
			fieldWin.AttrOn(goncurses.A_REVERSE)
		}

		// Show checkbox status
		checkbox := "[x]"
		if hiddenFields[field] {
			checkbox = "[ ]"
		}

		// Display field name (remove prefix for cleaner display)
		displayName := field
		if len(displayName) > boxWidth-10 {
			displayName = displayName[:boxWidth-10]
		}
		fieldWin.Printf(" %s %s", checkbox, displayName)

		// Pad to fill width
		for j := len(checkbox) + len(displayName) + 4; j < boxWidth-4; j++ {
			fieldWin.Printf(" ")
		}

		if fieldIdx == fieldSelectorCursor {
			fieldWin.AttrOff(goncurses.A_REVERSE)
		}
	}

	// Show scroll indicator if needed
	if len(availableFields) > visibleFields {
		fieldWin.Move(boxHeight-2, 2)
		fieldWin.Printf("(%d/%d fields)", fieldSelectorCursor+1, len(availableFields))
	}

	fieldWin.NoutRefresh()
	goncurses.Update()
}

// filterFieldsForSelector returns the list of fields for the current view mode
func filterFieldsForSelector(domainFields []string) []string {
	var filtered []string

	// For physical views, use fields from the collector directly
	switch currentViewMode {
	case ViewPhysNet:
		// Get fields from network collector (skip first "DEVICE" column)
		physFields := netcollector.HostNetFields()
		for i, field := range physFields {
			if i > 0 { // Skip DEVICE column - always visible
				filtered = append(filtered, field)
			}
		}
		return filtered
	case ViewPhysDisk, ViewLVM, ViewMpath:
		// Get fields from disk collector (skip first "DEVICE" column)
		physFields := diskcollector.HostDiskFields()
		for i, field := range physFields {
			if i > 0 { // Skip DEVICE column - always visible
				filtered = append(filtered, field)
			}
		}
		return filtered
	}

	// For other views, filter domain fields
	for _, field := range domainFields {
		switch currentViewMode {
		case ViewCPU:
			if strings.HasPrefix(field, "cpu_") {
				filtered = append(filtered, field)
			}
		case ViewMem:
			if strings.HasPrefix(field, "mem_") {
				filtered = append(filtered, field)
			}
		case ViewDisk:
			if strings.HasPrefix(field, "dsk_") {
				filtered = append(filtered, field)
			}
		case ViewNet:
			if strings.HasPrefix(field, "net_") {
				filtered = append(filtered, field)
			}
		case ViewIO:
			if strings.HasPrefix(field, "io_") || strings.HasPrefix(field, "psi_") {
				filtered = append(filtered, field)
			}
		default: // ViewAll
			filtered = append(filtered, field)
		}
	}
	return filtered
}

// Close terminates the printer
func (printer *NcursesPrinter) Close() {
	goncurses.End()
}

// CreateNcurses creates a new ncurses printer
func CreateNcurses() NcursesPrinter {
	return NcursesPrinter{}
}

func printHost(window *goncurses.Window, fields []string, values []string) {
	maxy, maxx := window.MaxYX()
	maxRows := maxy - 1 // Leave room for border/margin

	currentPosX := 1
	currentPosY := 1
	columnWidth := HOSTFIELDWIDTH + HOSTVALUEWIDTH + 2 // Total width per column group
	groupStartY := 1 // Track where current group started

	currentGroup := ""
	for columnID, field := range fields {
		// make group columns & print headline
		fieldParts := strings.Split(field, "_")
		groupLabel := ""
		if len(fieldParts) > 1 {
			groupLabel = strings.Join(fieldParts[0:len(fieldParts)-1], " ")
		}

		// Check if we need to start a new column (new group or out of vertical space)
		if groupLabel != currentGroup {
			// found new group! move to next column
			if currentGroup != "" {
				currentPosY = 1
				currentPosX += columnWidth
			}
			// Check if we've exceeded screen width
			if currentPosX+columnWidth > maxx {
				break // Stop if we can't fit more columns
			}
			// print group label header
			currentGroup = groupLabel
			groupStartY = currentPosY
			window.Move(currentPosY, currentPosX)
			window.AttrOn(goncurses.A_REVERSE)
			headline := padRight(groupLabel, columnWidth-1)
			window.Printf("%s", headline)
			window.AttrOff(goncurses.A_REVERSE)
			currentPosY++
		}

		// Check if we've run out of vertical space in this group
		if currentPosY >= maxRows {
			// Move to next column, continue same group
			currentPosY = groupStartY + 1
			currentPosX += columnWidth
			if currentPosX+columnWidth > maxx {
				break // Stop if we can't fit more columns
			}
			// Print group header for continuation
			window.Move(groupStartY, currentPosX)
			window.AttrOn(goncurses.A_REVERSE)
			headline := padRight(groupLabel+" (cont)", columnWidth-1)
			window.Printf("%s", headline)
			window.AttrOff(goncurses.A_REVERSE)
		}

		// print label
		fieldLabel := padRight(fieldParts[len(fieldParts)-1], HOSTFIELDWIDTH)
		window.Move(currentPosY, currentPosX)
		window.Printf("%s", fieldLabel)

		// print value
		value := ""
		if columnID < len(values) {
			value = padRight(values[columnID], HOSTVALUEWIDTH)
		}
		window.Move(currentPosY, currentPosX+HOSTFIELDWIDTH+1)
		window.Printf("%s", value)

		currentPosY++
	}
}

func printDomain(window *goncurses.Window, fields []string, values map[string][]string, sortByColumn int) {
	// Get terminal width
	_, maxx := window.MaxYX()
	availableWidth := maxx - 2 // Leave margin

	// Reset column widths for this view
	numColumns := len(fields)
	if numColumns == 0 {
		return
	}
	domainColumnWidths = make([]int, numColumns)
	desiredWidths := make([]int, numColumns) // Track what each column ideally wants

	// First pass: calculate desired widths for each column based on data
	for colID, field := range fields {
		fieldParts := strings.Split(field, "_")
		fieldName := fieldParts[len(fieldParts)-1]
		// Start with field name length
		desiredWidths[colID] = len(fieldName)
		if desiredWidths[colID] < DOMAINMINFIELDWIDTH {
			desiredWidths[colID] = DOMAINMINFIELDWIDTH
		}
	}

	// Check max data width for each column (no cap - we want actual desired size)
	for _, vals := range values {
		for colID, val := range vals {
			if colID < numColumns && len(val) > desiredWidths[colID] {
				desiredWidths[colID] = len(val)
			}
		}
	}

	// Calculate total desired width
	totalDesired := 0
	for _, w := range desiredWidths {
		totalDesired += w + 1 // +1 for separator
	}

	// If everything fits, use desired widths directly
	if totalDesired <= availableWidth {
		for colID := range domainColumnWidths {
			domainColumnWidths[colID] = desiredWidths[colID]
		}
	} else {
		// Not enough space - need to compress columns
		// Start with minimum widths
		for colID := range domainColumnWidths {
			domainColumnWidths[colID] = DOMAINMINFIELDWIDTH
		}

		// Calculate total width used (including 1 char separator per column)
		calcTotalWidth := func() int {
			total := 0
			for _, w := range domainColumnWidths {
				total += w + 1
			}
			return total
		}

		// Cap desired widths at DOMAINMAXFIELDWIDTH for constrained distribution
		cappedDesired := make([]int, numColumns)
		for colID := range desiredWidths {
			cappedDesired[colID] = desiredWidths[colID]
			if cappedDesired[colID] > DOMAINMAXFIELDWIDTH {
				cappedDesired[colID] = DOMAINMAXFIELDWIDTH
			}
		}

		// Distribute remaining space to columns that need it (up to capped max)
		for iteration := 0; iteration < 100; iteration++ {
			totalUsed := calcTotalWidth()
			remaining := availableWidth - totalUsed

			if remaining <= 0 {
				break
			}

			expanded := false
			for colID := range domainColumnWidths {
				if domainColumnWidths[colID] < cappedDesired[colID] && remaining > 0 {
					domainColumnWidths[colID]++
					remaining--
					expanded = true
				}
			}

			if !expanded {
				break
			}
		}
	}

	// Print group headers (row 1)
	window.Move(1, 1)
	currentGroup := ""
	currentPosX := 1
	groupStartX := make(map[string]int) // Track where each group starts

	for colID, field := range fields {
		fieldParts := strings.Split(field, "_")
		groupLabel := ""
		if len(fieldParts) > 1 {
			groupLabel = strings.Join(fieldParts[0:len(fieldParts)-1], " ")
		}
		if groupLabel != currentGroup {
			currentGroup = groupLabel
			groupStartX[groupLabel] = currentPosX
		}
		currentPosX += domainColumnWidths[colID] + 1
	}

	// Print group labels
	for group, startX := range groupStartX {
		if group != "" {
			window.Move(1, startX)
			window.Printf("%s", group)
		}
	}

	// Print field headers (row 2)
	window.Move(2, 1)
	for colID, field := range fields {
		fieldParts := strings.Split(field, "_")
		fieldName := fieldParts[len(fieldParts)-1]
		// Add sort direction indicator to sorted column
		if sortByColumn == colID {
			if sortAscending {
				fieldName = fieldName + "^"
			} else {
				fieldName = fieldName + "v"
			}
		}
		fieldLabel := padRight(fieldName, domainColumnWidths[colID])

		if sortByColumn == colID {
			window.AttrOn(goncurses.A_BOLD)
		} else {
			window.AttrOff(goncurses.A_BOLD)
		}

		window.AttrOn(goncurses.A_REVERSE)
		window.Printf("%s ", fieldLabel)
		window.AttrOff(goncurses.A_REVERSE)
	}
	window.AttrOff(goncurses.A_BOLD)

	// Create ordered domain list
	domainList := sortDomainIDsByField(values, sortByColumn)

	// Print domain rows
	rowCounter := 3
	for _, domain := range domainList {
		window.Move(rowCounter, 1)
		for colID, value := range values[domain.Key] {
			if sortByColumn == colID {
				window.AttrOn(goncurses.A_BOLD)
			} else {
				window.AttrOff(goncurses.A_BOLD)
			}

			width := DOMAINMAXFIELDWIDTH
			if colID < len(domainColumnWidths) {
				width = domainColumnWidths[colID]
			}
			val := padRight(value, width)
			window.Printf("%s ", val)
		}
		window.AttrOff(goncurses.A_BOLD)
		rowCounter++
	}
}

func sortDomainIDsByField(values map[string][]string, sortByColumn int) []KeyValue {
	var sorted []KeyValue
	for key, value := range values {
		if len(value) > sortByColumn {
			sorted = append(sorted, KeyValue{key, value[sortByColumn]})
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		// Try numeric comparison first
		vi, errI := strconv.ParseFloat(sorted[i].Value, 64)
		vj, errJ := strconv.ParseFloat(sorted[j].Value, 64)
		if errI == nil && errJ == nil {
			// Both are numeric
			if sortAscending {
				return vi < vj
			}
			return vi > vj
		}
		// Fall back to string comparison
		if sortAscending {
			return sorted[i].Value < sorted[j].Value
		}
		return sorted[i].Value > sorted[j].Value
	})
	return sorted
}

func prepareForCell(content string, columnID int) string {
	return expandCell(fitInCellWithWidth(content, columnID), columnID)
}

func fitInCellWithWidth(content string, columnID int) string {
	maxWidth := DOMAINMAXFIELDWIDTH
	if columnID < len(domainColumnWidths) {
		maxWidth = domainColumnWidths[columnID]
	}
	if len(content) > maxWidth {
		// Truncate with "..." to indicate overflow
		if maxWidth > 3 {
			return content[:maxWidth-3] + "..."
		}
		return content[:maxWidth]
	}
	return content
}

func fitInCell(content string) string {
	if len(content) > DOMAINMAXFIELDWIDTH {
		// Truncate with "..." to indicate overflow
		if DOMAINMAXFIELDWIDTH > 3 {
			return content[:DOMAINMAXFIELDWIDTH-3] + "..."
		}
		return content[:DOMAINMAXFIELDWIDTH]
	}
	return content
}

func expandCell(content string, columnID int) string {
	if len(domainColumnWidths) <= columnID {
		// set current width as column width
		domainColumnWidths = append(domainColumnWidths, len(content))
	} else if len(content) < domainColumnWidths[columnID] {
		// column width is larger, expand with spaces
		spaces := []string{""}
		diff := domainColumnWidths[columnID] - len(content)
		for i := 0; i < diff; i++ {
			spaces = append(spaces, " ")
		}
		content = content + strings.Join(spaces, "")
	} else if len(content) > domainColumnWidths[columnID] {
		// Truncate to column width
		width := domainColumnWidths[columnID]
		if width > 3 {
			content = content[:width-3] + "..."
		} else {
			content = content[:width]
		}
	}
	return content
}

// padRight pads a string to a fixed width, truncating with "..." if necessary
func padRight(s string, width int) string {
	if len(s) > width {
		// Truncate with "..." to indicate overflow
		if width > 3 {
			return s[:width-3] + "..."
		}
		return s[:width]
	}
	for len(s) < width {
		s += " "
	}
	return s
}

// printPhysicalNetDevices displays physical network interface statistics
func printPhysicalNetDevices(window *goncurses.Window) {
	maxy, maxx := window.MaxYX()

	// Get fields and data from collector
	allFields := netcollector.HostNetFields()
	allDeviceData := netcollector.HostPrintPerDevice()

	// Filter fields based on hiddenFields (but always keep DEVICE column)
	visibleFields := []string{}
	visibleIndices := []int{}
	for i, field := range allFields {
		if i == 0 || !hiddenFields[field] {
			visibleFields = append(visibleFields, field)
			visibleIndices = append(visibleIndices, i)
		}
	}

	// Build data rows with only visible columns
	type netDevRow struct {
		name   string
		values []string
	}
	rows := make([]netDevRow, 0, len(allDeviceData))
	for name, vals := range allDeviceData {
		visibleVals := make([]string, len(visibleIndices))
		for vi, origIdx := range visibleIndices {
			if origIdx < len(vals) {
				visibleVals[vi] = vals[origIdx]
			}
		}
		rows = append(rows, netDevRow{name: name, values: visibleVals})
	}

	// Sort by selected column
	sortCol := currentSortColumn
	if sortCol >= len(visibleFields) {
		sortCol = 0
	}
	sort.Slice(rows, func(i, j int) bool {
		if sortCol == 0 {
			// Sort by name alphabetically
			if sortAscending {
				return rows[i].name < rows[j].name
			}
			return rows[i].name > rows[j].name
		}
		// Parse as float for numeric comparison
		vi, _ := strconv.ParseFloat(rows[i].values[sortCol], 64)
		vj, _ := strconv.ParseFloat(rows[j].values[sortCol], 64)
		if sortAscending {
			return vi < vj
		}
		return vi > vj
	})

	// Calculate column widths dynamically (add space for sort indicator)
	widths := make([]int, len(visibleFields))
	for i, field := range visibleFields {
		fieldName := strings.TrimPrefix(field, "net_")
		widths[i] = len(fieldName) + 1 // +1 for sort indicator
		if widths[i] < 8 {
			widths[i] = 8
		}
	}
	for _, r := range rows {
		for i, v := range r.values {
			if i < len(widths) && len(v) > widths[i] {
				widths[i] = len(v)
			}
		}
	}

	// Print header with sort indicator
	window.Move(0, 0)
	for i, field := range visibleFields {
		fieldName := strings.TrimPrefix(field, "net_")
		// Add sort direction indicator
		if i == sortCol {
			if sortAscending {
				fieldName = fieldName + "^"
			} else {
				fieldName = fieldName + "v"
			}
			window.AttrOn(goncurses.A_BOLD | goncurses.A_REVERSE)
		} else {
			window.AttrOn(goncurses.A_BOLD)
		}
		format := fmt.Sprintf("%%%ds ", widths[i])
		if i == 0 {
			format = fmt.Sprintf("%%-%ds ", widths[i])
		}
		window.Printf(format, fieldName)
		window.AttrOff(goncurses.A_BOLD | goncurses.A_REVERSE)
	}

	// Print rows
	row := 1
	for _, r := range rows {
		if row >= maxy-1 {
			break
		}
		window.Move(row, 0)
		line := ""
		for i, v := range r.values {
			if i >= len(widths) {
				break
			}
			if i == 0 {
				line += fmt.Sprintf("%-*s ", widths[i], v)
			} else {
				line += fmt.Sprintf("%*s ", widths[i], v)
			}
		}
		if len(line) > maxx {
			line = line[:maxx]
		}
		window.Printf("%s", line)
		row++
	}

	window.NoutRefresh()
}

// printDiskDeviceView is a helper that displays disk devices from a specific category
func printDiskDeviceView(window *goncurses.Window, deviceData map[string][]string) {
	maxy, maxx := window.MaxYX()

	// Get fields from collector
	allFields := diskcollector.HostDiskFields()

	// Filter fields based on hiddenFields (but always keep DEVICE column)
	visibleFields := []string{}
	visibleIndices := []int{}
	for i, field := range allFields {
		if i == 0 || !hiddenFields[field] {
			visibleFields = append(visibleFields, field)
			visibleIndices = append(visibleIndices, i)
		}
	}

	// Build rows from device data
	type diskDevRow struct {
		name   string
		values []string
	}
	rows := make([]diskDevRow, 0, len(deviceData))
	for name, vals := range deviceData {
		visibleVals := make([]string, len(visibleIndices))
		for vi, origIdx := range visibleIndices {
			if origIdx < len(vals) {
				visibleVals[vi] = vals[origIdx]
			}
		}
		rows = append(rows, diskDevRow{name: name, values: visibleVals})
	}

	// Sort rows
	sortCol := currentSortColumn
	if sortCol >= len(visibleFields) {
		sortCol = 0
	}
	sort.Slice(rows, func(i, j int) bool {
		if sortCol == 0 {
			if sortAscending {
				return rows[i].name < rows[j].name
			}
			return rows[i].name > rows[j].name
		}
		vi, _ := strconv.ParseFloat(rows[i].values[sortCol], 64)
		vj, _ := strconv.ParseFloat(rows[j].values[sortCol], 64)
		if sortAscending {
			return vi < vj
		}
		return vi > vj
	})

	// Calculate column widths
	widths := make([]int, len(visibleFields))
	for i, field := range visibleFields {
		fieldName := strings.TrimPrefix(field, "dsk_")
		widths[i] = len(fieldName) + 1
		if widths[i] < 8 {
			widths[i] = 8
		}
	}
	for _, r := range rows {
		for i, v := range r.values {
			if i < len(widths) && len(v) > widths[i] {
				widths[i] = len(v)
			}
		}
	}

	// Print header
	window.Move(0, 0)
	for i, field := range visibleFields {
		fieldName := strings.TrimPrefix(field, "dsk_")
		if i == sortCol {
			if sortAscending {
				fieldName = fieldName + "^"
			} else {
				fieldName = fieldName + "v"
			}
			window.AttrOn(goncurses.A_BOLD | goncurses.A_REVERSE)
		} else {
			window.AttrOn(goncurses.A_BOLD)
		}
		format := fmt.Sprintf("%%%ds ", widths[i])
		if i == 0 {
			format = fmt.Sprintf("%%-%ds ", widths[i])
		}
		window.Printf(format, fieldName)
		window.AttrOff(goncurses.A_BOLD | goncurses.A_REVERSE)
	}

	// Print rows
	row := 1
	for _, r := range rows {
		if row >= maxy-1 {
			break
		}
		window.Move(row, 0)
		line := ""
		for i, v := range r.values {
			if i >= len(widths) {
				break
			}
			if i == 0 {
				line += fmt.Sprintf("%-*s ", widths[i], v)
			} else {
				line += fmt.Sprintf("%*s ", widths[i], v)
			}
		}
		if len(line) > maxx {
			line = line[:maxx]
		}
		window.Printf("%s", line)
		row++
	}

	window.NoutRefresh()
}

// printPhysicalDiskDevices displays physical disk device statistics (sd*, nvme*, vd*, etc.)
func printPhysicalDiskDevices(window *goncurses.Window) {
	categorized := diskcollector.HostPrintPerDeviceCategorized()
	printDiskDeviceView(window, categorized.Physical)
}

// printLVMDevices displays LVM logical volume statistics
func printLVMDevices(window *goncurses.Window) {
	categorized := diskcollector.HostPrintPerDeviceCategorized()
	printDiskDeviceView(window, categorized.LVM)
}

// printMpathDevices displays multipath device statistics
func printMpathDevices(window *goncurses.Window) {
	categorized := diskcollector.HostPrintPerDeviceCategorized()
	printDiskDeviceView(window, categorized.Mpath)
}
