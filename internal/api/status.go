package api

import (
	"net/http"
	"os"
)

// statusResponse is the body of GET /api/status.
type statusResponse struct {
	PID       int    `json:"pid"`
	Version   string `json:"version"`
	AdminPort int    `json:"adminPort"`
	MockPorts []int  `json:"mockPorts"` // the ports mock servers are listening on
}

// getStatus answers GET /api/status, which also tells clients fapi is running.
func (a *API) getStatus(writer http.ResponseWriter, request *http.Request) {
	ports := a.core.ListeningPorts()
	if ports == nil {
		ports = []int{} // so the JSON is [] rather than null
	}

	writeJSON(writer, http.StatusOK, statusResponse{
		PID:       os.Getpid(),
		Version:   a.version,
		AdminPort: a.core.Settings().AdminPort,
		MockPorts: ports,
	})
}

// getConfig answers GET /api/config: the effective settings and feature
// flags. UIs read it to hide features that are turned off.
func (a *API) getConfig(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, a.core.Settings())
}

// postShutdown answers POST /api/shutdown: stop fapi gracefully. The answer
// is sent before fapi stops, and the process exits shortly after.
func (a *API) postShutdown(writer http.ResponseWriter, request *http.Request) {
	writeError(writer, http.StatusAccepted, "fapi is shutting down")
	a.shutdown()
}
