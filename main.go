package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var ignoreImages = regexp.MustCompile(`(jpg|png|gif|\.js|\.aspx)`)
var siteMatch = regexp.MustCompile(`(http|https)://[a-zA-Z0-9./?=_%:-]*`)

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
		fmt.Fprintln(os.Stderr, "Must declare a target")
		os.Exit(1)
	}

	orc := NewOrchestrator(target, conc)

	go func() { orc.worker <- []string{target} }()

	timer := time.After(time.Duration(duration) * time.Second)

loop:
	for {
		select {
		case urls := <-orc.worker:
			for _, url := range urls {
				if !orc.seen[url] {
					orc.seen[url] = true
					fmt.Println(url)
					go func(url string) {
						orc.worker <- Fetch(url, orc)
					}(url)
				}
			}
		case <-timer:
			break loop
		}
	}
	close(orc.worker)
}

type Orchestrator struct {
	seen    map[string]bool
	tokens  chan struct{}
	worker  chan []string
	baseurl string
}

func NewOrchestrator(b string, p int) *Orchestrator {
	return &Orchestrator{
		seen:    make(map[string]bool),
		tokens:  make(chan struct{}, p),
		worker:  make(chan []string),
		baseurl: b,
	}
}

type UserAgent struct {
	Allowed    []string
	Disallowed []string
}

func Fetch(url string, o *Orchestrator) []string {
	// get a token
	o.tokens <- struct{}{}
	// release a token when we're done
	defer func() { <-o.tokens }()

	b, err := GetURL(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: couldn't read body: %v\n", err)
		return nil
	}

	return ExtractLinks(b, o)
}

func GetURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: couldn't get: %v\n", err)
		resp.Body.Close()
		return nil, nil
	}

	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	return b, err
}

func ExtractLinks(b []byte, o *Orchestrator) []string {
	matches := siteMatch.FindAllString(string(b), -1)
	var filtered []string
	for _, match := range matches {
		// keep the crawling contained
		if !ignoreImages.MatchString(match) && strings.HasPrefix(match, o.baseurl) {
			filtered = append(filtered, match)
		}
	}
	return filtered
}
