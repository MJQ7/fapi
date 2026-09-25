package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"fapi/internal/core"
)

// payloadRequest is the body of POST /api/payloads and PUT /api/payloads/{id}.
type payloadRequest struct {
	Name   string          `json:"name"`
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

func (body payloadRequest) toNewPayload() core.NewPayload {
	return core.NewPayload{Name: body.Name, Status: body.Status, Body: body.Body}
}

// getPayloads answers GET /api/payloads: the saved payloads.
func (a *API) getPayloads(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, a.core.Payloads())
}

// postPayload answers POST /api/payloads: save a new payload.
func (a *API) postPayload(writer http.ResponseWriter, request *http.Request) {
	var body payloadRequest
	if !readJSON(writer, request, &body) {
		return
	}

	payload, err := a.core.AddPayload(body.toNewPayload())
	if err != nil {
		writeCoreError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, payload)
}

// putPayload answers PUT /api/payloads/{id}: change a saved payload.
func (a *API) putPayload(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	var body payloadRequest
	if !readJSON(writer, request, &body) {
		return
	}

	payload, found, err := a.core.UpdatePayload(id, body.toNewPayload())
	if !found {
		writeError(writer, http.StatusNotFound, fmt.Sprintf("No payload with ID %s", id))
		return
	}
	if err != nil {
		writeCoreError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, payload)
}

// deletePayload answers DELETE /api/payloads/{id}: remove a saved payload.
func (a *API) deletePayload(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	found, err := a.core.RemovePayload(id)
	if err != nil {
		writeCoreError(writer, err)
		return
	}
	if !found {
		writeError(writer, http.StatusNotFound, fmt.Sprintf("No payload with ID %s", id))
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
