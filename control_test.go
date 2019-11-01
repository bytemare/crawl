package crawl

import (
	"os"
	"reflect"
	"testing"
	"time"
)

type controlTest struct {
	url     string
	timeout time.Duration
	errMsg  string
}

// TestFetchLinksFail tests cases where FetchLinks is supposed to fail and/or return an error
func TestFetchLinksFail(t *testing.T) {
	failing := []controlTest{
		{"", 0 * time.Second, "FetchLinks returned without error, but url is empty."},
		{"bytema.re", 0 * time.Second, "FetchLinks returned without error, but url is invalid."},
		{"https://bytema.re", -10 * time.Second, "FetchLinks returned without error, but timeout is invalid."},
	}

	for _, test := range failing {
		output, err := FetchLinks(test.url, test.timeout)
		if err == nil || output != nil {
			t.Errorf("%s URL : %s, timeout %d.", test.errMsg, test.url, test.timeout)
		}
	}

	// Set up failing condition for config initialisation
	urlBad := "https://example.com/%"
	timeout := 3 * time.Second
	test := getConfigTest()
	_ = os.Rename(test.validConfigFile, test.backupConfigFile)
	env := getEnv()
	os.Clearenv()

	// Place an invalid phony config file
	_ = os.Link(test.invalidConfigFile, test.validConfigFile)

	output, err := FetchLinks(urlBad, timeout)
	if err == nil || output != nil {
		t.Error("FetchLinks() should fail when config fails.")
	}

	// Restore config file and env vars
	restoreEnv(env)
	_ = os.Rename(test.backupConfigFile, test.validConfigFile)
}

// TestFetchLinksSuccess tests cases where FetchLinks is supposed to succeed
func TestFetchLinksSuccess(t *testing.T) {
	var succeed = []controlTest{
		{"https://bytema.re", 10 * time.Second, ""},
		{"https://bytema.re", 250 * time.Millisecond, ""},
		{"https://bytema.re", 0 * time.Second, ""},
	}
	errMsg := "FetchLinks returned with error, but url and timeout are valid. URL : %s, timeout : %0.3fs., error : %s"

	for _, test := range succeed {
		output, err := FetchLinks(test.url, test.timeout)
		if err != nil || output == nil {
			t.Errorf(errMsg, test.url, test.timeout.Seconds(), err)
		}
	}
}

//
func TestFetchLinks(t *testing.T) {
	url := "https://bytema.re"
	timeout := time.Duration(0)
	expected := []string{"https://bytema.re/author/bytemare/", "https://bytema.re/crypto/", "https://bytema.re/tutos/",
		"https://bytema.re/x/", "https://bytema.re/compiling/]"}

	output, err := FetchLinks(url, timeout)
	if len(output) == 0 || err != nil {
		t.Errorf("FetchLinks should return results for '%s'\n", url)
	} else if reflect.DeepEqual(output, expected) {
		t.Errorf("FetchLinks returned different result from what is expected."+
			"\n\t\tResult : '%s'\n\t\tExpected : '%s'\n", output, expected)
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
	_ = os.Rename(test.validConfigFile, test.backupConfigFile)
	env := getEnv()
	os.Clearenv()

	// Place an invalid phony config file
	_ = os.Link(test.invalidConfigFile, test.validConfigFile)

	_, err = ScrapLinks(urlBad, timeout)
	if err == nil {
		t.Error("ScrapLinks() should fail when config fails.")
	}

	// Restore config file and env vars
	restoreEnv(env)
	_ = os.Rename(test.backupConfigFile, test.validConfigFile)
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
