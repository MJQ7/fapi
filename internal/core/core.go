// Package core is fapi itself: the endpoints, the mock servers that answer
// requests on their ports, pass-through to the real API, CORS, the request
// log and saving the user's data.
//
// Everything outside this package (the admin API, and through it the web UI
// and CLI) changes fapi only by calling the exported methods of Core. All
// validation happens here, so every UI gets the same rules and messages.
package core

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"sync"

	"fapi/internal/config"
)

// Core holds the application's state. main creates one and passes it to the
// admin API; there are no global variables.
type Core struct {
	settings  config.Config
	dataFile  string // path of mocks.json
	requests  *RequestLog
	upstreams *http.Client // used to forward requests to the real API

	// mutex protects every field below it, since each request is handled
	// concurrently.
	mutex   sync.Mutex
	data    Data
	servers map[int]*http.Server // running mock servers, keyed by port
}

// New loads the saved endpoints and request log from dataDir. It doesn't
// start listening yet; call Start for that.
func New(settings config.Config, dataDir string) (*Core, error) {
	dataFile := filepath.Join(dataDir, "mocks.json")
	data, err := loadData(dataFile)
	if err != nil {
		return nil, err
	}

	requestsFile := ""
	if settings.Features.RequestLogPersistence {
		requestsFile = filepath.Join(dataDir, "requests.json")
	}
	requests, err := NewRequestLog(settings.RequestLog.MaxEntries, requestsFile)
	if err != nil {
		return nil, err
	}

	core := &Core{
		settings:  settings,
		dataFile:  dataFile,
		requests:  requests,
		upstreams: newUpstreamClient(),
		data:      data,
		servers:   map[int]*http.Server{},
	}
	return core, nil
}

// Start listens on every port that an endpoint or pass-through uses.
// A port that can't be opened (usually because another program uses it) is
// logged and skipped, so one busy port doesn't stop fapi from starting.
func (c *Core) Start() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, port := range c.usedPorts() {
		err := c.startServer(port)
		if err != nil {
			log.Printf("Could not start the mock server on port %d: %v", port, err)
		}
	}
}

// Stop closes every mock server.
func (c *Core) Stop() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for port := range c.servers {
		c.stopServer(port)
	}
}

// Settings returns the effective settings fapi started with.
func (c *Core) Settings() config.Config {
	return c.settings
}

// Requests returns the request log.
func (c *Core) Requests() *RequestLog {
	return c.requests
}

// Snapshot returns a copy of the endpoints, pass-throughs, saved payloads and
// on/off switch.
// It's a copy so the caller can read it without holding the mutex.
func (c *Core) Snapshot() Data {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	snapshot := Data{
		Enabled:           c.data.Enabled,
		Mocks:             append([]Mock{}, c.data.Mocks...),
		Upstreams:         map[int]int{},
		UpstreamHosts:     map[int]string{},
		DisabledUpstreams: map[int]bool{},
		Payloads:          append([]Payload{}, c.data.Payloads...),
	}
	for port, upstreamPort := range c.data.Upstreams {
		snapshot.Upstreams[port] = upstreamPort
	}
	for port, host := range c.data.UpstreamHosts {
		snapshot.UpstreamHosts[port] = host
	}
	for port, disabled := range c.data.DisabledUpstreams {
		snapshot.DisabledUpstreams[port] = disabled
	}
	return snapshot
}

// ListeningPorts returns the ports mock servers are listening on, in order.
func (c *Core) ListeningPorts() []int {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var ports []int
	for port := range c.servers {
		ports = append(ports, port)
	}
	sort.Ints(ports)
	return ports
}

// usedPorts lists the ports that need a mock server, in order.
// The caller must hold the mutex.
func (c *Core) usedPorts() []int {
	var ports []int
	for _, mock := range c.data.Mocks {
		ports = append(ports, mock.Port)
	}
	if c.settings.Features.PassThrough {
		for port := range c.data.Upstreams {
			ports = append(ports, port)
		}
	}
	return uniqueSorted(ports)
}

// portInUse reports whether an endpoint or pass-through uses port.
// The caller must hold the mutex.
func (c *Core) portInUse(port int) bool {
	for _, mock := range c.data.Mocks {
		if mock.Port == port {
			return true
		}
	}
	_, hasUpstream := c.data.Upstreams[port]
	return c.settings.Features.PassThrough && hasUpstream
}

// save writes the data to mocks.json. The caller must hold the mutex.
func (c *Core) save() error {
	err := saveData(c.dataFile, c.data)
	if err != nil {
		return fmt.Errorf("saving %s: %w", c.dataFile, err)
	}
	return nil
}

func uniqueSorted(numbers []int) []int {
	seen := map[int]bool{}
	var unique []int
	for _, number := range numbers {
		if !seen[number] {
			seen[number] = true
			unique = append(unique, number)
		}
	}
	sort.Ints(unique)
	return unique
}
