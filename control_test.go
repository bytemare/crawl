package crawl_test

import (
	"testing"
	"time"

	"github.com/bytemare/crawl"
)

type Test struct {
	url     string
	timeout time.Duration
	errMsg  string
}

// TestFetchLinksFail tests cases where FetchLinks is supposed to fail and/or return an error
func TestFetchLinksFail(t *testing.T) {
	failing := []Test{
		{"", 0 * time.Second, "FetchLinks returned without error, but url is empty."},
		{"bytema.re", 0 * time.Second, "FetchLinks returned without error, but url is invalid."},
		{"https://bytema.re", -10 * time.Second, "FetchLinks returned without error, but timeout is invalid."},
	}

	for _, test := range failing {
		output, err := crawl.FetchLinks(test.url, test.timeout)
		if err == nil || output != nil {
			t.Errorf("%s URL : %s, timeout %d.", test.errMsg, test.url, test.timeout)
		}
	}
}

// TestFetchLinksSuccess tests cases where FetchLinks is supposed to succeed
func TestFetchLinksSuccess(t *testing.T) {
	var succeed = []Test{
		{"https://bytema.re", 10 * time.Second, ""},
		{"https://bytema.re", 250 * time.Millisecond, ""},
		{"https://bytema.re", 0 * time.Second, ""},
	}
	errMsg := "FetchLinks returned with error, but url and timeout are valid. URL : %s, timeout : %0.3fs., error : %s"

	for _, test := range succeed {
		output, err := crawl.FetchLinks(test.url, test.timeout)
		if err != nil || output == nil {
			t.Errorf(errMsg, test.url, test.timeout.Seconds(), err)
		}
	}
}

// TestScrapLinksFail tests a failing condition for ScrapLinks()
func TestScrapLinksFail(t *testing.T) {
	urlBad := "https://example.com/%"
	timeout := 3 * time.Second

	_, err := crawl.ScrapLinks(urlBad, timeout)
	if err == nil {
		t.Errorf("ScrapLinks() should fail on invalid URL. URL : '%s'.", urlBad)
	}
}

/*
// TestFetchLinksInterrupt simulates a crawling with signal interrupt
func TestFetchLinksInterrupt(t *testing.T) {

	signalTime := 1 * time.Second

	var sendSignal = func(wait time.Duration) {
		time.Sleep(wait)
		p, _ := os.FindProcess(os.Getpid())
		_ = p.Signal(os.Interrupt)
	}

	tests := []Test{
		{"https://github.com/bytemare", signalTime + 2*time.Second, ""},
	}

	for _, test := range tests {
		sendSignal(signalTime)
		output, err := crawl.FetchLinks(test.url, test.timeout)
		if err != nil || output == nil {
			t.Errorf("Error in testing with signal. URL : %s, timeout : %0.3fs.", test.url, test.timeout.Seconds())
		}
	}
}
*/
