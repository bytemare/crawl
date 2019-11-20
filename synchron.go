package crawl

import (
	"sync"
	"time"
)

// synchron holds the synchronisation tools and parameters
type synchron struct {
	timeout     time.Duration
	results     chan *Result
	stopChan    chan struct{}
	mutex       *sync.Mutex
	group       sync.WaitGroup
	stopFlag    bool
	stopContext string
}

// newSynchron returns an initialised synchron struct
func newSynchron(timeout time.Duration, nbParties int) *synchron {
	s := &synchron{
		timeout:  timeout,
		results:  make(chan *Result),
		group:    sync.WaitGroup{},
		stopChan: make(chan struct{}, 2),
		stopFlag: false,
		mutex:    &sync.Mutex{},
	}

	s.group.Add(nbParties)
	return s
}

// checkout reads state of the stop flag, toggling it if it is called the first time, and returns true in that case
// So only First call of this function returns true.
func (syn *synchron) checkout() bool {
	syn.mutex.Lock()
	defer syn.mutex.Unlock()

	first := !syn.stopFlag // only true if it was false first
	syn.stopFlag = true
	return first
}

// notifyStop notifies only once, on first call, to shutdown
func (syn *synchron) notifyStop(stopContext string) {
	// Only the first caller of checkout will have true returned
	if syn.checkout() {
		log.Infof("Initiating shutdown : %s", stopContext)

		// Register the reason/context for the shutdown
		syn.stopContext = stopContext

		// Sending messages to the two other listeners
		syn.stopChan <- struct{}{}
		syn.stopChan <- struct{}{}
	}
}
