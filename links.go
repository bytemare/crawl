package crawl

import (
	"net/url"

	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

// extractLink tries to return the link inside the token
func extractLink(origin string, token html.Token) (string, error) {
	//var a *html.Attribute
	// get href value
	for _, a := range token.Attr {
		if a.Key == "href" {
			link, err := sanitise(origin, a.Val)
			if err != nil {
				err = errors.Wrap(err, "Error in sanitising href")
			}
			return link, err
		}
	}

	return "", nil
}

// sanitise fixes some things in supposed link :
// - rebuilds the absolute url if the given link is relative to origin
// - escapes invalid links
// - strips queries and fragments
func sanitise(origin, link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", errors.Wrap(err, "could not parse link url")
	}

	if u.Path == "" || u.Path == "/" {
		return "", nil
	}

	base, err := url.Parse(origin)
	if err != nil {
		return "", errors.Wrap(err, "could parse origin url")
	}
	u = base.ResolveReference(u)

	stripQuery(u)

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
