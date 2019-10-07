# crawl [![Build Status](https://travis-ci.com/bytemare/crawl.svg?branch=master)](https://travis-ci.com/bytemare/crawl) [![Go Report Card](https://goreportcard.com/badge/github.com/bytemare/gonetmon)](https://goreportcard.com/report/github.com/bytemare/gonetmon) [![codebeat badge](https://codebeat.co/badges/db89a587-9d35-49ef-96b1-d62b9cd1775b)](https://codebeat.co/projects/github-com-bytemare-crawl-dev) [![GolangCI](https://golangci.com/badges/github.com/bytemare/crawl.svg)](https://golangci.com/r/github.com/bytemare/crawl) [![GoDoc](https://godoc.org/github.com/bytemare/crawl?status.svg)](https://godoc.org/github.com/bytemare/crawl)
Simple web crawler with a single domain scope. Use it as a package or directly as an app.

The crawler scraps a page for links, follows them and scrapes them in the same fashion. 

You can launch the app with or without a timeout (in seconds), like this :

> go run app/crawl.go (-timeout=10) https://monzo.com

However the program was launched, you can interrupt it with ctrl+c.

## Features

- single domain scope
- parallel scrawling
- optional timeout
- scraps queries and fragments from url
- avoid loops on already visited links
- usable as a package by calling FetchLinks(), StreamLinks() and ScrapLinks() functions