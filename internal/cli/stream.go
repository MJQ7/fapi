package cli

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"

	"fapi/internal/core"
)

// streamEvent is one server-sent event from GET /api/requests/stream.
// Request is nil when the log was cleared.
type streamEvent struct {
	Request *core.LoggedRequest
}

// readRequestStream reads server-sent events from stream and sends them to
// events until the stream ends (usually because fapi stopped). It then
// closes events, which tells the receiver the stream is over.
//
// It runs in its own goroutine, because reading waits for fapi to write.
//
// Each event is a few lines followed by an empty line:
//
//	event: request
//	data: {"id": "...", ...}
func readRequestStream(stream io.ReadCloser, events chan<- streamEvent) {
	defer close(events)
	defer closeQuietly(stream)

	scanner := bufio.NewScanner(stream)
	// A logged request can be larger than the scanner's default 64 KB line.
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	eventName := ""
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "event: "):
			eventName = strings.TrimPrefix(line, "event: ")

		case strings.HasPrefix(line, "data: ") && eventName == "cleared":
			events <- streamEvent{Request: nil}

		case strings.HasPrefix(line, "data: ") && eventName == "request":
			var request core.LoggedRequest
			err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &request)
			if err != nil {
				continue // not expected from fapi; skip the event
			}
			events <- streamEvent{Request: &request}

		case line == "":
			eventName = "" // the end of one event
		}
	}
}
