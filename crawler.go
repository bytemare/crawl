package crawl

import (
	"net/url"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type crawler struct {
	task
	workers
	parameters
	output chan<- *LinkMap
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
	results chan *LinkMap
}

type workers struct {
	workerSync sync.WaitGroup
	workerStop chan struct{}
}

// LinkMap holds the links of the web page pointed to by url, of the same host as the url
type LinkMap struct {
	URL   string
	Links *[]string
	Error error
}

// newCrawler returns an initialised crawler struct
func newCrawler(domain string, output chan<- *LinkMap, timeout time.Duration, maxRetry int) (*crawler, error) {
	dURL, err := url.Parse(domain)
	if err != nil {
		return nil, err
	}

	return &crawler{
		task: task{
			linkStates: linkStates{
				visited: make(map[string]bool),
				pending: make(map[string]int),
				failed:  make(map[string]bool),
			},
			todo:    make(chan string, 100),
			results: make(chan *LinkMap, 100),
		},
		workers: workers{
			workerSync: sync.WaitGroup{},
			workerStop: make(chan struct{}),
		},
		parameters: parameters{
			domain:         dURL,
			requestTimeout: timeout,
			maxRetry:       maxRetry,
		},
		output: output,
	}, nil
}

//  newLinkMap returns an initialised LinkMap struct
func newLinkMap(url string, links *[]string) *LinkMap {
	return &LinkMap{
		URL:   url,
		Links: links,
		Error: nil,
	}
}

// scraper serves a worker goroutine.
// It retrieves a web page, parses it for links,
// keeps only domain or relative links, sanitises them, an returns the LinkMap
func (c *crawler) scraper(url string) {
	defer c.workerSync.Done()

	// LinkMap will hold the links on success, or send as is on error
	res := newLinkMap(url, nil)

	// Scrap and retrieve links
	log.WithField("url", url).Tracef("Attempting download.")
	links, err := cancellableScrapLinks(url, c.requestTimeout, c.workerStop)
	if err != nil {
		log.WithField("url", url).Tracef("Download failed : %s", err)
		res.Error = err
	} else if links != nil {
		// Filter links by current domain
		links = c.filterHost(links)
		res.Links = &links
	}

	// Don't send results if we're being asked to stop
	select {
	case <-c.workerStop:
		return

	// Enqueue results
	case c.results <- res:
	}
}

// filterHost filters out links that are different from the crawler's scope
func (c *crawler) filterHost(links []string) []string {
	n := 0
	for _, link := range links {
		linkURL, _ := url.Parse(link)
		if linkURL.Host == c.domain.Host {
			links[n] = link
			n++
		} else {
			log.WithField("host", c.domain.Host).Tracef("Filtering out link to %s.", link)
		}
	}
	return links[:n]
}

// filterLinks filters out links that have already been visited or are in pending treatment
func (c *crawler) filterLinks(links []string) []string {
	n := 0
	// Only keep links that are neither pending or visited
	for _, link := range links {
		// If pending, skip
		if _, ok := c.pending[link]; ok {
			log.WithField("status", "pending").Tracef("Discarding %s.", link)
			continue
		}

		// If visited, skip
		if _, ok := c.visited[link]; ok {
			log.WithField("status", "pending").Tracef("Discarding %s.", link)
			continue
		}

		// Keep the link
		links[n] = link
		n++
	}
	return links[:n]
}

// handleResultError handles the error a LinkMap has upon return of a link scraping attempt
func (c *crawler) handleResultError(res *LinkMap) {
	log.WithField("url", res.URL).Tracef("LinkMap returned with error : %s", res.Error)

	// If we tried to much, mark it as failed
	if c.pending[res.URL] >= c.maxRetry {
		c.failed[res.URL] = true
		delete(c.pending, res.URL)
		log.WithField("url", res.URL).Errorf("Discarding. Page unreachable after %d attempts.\n", c.maxRetry)
		return
	}

	// If we have not reached maximum retries, re-enqueue
	c.todo <- res.URL
}

// handleResult treats the LinkMap of scraping a page for links
func (c *crawler) handleResult(result *LinkMap) {
	if result.Error != nil {
		c.handleResultError(result)
		return
	}

	// Change state from pending to visited
	c.visited[result.URL] = true
	delete(c.pending, result.URL)

	// Filter out already visited links
	log.WithField("url", result.URL).Tracef("Filtering links.")
	filtered := c.filterLinks(*result.Links)
	result.Links = &filtered

	// Add filtered list in queue of links to visit
	for _, link := range filtered {
		c.todo <- link
	}

	// Log LinkMap and send them to caller
	log.WithFields(logrus.Fields{
		"url":   result.URL,
		"links": filtered,
	}).Infof("Found %d unvisited links.", len(filtered))
	c.output <- result
}

// newTask triggers a new visit on a link
func (c *crawler) newTask(url string) {
	// Add to pending tasks
	c.pending[url]++

	// Launch a worker goroutine on that link
	c.workerSync.Add(1)
	go c.scraper(url)
}

// checkProgress verifies if there are pages left to scrap or being scraped. Returns false if not.
func (c *crawler) checkProgress() bool {
	return len(c.todo) != 0 || len(c.pending) != 0
}

// initialiseCrawler initialises and returns a new crawler struct
func initialiseCrawler(domain string, syn *synchron, conf *config) *crawler {
	c, err := newCrawler(domain, syn.results, conf.Requests.Timeout, int(conf.Requests.Retries))
	if err != nil {
		log.WithField("url", domain).Error(err)
		syn.notifyStop(exitErrorInit)
		return nil
	}
	c.todo <- c.domain.String()
	return c
}

// quitCrawler initiates the shutdown process of the crawler
func (c *crawler) quitCrawler(syn *synchron) {
	// Declare intend to stop
	syn.notifyStop(exitLinks)

	// Inform launched workers to stop, and wait for them
	close(c.workerStop)
	c.workerSync.Wait()

	log.WithField("url", c.domain.String()).Infof("Visited %d links. %d failed.", len(c.visited), len(c.failed))
}

// crawl manages worker goroutines scraping pages and prints results
func crawl(domain string, syn *synchron, conf *config) {
	defer syn.group.Done()

	c := initialiseCrawler(domain, syn, conf)
	if c == nil {
		return
	}
	ticker := time.NewTicker(time.Second)
loop:
	for {
		select {
		// Upon receiving a stop signal
		case <-syn.stopChan:
			break loop

		// Upon receiving a resulting from a worker scraping a page
		case result := <-c.results:
			c.handleResult(result)

		// For every link that is left to visit in the queue
		case link := <-c.todo:
			c.newTask(link)

		// Every tick, verify if there are jobs or pending tasks left
		case <-ticker.C:
			if !c.checkProgress() {
				break loop
			}
		}
	}
	ticker.Stop()
	c.quitCrawler(syn)
}
