/*
Package crawl is a simple link scraper and web crawler with single domain scope.
It can be limited with a timeout and interrupted with signals.

Three public functions give access to single page link scraping (ScrapLinks) and
single host web crawling (FetchLinks and StreamLinks).
FetchLinks and StreamLinks have the same behaviour and result, as FetchLinks is a wrapper for StreamLinks.
The only difference is that FetchLinks is blocking,
and returns once a stopping condition is reached (link tree exhaustion, timeout, signals),
where StreamLinks immediately returns a channel on which the calling function can listen on to get results as they come.

The return values can be used for a site map.

Some precautions have been taken to prevent infinite loops, like stripping queries and fragments off urls.

A sample program calling the package is given in the project repository.
*/
package crawl
