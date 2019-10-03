package crawl

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
)

type crawler struct {
	domain  string
	visited map[string]bool
	pending map[string]bool
	todo    chan string
	results chan *result
}

type result struct {
	url   string
	links *[]string
}

func newCrawler(domain string) *crawler {
	// todo : retain radical domain from url
	return &crawler{
		domain:  domain,
		visited: make(map[string]bool),
		pending: make(map[string]bool),
		todo:    make(chan string, 100),
	}
}

func newResult(url string, links *[]string) *result {
	return &result{
		url:   url,
		links: links,
	}
}

// scrap retrieves a webpage, parses it for links, keeps only domain or relative links, sanitises them, an returns the result
func (c *crawler) scrap(url string) {

	// Retrieve page
	body, err := download(url)
	defer func() {
		err = body.Close()
		if err != nil {
			log.Error("Closing response body failed. But who cares ?")
		}
	}()
	if err != nil {
		log.Errorf("Encountered error on page '%s' : %s", url, err)
		c.results <- newResult(url, nil)
		return
	}

	// Retrieve links
	links := extractLinks(url, body)

	// Filter links by current domain
	log.Info("before filtering by domain : ", links)
	c.filterDomain(links)
	log.Info("after filtering by domain : ", links)

	// Enqueue results
	c.results <- newResult(url, &links)
}

// download retrieves the web page pointed to by the given url
func download(url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// filterDomain filters out links that are different from the crawler's scope
func (c *crawler) filterDomain(links []string) {
	log.Info("Filtering by domain")
	// todo : not sure if modifying in place will work here
	n := 0
	for _, link := range links {
		if strings.HasPrefix(link, c.domain) {
			links[n] = link
			n++
		} else {
			log.Trace("Filtering out element ", link)
		}
	}
	links = links[:n]
}

// filterVisited filters out links that have already been visited
func (c *crawler) filterVisited(links *[]string) []string {
	// todo : add check to filter out external domains

	filtered := make([]string, len(*links))
	// todo : modifying the slice in-place may be more efficient, setting a string to "" if don't keep
	//  it, and then only send to channel if it's non-""

	// filter out already encountered links
	for _, link := range *links {
		if _, ok := c.visited[link]; ok == false {
			// If value is not in map, we haven't visited it, thus keeping it
			filtered = append(filtered, link)
		}
	}

	return filtered
}

// handleResult treats the result of scraping a page for links
func (c *crawler) handleResult(result *result) {
	delete(c.pending, result.url)

	// If the download failed and links is nil
	if result.links == nil {
		// todo : handle pages that continuously fail on download
		c.todo <- result.url
		return
	}

	// Change state from pending to visited
	c.visited[result.url] = true

	// Filter out already visited links
	filtered := c.filterVisited(result.links)

	// Add filtered list in queue of links to visit
	for _, link := range filtered {
		c.todo <- link
	}

	// Print out result
	fmt.Printf("Found %d unvisited links on page %s : %s\n", len(filtered), result.url, *result.links)
}

// newTask triggers a new visit on a link
// todo : change that name
func (c *crawler) newTask(url string) {
	// Add to pending tasks
	c.pending[url] = true

	// Launch a worker goroutine on that link
	go c.scrap(url)
}

// crawl manages worker goroutines scraping pages and prints results
// todo : add a condition to quit when no more pages are to be visited
// todo : add a finer sync mechanism with workers when interrupting mid-request
// todo : keep a time tracker for stats
func crawl(domain string, syn *synchron) {
	defer syn.group.Done()

	c := newCrawler(domain)
	c.todo <- domain

loop:
	for {
		select {

		// Upon receiving a stop signal
		case <-syn.stopChan:
			log.Info("Stopping crawler.")
			close(c.todo)
			close(c.results)
			break loop

		// Upon receiving a resulting from a worker scraping a page
		case result := <-c.results:
			c.handleResult(result)

		// For every link that is left to visit in the queue
		case url := <-c.todo:
			c.newTask(url)
		}
	}
}
