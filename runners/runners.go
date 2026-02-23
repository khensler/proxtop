package runners

import (
	"sync"
)

var initialLookupDone chan bool

// CollectionPaused is set to true when overlays are shown to pause data collection
// This allows the UI to remain responsive during help/field selection
var CollectionPaused bool = false

// InitializeRunners starts necessary runners as threads
func InitializeRunners() {
	var wg sync.WaitGroup
	wg.Add(3) // terminate when all threads terminate

	initialLookupDone = make(chan bool, 1)

	go InitializeLookup(&wg)
	go InitializeCollect(&wg)
	go InitializePrinter(&wg)

	wg.Wait()
}
