package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bytemare/crawl"
)

func main() {
	logFilePath := "./log-crawler.log"

	// Define and parse command line arguments
	timeout := flag.Int("timeout", 0, "crawling time, in seconds. 0 or none is infinite.")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Printf("Expecting at least an url as entry point. e.g. './%s https://bytema.re'\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	domain := flag.Args()[0]

	// Launch crawler
	fmt.Printf("Starting web crawler. You can interrupt the program any time with ctrl+c. Logging to %s.\n", logFilePath)
	resultChan, err := crawl.StreamLinks(domain, time.Duration(*timeout)*time.Second)
	if err != nil {
		fmt.Printf("Error : %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Mapping only shows non-visited links.")
	for res := range resultChan {
		fmt.Printf("%s -> %s\n", res.URL, *res.Links)
	}

	os.Exit(0)
}
