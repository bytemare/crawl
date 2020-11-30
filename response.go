package crawl

import (
	"net/http"

	"golang.org/x/net/html"
)

// Response wraps an http.Response with the result
type Response struct {
	// associated http.Response
	Response *http.Response
	// links found in a web page in the authorised scope
	Links []string
	// error, if any
	Error error
	// URL from the associated request
	URL string

	// request points to the associated Request
	request *Request
}

// newResponse returns a new Response struct based on a response body
func newResponse(req *Request) *Response {
	resp := &Response{
		Response: nil,
		Links:    nil,
		Error:    nil,
		URL:      "",
		request:  req,
	}
	if req != nil {
		resp.URL = req.url.Path
	}
	return resp
}

// extractLinks returns a slice of all links from an http.Get response body like reader object.
// Links won't contain queries or fragments
// It does not close the reader.
func (resp *Response) extractLinks(origin string) error {
	var errs ErrorList
	tokens := html.NewTokenizer(resp.Response.Body)

	// This map is an intermediary container for found links, avoiding duplicates
	links := make(map[string]bool)

	for typ := tokens.Next(); typ != html.ErrorToken; typ = tokens.Next() {
		token := tokens.Token()
		if typ == html.StartTagToken && token.Data == "a" {
			// If it's an anchor, try get the link
			link, err := extractLink(origin, token)
			if err != nil {
				errs = append(errs, err)
			}

			if link != "" {
				links[link] = true
				continue
			}
		}
	}
	resp.Links = mapToSlice(links)
	return errs
}
