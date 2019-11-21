package crawl

import (
	"errors"
	"io"
	"testing"
	"time"
)

type testData struct {
	timeout    time.Duration
	syn        *synchron
	urlBad     string
	urlValid   string
	urlTimeout string
}

// getTestData returns default test data
func getTestData() *testData {
	timeout := 3 * time.Second
	return &testData{
		timeout:    timeout,
		syn:        newSynchron(timeout, 1),
		urlBad:     "https://example.com/%",
		urlValid:   "https://example.com",
		urlTimeout: "http://example.com:8000/submit",
	}
}

// getTestConfig returns a default configuration with logging turned off
func getTestConfig() *config {
	return configGetEmergencyConf()
}

// TestNewCrawlerFail tests a failing condition for the newCrawler() function
func TestNewCrawlerFail(t *testing.T) {
	test := getTestData()
	_, err := newCrawler(test.urlBad, test.syn.results, test.timeout, 3)
	if err == nil {
		t.Errorf("newCrawler() should fail with invalid domain. URL : '%s'.", test.urlBad)
	}
}

// TestInitialiseCrawler tests a failing condition for the initialiseCrawler() function
func TestInitialiseCrawlerFail(t *testing.T) {
	test := getTestData()
	conf := getTestConfig()

	c := initialiseCrawler(test.urlBad, test.syn, conf)
	if c != nil {
		t.Errorf("initialiseCrawler() should fail with invalid domain. URL : '%s'.", test.urlBad)
	}
}

// TestCrawlFail should immediately return when initialiseCrawler fails
func TestCrawlFail(t *testing.T) {
	test := getTestData()
	conf := getTestConfig()

	// use a timeout to measure if crawler is running
	done := make(chan struct{})

	go func() {
		go crawl(test.urlBad, test.syn, conf)
		test.syn.group.Wait()
		done <- struct{}{}
	}()

	// Wait for the crawler to return
	<-done
	if test.syn.exitContext != exitErrorInit {
		t.Error("crawler() should not run when calling initialiseCrawler() failed.")
	}
}

// TestScraperFail test a failing condition for the scraper method
func TestScraperFail(t *testing.T) {
	test := getTestData()
	conf := getTestConfig()

	c := initialiseCrawler(test.urlValid, test.syn, conf)

	c.workerSync.Add(1)
	go c.scraper(test.urlBad)
	c.workerSync.Wait()
	result := <-c.results
	if result.Error == nil {
		t.Errorf("scraper() should flag an error in the returning result. URL : '%s'.", test.urlBad)
	}
}

// TestDownloadFail tests a failing condition for the download function
func TestDownloadFail(t *testing.T) {
	var closeBody = func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}
	test := getTestData()

	// Should fail on request building
	body, err := download(test.urlBad, test.timeout)
	if err == nil {
		t.Errorf("download() should fail on invalid link. URL : '%s'", test.urlBad)
	}
	closeBody(body)

	// Should fail on request execution due to timeout
	body, err = download(test.urlTimeout, test.timeout)
	if err == nil {
		t.Errorf("download() should timeout on non-ending request. URL : '%s'", test.urlTimeout)
	}
	closeBody(body)
}

// TestHandleResult tests the right behaviour of handleResult() in case of an error in a result
func TestHandleResult(t *testing.T) {
	test := getTestData()
	conf := getTestConfig()

	c := initialiseCrawler(test.urlValid, test.syn, conf)
	badResult := newLinkMap(test.urlBad, nil)
	badResult.Error = errors.New("this a test error")
	c.handleResult(badResult)
	_, visited := c.visited[badResult.URL]
	if visited {
		t.Errorf("handleResult() should not mark a failing URL as visited.")
	}
}

// TestHandleResultError tests handleResultError
func TestHandleResultError(t *testing.T) {
	test := getTestData()
	conf := getTestConfig()

	c := initialiseCrawler(test.urlValid, test.syn, conf)

	badResult := newLinkMap(test.urlBad, nil)
	badResult.Error = errors.New("this a test error")

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
