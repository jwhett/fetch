package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"fetch/orchestration"
	"fetch/fetcher"
	fe "fetch/errors"
)

var target string
var duration int
var conc int

func init() {
	flag.StringVar(&target, "target", "", "target baseurl to crawl")
	flag.IntVar(&duration, "duration", 5, "how long to crawl")
	flag.IntVar(&conc, "conc", 10, "number of sites to crawl in parallel")
	flag.Parse()
}

func main() {
	if len(target) == 0 {
		fmt.Fprintln(os.Stderr, "fetch: Must declare a target")
		os.Exit(fe.NotEnoughArgs)
	}

	baseURL := strings.TrimSuffix(target, "/")
	userAgents, err := fetcher.GetAndParseRobots(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: Problem with robots.txt: %v\n", err)
		os.Exit(fe.RobotError)
	}

	disallowed := make(map[string]bool)
	for _, agent := range userAgents {
		if agent.Agent == "*" {
			for _, suffix := range agent.Disallowed {
				if suffix == "/" {
					fmt.Fprintf(os.Stderr, "fetch: cannot crawl %s, explicitly disallowed\n", baseURL)
					os.Exit(fe.ExplicitDisallow)
				}
				disallowed[baseURL+suffix] = true
			}
		}
	}

	orc := orchestration.NewOrchestrator(baseURL, conc, disallowed)

	go func() { orc.Worker <- []string{baseURL} }()

	timer := time.After(time.Duration(duration) * time.Second)
loop:
	for {
		select {
		case urls := <-orc.Worker:
			for _, url := range urls {
				if !orc.Seen[url] && !orc.Disallowed[url] {
					orc.Seen[url] = true
					fmt.Println(url)
					go func(url string) {
						orc.Worker <- fetcher.Fetch(url, orc)
					}(url)
				}
			}
		case <-timer:
			break loop
		}
	}
	close(orc.Worker)
}


