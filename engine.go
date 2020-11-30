package crawl

import (
	"net/url"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
)

const (
	defNbTodo    = 100
	defNbResults = 100
)

type engine struct {
	task
	workers
	parameters
	// connections map[string]*http.Client // map Client connections to hosts
	output  chan<- *Response
	crawler *Crawler
}

type parameters struct {
	domain         *url.URL
	requestTimeout time.Duration
	maxRetry       int
}

type linkStates struct {
	pending map[string]int
	visited map[string]bool
	failed  map[string]bool
}

type task struct {
	linkStates
	todo    chan string
	results chan *Response
}

type workers struct {
	workerSync sync.WaitGroup
	workerStop chan struct{}
}

// newEngine returns an initialised engine struct
func (c *Crawler) newEngine(URL *url.URL) {
	c.engine = &engine{
		task: task{
			linkStates: linkStates{
				visited: make(map[string]bool),
				pending: make(map[string]int),
				failed:  make(map[string]bool),
			},
			todo:    make(chan string, defNbTodo),
			results: make(chan *Response, defNbResults),
		},
		workers: workers{
			workerSync: sync.WaitGroup{},
			workerStop: make(chan struct{}),
		},
		parameters: parameters{
			domain:         URL,
			requestTimeout: c.config.Requests.Timeout,
			maxRetry:       int(c.config.Requests.Retries),
		},
		output:  c.syn.results,
		crawler: c,
	}
}

// initialiseEngine initialises and returns a new engine struct
func (c *Crawler) initialiseEngine(startURL string) error {
	// Verify URL
	dURL, err := url.Parse(startURL)
	if err != nil {
		c.syn.notifyStop(exitErrorInit)
		return errors.Wrapf(err, "%s", exitErrorInit)
	}

	// Build the engine
	c.newEngine(dURL)

	// Queue first task
	c.engine.todo <- dURL.String()
	return nil
}

// scraper serves a worker goroutine.
// It retrieves a web page, parses it for links,
// keeps only domain or relative links, sanitises them, an returns the LinkMap
func (e *engine) scraper(URL string) {
	defer e.workerSync.Done()

	// Scrap and retrieve links
	e.crawler.config.Logger.log.WithField("url", URL).Tracef("Attempting download.")
	resp, err := scrapLinks(URL, e.requestTimeout, e.workerStop)
	if err != nil {
		if merr, ok := err.(*multierror.Error); ok {
			e.crawler.config.Logger.log.WithField("url", URL).
				Tracef("Encountered errors in scraping page : %s", merr.Errors)
		} else {
			e.crawler.config.Logger.log.WithField("url", URL).
				Tracef("Encountered errors in scraping page : %s", err)
		}
		resp.Error = err
	} else if resp.Links != nil {
		// If links were found, filter authorised domains
		resp.Links = e.filterHost(resp.Links)
	}

	// Don't send results if we're being asked to stop
	select {
	case <-e.workerStop:
		return

	// Enqueue results
	case e.results <- resp:
	}
}

// filterHost filters out links that are different from the engine's scope
func (e *engine) filterHost(links []string) []string {
	n := 0
	for _, link := range links {
		linkURL, _ := url.Parse(link)
		// TODO : handle error
		if linkURL.Host == e.domain.Host {
			links[n] = link
			n++
		}
	}
	return links[:n]
}

// filterLinks filters out links that have already been visited or are in pending treatment
func (e *engine) filterLinks(links []string) []string {
	n := 0
	// Only keep links that are neither pending or visited
	for _, link := range links {
		// If pending, skip
		if _, ok := e.pending[link]; ok {
			continue
		}

		// If visited, skip
		if _, ok := e.visited[link]; ok {
			continue
		}

		// Keep the link
		links[n] = link
		n++
	}
	return links[:n]
}

// handleResultError handles the error a LinkMap has upon return of a link scraping attempt
func (e *engine) handleResultError(response *Response) {
	rurl := response.request.url.Path
	// If we tried to much, mark it as failed
	if e.pending[rurl] >= e.maxRetry {
		e.failed[rurl] = true
		delete(e.pending, response.request.url.Path)
		e.crawler.config.Logger.log.WithField("url", rurl).
			Errorf("Discarding. Page unreachable after %d attempts.\n", e.maxRetry)
		return
	}

	// If we have not reached maximum retries, re-enqueue
	e.todo <- rurl
}

// handleResult treats the LinkMap of scraping a page for links
func (e *engine) handleResult(response *Response) {
	if response.Error != nil {
		e.handleResultError(response)
		return
	}

	// Change state from pending to visited
	e.visited[response.request.url.Path] = true
	delete(e.pending, response.request.url.Path)

	// Filter out already visited links
	filtered := e.filterLinks(response.Links)
	response.Links = filtered

	// Add filtered list in queue of links to visit
	for _, link := range filtered {
		e.todo <- link
	}

	// Log LinkMap and send them to caller
	e.crawler.config.Logger.log.WithFields(logrus.Fields{
		"url":   response.request.url.Path,
		"links": filtered,
	}).Infof("Found %d unvisited links.", len(filtered))
	e.output <- response
}

// newTask triggers a new visit on a link
func (e *engine) newTask(URL string) {
	// Add to pending tasks
	e.pending[URL]++

	// Launch a worker goroutine on that link
	e.workerSync.Add(1)
	go e.scraper(URL)
}

// checkProgress verifies if there are pages left to scrap or being scraped. Returns false if not.
func (e *engine) checkProgress() bool {
	return len(e.todo) != 0 || len(e.pending) != 0
}

// stopEngine initiates the shutdown process of the engine
func (e *engine) stopEngine() {
	// Declare intend to stop
	e.crawler.syn.notifyStop(exitLinks)

	// Inform launched workers to stop, and wait for them
	close(e.crawler.engine.workerStop)
	e.workerSync.Wait()
	e.crawler.config.Logger.log.WithField("url", e.domain.String()).
		Infof("Visited %d links. %d failed.", len(e.visited), len(e.failed))
}

// run starts the crawler engine, manages worker goroutines scraping pages and streams results
func (e *engine) run() {
	defer e.crawler.syn.group.Done()

	ticker := time.NewTicker(time.Second)
loop:
	for {
		select {
		// Upon receiving a stop signal
		case <-e.crawler.syn.stopChan:
			break loop

		// Upon receiving a resulting from a worker scraping a page
		case result := <-e.results:
			e.handleResult(result)

		// For every link that is left to visit in the queue
		case link := <-e.todo:
			e.newTask(link)

		// Every tick, verify if there are jobs or pending tasks left
		case <-ticker.C:
			if !e.checkProgress() {
				break loop
			}
		}
	}
	ticker.Stop()
	e.stopEngine()
}
