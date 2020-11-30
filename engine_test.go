package crawl

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testData struct {
	timeout       time.Duration
	synchron      *synchron
	urlBad        string
	urlValid      string
	urlTimeout    string
	expectedLinks []string
}

// getTestData returns default test data
func getTestData() *testData {
	timeout := 3 * time.Second
	return &testData{
		timeout:       timeout,
		synchron:      newSynchron(timeout, 1),
		urlBad:        "https://example.com/%",
		urlValid:      "https://example.com",
		urlTimeout:    "http://example.com:8000/submit",
		expectedLinks: []string{"https://www.iana.org/domains/example"},
	}
}

// getTestConfig returns a default configuration with logging turned off
func getTestConfig() *config {
	return configGetEmergencyConf()
}

// TestInitialiseEngineFail tests a failing condition for the initialiseEngine() function
func TestInitialiseEngineFail(t *testing.T) {
	test := getTestData()

	// Get a crawler
	crawler := NewDefaultCrawler()

	err := crawler.initialiseEngine(test.urlBad)
	if err == nil {
		t.Errorf("initialiseEngine() should fail with invalid domain. URL : '%s'.", test.urlBad)
	}
}

// TestScraperFail test a failing condition for the scraper method
func TestScraperFail(t *testing.T) {
	test := getTestData()

	crawler, _ := NewCrawler(
		Scope(test.urlValid),
	)

	_ = crawler.initialiseEngine(test.urlValid)

	// Launch and wait scraper
	crawler.engine.workerSync.Add(1)
	go crawler.engine.scraper(test.urlBad)
	crawler.engine.workerSync.Wait()
	result := <-crawler.engine.results
	if result.Error == nil {
		t.Errorf("scraper() should flag an error in the returning result. URL : '%s'.", test.urlBad)
	}
}

// TestFilterLinks
func TestFilterLinks(t *testing.T) {

}

// TestHandleResult tests the right behaviour of handleResult() in case of an error in a result
func TestHandleResult(t *testing.T) {
	test := getTestData()

	crawler, _ := NewCrawler(
		Scope(test.urlValid),
	)

	_ = crawler.initialiseEngine(test.urlValid)
	badResult := newResponse(nil)
	badResult.URL = test.urlBad
	badResult.Error = errors.New("this a test error")

	crawler.engine.handleResult(badResult)
	_, visited := crawler.engine.visited[badResult.URL]
	if visited {
		t.Errorf("handleResult() should not mark a failing URL as visited.")
	}
}

// TestHandleResultError tests handleResultError
func TestHandleResultError(t *testing.T) {
	test := getTestData()

	crawler, _ := NewCrawler(
		Scope(test.urlValid),
	)

	_ = crawler.initialiseEngine(test.urlValid)

	badResult := newResponse(nil)
	badResult.URL = test.urlBad
	badResult.Error = errors.New("this a test error")

	// Test case we re-enqueue the result
	crawler.engine.pending[badResult.URL] = crawler.engine.maxRetry - 1
	crawler.engine.handleResultError(badResult)
	if crawler.engine.failed[badResult.URL] {
		t.Error("URL retries have not hit the maximum, should be marked as failed.")
	}

	// Test case we decide to mark a URL as failed
	crawler.engine.pending[badResult.URL] = crawler.engine.maxRetry
	crawler.engine.handleResultError(badResult)
	_, failed := crawler.engine.failed[badResult.URL]
	_, pending := crawler.engine.pending[badResult.URL]
	if pending || !failed {
		t.Error("HandleResultError doesn't correctly switch the URL from pending to failed.")
	}
}

// TestCancellableScrapLinksFail tests failing conditions for the download function
func TestCancellableScrapLinksFail(t *testing.T) {
	test := getTestData()

	// Should fail on request building
	_, err := scrapLinks(test.urlBad, test.timeout, nil)
	if err == nil {
		t.Errorf("scrapLinks() should fail on invalid link. URL : '%s'", test.urlBad)
	}

	// Should fail on request execution due to timeout
	_, err = scrapLinks(test.urlTimeout, test.timeout, nil)
	if err == nil {
		t.Errorf("scrapLinks() should fail on timeout. Timeout : '%s'", test.timeout)
	}
}

// TestCancellableScrapLinksSuccess verifies the function behaves appropriately on success cases
func TestCancellableScrapLinksSuccess(t *testing.T) {
	test := getTestData()

	// Should return immediately because stop is requested
	stop := make(chan struct{})
	close(stop)
	res, err := scrapLinks(test.urlValid, test.timeout, stop)
	if err != nil || res != nil {
		t.Error("scrapLinks() should return nil only when stop is requested.")
	}

	// Should return nothing when stop is requested
	stop = make(chan struct{})
	cancelWait := 50 * time.Millisecond
	go func() {
		time.Sleep(cancelWait)
		close(stop)
	}()

	res, err = scrapLinks(test.urlTimeout, test.timeout, stop)
	if err != nil || res != nil {
		t.Errorf("scrapLinks() should return nil only when stop is requested : %s", err)
	}

	// Should return expected result
	res, err = scrapLinks(test.urlValid, test.timeout, nil)
	if err != nil {
		t.Errorf("scrapLinks() should not return an error and return expected result : %s", err)
	} else {
		assert.ElementsMatch(t, test.expectedLinks, res)
	}
}
