package cli

import (
	"encoding/json"

	"charm.land/lipgloss/v2"
)

// Text styles, made with Lip Gloss. The colours are the terminal's own
// numbered colours (1 red, 2 green, 3 yellow), so they suit both light and
// dark terminal themes. These are set once and never changed.
var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	faintStyle    = lipgloss.NewStyle().Faint(true)
	selectedStyle = lipgloss.NewStyle().Reverse(true)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	warningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

// statusNames names the statuses the web UI offers; others get no name.
var statusNames = map[int]string{
	200: "OK",
	201: "Created",
	400: "Bad Request",
	401: "Unauthorized",
	403: "Forbidden",
	404: "Not Found",
	409: "Conflict",
	418: "I'm a teapot",
	422: "Unprocessable Entity",
	500: "Internal Server Error",
	502: "Bad Gateway",
	503: "Service Unavailable",
	504: "Gateway Timeout",
}

// sampleBody returns a sample response body for a status, on one line, the
// same as the web UI's (web/src/lib/statuses.ts).
func sampleBody(code int) string {
	name := statusNames[code]

	// Structs keep the fields in this order in the JSON; maps would sort them.
	var sample any = struct {
		Message string `json:"message"`
	}{Message: name}

	if code >= 400 {
		sample = struct {
			StatusCode int    `json:"statusCode"`
			Error      string `json:"error"`
			Message    string `json:"message"`
		}{StatusCode: code, Error: name, Message: "An error occurred"}
	}

	encoded, err := json.Marshal(sample)
	if err != nil {
		return "" // not possible for these values
	}
	return string(encoded)
}
