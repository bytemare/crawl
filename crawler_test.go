package crawl

import (
	"errors"
	"io"
	"testing"
	"time"
)

var (
	timeout    = 3 * time.Second
	syn        = newSynchron(timeout, 1)
	urlBad     = "https://example.com/%"
	urlValid   = "https://example.com"
	urlTimeout = "http://example.com:8000/submit"
)

// TestNewCrawlerFail tests a failing condition for the newCrawler() function
func TestNewCrawlerFail(t *testing.T) {
	_, err := newCrawler(urlBad, syn.results, timeout, 3)
	if err == nil {
		t.Errorf("newCrawler() should fail with invalid domain. URL : '%s'.", urlBad)
	}
}

// TestInitialiseCrawler tests a failing condition for the initialiseCrawler() function
func TestInitialiseCrawlerFail(t *testing.T) {
	go signalHandler(syn)

	time.Sleep(200 * time.Millisecond)

	c := initialiseCrawler(urlBad, syn)
	if c != nil {
		t.Errorf("initialiseCrawler() should fail with invalid domain. URL : '%s'.", urlBad)
	}
	syn.group.Wait()
}

// TestScraperFail test a failing condition for the scraper method
func TestScraperFail(t *testing.T) {
	c := initialiseCrawler(urlValid, syn)
	c.workerSync.Add(1)
	go c.scraper(urlBad)
	c.workerSync.Wait()
	result := <-c.results
	if result.err == nil {
		t.Errorf("scraper() should flag an error in the returning result. URL : '%s'.", urlBad)
	}
}

// TestDownloadFail tests a failing condition for the download function
func TestDownloadFail(t *testing.T) {
	var close = func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}

	// Should fail on request building
	body, err := download(urlBad, timeout)
	if err == nil {
		t.Errorf("download() should fail on invalid link. URL : '%s'", urlBad)
	}
	close(body)

	// Should fail on request execution due to timeout
	body, err = download(urlTimeout, timeout)
	if err == nil {
		t.Errorf("download() should timeout on non-ending request. URL : '%s'", urlTimeout)
	}
	close(body)
}

// TestHandleResult tests the right behaviour of handleResult() in case of an error in a result
func TestHandleResult(t *testing.T) {
	c := initialiseCrawler(urlValid, syn)
	badResult := newResult(urlBad, nil)
	badResult.err = errors.New("this a test error")
	c.handleResult(badResult)
	_, visited := c.visited[badResult.URL]
	if visited {
		t.Errorf("handleResult() should not mark a failing URL as visited.")
	}
}

// TestHandleResultError tests handleResultError
func TestHandleResultError(t *testing.T) {
	c := initialiseCrawler(urlValid, syn)

	badResult := newResult(urlBad, nil)
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
