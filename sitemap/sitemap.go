package sitemap

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

// SiteMap represents a full sitemap provided by robots.txt
type SiteMap struct {
	XMLName xml.Name `xml:"urlset"`
	Urls    []Url    `xml:"url"`
}

// Url represents fields of interest of each URL listed
// within a sitemap.
type Url struct {
	XMLName    xml.Name `xml:"url"`
	Location   string   `xml:"loc"`
	LastMod    string   `xml:"lastmod"`
	ChangeFreq string   `xml:"changefreq"`
}

// ReadSiteMap gets and parses a sitemap.xml at the
// base of u if one exists.
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
