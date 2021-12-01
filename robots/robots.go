// Package robots includes types and functions that
// deal with robots.txt.
package robots

// UserAgent tracks restrictions as defined
// in robots.txt.
type UserAgent struct {
	Agent      string
	Allowed    []string
	Disallowed []string
}

// AddAllowed adds a site suffix to the
// list of allowed suffixes.
func (ua *UserAgent) AddAllowed(a string) {
	ua.Allowed = append(ua.Allowed, a)
}

// AddDisallowed adds a site suffix to the
// list of disallowed suffixes.
func (ua *UserAgent) AddDisallowed(d string) {
	ua.Disallowed = append(ua.Disallowed, d)
}

// NewUserAgent returns an initialized UserAgent
// given a user-agent string from robots.txt.
func NewUserAgent(agent string) *UserAgent {
	return &UserAgent{Agent: agent}
}
