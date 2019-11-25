package crawl

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// todo comments
func cancellableRequest(ctx context.Context, client *http.Client, req *http.Request,
	respChan chan<- io.ReadCloser, errChan chan<- error) {
	// Send request
	resp, err := client.Do(req)

	select {
	case <-ctx.Done(): // Cancelled
		return
	default: // Success or timeout
		break
	}

	if err != nil {
		errChan <- errors.Wrapf(err, "Error in downloading resource")
	} else {
		respChan <- resp.Body
	}
}

// private scrapLinks with additional argument to indicate cancellation

// todo : explanations
// make get requests that are cancellable in-flight
// if returns error, all others are nil
// either errChan xor respChan send a message,
// if no error, then body from respChan must be closed
func cancellableDownload(url string, timeout time.Duration) (
	<-chan io.ReadCloser, <-chan error, context.CancelFunc, error) {
	// Build request and client
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "Could not make a GET Request for %s", url)
	}
	var client = &http.Client{
		Timeout: timeout,
	}
	var ctx context.Context
	var cancel context.CancelFunc

	respChan := make(chan io.ReadCloser)
	errChan := make(chan error)

	ctx, cancel = context.WithCancel(context.Background())
	req = req.WithContext(ctx)

	// Send request
	go cancellableRequest(ctx, client, req, respChan, errChan)

	return respChan, errChan, cancel, nil
}

// scrapLinks returns the links found in the web page pointed to by url
// todo : add statement that if stop is given closed, it will return nil, nil
func cancellableScrapLinks(url string, timeout time.Duration, stop <-chan struct{}) ([]string, error) {
	// If stop was already ordered, quit immediately
	select {
	case <-stop:
		return nil, nil
	default:
	}

	// If stop is not a nil channel, we want to be it cancellable
	respChan, errChan, cancel, err := cancellableDownload(url, timeout)
	if err != nil {
		return nil, err
	}

	select {
	// We need to stop / cancel request
	case <-stop:
		cancel()
		return nil, nil

	// We have a result
	case body := <-respChan:
		defer func() {
			if body != nil {
				_ = body.Close()
			}
		}()
		// Retrieve links
		return extractLinks(url, body), nil

	// Request encountered an error
	case err := <-errChan:
		return nil, err
	}
}
