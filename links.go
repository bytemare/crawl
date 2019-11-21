package crawl

import (
	"io"
	"net/url"

	"github.com/pkg/errors"
	"golang.org/x/net/html"

	"github.com/sirupsen/logrus"
)

// extractLinks returns a slice of all links from an http.Get response body like reader object.
// Links won't contain queries or fragments
// It does not close the reader.
func extractLinks(origin string, body io.Reader) []string {
	tokens := html.NewTokenizer(body)

	// This map is an intermediary container for found links, avoiding duplicates
	links := make(map[string]bool)

	for typ := tokens.Next(); typ != html.ErrorToken; typ = tokens.Next() {
		token := tokens.Token()
		if typ == html.StartTagToken && token.Data == "a" {
			// If it's an anchor, try get the link
			if link := extractLink(origin, token); link != "" {
				links[link] = true
				continue
			}
		}
	}
	return mapToSlice(links)
}

// extractLink tries to return the link inside the token
func extractLink(origin string, token html.Token) string {
	// get href value
	for _, a := range token.Attr {
		if a.Key == "href" {
			link, err := sanitise(origin, a.Val)
			if err != nil {
				log.WithFields(logrus.Fields{
					"url":   origin,
					"token": token.String(),
				}).Tracef("Error in parsing token : %s", err)
			}
			return link
		}
	}

	return ""
}

// sanitise fixes some things in supposed link :
// - rebuilds the absolute url if the given link is relative to origin
// - escapes invalid links
// - strips queries and fragments
func sanitise(origin string, link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", errors.Wrap(err, "Error in parsing url")
	}

	if u.Path == "" || u.Path == "/" {
		return "", nil
	}

	base, err := url.Parse(origin)
	if err != nil {
		return "", errors.Wrap(err, "Error in parsing url")
	}
	u = base.ResolveReference(u)

	stripQuery(u)

	log.WithField("url", origin).Tracef("Rewrote '%s' to '%s'", link, u.String())

	return u.String(), nil
}

// stripQuery strips the query and fragments from an URL
func stripQuery(link *url.URL) {
	link.RawQuery = ""
	link.Fragment = ""
}

// mapToSlice returns a slice of strings containing the map's keys
func mapToSlice(links map[string]bool) []string {
	// Extract the keys from map into a slice
	keys := make([]string, len(links))
	i := 0
	for k := range links {
		keys[i] = k
		i++
	}
	return keys
}
