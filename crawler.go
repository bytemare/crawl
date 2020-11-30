package crawl

import (
	"fmt"
	"net/url"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
)

type CrawlerMode uint

const (
	StreamMode CrawlerMode = 1
	FetchMode  CrawlerMode = 2

	defRequestTimeout = 20 * time.Second
	defCrawlerTimeout = 0
	defMaxRetries     = 5
)

// Crawler holds the crawling instance configuration
type Crawler struct {
	// Crawling scope
	// Domain whitelist
	Domains []string

	// Blacklist of domains not to be visited
	DomainBlacklist []string

	// User Agent to send during requests
	UserAgent string

	// Timeout per request
	RequestTimeout time.Duration

	// Timeout for total crawling
	CrawlerTimeout time.Duration

	// Maximum retries before abandoning to pull a page
	MaxRetries int

	/*
		Private structures for the engine
	*/

	// Crawler engine
	engine *engine

	// Synchronisation
	syn *synchron

	// Callback functions
	requestCallback  *func(*Request)
	responseCallback *func(*Response)
	successCallback  *func(*Response)
	errorCallback    *func(*Response)
	finishCallback   *func()

	// Configuration
	config *config
}

// NewCrawler returns a default initialised Crawler, or parameterised with given options
func NewCrawler(options ...func(*Crawler)) (*Crawler, error) {
	// Check env and initialise logging
	conf, err := initialiseCrawlerConfiguration()
	if err != nil && conf == nil {
		return nil, errors.Wrap(err, exitErrorConf)
	}

	// Set up new engine
	c := NewDefaultCrawler()
	for _, function := range options {
		function(c)
	}

	c.config = conf
	return c, nil
}

// UserAgent returns pre-registered common User Agents to be send in the request headers
func UserAgent(ua string) string {
	userAgents := map[string]string{
		"Crawler": "Crawler",
		"Chrome": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko)" +
			" Chrome/78.0.3904.108 Safari/537.36",
		"Firefox": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:70.0) Gecko/20100101 Firefox/70.0",
	}

	return userAgents[ua]
}

// defaultCrawler returns a default initialised Crawler
func NewDefaultCrawler() *Crawler {
	c := &Crawler{
		Domains:          nil,
		DomainBlacklist:  nil,
		UserAgent:        UserAgent("Crawler"),
		RequestTimeout:   defRequestTimeout,
		CrawlerTimeout:   defCrawlerTimeout,
		MaxRetries:       defMaxRetries,
		engine:           nil,
		syn:              nil,
		requestCallback:  nil,
		responseCallback: nil,
		successCallback:  nil,
		errorCallback:    nil,
		finishCallback:   nil,
		config:           nil,
	}
	c.config = configGetEmergencyConf()
	return c
}

// Scope sets the scope - authorised domains - for the crawler
func Scope(domains ...string) func(*Crawler) {
	return func(crawler *Crawler) {
		crawler.Domains = append(crawler.Domains, domains...)
	}
}

// SetTimeout sets the timeout for the associated engine
func Timeout(t time.Duration) func(*Crawler) {
	return func(crawler *Crawler) {
		crawler.CrawlerTimeout = t
	}
}

// StopOnSigInt will set SIGINT interception when possible, and halts when signal is intercepted
func (c *Crawler) StopOnSigInt() error {
	// todo
	return nil
}

//
func (c *Crawler) OnRequest() {
	// todo
}

//
func (c *Crawler) OnResponse() {
	// todo
}

//
func (c *Crawler) OnSuccess() {
	// todo
}

//
func (c *Crawler) OnError() {

}

//
func (c *Crawler) OnFinish() {
	// todo
}

//
func (c *Crawler) SetUserAgent(ua string) {
	// todo
}

//
func (c *Crawler) Log(ua string) {
	// todo
}

// Run starts the engine, starting with the given address
func (c *Crawler) Run(mode CrawlerMode, start string) (*CrawlerResults, error) {
	switch mode {
	case StreamMode:
		return c.StreamLinks(start, c.RequestTimeout)
	case FetchMode:
		return c.FetchLinks(start, c.RequestTimeout)
	default:
		return nil, errors.New("unknown crawler mode")
	}
}

// validateInput returns whether input is valid and can be worked with
func validateInput(domain string, timeout time.Duration) error {
	// We can't crawl without a target domain
	if domain == "" {
		return errors.New("if you want to crawl something, please specify the target domain as argument")
	}

	// Check whether domain is of valid form
	if _, err := url.ParseRequestURI(domain); err != nil {
		return errors.Wrap(err, "Invalid url : you must specify a valid target domain/url to crawl")
	}

	// Invalid timeout values are handled later, but let's not the user mess with us
	if timeout < 0 {
		msg := fmt.Sprintf("Invalid timeout value '%d' (accepted values [0 ; +yourpatience [, in seconds)", timeout)
		return errors.New(msg)
	}

	return nil
}

// StreamLinks returns a channel on which it will report links as they come during the crawling.
// The caller should range over that channel to continuously retrieve messages. StreamLinks will close that channel
// when all encountered links have been visited and none is left, when the deadline on the timeout parameter is reached,
// or if a SIGINT or SIGTERM signals is received.
func (c *Crawler) StreamLinks(domain string, timeout time.Duration) (*CrawlerResults, error) {
	// check input
	if err := validateInput(domain, timeout); err != nil {
		return nil, errors.Wrap(err, exitErrorInput)
	}

	c.config.Logger.log.WithField("url", domain).Info("Starting web crawler.")
	c.syn = newSynchron(timeout, 3)
	res := newCrawlerResults(c.syn)

	go c.startCrawling(domain)

	return res, nil
}

// FetchLinks is a wrapper around StreamLinks and does the same, except it blocks and accumulates all links before
// returning them to the caller.
func (c *Crawler) FetchLinks(domain string, timeout time.Duration) (*CrawlerResults, error) {
	res, err := c.StreamLinks(domain, timeout)
	if err != nil {
		return nil, err
	}

	res.links = make([]string, 0, 100) // todo : trade-off here, look if we really need that
	for linkMap := range res.Stream() {
		res.links = append(res.links, linkMap.Links...)
	}

	return res, nil
}

// ScrapLinks returns the links found in the web page pointed to by url
func ScrapLinks(URL string, timeout time.Duration) ([]string, error) {
	// Check env and initialise logging
	conf, err := initialiseCrawlerConfiguration()
	if err != nil && conf == nil {
		return nil, errors.Wrap(err, exitErrorConf)
	}
	resp, err := scrapLinks(URL, timeout, nil)
	if err != nil {
		if merr, ok := err.(*multierror.Error); ok {
			conf.Logger.log.WithFields(logrus.Fields{
				"url": URL,
			}).Errorf("Encountered errors in scraping page : %s", merr.Errors)
		}
	}

	if resp == nil {
		return nil, err
	}

	return resp.Links, err
}
