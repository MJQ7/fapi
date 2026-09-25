package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"fapi/internal/files"
)

// LoggedRequest is one request that arrived on a fapi port.
type LoggedRequest struct {
	ID          string `json:"id"`
	Time        string `json:"time"` // RFC 3339, such as 2026-09-21T14:03:12.345Z
	Port        int    `json:"port"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Search      string `json:"search"` // the query string with its "?", or ""
	ContentType string `json:"contentType"`
	Body        string `json:"body"`

	// BodyTruncated is true when Body was cut at the size limit. Body then
	// also ends with "…" so the cut is visible wherever it's shown.
	BodyTruncated bool `json:"bodyTruncated"`

	Status   int    `json:"status"`
	Outcome  string `json:"outcome"`            // mocked, proxied, unmatched or preflight
	MockID   string `json:"mockId,omitempty"`   // the endpoint that answered, when mocked
	Upstream int    `json:"upstream,omitempty"` // the real API port, when proxied
	FromPort int    `json:"fromPort"`           // the sender's port; the address is assumed to be localhost
}

// The outcomes a logged request can have.
const (
	OutcomeMocked    = "mocked"
	OutcomeProxied   = "proxied"
	OutcomeUnmatched = "unmatched"
	OutcomePreflight = "preflight"
)

// RequestLog keeps the most recent requests, newest first.
type RequestLog struct {
	maxEntries int
	file       string // requests.json, or "" to keep the log in memory only

	mutex       sync.Mutex // protects the fields below
	entries     []LoggedRequest
	subscribers map[chan LogEvent]bool
}

// LogEvent is sent to subscribers when the log changes. Request is nil when
// the log was cleared.
type LogEvent struct {
	Request *LoggedRequest
}

// subscriberBuffer is how many events a subscriber can fall behind by before
// it starts missing them. Recording a request never waits for a slow client.
const subscriberBuffer = 64

// NewRequestLog creates the log. When file isn't "", the log is loaded from
// it and saved to it after every change.
func NewRequestLog(maxEntries int, file string) (*RequestLog, error) {
	requestLog := &RequestLog{
		maxEntries:  maxEntries,
		file:        file,
		entries:     []LoggedRequest{},
		subscribers: map[chan LogEvent]bool{},
	}
	if file == "" {
		return requestLog, nil
	}

	contents, err := os.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		return requestLog, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", file, err)
	}

	err = json.Unmarshal(contents, &requestLog.entries)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", file, err)
	}
	if len(requestLog.entries) > maxEntries {
		requestLog.entries = requestLog.entries[:maxEntries]
	}
	return requestLog, nil
}

// Add records a request at the front of the log and tells subscribers.
func (l *RequestLog) Add(entry LoggedRequest) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Put entry first, then drop the oldest ones over the limit.
	l.entries = append([]LoggedRequest{entry}, l.entries...)
	if len(l.entries) > l.maxEntries {
		l.entries = l.entries[:l.maxEntries]
	}

	l.save()
	l.notify(LogEvent{Request: &entry})
}

// All returns a copy of the log, newest first.
func (l *RequestLog) All() []LoggedRequest {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	return append([]LoggedRequest{}, l.entries...)
}

// Clear empties the log and tells subscribers.
func (l *RequestLog) Clear() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.entries = []LoggedRequest{}
	l.save()
	l.notify(LogEvent{Request: nil})
}

// Subscribe returns a channel that receives an event for every change to the
// log, and a function to call when done listening.
func (l *RequestLog) Subscribe() (<-chan LogEvent, func()) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	events := make(chan LogEvent, subscriberBuffer)
	l.subscribers[events] = true

	unsubscribe := func() {
		l.mutex.Lock()
		defer l.mutex.Unlock()
		delete(l.subscribers, events)
	}
	return events, unsubscribe
}

// notify sends event to every subscriber. The caller must hold the mutex.
func (l *RequestLog) notify(event LogEvent) {
	for events := range l.subscribers {
		// select with a default case sends only if there's room in the
		// channel, so a subscriber that stopped reading can't block fapi.
		select {
		case events <- event:
		default:
		}
	}
}

// save writes the log to its file, when it has one. A failure is logged
// rather than returned: losing the saved copy shouldn't fail a request.
// The caller must hold the mutex.
func (l *RequestLog) save() {
	if l.file == "" {
		return
	}

	contents, err := json.Marshal(l.entries)
	if err != nil {
		log.Printf("Saving the request log: %v", err)
		return
	}

	err = files.WriteAtomic(l.file, contents)
	if err != nil {
		log.Printf("Saving the request log: %v", err)
	}
}
