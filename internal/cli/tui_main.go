package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"fapi/internal/core"
)

// confirmAction is what the main screen is asking y/n about.
type confirmAction int

const (
	confirmNothing confirmAction = iota
	confirmDelete
	confirmStop
)

// handleMainKey handles keys on the main screen: the endpoint list.
func (m *model) handleMainKey(key tea.KeyPressMsg) tea.Cmd {
	if m.confirming != confirmNothing {
		return m.handleConfirmKey(key)
	}

	features := m.settings.Features
	switch key.String() {
	case "q":
		return tea.Quit
	case "up", "k":
		m.selected = max(m.selected-1, 0)
	case "down", "j":
		m.selected = min(m.selected+1, max(len(m.data.Mocks)-1, 0))
	case "a":
		m.openAddEndpointForm()
		return m.form.focusField(0)
	case "d":
		if len(m.data.Mocks) > 0 {
			m.confirming = confirmDelete
		}
	case "e":
		if len(m.data.Mocks) > 0 {
			mock := m.data.Mocks[m.selected]
			enabled := mock.Disabled // turning an off endpoint on, or an on one off
			message := onOff(fmt.Sprintf("%s %s on port %d turned", mock.Method, mock.Path, mock.Port), enabled)
			client := m.client
			return run(message, func() error { return client.SetMockEnabled(mock.ID, enabled) })
		}
	case "o":
		enabled := !m.data.Enabled
		client := m.client
		return run(onOff("Endpoints turned", enabled), func() error { return client.SetEnabled(enabled) })
	case "p":
		if features.PassThrough {
			m.openPassThroughForm()
			return m.form.focusField(0)
		}
	case "enter":
		if features.RequestLog && len(m.data.Mocks) > 0 {
			m.openRequests(m.data.Mocks[m.selected].ID)
		}
	case "r":
		if features.RequestLog {
			m.openRequests("")
		}
	case "s":
		m.confirming = confirmStop
	}
	return nil
}

// handleConfirmKey handles the answer to "Delete ...?" or "Stop fapi?".
func (m *model) handleConfirmKey(key tea.KeyPressMsg) tea.Cmd {
	action := m.confirming
	m.confirming = confirmNothing
	if key.String() != "y" {
		return nil
	}

	client := m.client
	if action == confirmStop {
		return func() tea.Msg {
			return stoppedMsg{err: client.Shutdown()}
		}
	}

	mock := m.data.Mocks[m.selected]
	message := fmt.Sprintf("Deleted %s %s on port %d", mock.Method, mock.Path, mock.Port)
	return run(message, func() error { return client.RemoveMock(mock.ID) })
}

// viewMain shows the proxies, and the endpoints with their request counts.
func (m *model) viewMain() string {
	var view strings.Builder

	view.WriteString(titleStyle.Render("Endpoints") + "  ")
	if m.data.Enabled {
		view.WriteString(successStyle.Render("on"))
	} else {
		view.WriteString(warningStyle.Render("off: every request goes to the real API"))
	}
	view.WriteString("\n\n")
	view.WriteString(m.endpointTable())

	if m.settings.Features.PassThrough {
		view.WriteString("\n" + titleStyle.Render("Proxies") + "\n\n")
		view.WriteString(m.passThroughList())
	}

	view.WriteString("\n")
	switch m.confirming {
	case confirmDelete:
		mock := m.data.Mocks[m.selected]
		view.WriteString(warningStyle.Render(fmt.Sprintf("Delete %s %s on port %d? (y/n)", mock.Method, mock.Path, mock.Port)))
	case confirmStop:
		view.WriteString(warningStyle.Render("Stop fapi? The mock servers and web UI stop too. (y/n)"))
	default:
		view.WriteString(m.mainHelp())
	}
	return view.String()
}

// endpointTable lists the endpoints in columns, with the selected one marked.
func (m *model) endpointTable() string {
	if len(m.data.Mocks) == 0 {
		return faintStyle.Render("No endpoints yet. Press a to add one.") + "\n"
	}

	// Make the path column as wide as the longest path.
	pathWidth := len("Path")
	for _, mock := range m.data.Mocks {
		pathWidth = max(pathWidth, len(mock.Path))
	}
	row := "%s %-3s %-6s %-7s %-" + strconv.Itoa(pathWidth) + "s  %-6s %s"

	var table strings.Builder
	requestsHeading := ""
	if m.settings.Features.RequestLog {
		requestsHeading = "Requests"
	}
	table.WriteString(faintStyle.Render(fmt.Sprintf(row, " ", "On", "Port", "Method", "Path", "Status", requestsHeading)) + "\n")

	for index, mock := range m.data.Mocks {
		requests := ""
		if m.settings.Features.RequestLog {
			requests = countRequests(m.requestCount(mock.ID))
		}
		on := "on"
		if mock.Disabled {
			on = "off"
		}
		line := fmt.Sprintf(row, " ", on, strconv.Itoa(mock.Port), mock.Method, mock.Path, strconv.Itoa(mock.Status), requests)
		switch {
		case index == m.selected:
			line = selectedStyle.Render(">" + line[1:])
		case mock.Disabled:
			line = faintStyle.Render(line)
		}
		table.WriteString(line + "\n")
	}
	return table.String()
}

// passThroughList shows one line per fapi port that proxies to the real API.
func (m *model) passThroughList() string {
	if len(m.data.Upstreams) == 0 {
		return faintStyle.Render("None. Press p to proxy a fapi port to your real API.") + "\n"
	}

	var ports []int
	for port := range m.data.Upstreams {
		ports = append(ports, port)
	}
	sort.Ints(ports) // map order is random

	var list strings.Builder
	for _, port := range ports {
		host := m.data.UpstreamHosts[port]
		if host == "" {
			host = m.settings.PassThrough.Host
		}
		line := fmt.Sprintf("  %d → %s", port, core.UpstreamURL(host, m.data.Upstreams[port]))
		if m.data.DisabledUpstreams[port] {
			line = faintStyle.Render(line + " (off)")
		}
		list.WriteString(line + "\n")
	}
	return list.String()
}

// mainHelp lists the keys for the main screen, leaving out turned-off features.
func (m *model) mainHelp() string {
	keys := []string{"↑/↓ select", "a override an endpoint", "e endpoint on/off", "d delete", "o all endpoints on/off"}
	if m.settings.Features.PassThrough {
		keys = append(keys, "p proxy")
	}
	if m.settings.Features.RequestLog {
		keys = append(keys, "enter endpoint's requests", "r all requests")
	}
	keys = append(keys, "s stop fapi", "q quit")
	return faintStyle.Render(strings.Join(keys, " · "))
}

// requestCount counts the logged requests answered by the endpoint with id,
// or all of them when id is "".
func (m *model) requestCount(id string) int {
	count := 0
	for _, request := range m.requests {
		if id == "" || request.MockID == id {
			count++
		}
	}
	return count
}

// countRequests writes a request count: "none yet", "1 request", "2 requests".
func countRequests(count int) string {
	switch count {
	case 0:
		return "none yet"
	case 1:
		return "1 request"
	default:
		return fmt.Sprintf("%d requests", count)
	}
}

// onOff writes "<prefix> on" or "<prefix> off".
func onOff(prefix string, on bool) string {
	if on {
		return prefix + " on"
	}
	return prefix + " off"
}
