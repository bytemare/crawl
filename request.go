package crawl

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
)

// Request wraps an http.Request
type Request struct {
	// Client used to carry the request
	client *http.Client
	// associated http.Request
	request *http.Request

	// URL is the URL the request is headed to
	url *url.URL
	// maxRetries
	maxRetries int
	// timeout
	timeout time.Duration
	// ctx allows additional context to requests, like cancelling
	ctx context.Context
	// cancel function to call when request is being interrupted
	cancel context.CancelFunc

	// Header for the HTTP request header
	header *http.Header
	// Response
	response *Response
}

// newRequest returns an initialised request containing a client and a new context
func newRequest(method string, URL string, maxRetries int, timeout time.Duration) (*Request, error) {
	// Build request and client
	req, err := http.NewRequest(method, URL, nil)
	if err != nil {
		return nil, err
	}
	var client = &http.Client{
		Timeout: timeout,
	}
	var ctx context.Context
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(context.Background())
	req = req.WithContext(ctx)

	request := &Request{
		client:     client,
		request:    req,
		url:        req.URL,
		maxRetries: maxRetries,
		timeout:    timeout,
		ctx:        ctx,
		cancel:     cancel,
		header:     nil,
		response:   nil,
	}
	request.response = newResponse(request)
	return request, nil
}

// closeBody wraps the closing function of a body
func closeBody(response *http.Response) {
	if response != nil && response.Body != nil {
		_ = response.Body.Close() // nolint:errcheck // one does not really care about that
	}
}

// todo comments
func (req *Request) doCancellable(errChan chan<- error) {
	// Send request
	var err error
	req.response.Response, err = req.client.Do(req.request) // nolint:bodyclose // body is closed through function call

	// Depending on how the request finished
	select {
	case <-req.ctx.Done(): // Cancelled
		closeBody(req.response.Response)
		return
	default: // Success or timeout
		break
	}

	// There was no forced cancellation
	if err != nil {
		errChan <- errors.Wrapf(err, "Error in downloading resource")
	} else {
		errChan <- nil
	}
}

// todo : explanations
// get requests that are cancellable in-flight
// if returns error, all others are nil
// either errChan xor respChan send a message,
// if no error, then body from respChan must be closed
func download(URL string, timeout time.Duration) (*Request, <-chan error) {
	// errChan will send an error if encountered
	errChan := make(chan error)

	// Build request and client
	req, err := newRequest("GET", URL, 3, timeout)
	if err != nil {
		errChan <- errors.Wrapf(err, "Could not make a GET Request for %s", URL)
		return nil, errChan
	}

	// Send request
	go req.doCancellable(errChan)

	return req, errChan
}

// scrapLinks returns the links found in the web page pointed to by url
// todo : add statement that if stop is given closed, it will return nil, nil
func scrapLinks(URL string, timeout time.Duration, stop <-chan struct{}) (*Response, error) {
	// Empty response
	_resp := newResponse(nil)

	// If stop was already ordered, quit immediately
	select {
	case <-stop:
		return _resp, nil
	default:
	}

	request, errChan := download(URL, timeout)

	select {
	// If stop is not a nil channel, we want to be it cancellable
	// We need to stop / cancel request
	case <-stop:
		request.cancel()
		return request.response, nil

	// Request encountered an error
	case err := <-errChan:
		// Download encountered an error
		if err != nil {
			return request.response, err
		}

		// No error was encountered during download
		// Retrieve links
		err = request.response.extractLinks(URL)
		closeBody(request.response.Response)
		return request.response, err
	}
}
