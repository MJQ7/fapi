package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"fapi/internal/core"
)

// mocksResponse is the body of GET /api/mocks.
type mocksResponse struct {
	Enabled   bool        `json:"enabled"`
	Mocks     []core.Mock `json:"mocks"`
	Upstreams map[int]int `json:"upstreams"`

	// The real API host of the proxies that don't use passThrough.host, and
	// the ports whose proxy is turned off.
	UpstreamHosts     map[int]string `json:"upstreamHosts"`
	DisabledUpstreams map[int]bool   `json:"disabledUpstreams"`
}

// getMocks answers GET /api/mocks: the endpoints, pass-throughs and switch.
func (a *API) getMocks(writer http.ResponseWriter, request *http.Request) {
	data := a.core.Snapshot()

	// Saved pass-throughs are kept but unused while the feature is off, so
	// they aren't shown.
	upstreams := data.Upstreams
	upstreamHosts := data.UpstreamHosts
	disabledUpstreams := data.DisabledUpstreams
	if !a.core.Settings().Features.PassThrough {
		upstreams = map[int]int{}
		upstreamHosts = map[int]string{}
		disabledUpstreams = map[int]bool{}
	}

	writeJSON(writer, http.StatusOK, mocksResponse{
		Enabled:           data.Enabled,
		Mocks:             data.Mocks,
		Upstreams:         upstreams,
		UpstreamHosts:     upstreamHosts,
		DisabledUpstreams: disabledUpstreams,
	})
}

// newMockRequest is the body of POST /api/mocks.
type newMockRequest struct {
	Port   int             `json:"port"`
	Method string          `json:"method"`
	Path   string          `json:"path"`
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

// postMock answers POST /api/mocks: add an endpoint.
func (a *API) postMock(writer http.ResponseWriter, request *http.Request) {
	var body newMockRequest
	if !readJSON(writer, request, &body) {
		return
	}

	mock, err := a.core.AddMock(core.NewMock{
		Port:   body.Port,
		Method: body.Method,
		Path:   body.Path,
		Status: body.Status,
		Body:   body.Body,
	})
	if err != nil {
		writeCoreError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, mock)
}

// deleteMock answers DELETE /api/mocks/{id}: remove an endpoint.
func (a *API) deleteMock(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	found, err := a.core.RemoveMock(id)
	if err != nil {
		writeCoreError(writer, err)
		return
	}
	if !found {
		writeError(writer, http.StatusNotFound, fmt.Sprintf("No endpoint with ID %s", id))
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

// enabledRequest is the body of PUT /api/enabled. Enabled is a pointer so a
// missing value (nil) can be told apart from false.
type enabledRequest struct {
	Enabled *bool `json:"enabled"`
}

// readEnabled reads an enabledRequest body. If it's missing or not valid, it
// answers the request with the error and returns false.
func readEnabled(writer http.ResponseWriter, request *http.Request) (bool, bool) {
	var body enabledRequest
	if !readJSON(writer, request, &body) {
		return false, false
	}
	if body.Enabled == nil {
		writeError(writer, http.StatusBadRequest, "enabled must be true or false")
		return false, false
	}
	return *body.Enabled, true
}

// putEnabled answers PUT /api/enabled: turn all endpoints on or off.
func (a *API) putEnabled(writer http.ResponseWriter, request *http.Request) {
	enabled, ok := readEnabled(writer, request)
	if !ok {
		return
	}

	err := a.core.SetEnabled(enabled)
	if err != nil {
		writeCoreError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

// upstreamRequest is the body of PUT /api/upstreams/{port}. The real API port
// is kept as raw JSON because a number, a numeric string, "" and null are
// all accepted; see parseUpstreamPort.
type upstreamRequest struct {
	UpstreamPort json.RawMessage `json:"upstreamPort"`

	// Host is the real API host, starting with https:// if it uses HTTPS.
	// Missing or "" means passThrough.host from fapi's settings.
	Host string `json:"host"`
}

// putMockEnabled answers PUT /api/mocks/{id}/enabled: turn one endpoint on
// or off.
func (a *API) putMockEnabled(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	enabled, ok := readEnabled(writer, request)
	if !ok {
		return
	}

	found, err := a.core.SetMockEnabled(id, enabled)
	if err != nil {
		writeCoreError(writer, err)
		return
	}
	if !found {
		writeError(writer, http.StatusNotFound, fmt.Sprintf("No endpoint with ID %s", id))
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

// putUpstreamEnabled answers PUT /api/upstreams/{port}/enabled: turn one
// proxy on or off.
func (a *API) putUpstreamEnabled(writer http.ResponseWriter, request *http.Request) {
	fapiPort, err := strconv.Atoi(request.PathValue("port"))
	if err != nil {
		writeError(writer, http.StatusBadRequest, "The fapi port must be a number")
		return
	}

	enabled, ok := readEnabled(writer, request)
	if !ok {
		return
	}

	found, err := a.core.SetUpstreamEnabled(fapiPort, enabled)
	if err != nil {
		writeCoreError(writer, err)
		return
	}
	if !found {
		writeError(writer, http.StatusNotFound, fmt.Sprintf("Port %d has no proxy", fapiPort))
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

// putUpstream answers PUT /api/upstreams/{port}: set or remove a pass-through.
func (a *API) putUpstream(writer http.ResponseWriter, request *http.Request) {
	fapiPort, err := strconv.Atoi(request.PathValue("port"))
	if err != nil {
		writeError(writer, http.StatusBadRequest, "The fapi port must be a number")
		return
	}

	var body upstreamRequest
	if !readJSON(writer, request, &body) {
		return
	}

	upstreamPort, ok := parseUpstreamPort(body.UpstreamPort)
	if !ok {
		writeError(writer, http.StatusBadRequest, "Real API port must be a number between 1 and 65535, or empty to remove it")
		return
	}

	err = a.core.SetUpstream(fapiPort, upstreamPort, body.Host)
	if err != nil {
		writeCoreError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

// parseUpstreamPort reads the real API port. A missing value, null or ""
// mean "remove the pass-through" and give 0. Form inputs send numbers as
// strings, so "3000" is accepted as well as 3000.
func parseUpstreamPort(raw json.RawMessage) (int, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, true
	}

	var number int
	err := json.Unmarshal(raw, &number)
	if err == nil {
		return number, true
	}

	var text string
	err = json.Unmarshal(raw, &text)
	if err != nil {
		return 0, false
	}
	if text == "" {
		return 0, true
	}
	number, err = strconv.Atoi(text)
	if err != nil {
		return 0, false
	}
	return number, true
}
