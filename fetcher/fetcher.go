// Package fetch handles fetching data from
// crawler target URLS.
package fetcher

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	fe "fetch/errors"
	"fetch/orchestration"
	"fetch/robots"
	"fetch/sitemap"
)

// ignoreTypes matches a set of file types we don't care about.
var ignoreTypes = regexp.MustCompile(`(jpg|png|gif|\.js|\.aspx)`)

// siteMatch matches all URLs.
var siteMatch = regexp.MustCompile(`(http|https)://[a-zA-Z0-9./?=_%:-]*`)

// Fetch returns a list of string URLs contained in a given url.
func Fetch(url string, o *orchestration.Orchestrator) []string {
	// get a token
	o.Tokens <- struct{}{}
	// release a token when we're done
	defer func() { <-o.Tokens }()

	b, err := GetURL(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: couldn't read url: %v\n", err)
		return nil
	}

	time.Sleep(10 * time.Second)
	return ExtractLinks(b, o)
}

// GetURL returns a byte slice representation of target url
// or an error.
func GetURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil || !strings.HasSuffix(resp.Status, "OK") {
		fmt.Fprintf(os.Stderr, "GetURL: couldn't get: %s\nStatus: %s\nError: %v\n", url, resp.Status, err)
		resp.Body.Close()
		return nil, fmt.Errorf("GetURL: couldn't get: %s\nStatus: %s\nError: %v\n", url, resp.Status, err)
	}
	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	return b, err
}

// ExtractLinks returns a list of string URLs found
// within a given byte slice representation of a page.
func ExtractLinks(b []byte, o *orchestration.Orchestrator) []string {
	matches := siteMatch.FindAllString(string(b), -1)
	var filtered []string
	for _, match := range matches {
		// keep the crawling contained
		if !ignoreTypes.MatchString(match) && strings.HasPrefix(match, o.Baseurl) {
			filtered = append(filtered, match)
		}
	}
	return filtered
}

// GetAndParseRobots returns a list of discovered UserAgents
// if there is a robots.txt file provided by the target u.
func GetAndParseRobots(u string) ([]*robots.UserAgent, error) {
	var agent *robots.UserAgent
	var userAgents []*robots.UserAgent
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
				agent = robots.NewUserAgent(value)

			}
		case "allow:":
			agent.AddAllowed(value)
		case "disallow:":
			agent.AddDisallowed(value)
		case "sitemap:":
			count, siteMap := sitemap.ReadSiteMap(value)
			fmt.Fprintf(os.Stderr, "Found %d links from the site map. Crawl skipped.\n", count)
			for _, site := range siteMap.Urls {
				fmt.Println(site.Location)
			}
			os.Exit(fe.SuccesfulSiteMap)
		}
	}
	return userAgents, nil
}
