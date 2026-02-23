package printers

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/cha87de/goncurses"
	"proxtop/collectors/diskcollector"
	"proxtop/collectors/netcollector"
	"proxtop/config"
	"proxtop/models"
	"proxtop/runners"
	"proxtop/util"
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
const DOMAINMAXFIELDWIDTH = 12  // Increased from 8 for better alignment
const HOSTFIELDWIDTH = 10       // Width for host field names
const HOSTVALUEWIDTH = 12       // Width for host values

type KeyValue struct {
	Key   string
	Value string
}

var domainColumnWidths []int
var currentViewMode ViewMode = ViewAll
var currentSortColumn int = 3
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

	// Initialize field selection state
	hiddenFields = make(map[string]bool)
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
	case '<':
		if currentSortColumn > 0 {
			currentSortColumn--
		}
	case '>':
		currentSortColumn++
	case '+', '=': // Increase refresh interval
		if config.Options.Frequency < 60 {
			config.Options.Frequency++
		}
	case '-', '_': // Decrease refresh interval
		if config.Options.Frequency > 1 {
			config.Options.Frequency--
		}
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
	default:
		return "ALL"
	}
}

// filterFieldsByView filters fields and values based on current view mode and hidden fields
func filterFieldsByView(fields []string, values map[string][]string) ([]string, map[string][]string) {
	var prefix string
	switch currentViewMode {
	case ViewCPU:
		prefix = "cpu_"
	case ViewMem:
		prefix = "mem_"
	case ViewDisk:
		prefix = "dsk_"
	case ViewNet:
		prefix = "net_"
	case ViewIO:
		prefix = "io_"
	default:
		prefix = "" // ViewAll - no prefix filtering
	}

	// Always include UUID and name (first two columns)
	filteredFields := []string{}
	includeIndices := []int{}

	for i, field := range fields {
		// Skip hidden fields (but always include first two columns)
		if i >= 2 && hiddenFields[field] {
			continue
		}

		// Filter by prefix (or include all if no prefix)
		if i < 2 || prefix == "" || strings.HasPrefix(strings.ToLower(field), prefix) {
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
	statusLine := fmt.Sprintf(" proxtop | View: %s | Refresh: %.1fs (target: %ds, +/-) | 'h' help, 'q' quit ",
		getViewModeName(), actualInterval, config.Options.Frequency)
	for len(statusLine) < maxx {
		statusLine += " "
	}
	if len(statusLine) > maxx {
		statusLine = statusLine[:maxx]
	}
	screen.Printf("%s", statusLine)
	screen.AttrOff(goncurses.A_REVERSE)

	// Handle physical device views differently
	if currentViewMode == ViewPhysNet || currentViewMode == ViewPhysDisk {
		// Use full screen for device list (no host panel)
		deviceWin, _ := goncurses.NewWindow(maxy-1, maxx, 1, 0)
		goncurses.UpdatePanels()
		goncurses.Update()
		goncurses.NewPanel(deviceWin)

		if currentViewMode == ViewPhysNet {
			printPhysicalNetDevices(deviceWin)
		} else {
			printPhysicalDiskDevices(deviceWin)
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

	var prefix string
	switch currentViewMode {
	case ViewCPU:
		prefix = "cpu_"
	case ViewMem:
		prefix = "mem_"
	case ViewDisk:
		prefix = "dsk_"
	case ViewNet:
		prefix = "net_"
	case ViewIO:
		prefix = "io_"
	default:
		return fields, values
	}

	filteredFields := []string{}
	filteredValues := []string{}

	for i, field := range fields {
		if strings.HasPrefix(strings.ToLower(field), prefix) {
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
	helpHeight := 26
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
		screen.Printf("a/c/m/d/n/i - View modes | +/- Refresh | q - Quit")
		screen.Refresh()
		return
	}

	helpWin.Box(0, 0)

	helpWin.Move(1, 2)
	helpWin.AttrOn(goncurses.A_BOLD)
	helpWin.Printf("proxtop - Keybindings")
	helpWin.AttrOff(goncurses.A_BOLD)

	helpWin.Move(3, 2)
	helpWin.Printf("View Modes:")
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
	helpWin.Move(10, 4)
	helpWin.Printf("p - Show PHYSICAL NETWORK interfaces")
	helpWin.Move(11, 4)
	helpWin.Printf("s - Show PHYSICAL DISK devices")

	helpWin.Move(13, 2)
	helpWin.Printf("Sorting:")
	helpWin.Move(14, 4)
	helpWin.Printf("< - Sort by previous column")
	helpWin.Move(15, 4)
	helpWin.Printf("> - Sort by next column")

	helpWin.Move(17, 2)
	helpWin.Printf("Refresh Interval:")
	helpWin.Move(18, 4)
	helpWin.Printf("+ - Increase interval (slower)")
	helpWin.Move(19, 4)
	helpWin.Printf("- - Decrease interval (faster)")

	helpWin.Move(21, 2)
	helpWin.Printf("Other:")
	helpWin.Move(22, 4)
	helpWin.Printf("f - Field selector (show/hide columns)")
	helpWin.Move(23, 4)
	helpWin.Printf("h/? - Toggle this help")
	helpWin.Move(24, 4)
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
	for _, field := range domainFields {
		// Filter by view mode prefix
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
	// Reset column widths for this view
	domainColumnWidths = make([]int, len(fields))

	// Calculate optimal column widths based on field names and data
	for colID, field := range fields {
		fieldParts := strings.Split(field, "_")
		fieldName := fieldParts[len(fieldParts)-1]
		domainColumnWidths[colID] = len(fieldName)
		if domainColumnWidths[colID] < DOMAINMAXFIELDWIDTH {
			domainColumnWidths[colID] = DOMAINMAXFIELDWIDTH
		}
	}

	// Check data widths
	for _, vals := range values {
		for colID, val := range vals {
			if colID < len(domainColumnWidths) && len(val) > domainColumnWidths[colID] {
				domainColumnWidths[colID] = len(val)
				if domainColumnWidths[colID] > DOMAINMAXFIELDWIDTH {
					domainColumnWidths[colID] = DOMAINMAXFIELDWIDTH
				}
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
		if len(value) >= sortByColumn {
			sorted = append(sorted, KeyValue{key, value[sortByColumn]})
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})
	return sorted
}

func prepareForCell(content string, columnID int) string {
	return expandCell(fitInCell(content), columnID)
}

func fitInCell(content string) string {
	if len(content) > DOMAINMAXFIELDWIDTH {
		tmp := strings.Split(content, "")
		content = strings.Join(tmp[0:DOMAINMAXFIELDWIDTH], "")
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
		// column width is smaller, store larger width
		domainColumnWidths[columnID] = len(content)
	}
	return content
}

// padRight pads a string to a fixed width, truncating if necessary
func padRight(s string, width int) string {
	if len(s) > width {
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

	// Get physical network devices
	devices := util.GetPhysicalNetDevices()

	// Column headers
	headers := []string{"INTERFACE", "RX-Bytes", "RX-Pkts", "RX-Errs", "RX-Drop", "TX-Bytes", "TX-Pkts", "TX-Errs", "TX-Drop"}
	widths := []int{15, 12, 12, 12, 12, 12, 12, 10, 10}

	// Build sortable list
	type netDevRow struct {
		name   string
		values []uint64
	}
	rows := make([]netDevRow, 0, len(devices))
	for name, dev := range devices {
		rows = append(rows, netDevRow{
			name: name,
			values: []uint64{
				dev.ReceivedBytes, dev.ReceivedPackets, dev.ReceivedErrs, dev.ReceivedDrop,
				dev.TransmittedBytes, dev.TransmittedPackets, dev.TransmittedErrs, dev.TransmittedDrop,
			},
		})
	}

	// Sort by selected column (0=name, 1-8=numeric columns)
	sortCol := currentSortColumn
	if sortCol >= len(headers) {
		sortCol = 0
	}
	sort.Slice(rows, func(i, j int) bool {
		if sortCol == 0 {
			return rows[i].name < rows[j].name
		}
		return rows[i].values[sortCol-1] > rows[j].values[sortCol-1] // descending for numeric
	})

	// Print header with sort indicator
	window.Move(0, 0)
	col := 0
	for i, h := range headers {
		if i == sortCol {
			window.AttrOn(goncurses.A_BOLD | goncurses.A_REVERSE)
		} else {
			window.AttrOn(goncurses.A_BOLD)
		}
		format := fmt.Sprintf("%%-%ds", widths[i])
		if i > 0 {
			format = fmt.Sprintf("%%%ds", widths[i])
		}
		window.Printf(format+" ", h)
		if i == sortCol {
			window.AttrOff(goncurses.A_BOLD | goncurses.A_REVERSE)
		} else {
			window.AttrOff(goncurses.A_BOLD)
		}
		col += widths[i] + 1
	}

	// Print rows
	row := 1
	for _, r := range rows {
		if row >= maxy-1 {
			break
		}
		window.Move(row, 0)
		line := fmt.Sprintf("%-15s %12d %12d %12d %12d %12d %12d %10d %10d",
			r.name,
			r.values[0], r.values[1], r.values[2], r.values[3],
			r.values[4], r.values[5], r.values[6], r.values[7])
		if len(line) > maxx {
			line = line[:maxx]
		}
		window.Printf("%s", line)
		row++
	}

	window.NoutRefresh()
}

// printPhysicalDiskDevices displays physical disk device statistics
func printPhysicalDiskDevices(window *goncurses.Window) {
	maxy, maxx := window.MaxYX()

	// Get disk stats from /proc/diskstats
	devices := util.GetProcDiskstats()

	// Column headers
	headers := []string{"DEVICE", "READS", "RD-MRGD", "SECT-RD", "WRITES", "WR-MRGD", "SECT-WR", "IO-OPS", "IO-TIME"}
	widths := []int{12, 10, 10, 12, 10, 10, 12, 8, 10}

	// Build sortable list (filtering as we go)
	type diskDevRow struct {
		name   string
		values []uint64
	}
	rows := make([]diskDevRow, 0)
	for name, dev := range devices {
		// Filter out partitions (only show base devices) and loop/ram devices
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") {
			continue
		}
		// Skip dm- devices
		if strings.HasPrefix(name, "dm-") {
			continue
		}
		rows = append(rows, diskDevRow{
			name: name,
			values: []uint64{
				dev.Reads, dev.ReadsMerged, dev.SectorsRead,
				dev.Writes, dev.WritesMerged, dev.SectorsWritten,
				dev.CurrentOps, dev.TimeForOps,
			},
		})
	}

	// Sort by selected column (0=name, 1-8=numeric columns)
	sortCol := currentSortColumn
	if sortCol >= len(headers) {
		sortCol = 0
	}
	sort.Slice(rows, func(i, j int) bool {
		if sortCol == 0 {
			return rows[i].name < rows[j].name
		}
		return rows[i].values[sortCol-1] > rows[j].values[sortCol-1] // descending for numeric
	})

	// Print header with sort indicator
	window.Move(0, 0)
	col := 0
	for i, h := range headers {
		if i == sortCol {
			window.AttrOn(goncurses.A_BOLD | goncurses.A_REVERSE)
		} else {
			window.AttrOn(goncurses.A_BOLD)
		}
		format := fmt.Sprintf("%%-%ds", widths[i])
		if i > 0 {
			format = fmt.Sprintf("%%%ds", widths[i])
		}
		window.Printf(format+" ", h)
		if i == sortCol {
			window.AttrOff(goncurses.A_BOLD | goncurses.A_REVERSE)
		} else {
			window.AttrOff(goncurses.A_BOLD)
		}
		col += widths[i] + 1
	}

	// Print rows
	row := 1
	for _, r := range rows {
		if row >= maxy-1 {
			break
		}
		window.Move(row, 0)
		line := fmt.Sprintf("%-12s %10d %10d %12d %10d %10d %12d %8d %10d",
			r.name,
			r.values[0], r.values[1], r.values[2], r.values[3],
			r.values[4], r.values[5], r.values[6], r.values[7])
		if len(line) > maxx {
			line = line[:maxx]
		}
		window.Printf("%s", line)
		row++
	}

	window.NoutRefresh()
}
