package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var ignoreImages = regexp.MustCompile(`(jpg|png|gif|\.js|\.aspx)`)
var siteMatch = regexp.MustCompile(`(http|https)://[a-zA-Z0-9./?=_%:-]*`)

type SiteTracker struct {
    mut sync.Mutex
    sites map[string]string
}

func (st *SiteTracker) Add(url, body string) {
    st.mut.Lock()
    defer st.mut.Unlock()
    if _, ok := st.sites[url]; !ok {
        st.sites[url] = body
    }
}

func (st *SiteTracker) PrintSites() {
    var str string
    for url := range st.sites {
        str += fmt.Sprintf("%s ", url)
    }
    fmt.Println(str)
}

func Fetch(baseurl, url string, depth int, wg *sync.WaitGroup, st *SiteTracker) {
    defer wg.Done()

    if depth == 0 {
        return
    }
    depth--

	resp, err := http.Get(url)
	if err != nil {
    	fmt.Fprintf(os.Stderr, "couldn't get %s: %v\n", url, err)
		return 
	}

    b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
    	fmt.Fprintf(os.Stderr, "couldn't read body for %s: %v\n", url, err)
		return 
	}

	st.Add(url, string(b))

	matches := siteMatch.FindAllString(string(b), -1)
	for _, match := range matches {
		if !ignoreImages.MatchString(match) && strings.HasPrefix(match, baseurl){
    		wg.Add(1)
    		go Fetch(baseurl, match, depth, wg, st)
		}
	}
}

func main() {
    if len(os.Args) < 3 {
        fmt.Println("fetch: not enough args")
        os.Exit(1)
    }

	url := os.Args[1]
    depth, err := strconv.Atoi(os.Args[2])
    if err != nil {
        fmt.Fprintf(os.Stderr, "fetch: cannot convert depth (%v) to int: %v\n", os.Args[2], err)
        os.Exit(1)
    }

    var wg sync.WaitGroup
    st := SiteTracker{ sites: make(map[string]string) }

    wg.Add(1)
    go Fetch(url, url, depth, &wg, &st)

    fmt.Fprintln(os.Stderr, "Working...")
    wg.Wait()
    st.PrintSites()
    fmt.Printf("We have site content for %d sites.\n", len(st.sites))
}
