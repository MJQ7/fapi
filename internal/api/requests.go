package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"fapi/internal/core"
)

// getRequests answers GET /api/requests: the request log, newest first.
func (a *API) getRequests(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, a.core.Requests().All())
}

// deleteRequests answers DELETE /api/requests: clear the request log.
func (a *API) deleteRequests(writer http.ResponseWriter, request *http.Request) {
	a.core.Requests().Clear()
	writer.WriteHeader(http.StatusNoContent)
}

// streamRequests answers GET /api/requests/stream with server-sent events:
// the response stays open, and fapi writes an event to it whenever the log
// changes. Browsers read it with EventSource. There are two kinds of event:
//
//	event: request   data is the new request, as in GET /api/requests
//	event: cleared   the log was cleared; data is {}
func (a *API) streamRequests(writer http.ResponseWriter, request *http.Request) {
	events, unsubscribe := a.core.Requests().Subscribe()
	defer unsubscribe()

	headers := writer.Header()
	headers.Set("Content-Type", "text/event-stream")
	headers.Set("Cache-Control", "no-cache")
	writer.WriteHeader(http.StatusOK)

	// A ResponseController sends what's been written so far (Flush) instead
	// of waiting for the response to end.
	controller := http.NewResponseController(writer)
	err := controller.Flush()
	if err != nil {
		log.Printf("Admin API: starting the request stream: %v", err)
		return
	}

	for {
		// select waits for whichever happens first: the client going away
		// (or fapi shutting down), or a change to the log.
		select {
		case <-request.Context().Done():
			return
		case event := <-events:
			err := writeEvent(writer, event.Request)
			if err == nil {
				err = controller.Flush()
			}
			if err != nil {
				return // the client has gone
			}
		}
	}
}

// writeEvent writes one server-sent event. loggedRequest is nil when the
// log was cleared.
func writeEvent(writer http.ResponseWriter, loggedRequest *core.LoggedRequest) error {
	if loggedRequest == nil {
		_, err := fmt.Fprint(writer, "event: cleared\ndata: {}\n\n")
		return err
	}

	data, err := json.Marshal(loggedRequest)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(writer, "event: request\ndata: %s\n\n", data)
	return err
}
