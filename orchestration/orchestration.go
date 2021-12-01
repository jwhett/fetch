// Package orchestration contains types and
// methods for safe concurrency.
package orchestration

// Orchestrator tracks sites seen thus far,
// a buffered channel as a token rate limit
// bucket, a worker stream of URLs, the
// base URL of the site, and a map of sites
// that cannot be crawled.
type Orchestrator struct {
	Seen       map[string]bool
	Tokens     chan struct{}
	Worker     chan []string
	Baseurl    string
	Disallowed map[string]bool
}

// NewOrchestrator returns an initialized Orchstrator given a
// base URL, count of concurrent workers allowed (tokens),
// and a map of forbidden URLs.
func NewOrchestrator(b string, p int, d map[string]bool) *Orchestrator {
	return &Orchestrator{
		Seen:       make(map[string]bool),
		Tokens:     make(chan struct{}, p),
		Worker:     make(chan []string),
		Baseurl:    b,
		Disallowed: d,
	}
}
