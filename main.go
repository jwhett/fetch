package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
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

const (
	NotEnoughArgs = iota
	RobotError
	ExplicitDisallow
	SuccesfulSiteMap
)

func init() {
	flag.StringVar(&target, "target", "", "target baseurl to crawl")
	flag.IntVar(&duration, "duration", 5, "how long to crawl")
	flag.IntVar(&conc, "conc", 10, "number of sites to crawl in parallel")
	flag.Parse()
}

func main() {
	if len(target) == 0 {
		fmt.Fprintln(os.Stderr, "fetch: Must declare a target")
		os.Exit(NotEnoughArgs)
	}

	baseURL := strings.TrimSuffix(target, "/")
	userAgents, err := GetAndParseRobots(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: Problem with robots.txt: %v\n", err)
		os.Exit(RobotError)
	}

	disallowed := make(map[string]bool)
	for _, agent := range userAgents {
		if agent.Agent == "*" {
			for _, suffix := range agent.Disallowed {
				if suffix == "/" {
					fmt.Fprintf(os.Stderr, "fetch: cannot crawl %s, explicitly disallowed\n", baseURL)
					os.Exit(ExplicitDisallow)
				}
				disallowed[baseURL+suffix] = true
			}
		}
	}

	orc := NewOrchestrator(baseURL, conc, disallowed)

	go func() { orc.worker <- []string{baseURL} }()

	timer := time.After(time.Duration(duration) * time.Second)
loop:
	for {
		select {
		case urls := <-orc.worker:
			for _, url := range urls {
				if !orc.seen[url] && !orc.disallowed[url] {
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
	seen       map[string]bool
	tokens     chan struct{}
	worker     chan []string
	baseurl    string
	disallowed map[string]bool
}

func NewOrchestrator(b string, p int, d map[string]bool) *Orchestrator {
	return &Orchestrator{
		seen:       make(map[string]bool),
		tokens:     make(chan struct{}, p),
		worker:     make(chan []string),
		baseurl:    b,
		disallowed: d,
	}
}

type UserAgent struct {
	Agent      string
	Allowed    []string
	Disallowed []string
}

func (ua *UserAgent) AddAllowed(a string) {
	ua.Allowed = append(ua.Allowed, a)
}

func (ua *UserAgent) AddDisallowed(d string) {
	ua.Disallowed = append(ua.Disallowed, d)
}

func NewUserAgent(agent string) *UserAgent {
	return &UserAgent{Agent: agent}
}

func Fetch(url string, o *Orchestrator) []string {
	// get a token
	o.tokens <- struct{}{}
	// release a token when we're done
	defer func() { <-o.tokens }()

	b, err := GetURL(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: couldn't read url: %v\n", err)
		return nil
	}

	time.Sleep(10 * time.Second)
	return ExtractLinks(b, o)
}

func GetURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil || !strings.HasSuffix(resp.Status, "OK") {
		fmt.Fprintf(os.Stderr, "fetch: couldn't get: %s\nStatus: %s\nError: %v\n", url, resp.Status, err)
		resp.Body.Close()
		return nil, fmt.Errorf("fetch: couldn't get: %s\nStatus: %s\nError: %v\n", url, resp.Status, err)
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

func GetAndParseRobots(u string) ([]*UserAgent, error) {
	var agent *UserAgent
	var userAgents []*UserAgent
	var directive, value string
	data, err := GetURL(u + "/robots.txt")
	if err != nil {
		return nil, err
	}
	dataReader := bytes.NewReader(data)
	for {
		if _, err := fmt.Fscanln(dataReader, &directive, &value); err != nil {
			if err == io.EOF {
				userAgents = append(userAgents, agent)
				break
			}
			continue
		}
		switch strings.ToLower(directive) {
		case "user-agent:":
			{
				if agent != nil && agent.Agent != value {
					userAgents = append(userAgents, agent)
				}
				agent = NewUserAgent(value)

			}
		case "allow:":
			agent.AddAllowed(value)
		case "disallow:":
			agent.AddDisallowed(value)
		case "sitemap:":
			count, siteMap := ReadSiteMap(value)
			fmt.Fprintf(os.Stderr, "Found %d links from the site map. Crawl skipped.\n", count)
			for _, site := range siteMap.Urls {
				fmt.Println(site.Location)
			}
			os.Exit(SuccesfulSiteMap)
		}
	}
	return userAgents, nil
}

type SiteMap struct {
	XMLName xml.Name `xml:"urlset"`
	Urls    []Url    `xml:"url"`
}

type Url struct {
	XMLName    xml.Name `xml:"url"`
	Location   string   `xml:"loc"`
	LastMod    string   `xml:"lastmod"`
	ChangeFreq string   `xml:"changefreq"`
}

func ReadSiteMap(u string) (int, SiteMap) {
	var client http.Client
	client.Timeout = 30 * time.Second
	xmlResponse, err := client.Get(u)
	if err != nil {
		fmt.Printf("Error getting site map! %v", err)
		xmlResponse.Body.Close()
		os.Exit(1)
	}

	xmlData, err := ioutil.ReadAll(xmlResponse.Body)
	xmlResponse.Body.Close()

	var siteMap SiteMap
	xml.Unmarshal(xmlData, &siteMap)

	return len(siteMap.Urls), siteMap
}
