// Package api is the admin HTTP API: JSON routes under /api/ that let the
// web UI and the terminal UI read and change fapi. It is a thin layer:
// each handler reads the request, calls one method of core.Core, and writes
// the result as JSON. The web UI's files are served from here too.
//
// This API is the only way a UI changes fapi; no UI calls core directly.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"fapi/internal/core"
	"fapi/internal/updates"
)

// maxRequestBytes limits the JSON a client can send to the admin API.
const maxRequestBytes = 1 << 20 // 1 MB

// API answers admin requests. Create it with New.
type API struct {
	core         *core.Core
	version      string
	shutdown     func() // stops fapi gracefully; called by POST /api/shutdown
	settingsFile string // where the Settings screen saves ports (see config.SettingsFile)
	updates      *updates.Checker
}

// New returns the handler for the admin port: the API routes, and the web UI
// for every other path. settingsFile is the file port settings are saved to.
func New(fapi *core.Core, version string, shutdown func(), settingsFile string) http.Handler {
	a := &API{
		core:         fapi,
		version:      version,
		shutdown:     shutdown,
		settingsFile: settingsFile,
		updates:      updates.NewChecker(updates.ReleasesURL, version),
	}
	features := fapi.Settings().Features

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/status", a.getStatus)
	mux.HandleFunc("GET /api/config", a.getConfig)
	mux.HandleFunc("POST /api/shutdown", a.postShutdown)
	mux.HandleFunc("GET /api/settings/ports", a.getPortSettings)
	mux.HandleFunc("PUT /api/settings/ports", a.putPortSettings)
	mux.HandleFunc("GET /api/updates", a.getUpdates)

	mux.HandleFunc("GET /api/mocks", a.getMocks)
	mux.HandleFunc("POST /api/mocks", a.postMock)
	mux.HandleFunc("DELETE /api/mocks/{id}", a.deleteMock)
	mux.HandleFunc("PUT /api/mocks/{id}/enabled", a.putMockEnabled)
	mux.HandleFunc("PUT /api/enabled", a.putEnabled)

	mux.HandleFunc("GET /api/payloads", a.getPayloads)
	mux.HandleFunc("POST /api/payloads", a.postPayload)
	mux.HandleFunc("PUT /api/payloads/{id}", a.putPayload)
	mux.HandleFunc("DELETE /api/payloads/{id}", a.deletePayload)

	mux.HandleFunc("PUT /api/upstreams/{port}", requireFeature(features.PassThrough, "passThrough", a.putUpstream))
	mux.HandleFunc("PUT /api/upstreams/{port}/enabled", requireFeature(features.PassThrough, "passThrough", a.putUpstreamEnabled))

	mux.HandleFunc("GET /api/traffic", a.getTraffic)

	mux.HandleFunc("GET /api/requests", requireFeature(features.RequestLog, "requestLog", a.getRequests))
	mux.HandleFunc("DELETE /api/requests", requireFeature(features.RequestLog, "requestLog", a.deleteRequests))

	stream := requireFeature(features.LiveUpdates, "liveUpdates", a.streamRequests)
	mux.HandleFunc("GET /api/requests/stream", requireFeature(features.RequestLog, "requestLog", stream))

	// Anything else under /api/ is a mistake, so answer in JSON rather than
	// with the web UI's page.
	mux.HandleFunc("/api/", notFound)

	mux.Handle("/", webUIRoute(features.WebUI))

	return rejectOtherSites(mux)
}

// requireFeature returns handler if its feature is on, and otherwise a
// handler that says the feature is turned off.
func requireFeature(enabled bool, feature string, handler http.HandlerFunc) http.HandlerFunc {
	if enabled {
		return handler
	}
	return func(writer http.ResponseWriter, request *http.Request) {
		writeError(writer, http.StatusNotFound, fmt.Sprintf("%s is turned off in the fapi config", feature))
	}
}

func notFound(writer http.ResponseWriter, request *http.Request) {
	writeError(writer, http.StatusNotFound, fmt.Sprintf("No admin API route for %s %s", request.Method, request.URL.Path))
}

// rejectOtherSites refuses requests sent by a web page from another site.
// fapi has no login, so without this any page open in the browser could
// change endpoints or shut fapi down. Browsers add an Origin header to such
// requests; the web UI is on the same origin, so it isn't affected.
func rejectOtherSites(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if origin != "" && !isSameOrigin(origin, request.Host) {
			writeError(writer, http.StatusForbidden, "The fapi admin API only accepts requests from its own web UI, not from other sites")
			return
		}
		next.ServeHTTP(writer, request)
	})
}

// isSameOrigin reports whether origin (such as "http://localhost:3100")
// names the host the request was sent to (such as "localhost:3100").
func isSameOrigin(origin string, host string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Host, host)
}

// readJSON decodes the request body into target. If that fails, it answers
// 400 with a message for the user and returns false; the handler then stops.
func readJSON(writer http.ResponseWriter, request *http.Request, target any) bool {
	request.Body = http.MaxBytesReader(writer, request.Body, maxRequestBytes)
	err := json.NewDecoder(request.Body).Decode(target)
	if err == nil {
		return true
	}

	var typeError *json.UnmarshalTypeError
	if errors.As(err, &typeError) {
		message := fmt.Sprintf("%s must be %s", typeError.Field, describeType(typeError.Type.Kind().String()))
		writeError(writer, http.StatusBadRequest, message)
		return false
	}
	writeError(writer, http.StatusBadRequest, "The request body must be a JSON object")
	return false
}

// describeType names a Go type the way a JSON user would.
func describeType(goType string) string {
	switch goType {
	case "int":
		return "a number"
	case "bool":
		return "true or false"
	case "string":
		return "a string"
	default:
		return goType
	}
}

// writeJSON sends value as JSON with status.
func writeJSON(writer http.ResponseWriter, status int, value any) {
	contents, err := json.Marshal(value)
	if err != nil {
		log.Printf("Admin API: encoding a response: %v", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_, err = writer.Write(contents)
	if err != nil {
		log.Printf("Admin API: sending a response: %v", err)
	}
}

// writeError sends {"message": "..."} with status.
func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"message": message})
}

// writeCoreError answers an error from core: 400 with core's message when
// the user's input was invalid, and 500 for anything else.
func writeCoreError(writer http.ResponseWriter, err error) {
	var validationError *core.ValidationError
	if errors.As(err, &validationError) {
		writeError(writer, http.StatusBadRequest, validationError.Message)
		return
	}

	log.Printf("Admin API: %v", err)
	writeError(writer, http.StatusInternalServerError, err.Error())
}
