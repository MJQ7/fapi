package core

import "strings"

// route decides what happens to a request on port: the endpoint that
// answers it, or failing that the real API to forward it to (port 0 for
// none). Taking a copy under the mutex lets the request carry on without it.
func (c *Core) route(port int, method string, path string) (*Mock, upstream) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.data.Enabled {
		// The endpoint added first wins, because Mocks is in the order added.
		for _, mock := range c.data.Mocks {
			if !mock.Disabled && mock.Port == port && mock.Method == method && pathMatches(mock.Path, path) {
				found := mock
				return &found, upstream{}
			}
		}
	}

	if !c.settings.Features.PassThrough || c.data.DisabledUpstreams[port] {
		return nil, upstream{}
	}
	return nil, upstream{port: c.data.Upstreams[port], host: c.data.UpstreamHosts[port]}
}

// pathMatches reports whether a request path matches an endpoint's path.
// Paths are compared segment by segment, and a segment starting with ":"
// matches any value, so "/api/users/:id" matches "/api/users/42".
//
// The number of segments must be the same, so a trailing slash makes a
// different path: "/api/users/" does not match "/api/users".
func pathMatches(pattern string, path string) bool {
	patternSegments := strings.Split(pattern, "/")
	pathSegments := strings.Split(path, "/")

	if len(patternSegments) != len(pathSegments) {
		return false
	}

	for index, patternSegment := range patternSegments {
		if strings.HasPrefix(patternSegment, ":") {
			continue
		}
		if patternSegment != pathSegments[index] {
			return false
		}
	}
	return true
}
