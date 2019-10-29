package crawl

import (
	"os"
	"sync"
	"time"
)

// synchron holds the synchronisation tools and parameters
type synchron struct {
	timeout  time.Duration
	results  chan *Result
	stopChan chan struct{}
	mutex    *sync.Mutex
	group    sync.WaitGroup
	stopFlag bool
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

// checkout allows checks on the state of timeout for synchronisation. Only First call of this function returns true.
func (syn *synchron) checkout() bool {
	syn.mutex.Lock()
	defer syn.mutex.Unlock()

	first := !syn.stopFlag // only true if it was false first
	syn.stopFlag = true
	return first
}

// sendQuitSignal sends a signal only once, to signal shutdown
func (syn *synchron) sendQuitSignal() {
	if syn.checkout() {
		log.Info("Initiating shutdown.")

		// Send interrupt signal to ourselves, intercepted by signalHandler
		p, _ := os.FindProcess(os.Getpid())
		_ = p.Signal(os.Interrupt)

		// If timer is calling this function, crawler will pick the message, and inversely
		syn.stopChan <- struct{}{}
	}
}
