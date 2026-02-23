package runners

import (
	"sync"
	"time"

	"proxtop/config"
	"proxtop/models"
)

var collectors []string

// LastRefreshDuration holds the actual measured time of the last refresh cycle
var LastRefreshDuration time.Duration

// InitializePrinter starts the periodic print calls
func InitializePrinter(wg *sync.WaitGroup) {
	// open configured printer
	models.Collection.Printer.Open()

	// define collectors and their order
	models.Collection.Collectors.Range(func(key interface{}, collector models.Collector) bool {
		collectorName := key.(string)
		collectors = append(collectors, collectorName)
		return true
	})

	// Wait for domains to be discovered before first print
	// Poll until we have at least one domain, with timeout
	waitStart := time.Now()
	for models.Collection.Domains.Length() == 0 {
		if time.Since(waitStart) > 5*time.Second {
			// Give up waiting - print headers anyway
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// start continuously printing values
	lastDataRefresh := time.Now()
	for n := 0; config.Options.Runs == -1 || n < config.Options.Runs; n++ {
		// For first print, don't wait - print immediately after collectors have run
		// For subsequent prints, wait for the configured interval
		if n > 0 {
			nextDataRun := lastDataRefresh.Add(time.Duration(config.Options.Frequency) * time.Second)

			// Wait until next data run time
			// If collection is paused (overlay shown), use short intervals for responsive UI
			// Otherwise, sleep until the next scheduled time
			for {
				if ForceRefresh {
					// Settings changed - refresh immediately
					ForceRefresh = false
					Print()
					break
				} else if CollectionPaused {
					// Overlay shown - redraw frequently for responsive UI
					time.Sleep(100 * time.Millisecond)
					Print()
				} else if time.Now().Before(nextDataRun) {
					// Normal mode - sleep in short intervals to check for ForceRefresh
					time.Sleep(50 * time.Millisecond)
				} else {
					break
				}
			}
		}

		// Measure actual refresh interval (only when data is collected)
		now := time.Now()
		LastRefreshDuration = now.Sub(lastDataRefresh)
		lastDataRefresh = now

		Print()
	}

	// close configured printer
	models.Collection.Printer.Close()

	// return from runner
	wg.Done()
}

// Print runs one printing cycle
func Print() {
	printable := models.Printable{}

	// add general domain fields first
	printable.DomainFields = []string{"UUID", "name"}
	printable.DomainValues = make(map[string][]string)
	models.Collection.Domains.Range(func(key, value interface{}) bool {
		uuid := key.(string)
		domain := value.(models.Domain)
		printable.DomainValues[uuid] = []string{
			uuid,
			domain.Name,
		}
		return true
	})

	// collect fields for each collector and merge together
	for _, collectorName := range collectors {
		collector, ok := models.Collection.Collectors.Load(collectorName)
		if !ok {
			continue
		}
		collectorPrintable := collector.Print()

		// merge host data
		printable.HostFields = append(printable.HostFields, collectorPrintable.HostFields[0:]...)
		printable.HostValues = append(printable.HostValues, collectorPrintable.HostValues[0:]...)

		// merge domain data
		printable.DomainFields = append(printable.DomainFields, collectorPrintable.DomainFields[0:]...)
		for uuid := range collectorPrintable.DomainValues {
			printable.DomainValues[uuid] = append(printable.DomainValues[uuid], collectorPrintable.DomainValues[uuid][0:]...)
		}
	}

	models.Collection.Printer.Screen(printable)
}
