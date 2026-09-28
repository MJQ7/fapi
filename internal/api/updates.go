package api

import (
	"context"
	"errors"
	"net/http"

	"fapi/internal/updates"
)

// getUpdates answers GET /api/updates: whether GitHub has a newer release,
// and how the running fapi was installed. The answer is remembered for an
// hour; ?force=true asks GitHub again.
func (a *API) getUpdates(writer http.ResponseWriter, request *http.Request) {
	force := request.URL.Query().Get("force") == "true"
	// WithoutCancel: finish the check even if the browser leaves the page,
	// so the answer is remembered rather than a "cancelled" error.
	result := a.updates.Check(context.WithoutCancel(request.Context()), force)
	writeJSON(writer, http.StatusOK, result)
}

// getUpdateInstall answers GET /api/updates/install: how the update being
// installed is getting on, or how the last one went.
func (a *API) getUpdateInstall(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, a.installer.Status())
}

// installRequest is the body of POST /api/updates/install.
type installRequest struct {
	Version string `json:"version"` // the newest release, as GET /api/updates found it
}

// postUpdateInstall answers POST /api/updates/install: download the newest
// release and, where fapi can, install it and restart. It answers at once;
// GET /api/updates/install follows the progress.
func (a *API) postUpdateInstall(writer http.ResponseWriter, request *http.Request) {
	var body installRequest
	if !readJSON(writer, request, &body) {
		return
	}

	job, err := a.installer.Start(context.WithoutCancel(request.Context()), body.Version)
	switch {
	case errors.Is(err, updates.ErrBusy):
		writeError(writer, http.StatusConflict, "An update is already being installed")
	case err != nil:
		writeError(writer, http.StatusBadRequest, err.Error())
	default:
		writeJSON(writer, http.StatusAccepted, job)
	}
}
