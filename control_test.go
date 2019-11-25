package crawl

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
)

type controlTest struct {
	url         string
	timeout     time.Duration
	exitContext string
	errMsg      string
}

// TestFetchLinksFail tests cases where FetchLinks is supposed to fail and/or return an error
func TestFetchLinksFail(t *testing.T) {
	var err error
	var crawlerResults *CrawlerResults
	failing := []controlTest{
		{"", 0 * time.Second, "", "FetchLinks returned without error, but url is empty."},
		{"bytema.re", 0 * time.Second, "", "FetchLinks returned without error, but url is invalid."},
		{"https://bytema.re", -10 * time.Second, "", "FetchLinks returned without error, but timeout is invalid."},
	}

	for _, test := range failing {
		crawlerResults, err = FetchLinks(test.url, test.timeout)
		if err == nil || crawlerResults != nil {
			t.Errorf("%s URL : %s, timeout %d.", test.errMsg, test.url, test.timeout)
		}
	}

	// Set up failing condition for config initialisation
	urlBad := "https://example.com/%"
	timeout := 3 * time.Second
	test := getConfigTest()
	env := getEnv()
	os.Clearenv()

	// Place an invalid phony config file
	if !test.makeInvalidConfigFile(t) {
		goto restore
	}

	crawlerResults, err = FetchLinks(urlBad, timeout)
	if err == nil || crawlerResults != nil {
		t.Error("FetchLinks() should fail when config fails.")
	}

	// Restore config file and env vars
restore:
	restoreConfigFileAndEnv(t, test, env)
}

// TestFetchLinksSuccess tests cases where FetchLinks is supposed to succeed
func TestFetchLinksSuccess(t *testing.T) {
	var succeed = []controlTest{
		{"https://bytema.re", 10 * time.Second, exitLinks, ""},
		{"https://bytema.re", 250 * time.Millisecond, exitTimeout, ""},
		{"https://bytema.re", 0 * time.Second, exitLinks, ""},
	}
	errMsg := "FetchLinks returned with error, but url and timeout are valid. URL : %s, timeout : %0.3fs."

	for _, test := range succeed {
		crawlerResult, err := FetchLinks(test.url, test.timeout)
		if err != nil || crawlerResult == nil {
			t.Errorf("%s", errors.Wrapf(err, errMsg, test.url, test.timeout.Seconds()))
		} else {
			assert.Equal(t, test.exitContext, crawlerResult.ExitContext())
		}
	}
}

//
func TestFetchLinks(t *testing.T) {
	url := "https://bytema.re"
	timeout := time.Duration(0)
	expected := []string{"https://bytema.re/author/bytemare/", "https://bytema.re/crypto/", "https://bytema.re/tutos/",
		"https://bytema.re/x/", "https://bytema.re/compiling/"}

	crawlerResult, err := FetchLinks(url, timeout)
	if err != nil || crawlerResult == nil || len(crawlerResult.Links()) == 0 {
		t.Errorf("FetchLinks should return results for '%s' : %s", url, err)
	} else {
		assert.ElementsMatch(t, expected, crawlerResult.Links())
	}
}

// TestScrapLinksSuccess tests if the functions succeeds with expected results
func TestScrapLinksSuccess(t *testing.T) {
	url := "https://bytema.re"
	timeout := time.Duration(0)
	expected := []string{"https://bytema.re/author/bytemare/", "https://bytema.re/crypto/", "https://bytema.re/tutos/",
		"https://bytema.re/x/", "https://bytema.re/compiling/", "https://twitter.com/_bytemare"}

	links, err := ScrapLinks(url, timeout)
	if err != nil || len(links) == 0 {
		t.Errorf("FetchLinks should return results for '%s' : %s", url, err)
	} else {
		assert.ElementsMatch(t, expected, links)
	}
}

// TestScrapLinksFail tests failing conditions for ScrapLinks()
func TestScrapLinksFail(t *testing.T) {
	urlBad := "https://example.com/%"
	timeout := 3 * time.Second

	_, err := ScrapLinks(urlBad, timeout)
	if err == nil {
		t.Errorf("ScrapLinks() should fail on invalid URL. URL : '%s'.", urlBad)
	}

	// Set up failing condition for config initialisation
	test := getConfigTest()
	env := getEnv()
	os.Clearenv()

	// Place an invalid phony config file
	if !test.makeInvalidConfigFile(t) {
		goto restore
	}

	_, err = ScrapLinks(urlBad, timeout)
	if err == nil {
		t.Error("ScrapLinks() should fail when config fails.")
	}

	// Restore config file and env vars
restore:
	restoreConfigFileAndEnv(t, test, env)
}

// TestFetchLinksInterrupt simulates a crawling with signal interrupt
func TestFetchLinksInterrupt(t *testing.T) {
	// Don't run this test on windows, since signals are not supported
	if runtime.GOOS == "windows" {
		return
	}

	signalTime := 2 * time.Second
	done := make(chan struct{})

	var sendSignal = func(wait time.Duration) {
		time.Sleep(wait)
		pid := os.Getpid()
		p, err := os.FindProcess(pid)
		if err != nil {
			t.Logf("Couldn't find process : %s\n", err)
		}

		if err := p.Signal(os.Interrupt); err != nil {
			t.Logf("Couldn't send signal : %s\n", err)
		}
		done <- struct{}{}
	}

	test := getTestData()

	go sendSignal(signalTime)
	crawlerResult, err := FetchLinks(test.urlTimeout, test.timeout)
	if err != nil || crawlerResult == nil {
		t.Errorf("Error in testing with signal. URL : %s, timeout : %0.3fs. : %s",
			test.urlTimeout, test.timeout.Seconds(), err)
	} else if crawlerResult.ExitContext() != exitSignal {
		t.Errorf("Error in testing with signal. Signal was not caught. URL : %s, timeout : %0.3fs.",
			test.urlTimeout, test.timeout.Seconds())
	}
	<-done
}
