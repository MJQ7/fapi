package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"

	"fapi/internal/core"
)

// maxBodyLines is how many lines of each request body the requests screen
// shows, so one large body doesn't fill the screen.
const maxBodyLines = 8

// openRequests shows the requests screen, for one endpoint or ("") for all.
func (m *model) openRequests(mockID string) {
	m.requestsFor = mockID
	m.requestsOffset = 0
	m.screen = screenRequests
}

// handleRequestsKey handles keys on the requests screen.
func (m *model) handleRequestsKey(key tea.KeyPressMsg) tea.Cmd {
	switch key.String() {
	case "esc", "q":
		m.screen = screenMain
	case "up", "k":
		m.requestsOffset = max(m.requestsOffset-1, 0)
	case "down", "j":
		m.requestsOffset = min(m.requestsOffset+1, max(len(m.shownRequests())-1, 0))
	case "c":
		client := m.client
		return run("Cleared the request log", client.ClearRequests)
	}
	return nil
}

// shownRequests returns the requests the screen is showing, newest first.
func (m *model) shownRequests() []core.LoggedRequest {
	if m.requestsFor == "" {
		return m.requests
	}
	var shown []core.LoggedRequest
	for _, request := range m.requests {
		if request.MockID == m.requestsFor {
			shown = append(shown, request)
		}
	}
	return shown
}

// viewRequests shows the requests, newest first, as many as fit.
func (m *model) viewRequests() string {
	var view strings.Builder

	title := "All requests"
	if m.requestsFor != "" {
		title = "Requests to " + m.describeMock(m.requestsFor)
	}
	shown := m.shownRequests()
	view.WriteString(titleStyle.Render(title) + "  " + faintStyle.Render(countRequests(len(shown))) + "\n\n")

	if len(shown) == 0 {
		view.WriteString(faintStyle.Render("Requests appear here as they arrive.") + "\n")
	}

	// Leave room for the header, the status line and the help.
	space := m.height - 8
	lines := 0
	for _, request := range shown[min(m.requestsOffset, len(shown)):] {
		block := formatRequest(request, m.width)
		blockLines := strings.Count(block, "\n")
		if lines > 0 && space > 0 && lines+blockLines > space {
			view.WriteString(faintStyle.Render("↓ more") + "\n")
			break
		}
		view.WriteString(block)
		lines += blockLines
	}

	view.WriteString("\n" + faintStyle.Render("↑/↓ scroll · c clear the request log · esc back"))
	return view.String()
}

// describeMock names an endpoint by method, path and port.
func (m *model) describeMock(id string) string {
	for _, mock := range m.data.Mocks {
		if mock.ID == id {
			return fmt.Sprintf("%s %s on port %d", mock.Method, mock.Path, mock.Port)
		}
	}
	return "a deleted endpoint"
}

// formatRequest writes one request: a summary line, then its body, indented.
// Everything is plain text, so nothing a request sends can change the screen.
func formatRequest(request core.LoggedRequest, width int) string {
	var block strings.Builder

	summary := fmt.Sprintf("%s  %s %s%s → %d %s", formatTime(request.Time), request.Method, request.Path, request.Search, request.Status, request.Outcome)
	block.WriteString(strings.Map(removeControlCharacters, summary) + "\n")

	details := fmt.Sprintf("port %d · from port %d", request.Port, request.FromPort)
	if request.ContentType != "" {
		details += " · " + request.ContentType
	}
	block.WriteString("  " + faintStyle.Render(strings.Map(removeControlCharacters, details)) + "\n")

	if request.Body == "" {
		block.WriteString("  " + faintStyle.Render("no payload") + "\n\n")
		return block.String()
	}

	bodyLines := strings.Split(formatBody(request.Body), "\n")
	if len(bodyLines) > maxBodyLines {
		bodyLines = append(bodyLines[:maxBodyLines], "…")
	}
	for _, line := range bodyLines {
		block.WriteString("  " + cut(line, width-2) + "\n")
	}
	block.WriteString("\n")
	return block.String()
}

// formatTime shows the time of day a request arrived, in local time.
func formatTime(text string) string {
	arrived, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return text
	}
	return arrived.Local().Format("15:04:05")
}

// formatBody pretty-prints a JSON body. Anything else is shown as it came,
// with control characters (which could move the cursor) removed.
func formatBody(body string) string {
	var value any
	err := json.Unmarshal([]byte(body), &value)
	if err == nil {
		pretty, err := json.MarshalIndent(value, "", "  ")
		if err == nil {
			return string(pretty)
		}
	}
	return strings.Map(removeControlCharacters, body)
}

// removeControlCharacters keeps line breaks, tabs and printable characters.
// Returning -1 tells strings.Map to drop the character.
func removeControlCharacters(character rune) rune {
	if character == '\n' || character == '\t' {
		return character
	}
	if unicode.IsControl(character) {
		return -1
	}
	return character
}

// cut shortens text to width characters, when width is known.
func cut(text string, width int) string {
	characters := []rune(text)
	if width <= 1 || len(characters) <= width {
		return text
	}
	return string(characters[:width-1]) + "…"
}
