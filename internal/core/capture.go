package core

import "net/http"

// responseCapture passes a response on to the client, keeping its content
// type and the start of its body for the request log.
type responseCapture struct {
	http.ResponseWriter
	maxBytes int
	body     []byte // up to one byte more than maxBytes, so bodyText can tell it was cut
}

func newResponseCapture(writer http.ResponseWriter, maxBytes int) *responseCapture {
	return &responseCapture{ResponseWriter: writer, maxBytes: maxBytes}
}

func (r *responseCapture) Write(data []byte) (int, error) {
	room := max(r.maxBytes+1-len(r.body), 0)
	r.body = append(r.body, data[:min(len(data), room)]...)
	return r.ResponseWriter.Write(data)
}

// Unwrap lets http.ResponseController reach the client's ResponseWriter.
func (r *responseCapture) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// logged returns the response for the request log, once it's been sent.
func (r *responseCapture) logged() *LoggedResponse {
	response := &LoggedResponse{ContentType: r.Header().Get("Content-Type")}
	response.Body, response.BodyTruncated = bodyText(r.body, r.maxBytes)
	return response
}
