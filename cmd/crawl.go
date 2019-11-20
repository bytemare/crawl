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
	// Define and parse command line arguments
	timeout := flag.Int("timeout", 0, "crawling time, in seconds. 0 or none is infinite.")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Printf("Expecting at least an url as entry point. e.g. './%s https://bytema.re'\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	domain := flag.Args()[0]

	// Launch crawler
	fmt.Println("Starting web crawler. You can interrupt the program any time with ctrl+c.")
	crawlerResult, err := crawl.StreamLinks(domain, time.Duration(*timeout)*time.Second)
	if err != nil {
		fmt.Printf("Error : %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Mapping only shows not yet visited links.")
	for res := range crawlerResult.Stream() {
		fmt.Printf("%s -> %s\n", res.URL, *res.Links)
	}

	os.Exit(0)
}
