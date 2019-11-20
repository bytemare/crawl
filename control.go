package crawl

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// timer implements a timeout (should be called as a goroutine)
func timer(syn *synchron) {
	defer syn.group.Done()

	if syn.timeout <= 0 {
		log.Info("No value assigned for timeout. Timer will not run.")
		return
	}

	timer := time.After(syn.timeout)

loop:
	for {
		select {
		// Quit if keyboard interruption
		case <-syn.stopChan:
			log.Trace("Timer received stop message. Stopping Timer.")
			break loop

		// When timeout is reached, inform of timeout, send signal, and quit
		case t := <-timer:
			log.Infof("Timing out after %0.3f seconds. time passed : %s\n", syn.timeout.Seconds(), t.String())
			syn.notifyStop("Timeout")
			break loop
		}
	}
}

// signalHandler is called as a goroutine to intercept signals and stop the program
func signalHandler(syn *synchron) {
	defer syn.group.Done()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal or stop is received
	select {
	case <-sig:
		syn.notifyStop("Received Signal")
		break

	case <-syn.stopChan:
		break
	}

	signal.Stop(sig)
}

// validateInput returns whether input is valid and can be worked with
func validateInput(domain string, timeout time.Duration) error {
	// We can't crawl without a target domain
	if domain == "" {
		return errors.New("if you want to crawl something, please specify the target domain as argument")
	}

	// Check whether domain is of valid form
	if _, err := url.ParseRequestURI(domain); err != nil {
		msg := fmt.Sprintf("Invalid url : you must specify a valid target domain/url to crawl : %s", err)
		return errors.New(msg)
	}

	// Invalid timeout values are handled later, but let's not the user mess with us
	if timeout < 0 {
		msg := fmt.Sprintf("Invalid timeout value '%d' (accepted values [0 ; +yourpatience [, in seconds)", timeout)
		return errors.New(msg)
	}

	return nil
}

// startCrawling launches the goroutines that constitute the crawler implementation.
func startCrawling(domain string, syn *synchron, config *config) {
	go signalHandler(syn)
	go timer(syn)
	go crawl(domain, syn, config)

	syn.group.Wait()

	log.WithField("url", domain).Infof("Shutting down : %s", syn.stopContext)
	close(syn.results)
}

// StreamLinks returns a channel on which it will report links as they come during the crawling.
// The caller should range over than channel to continuously retrieve messages. StreamLinks will close that channel
// when all encountered links have been visited and none is left, when the deadline on the timeout parameter is reached,
// or if a SIGINT or SIGTERM signals is received.
func StreamLinks(domain string, timeout time.Duration) (chan *Result, error) {
	// Check env and initialise logging
	conf, err := initialiseCrawlConfiguration()
	if err != nil && conf == nil {
		return nil, err
	}

	if err = validateInput(domain, timeout); err != nil {
		return nil, err
	}

	syn := newSynchron(timeout, 3)
	log.WithField("url", domain).Info("Starting web crawler.")

	go startCrawling(domain, syn, conf)

	return syn.results, nil
}

// FetchLinks is a wrapper around StreamLinks and does the same, except it blocks and accumulates all links before
// returning them to the caller.
// todo : return indicator of stop condition : signal, timeout, end of tree
func FetchLinks(domain string, timeout time.Duration) ([]string, error) {
	results, err := StreamLinks(domain, timeout)
	if err != nil {
		return nil, err
	}
	links := make([]string, 0, 100) // todo : trade-off here, look if we really need that

	for res := range results {
		links = append(links, *res.Links...)
	}

	return links, nil
}

// ScrapLinks returns the links found in the web page pointed to by url
func ScrapLinks(url string, timeout time.Duration) ([]string, error) {
	// Check env and initialise logging
	conf, err := initialiseCrawlConfiguration()
	if err != nil && conf == nil {
		return nil, err
	}
	return scrapLinks(url, timeout)
}
