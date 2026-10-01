package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"

	"fapi/internal/files"
)

// Mock is one endpoint: requests to Method and Path on Port get Status and
// Body back.
type Mock struct {
	ID     string `json:"id"`
	Port   int    `json:"port"`
	Method string `json:"method"`
	Path   string `json:"path"`
	Status int    `json:"status"`

	// json.RawMessage keeps the body exactly as JSON text, whatever shape
	// it has. "omitempty" leaves it out of the file when there's no body.
	Body json.RawMessage `json:"body,omitempty"`

	// Disabled endpoints are kept but don't answer requests, which go to the
	// proxy instead. It's "disabled" rather than "enabled" so that files
	// written before the switch existed, which don't have it, load with
	// every endpoint on.
	Disabled bool `json:"disabled,omitempty"`
}

// Proxy forwards the requests to a fapi port that no endpoint answers to
// the real API. A port can have several proxies, but only one of them is on
// at a time (see proxies.go).
type Proxy struct {
	ID           string `json:"id"`
	Port         int    `json:"port"`         // the fapi port
	UpstreamPort int    `json:"upstreamPort"` // the real API port

	// Host is the real API host, starting with https:// if it uses HTTPS,
	// or "" for passThrough.host in fapi's settings.
	Host string `json:"host,omitempty"`

	// Disabled proxies are kept but don't forward anything. Like Mock's,
	// it's "disabled" so a missing value means on.
	Disabled bool `json:"disabled,omitempty"`
}

// Data is the user's data, saved in mocks.json.
type Data struct {
	// When Enabled is false, endpoints are ignored and every request goes to
	// the pass-through, or gets a 404.
	Enabled bool `json:"enabled"`

	Mocks []Mock `json:"mocks"`

	// Proxies are in the order they were added.
	Proxies []Proxy `json:"proxies"`

	// The way fapi saved proxies before a port could have more than one:
	// Upstreams maps a fapi port to its real API port ({"3001": 3000}),
	// UpstreamHosts holds the real API hosts that aren't passThrough.host,
	// and DisabledUpstreams the ports whose proxy is off. A file without
	// "proxies" is read from these. They're still written, with each port's
	// shown proxy (see portProxy), so an older fapi can read the file.
	Upstreams         map[int]int    `json:"upstreams"`
	UpstreamHosts     map[int]string `json:"upstreamHosts,omitempty"`
	DisabledUpstreams map[int]bool   `json:"disabledUpstreams,omitempty"`

	// Payloads are saved responses that UIs offer when adding an endpoint,
	// so the same JSON doesn't have to be typed each time.
	Payloads []Payload `json:"payloads"`
}

// loadData reads mocks.json. A missing file means no endpoints yet.
func loadData(path string) (Data, error) {
	data := Data{
		Enabled:  true,
		Mocks:    []Mock{},
		Payloads: []Payload{},
	}

	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		data.Proxies = []Proxy{}
		return data, nil
	}
	if err != nil {
		return Data{}, fmt.Errorf("reading %s: %w", path, err)
	}

	// Fields missing from the file keep the values set above.
	err = json.Unmarshal(contents, &data)
	if err != nil {
		return Data{}, fmt.Errorf("reading %s: %w", path, err)
	}

	// "null" in the file would leave these as nil; use empty values instead
	// so the API always returns [] and {}.
	if data.Mocks == nil {
		data.Mocks = []Mock{}
	}
	if data.Proxies == nil {
		// No "proxies": a file from an older fapi.
		data.Proxies = proxiesFromPorts(data.Upstreams, data.UpstreamHosts, data.DisabledUpstreams)
	}
	data.Upstreams, data.UpstreamHosts, data.DisabledUpstreams = nil, nil, nil
	if data.Payloads == nil {
		data.Payloads = []Payload{}
	}
	for index := range data.Mocks {
		data.Mocks[index].Body = normalizeBody(data.Mocks[index].Body)
	}
	for index := range data.Payloads {
		data.Payloads[index].Body = normalizeBody(data.Payloads[index].Body)
	}
	return data, nil
}

// saveData writes mocks.json, indented so it's readable.
func saveData(path string, data Data) error {
	data.Upstreams, data.UpstreamHosts, data.DisabledUpstreams = ProxiesByPort(data.Proxies)
	contents, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return files.WriteAtomic(path, contents)
}

// proxiesFromPorts converts proxies saved the old way, one per port, to
// Proxies, in port order.
func proxiesFromPorts(upstreams map[int]int, hosts map[int]string, disabled map[int]bool) []Proxy {
	proxies := []Proxy{}
	for port, upstreamPort := range upstreams {
		proxies = append(proxies, Proxy{
			ID:           newID(),
			Port:         port,
			UpstreamPort: upstreamPort,
			Host:         hosts[port],
			Disabled:     disabled[port],
		})
	}
	slices.SortFunc(proxies, func(a, b Proxy) int { return a.Port - b.Port })
	return proxies
}

// ProxiesByPort converts proxies to the old way of saving them, which the
// admin API also still sends: for each port, the proxy that's on, or else
// the last one added (see portProxy).
func ProxiesByPort(proxies []Proxy) (map[int]int, map[int]string, map[int]bool) {
	upstreams := map[int]int{}
	hosts := map[int]string{}
	disabled := map[int]bool{}
	for _, port := range proxyPorts(proxies) {
		proxy := proxies[portProxy(proxies, port)]
		upstreams[port] = proxy.UpstreamPort
		if proxy.Host != "" {
			hosts[port] = proxy.Host
		}
		if proxy.Disabled {
			disabled[port] = true
		}
	}
	return upstreams, hosts, disabled
}

// normalizeBody removes the spaces and line breaks from a JSON body, so it's
// sent the same way however it was typed, and treats null as no body.
func normalizeBody(body json.RawMessage) json.RawMessage {
	var compact bytes.Buffer
	err := json.Compact(&compact, body)
	if err != nil {
		return body // not valid JSON; validation reports that separately
	}
	if compact.String() == "null" {
		return nil
	}
	return compact.Bytes()
}
