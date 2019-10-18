package crawl

import (
	"errors"
	"testing"
	"time"
)

var timeout = 5 * time.Second
var syn = newSynchron(timeout, 1)
var badURL = "https://example.com/%"

// TestNewCrawlerFail tests a failing condition for the newCrawler() function
func TestNewCrawlerFail(t *testing.T) {
	_, err := newCrawler(badURL, syn.results, timeout, 3)
	if err == nil {
		t.Errorf("newCrawler() should fail with invalid domain. URL : '%s'.", badURL)
	}
}

// TestInitialiseCrawler tests a failing condition for the initialiseCrawler() function
func TestInitialiseCrawlerFail(t *testing.T) {
	go signalHandler(syn)

	c := initialiseCrawler(badURL, syn)
	if c != nil {
		t.Errorf("initialiseCrawler() should fail with invalid domain. URL : '%s'.", badURL)
	}
	syn.group.Wait()
}

// TestHandleResult tests the right behaviour of handleResult() in case of an error in a result
func TestHandleResult(t *testing.T) {
	c := initialiseCrawler("http://example.com:8000/submit", syn)
	badResult := newResult("http://example.com", nil)
	badResult.err = errors.New("this a test error")
	c.handleResult(badResult)
	_, visited := c.visited[badResult.URL]
	if visited {
		t.Errorf("handleResult() should not mark a failing URL as visited.")
	}
}

// TestHandleResultError tests handleResultError
func TestHandleResultError(t *testing.T) {
	c := initialiseCrawler("http://example.com:8000/submit", syn)

	badResult := newResult("http://example.com", nil)
	badResult.err = errors.New("this a test error")

	// Test case we re-enqueue the result
	c.pending[badResult.URL] = c.maxRetry - 1
	c.handleResultError(badResult)
	if c.failed[badResult.URL] {
		t.Error("URL retries have not hit the maximum, should be marked as failed.")
	}

	// Test case we decide to mark a URL as failed
	c.pending[badResult.URL] = c.maxRetry
	c.handleResultError(badResult)
	_, failed := c.failed[badResult.URL]
	_, pending := c.pending[badResult.URL]
	if pending || !failed {
		t.Error("HandleResultError doesn't correctly switch the URL from pending to failed.")
	}
}
