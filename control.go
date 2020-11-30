package crawl

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Exit statuses indicate the context from which in the crawler returns
const (
	exitSignal  = "Received Signal"
	exitTimeout = "Timeout"
	exitLinks   = "Explored all Links"
)

const (
	exitErrorInit  = "Error in initialising crawler"
	exitErrorConf  = "Error in loading configuration"
	exitErrorInput = "Error in input validation"
)

// CrawlerResults is send back to the caller, containing results and information about the crawling
type CrawlerResults struct {
	links       []string       // list of all encountered links
	stream      chan *Response // channel streaming results as they arrive
	exitContext *string        // when the crawler returns, will hold the reason
	contextLock *sync.Mutex
}

func newCrawlerResults(syn *synchron) *CrawlerResults {
	return &CrawlerResults{
		links:       nil,
		stream:      syn.results,
		exitContext: &syn.exitContext,
		contextLock: &sync.Mutex{},
	}
}

func (cr *CrawlerResults) Links() []string {
	return cr.links
}

func (cr *CrawlerResults) Stream() <-chan *Response {
	return cr.stream
}

func (cr *CrawlerResults) ExitContext() string {
	cr.contextLock.Lock()
	defer cr.contextLock.Unlock()
	return *cr.exitContext
}

// timer implements a timeout (should be called as a goroutine)
func timer(syn *synchron, conf *config) {
	defer syn.group.Done()

	if syn.timeout <= 0 {
		return
	}

	timer := time.After(syn.timeout)

loop:
	for {
		select {
		// Quit if keyboard interruption
		case <-syn.stopChan:
			break loop

		// When timeout is reached, inform of timeout, send signal, and quit
		case t := <-timer:
			conf.Logger.log.Infof("Timing out after %0.3f seconds. time passed : %s\n", syn.timeout.Seconds(), t.String())
			syn.notifyStop(exitTimeout)
			break loop
		}
	}
}

// signalHandler is called as a goroutine to intercept signals and stop the program
func signalHandler(syn *synchron) {
	defer syn.group.Done()

	sig := make(chan os.Signal, 1)
	// os.Interrupt and os.Kill are the only signal values guaranteed to be present on all systems (except windows)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	// Block until a signal or stop is received
	select {
	case <-sig:
		syn.notifyStop(exitSignal)
		break

	case <-syn.stopChan:
		break
	}

	signal.Stop(sig)
}

// startCrawling launches the goroutines that constitute the crawler implementation.
func (c *Crawler) startCrawling(domain string) {
	if err := c.initialiseEngine(domain); err != nil {
		return
	}
	go signalHandler(c.syn)
	go timer(c.syn, c.config)
	go c.engine.run()

	c.syn.group.Wait()

	c.config.Logger.log.WithField("url", domain).Infof("Shutting down : %s", c.syn.exitContext)
	close(c.syn.results)
}
