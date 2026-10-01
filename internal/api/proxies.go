package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// newProxyRequest is the body of POST /api/proxies. The ports are kept as
// raw JSON because form inputs send numbers as strings; see
// parseUpstreamPort.
type newProxyRequest struct {
	Port         json.RawMessage `json:"port"`
	UpstreamPort json.RawMessage `json:"upstreamPort"`

	// Host is the real API host, starting with https:// if it uses HTTPS.
	// Missing or "" means passThrough.host from fapi's settings.
	Host string `json:"host"`
}

// postProxy answers POST /api/proxies: add a proxy and turn it on, turning
// off any other on its port (see core.AddProxy).
func (a *API) postProxy(writer http.ResponseWriter, request *http.Request) {
	var body newProxyRequest
	if !readJSON(writer, request, &body) {
		return
	}

	port, portOK := parseUpstreamPort(body.Port)
	upstreamPort, upstreamOK := parseUpstreamPort(body.UpstreamPort)
	switch {
	case !portOK || port == 0:
		writeError(writer, http.StatusBadRequest, "The fapi port must be a number between 1 and 65535")
		return
	case !upstreamOK || upstreamPort == 0:
		writeError(writer, http.StatusBadRequest, "Real API port must be a number between 1 and 65535")
		return
	}

	proxy, err := a.core.AddProxy(port, upstreamPort, body.Host)
	if err != nil {
		writeCoreError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, proxy)
}

// putProxyEnabled answers PUT /api/proxies/{id}/enabled: turn a proxy on or
// off. Turning one on turns the others on its port off.
func (a *API) putProxyEnabled(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	enabled, ok := readEnabled(writer, request)
	if !ok {
		return
	}

	found, err := a.core.SetProxyEnabled(id, enabled)
	if err != nil {
		writeCoreError(writer, err)
		return
	}
	if !found {
		writeError(writer, http.StatusNotFound, fmt.Sprintf("No proxy with ID %s", id))
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

// deleteProxy answers DELETE /api/proxies/{id}: remove a proxy.
func (a *API) deleteProxy(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	found, err := a.core.RemoveProxy(id)
	if err != nil {
		writeCoreError(writer, err)
		return
	}
	if !found {
		writeError(writer, http.StatusNotFound, fmt.Sprintf("No proxy with ID %s", id))
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
