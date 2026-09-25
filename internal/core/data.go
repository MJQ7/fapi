package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"

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

// Data is the user's data, saved in mocks.json.
type Data struct {
	// When Enabled is false, endpoints are ignored and every request goes to
	// the pass-through, or gets a 404.
	Enabled bool `json:"enabled"`

	Mocks []Mock `json:"mocks"`

	// Upstreams maps a fapi port to the real API port its unmatched requests
	// are forwarded to. JSON object keys are always strings, so the file
	// holds {"3001": 3000}; encoding/json converts the keys to and from int.
	Upstreams map[int]int `json:"upstreams"`

	// UpstreamHosts holds the real API host of the fapi ports whose real API
	// isn't on passThrough.host, such as {"3001": "https://api.example.com"}.
	// It's kept apart from Upstreams so mocks.json files and API clients
	// that only know about ports keep working.
	UpstreamHosts map[int]string `json:"upstreamHosts,omitempty"`

	// DisabledUpstreams lists the fapi ports whose proxy is kept but turned
	// off: their unmatched requests get a 404 instead of being forwarded.
	DisabledUpstreams map[int]bool `json:"disabledUpstreams,omitempty"`

	// Payloads are saved responses that UIs offer when adding an endpoint,
	// so the same JSON doesn't have to be typed each time.
	Payloads []Payload `json:"payloads"`
}

// loadData reads mocks.json. A missing file means no endpoints yet.
func loadData(path string) (Data, error) {
	data := Data{
		Enabled:           true,
		Mocks:             []Mock{},
		Upstreams:         map[int]int{},
		UpstreamHosts:     map[int]string{},
		DisabledUpstreams: map[int]bool{},
		Payloads:          []Payload{},
	}

	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
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
	if data.Upstreams == nil {
		data.Upstreams = map[int]int{}
	}
	if data.UpstreamHosts == nil {
		data.UpstreamHosts = map[int]string{}
	}
	if data.DisabledUpstreams == nil {
		data.DisabledUpstreams = map[int]bool{}
	}
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
	contents, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return files.WriteAtomic(path, contents)
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
