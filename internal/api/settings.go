package api

import (
	"context"
	"errors"
	"log"
	"net/http"

	"fapi/internal/config"
)

// portSettingsResponse is the body of GET and PUT /api/settings/ports.
// Settings are read when fapi starts, so a change is saved to the file but
// only used after a restart: Active and Saved differ until then.
type portSettingsResponse struct {
	File          string       `json:"file"`          // the settings file changes are saved to
	Active        config.Ports `json:"active"`        // what fapi is using now
	Saved         config.Ports `json:"saved"`         // what fapi will use after a restart
	RestartNeeded bool         `json:"restartNeeded"` // Active and Saved differ
}

// getPortSettings answers GET /api/settings/ports.
func (a *API) getPortSettings(writer http.ResponseWriter, request *http.Request) {
	saved, err := config.LoadFile(a.settingsFile)
	if err != nil {
		// The file was changed by hand since fapi started, and has a mistake.
		log.Printf("Admin API: %v", err)
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, a.portSettings(saved.Ports()))
}

// putPortSettings answers PUT /api/settings/ports: save new port settings to
// the settings file, for fapi's next start.
func (a *API) putPortSettings(writer http.ResponseWriter, request *http.Request) {
	var ports config.Ports
	if !readJSON(writer, request, &ports) {
		return
	}

	saved, err := config.SavePorts(a.settingsFile, ports)
	var settingError *config.SettingError
	if errors.As(err, &settingError) {
		writeError(writer, http.StatusBadRequest, settingError.Message)
		return
	}
	if err != nil {
		log.Printf("Admin API: %v", err)
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("Saved new port settings to %s; they take effect when fapi restarts", a.settingsFile)
	writeJSON(writer, http.StatusOK, a.portSettings(saved.Ports()))
}

func (a *API) portSettings(saved config.Ports) portSettingsResponse {
	active := a.core.Settings().Ports()
	return portSettingsResponse{
		File:          a.settingsFile,
		Active:        active,
		Saved:         saved,
		RestartNeeded: saved != active,
	}
}

// getUpdates answers GET /api/updates: whether GitHub has a newer release.
// The answer is remembered for an hour; ?force=true asks GitHub again.
func (a *API) getUpdates(writer http.ResponseWriter, request *http.Request) {
	force := request.URL.Query().Get("force") == "true"
	// WithoutCancel: finish the check even if the browser leaves the page,
	// so the answer is remembered rather than a "cancelled" error.
	result := a.updates.Check(context.WithoutCancel(request.Context()), force)
	writeJSON(writer, http.StatusOK, result)
}
