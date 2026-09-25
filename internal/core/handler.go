package core

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// timeFormat is how times are written in the request log: RFC 3339 with
// milliseconds, as JavaScript writes them, such as 2026-09-21T14:03:12.345Z.
const timeFormat = "2006-01-02T15:04:05.000Z07:00"

// handleMockRequest answers one request on a fapi port: a CORS preflight,
// the matching endpoint, the real API, or a 404, in that order.
func (c *Core) handleMockRequest(port int, writer http.ResponseWriter, request *http.Request) {
	// Go reads the body of every method, including GET and HEAD, so bodies
	// sent with those are logged too.
	body, err := io.ReadAll(request.Body)
	if err != nil {
		log.Printf("[%d] %s %s -> could not read the request body: %v", port, request.Method, request.URL.Path, err)
		writeJSON(writer, http.StatusBadRequest, map[string]string{"message": "Could not read the request body"})
		return
	}

	entry := c.newLogEntry(port, request, body)
	origin := request.Header.Get("Origin")

	if c.settings.Features.CORS && isPreflight(request) {
		writePreflight(writer, request)
		log.Printf("[%d] %s %s -> CORS preflight", port, request.Method, request.URL.Path)
		c.record(entry, OutcomePreflight, http.StatusNoContent)
		return
	}

	mock, target := c.route(port, request.Method, request.URL.Path)

	if mock != nil {
		if c.settings.Features.CORS && origin != "" {
			addCORSHeaders(writer.Header(), origin)
		}
		writeMock(writer, *mock)
		log.Printf("[%d] %s %s -> mocked %d", port, request.Method, request.URL.Path, mock.Status)
		entry.MockID = mock.ID
		c.record(entry, OutcomeMocked, mock.Status)
		return
	}

	if target.port != 0 {
		status := c.proxy(port, target, writer, request, body)
		entry.Upstream = target.port
		c.record(entry, OutcomeProxied, status)
		return
	}

	if c.settings.Features.CORS && origin != "" {
		addCORSHeaders(writer.Header(), origin)
	}
	message := fmt.Sprintf("No mock for %s %s", request.Method, request.URL.Path)
	writeJSON(writer, http.StatusNotFound, map[string]string{"message": message})
	log.Printf("[%d] %s %s -> no mock found", port, request.Method, request.URL.Path)
	c.record(entry, OutcomeUnmatched, http.StatusNotFound)
}

// writeMock sends an endpoint's status and JSON body.
func writeMock(writer http.ResponseWriter, mock Mock) {
	if len(mock.Body) == 0 || !statusAllowsBody(mock.Status) {
		// Note: for 1xx statuses, Go sends the status as an informational
		// response followed by 200 OK, because HTTP requires a final status.
		writer.WriteHeader(mock.Status)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(mock.Status)
	_, err := writer.Write(mock.Body)
	if err != nil {
		log.Printf("Sending the response for %s %s: %v", mock.Method, mock.Path, err)
	}
}

// statusAllowsBody reports whether HTTP allows a response body with status.
func statusAllowsBody(status int) bool {
	return status >= 200 && status != http.StatusNoContent && status != http.StatusNotModified
}

// writeJSON sends value as a JSON response with status.
func writeJSON(writer http.ResponseWriter, status int, value any) {
	contents, err := json.Marshal(value)
	if err != nil {
		log.Printf("Encoding a response: %v", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_, err = writer.Write(contents)
	if err != nil {
		log.Printf("Sending a response: %v", err)
	}
}

// newLogEntry fills in what's known about a request before it's answered.
func (c *Core) newLogEntry(port int, request *http.Request, body []byte) LoggedRequest {
	entry := LoggedRequest{
		ID:     newID(),
		Time:   time.Now().UTC().Format(timeFormat),
		Port:   port,
		Method: request.Method,
		Path:   request.URL.Path,

		// Only the content type is kept: other headers can hold credentials.
		ContentType: request.Header.Get("Content-Type"),
		FromPort:    senderPort(request.RemoteAddr),
	}
	if request.URL.RawQuery != "" {
		entry.Search = "?" + request.URL.RawQuery
	}
	entry.Body, entry.BodyTruncated = bodyText(body, c.settings.RequestLog.MaxBodyBytes)
	return entry
}

// record completes entry and adds it to the request log, if it's turned on.
func (c *Core) record(entry LoggedRequest, outcome string, status int) {
	if !c.settings.Features.RequestLog {
		return
	}
	entry.Outcome = outcome
	entry.Status = status
	c.requests.Add(entry)
}

// bodyText turns a request body into text for the log, cut at maxBytes.
// It reports whether the body was cut.
func bodyText(body []byte, maxBytes int) (string, bool) {
	truncated := len(body) > maxBytes
	if truncated {
		body = body[:maxBytes]
	}

	// Bodies can be binary, and the cut can split a character in two, so
	// replace anything that isn't valid text.
	text := strings.ToValidUTF8(string(body), string(utf8.RuneError))
	if truncated {
		text += "…"
	}
	return text, truncated
}

// senderPort returns the port from an address such as "127.0.0.1:54321".
func senderPort(remoteAddress string) int {
	_, portText, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		return 0 // not expected from net/http; 0 means unknown
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return 0
	}
	return port
}
