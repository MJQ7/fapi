package core

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// connectionHeaders only describe the connection between two machines, so
// they're dropped instead of being copied from one connection to the other.
var connectionHeaders = []string{"Host", "Connection", "Content-Length", "Transfer-Encoding", "Keep-Alive"}

// Accept-Encoding isn't forwarded either, so the real API's response isn't
// compressed in a way fapi can't read (such as Brotli) and can be logged
// (see capture.go).
// Go asks for gzip instead and decompresses it, so the frontend gets the
// response uncompressed.
const acceptEncoding = "Accept-Encoding"

// newUpstreamClient returns the HTTP client used to call the real API.
func newUpstreamClient() *http.Client {
	return &http.Client{
		// Pass redirects back to the client as they are, instead of
		// following them, so the frontend sees what the real API sent.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// proxy forwards request to the real API at target and copies its response
// back. It returns the status sent to the client and, when the real API
// couldn't be reached, why.
func (c *Core) proxy(port int, target upstream, writer http.ResponseWriter, request *http.Request, body []byte) (int, string) {
	address := c.upstreamURL(target)
	url := address + request.URL.EscapedPath()
	if request.URL.RawQuery != "" {
		url += "?" + request.URL.RawQuery
	}

	// Using the incoming request's context cancels the forwarded request if
	// the client disconnects.
	upstreamRequest, err := http.NewRequestWithContext(request.Context(), request.Method, url, bodyReader(request.Method, body))
	if err != nil {
		return c.upstreamFailed(port, address, writer, request, err)
	}
	upstreamRequest.Header = copyHeaders(request.Header)
	upstreamRequest.Header.Del(acceptEncoding)

	response, err := c.upstreams.Do(upstreamRequest)
	if err != nil {
		return c.upstreamFailed(port, address, writer, request, err)
	}
	defer closeBody(response.Body)

	headers := writer.Header()
	for name, values := range copyHeaders(response.Header) {
		headers[name] = values
	}

	// Keep the real API's own CORS headers; add fapi's only if it sent none.
	origin := request.Header.Get("Origin")
	if c.settings.Features.CORS && origin != "" && response.Header.Get("Access-Control-Allow-Origin") == "" {
		addCORSHeaders(headers, origin)
	}

	writer.WriteHeader(response.StatusCode)
	_, err = io.Copy(writer, response.Body)
	if err != nil {
		log.Printf("[%d] %s %s -> copying the response from %s: %v", port, request.Method, request.URL.Path, address, err)
	}

	log.Printf("[%d] %s %s -> proxied to %s (%d)", port, request.Method, request.URL.Path, address, response.StatusCode)
	return response.StatusCode, ""
}

// upstreamFailed answers 502 Bad Gateway when the real API can't be reached.
func (c *Core) upstreamFailed(port int, address string, writer http.ResponseWriter, request *http.Request, err error) (int, string) {
	log.Printf("[%d] %s %s -> could not reach the real API at %s: %v", port, request.Method, request.URL.Path, address, err)

	origin := request.Header.Get("Origin")
	if c.settings.Features.CORS && origin != "" {
		addCORSHeaders(writer.Header(), origin)
	}
	message := fmt.Sprintf("fapi could not reach the real API at %s: %v", address, err)
	writeJSON(writer, http.StatusBadGateway, map[string]string{"message": message})
	return http.StatusBadGateway, message
}

// copyHeaders copies headers, leaving out the connection-level ones.
func copyHeaders(source http.Header) http.Header {
	copied := http.Header{}
	for name, values := range source {
		if isConnectionHeader(name) {
			continue
		}
		copied[name] = append([]string{}, values...)
	}
	return copied
}

func isConnectionHeader(name string) bool {
	for _, connectionHeader := range connectionHeaders {
		if strings.EqualFold(name, connectionHeader) {
			return true
		}
	}
	return false
}

// bodyReader returns the body to forward, or nil for GET and HEAD, which
// aren't forwarded with a body.
func bodyReader(method string, body []byte) io.Reader {
	if method == http.MethodGet || method == http.MethodHead {
		return nil
	}
	return bytes.NewReader(body)
}

// closeBody closes a response body when fapi is done reading it. There's
// nothing useful to do if closing fails, so the error is ignored.
func closeBody(body io.Closer) {
	_ = body.Close() // ignored: see above
}
