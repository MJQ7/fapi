package core

import (
	"encoding/json"
	"slices"
	"strings"
)

// Payload is a saved response: a name, a status and a JSON body. Choosing one
// when adding an endpoint fills in its status and body, so the same JSON
// doesn't have to be typed each time. Endpoints copy the values, so changing
// or removing a payload doesn't change endpoints added from it.
type Payload struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body,omitempty"`
}

// NewPayload is what's needed to add or change a payload. Core adds the ID.
type NewPayload struct {
	Name   string
	Status int
	Body   json.RawMessage
}

// maxPayloadNameLength keeps names short enough to fit in a menu.
const maxPayloadNameLength = 80

// Payloads returns a copy of the saved payloads, in the order they were added.
func (c *Core) Payloads() []Payload {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return append([]Payload{}, c.data.Payloads...)
}

// AddPayload validates and saves a new payload.
func (c *Core) AddPayload(newPayload NewPayload) (Payload, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	payload, err := c.makePayload(newID(), newPayload)
	if err != nil {
		return Payload{}, err
	}
	c.data.Payloads = append(c.data.Payloads, payload)
	return payload, c.save()
}

// UpdatePayload replaces the payload with that ID. It returns false if there
// was no payload with that ID.
func (c *Core) UpdatePayload(id string, newPayload NewPayload) (Payload, bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	index := c.payloadIndex(id)
	if index < 0 {
		return Payload{}, false, nil
	}

	payload, err := c.makePayload(id, newPayload)
	if err != nil {
		return Payload{}, true, err
	}
	c.data.Payloads[index] = payload
	return payload, true, c.save()
}

// RemovePayload removes a payload. It returns false if there was no payload
// with that ID.
func (c *Core) RemovePayload(id string) (bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	index := c.payloadIndex(id)
	if index < 0 {
		return false, nil
	}

	c.data.Payloads = slices.Delete(c.data.Payloads, index, index+1)
	return true, c.save()
}

// makePayload validates newPayload and returns it as a Payload with id.
// The caller must hold the mutex.
func (c *Core) makePayload(id string, newPayload NewPayload) (Payload, error) {
	name := strings.TrimSpace(newPayload.Name)
	if name == "" {
		return Payload{}, invalid("Name must not be empty")
	}
	if len([]rune(name)) > maxPayloadNameLength {
		return Payload{}, invalid("Name must be at most %d characters", maxPayloadNameLength)
	}
	for _, payload := range c.data.Payloads {
		if payload.ID != id && strings.EqualFold(payload.Name, name) {
			return Payload{}, invalid("There is already a payload named %q", payload.Name)
		}
	}

	err := validateResponse(newPayload.Status, newPayload.Body)
	if err != nil {
		return Payload{}, err
	}

	return Payload{
		ID:     id,
		Name:   name,
		Status: newPayload.Status,
		Body:   normalizeBody(newPayload.Body),
	}, nil
}

// payloadIndex returns the position of the payload with id, or -1.
// The caller must hold the mutex.
func (c *Core) payloadIndex(id string) int {
	for index, payload := range c.data.Payloads {
		if payload.ID == id {
			return index
		}
	}
	return -1
}
