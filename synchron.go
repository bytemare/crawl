package crawl

import (
	"sync"
	"time"
)

const (
	stopChanParties = 2
)

// synchron holds the synchronisation tools and parameters
type synchron struct {
	timeout     time.Duration
	results     chan *Response
	stopChan    chan struct{}
	mutex       *sync.Mutex
	group       sync.WaitGroup
	stopFlag    bool
	exitContext string
}

// newSynchron returns an initialised synchron struct
func newSynchron(timeout time.Duration, nbParties int) *synchron {
	s := &synchron{
		timeout:  timeout,
		results:  make(chan *Response),
		group:    sync.WaitGroup{},
		stopChan: make(chan struct{}, stopChanParties),
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
func (syn *synchron) notifyStop(exitContext string) {
	// Only the first caller of checkout will have true returned, thus entering here
	if syn.checkout() {
		// todo log.Infof("Initiating shutdown : %s", exitContext)

		// Register the reason/context for the shutdown
		syn.exitContext = exitContext

		// Sending messages to the two other listeners
		syn.stopChan <- struct{}{}
		syn.stopChan <- struct{}{}
	}
}
