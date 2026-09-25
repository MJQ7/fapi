package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"fapi/internal/config"
	"fapi/internal/core"
)

// Client calls fapi's admin API over HTTP. The CLI changes fapi only through
// it, like the web UI does. It reuses the core and config types only to
// describe the JSON, never to call core directly.
type Client struct {
	baseURL string // such as http://127.0.0.1:3100
	http    *http.Client
	stream  *http.Client // no timeout, since the request stream stays open
}

// Status is the body of GET /api/status.
type Status struct {
	PID       int    `json:"pid"`
	Version   string `json:"version"`
	AdminPort int    `json:"adminPort"`
	MockPorts []int  `json:"mockPorts"`
}

// MocksData is the body of GET /api/mocks.
type MocksData struct {
	Enabled           bool           `json:"enabled"`
	Mocks             []core.Mock    `json:"mocks"`
	Upstreams         map[int]int    `json:"upstreams"`
	UpstreamHosts     map[int]string `json:"upstreamHosts"`     // only for proxies not using passThrough.host
	DisabledUpstreams map[int]bool   `json:"disabledUpstreams"` // the ports whose proxy is turned off
}

// NewMock is the body of POST /api/mocks.
type NewMock struct {
	Port   int             `json:"port"`
	Method string          `json:"method"`
	Path   string          `json:"path"`
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body,omitempty"`
}

// NewClient returns a client for the admin API at baseURL.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
		stream:  &http.Client{},
	}
}

// Status asks whether fapi is running. An error usually means it isn't.
func (c *Client) Status() (Status, error) {
	var status Status
	err := c.send(http.MethodGet, "/api/status", nil, &status)
	return status, err
}

// Config returns the settings fapi is running with.
func (c *Client) Config() (config.Config, error) {
	var settings config.Config
	err := c.send(http.MethodGet, "/api/config", nil, &settings)
	return settings, err
}

// Mocks returns the endpoints, proxies and on/off switch.
func (c *Client) Mocks() (MocksData, error) {
	var data MocksData
	err := c.send(http.MethodGet, "/api/mocks", nil, &data)
	return data, err
}

// AddMock adds an endpoint.
func (c *Client) AddMock(mock NewMock) error {
	return c.send(http.MethodPost, "/api/mocks", mock, nil)
}

// RemoveMock removes the endpoint with id.
func (c *Client) RemoveMock(id string) error {
	return c.send(http.MethodDelete, "/api/mocks/"+url.PathEscape(id), nil, nil)
}

// SetMockEnabled turns the endpoint with id on or off.
func (c *Client) SetMockEnabled(id string, enabled bool) error {
	return c.send(http.MethodPut, "/api/mocks/"+url.PathEscape(id)+"/enabled", map[string]bool{"enabled": enabled}, nil)
}

// SetEnabled turns all endpoints on or off.
func (c *Client) SetEnabled(enabled bool) error {
	return c.send(http.MethodPut, "/api/enabled", map[string]bool{"enabled": enabled}, nil)
}

// SetUpstream sets the real API host and port for port. Both are sent as the
// user typed them: fapi checks them, an empty host means passThrough.host,
// and an empty upstreamPort removes the proxy.
func (c *Client) SetUpstream(port int, upstreamPort string, host string) error {
	path := "/api/upstreams/" + strconv.Itoa(port)
	return c.send(http.MethodPut, path, map[string]string{"upstreamPort": upstreamPort, "host": host}, nil)
}

// Requests returns the request log, newest first.
func (c *Client) Requests() ([]core.LoggedRequest, error) {
	var requests []core.LoggedRequest
	err := c.send(http.MethodGet, "/api/requests", nil, &requests)
	return requests, err
}

// ClearRequests empties the request log.
func (c *Client) ClearRequests() error {
	return c.send(http.MethodDelete, "/api/requests", nil, nil)
}

// Shutdown asks fapi to stop.
func (c *Client) Shutdown() error {
	return c.send(http.MethodPost, "/api/shutdown", nil, nil)
}

// OpenRequestStream opens GET /api/requests/stream. The caller reads the
// server-sent events from it (see stream.go) and closes it when done.
func (c *Client) OpenRequestStream() (io.ReadCloser, error) {
	response, err := c.stream.Get(c.baseURL + "/api/requests/stream")
	if err != nil {
		return nil, fmt.Errorf("opening the request stream: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		defer closeQuietly(response.Body)
		return nil, readError(response)
	}
	return response.Body, nil
}

// send calls the API with body encoded as JSON (nil for none), and decodes
// the response into result (nil to ignore it).
func (c *Client) send(method string, path string, body any, result any) error {
	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding the request: %w", err)
		}
		requestBody = bytes.NewReader(encoded)
	}

	request, err := http.NewRequest(method, c.baseURL+path, requestBody)
	if err != nil {
		return fmt.Errorf("creating the request: %w", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("could not reach fapi at %s: %w", c.baseURL, err)
	}
	defer closeQuietly(response.Body)

	if response.StatusCode >= 400 {
		return readError(response)
	}
	if result == nil {
		return nil
	}
	err = json.NewDecoder(response.Body).Decode(result)
	if err != nil {
		return fmt.Errorf("reading fapi's answer to %s %s: %w", method, path, err)
	}
	return nil
}

// readError turns an error response, {"message": "..."}, into an error whose
// text is fapi's message, worded for the user.
func readError(response *http.Response) error {
	var body struct {
		Message string `json:"message"`
	}
	err := json.NewDecoder(response.Body).Decode(&body)
	if err != nil || body.Message == "" {
		return fmt.Errorf("fapi answered %s", response.Status)
	}
	return errors.New(body.Message)
}

// closeQuietly closes a response body. Nothing useful can be done if that
// fails, so the error is ignored.
func closeQuietly(body io.Closer) {
	_ = body.Close() // ignored: see above
}
