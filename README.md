# crawl
[![Build Status](https://travis-ci.com/bytemare/crawl.svg?branch=master)](https://travis-ci.com/bytemare/crawl) [![Coverage](https://sonarcloud.io/api/project_badges/measure?project=bytemare_crawl&metric=coverage)](https://sonarcloud.io/dashboard?id=bytemare_crawl) [![Go Report Card](https://goreportcard.com/badge/github.com/bytemare/crawl)](https://goreportcard.com/report/github.com/bytemare/crawl) [![codebeat badge](https://codebeat.co/badges/db89a587-9d35-49ef-96b1-d62b9cd1775b)](https://codebeat.co/projects/github-com-bytemare-crawl-master) [![GolangCI](https://golangci.com/badges/github.com/bytemare/crawl.svg)](https://golangci.com/r/github.com/bytemare/crawl) [![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/3285/badge)](https://bestpractices.coreinfrastructure.org/projects/3285) [![GoDoc](https://godoc.org/github.com/bytemare/crawl?status.svg)](https://godoc.org/github.com/bytemare/crawl)

The crawler scraps a page for links, follows them and scrapes them in the same fashion.

You can launch the app with or without a timeout (in seconds), like this :

```go
go run app/crawl.go (-timeout=10) https://bytema.re
```

However the program was launched, you can interrupt it with ctrl+c.

## Features

* single domain scope
* parallel scrawling
* optional timeout
* scraps queries and fragments from url
* avoid loops on already visited links
* usable as a package by calling FetchLinks(), StreamLinks() and ScrapLinks() functions
* logs to file in JSON for log aggregation

## Get the Crawler : Installation and update

It's as easy as it gets with Go :

```shell script
go get -u github.com/bytemare/crawl
```

## Usage and examples

The scraper and crawler functions are rather easy to use. The timeout parameter is optional, if you don't need to timeout,
just set it to 0.

### Calling the crawler from your code

You can call the crawler from your own code with StreamLinks or FetchLinks.

StreamLinks returns a channel you can listen on for continuous results as they arrive

```go
import "github.com/bytemare/crawl"

func myCrawler() {
	
	domain := "https://bytema.re"
	timeout := 10 * time.Second
	
	resultChan, err := crawl.StreamLinks(domain, timeout)
	if err != nil {
		fmt.Printf("Error : %s\n", err)
		os.Exit(1)
	}

	for res := range resultChan {
		fmt.Printf("%s -> %s\n", res.URL, *res.Links)
	}
}
```

FetchLinks blocks, collects, explores, then returns all encountered links

```go
import "github.com/bytemare/crawl"

func myCrawler() {

	domain := "https://bytema.re"
	timeout := 10 * time.Second

	links, err := crawl.FetchLinks(domain, timeout)
	if err != nil {
		fmt.Printf("Error : %s\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Starting from %s, encountered following links :\n%s\n", domain, links)
}
```

### Scraping a single page for links

If you simply want to scrap all links for a single web page, use the ScrapLinks function :

```go
import "github.com/bytemare/crawl"

func myScraper() {

	domain := "https://bytema.re"
	timeout := 10 * time.Second

	links, err := crawl.ScrapLinks(domain, timeout)
	if err != nil {
		fmt.Printf("Error : %s\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Found following links on %s :\n%s\n", domain, links)
}
```

## Supported go versions

We support the last three major Go versions, which are 1.11, 1.12 and 1.13 at the moment.

## Contributing

Please feel free to submit issues, fork the repository and send pull requests!
Take a look at the [contributing guidelines](https://github.com/bytemare/crawl/blob/master/contributing.md) !

## License

This project is licensed under the terms of the MIT license.