package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// The fields of the add-endpoint form, in order.
const (
	fieldPort = iota
	fieldMethod
	fieldPath
	fieldStatus
	fieldBody
)

// The fields of the proxy form, in order.
const (
	fieldFapiPort = iota
	fieldUpstreamHost
	fieldUpstreamPort
)

// openAddEndpointForm shows the add-endpoint form. The port starts as the
// selected endpoint's, since endpoints are usually added to the same port,
// or else the default mock port from fapi's settings.
func (m *model) openAddEndpointForm() {
	port := m.defaultMockPort()
	if len(m.data.Mocks) > 0 {
		port = strconv.Itoa(m.data.Mocks[m.selected].Port)
	}

	m.sampleStatus = "400"
	m.form = newForm("Override an endpoint",
		formField{label: "Port", value: port},
		formField{label: "Method (GET, POST, PUT, PATCH or DELETE)", value: "GET"},
		formField{label: "Path (a segment starting with : matches any value)", placeholder: "/api/users/:id"},
		formField{label: "Response status", value: m.sampleStatus},
		formField{label: "Response body (JSON, optional)", value: sampleBody(400)},
	)
	m.screen = screenAddEndpoint
}

// openPassThroughForm shows the proxy form for the selected
// endpoint's port, with its current real API host and port if it has them.
func (m *model) openPassThroughForm() {
	port, host, upstreamPort := "", "", ""
	if len(m.data.Mocks) > 0 {
		selectedPort := m.data.Mocks[m.selected].Port
		port = strconv.Itoa(selectedPort)
		if current, found := m.data.Upstreams[selectedPort]; found {
			upstreamPort = strconv.Itoa(current)
			host = m.data.UpstreamHosts[selectedPort]
		}
	}

	m.form = newForm("Proxy: forward requests that no endpoint overrides to the real API",
		formField{label: "fapi port", value: port, placeholder: m.defaultMockPort()},
		formField{label: "Real API host (empty for " + m.settings.PassThrough.Host + "; start with https:// for HTTPS)", value: host, placeholder: m.settings.PassThrough.Host},
		formField{label: "Real API port (empty removes the proxy)", value: upstreamPort, placeholder: "3000"},
	)
	m.screen = screenPassThrough
}

// defaultMockPort is the port the forms suggest (mockPorts.default in fapi's
// settings). A fapi older than that setting sends 0, so fall back to 3001.
func (m *model) defaultMockPort() string {
	if m.settings.MockPorts.Default == 0 {
		return "3001"
	}
	return strconv.Itoa(m.settings.MockPorts.Default)
}

// handleFormKey handles keys on either form screen.
func (m *model) handleFormKey(key tea.KeyPressMsg) tea.Cmd {
	if m.form.saving {
		return nil // wait for fapi's answer
	}

	result, command := m.form.handleKey(key)
	switch result {
	case formCancelled:
		m.screen = screenMain
		return nil
	case formSubmitted:
		if m.screen == screenAddEndpoint {
			return m.submitAddEndpoint()
		}
		return m.submitPassThrough()
	}

	if m.screen == screenAddEndpoint {
		m.updateSampleBody()
	}
	return command
}

// updateSampleBody replaces the body with a sample for the status typed, as
// long as the body is still the sample for the previous status (so a body
// the user wrote is never replaced).
func (m *model) updateSampleBody() {
	status := m.form.value(fieldStatus)
	if status == m.sampleStatus {
		return
	}

	previousCode, _ := strconv.Atoi(m.sampleStatus) // not a number gives 0, which still has a sample
	if m.form.value(fieldBody) != sampleBody(previousCode) {
		return
	}

	code, _ := strconv.Atoi(status) // as above
	m.form.inputs[fieldBody].SetValue(sampleBody(code))
	// Show the new body from its start, not scrolled to where the old one ended.
	m.form.inputs[fieldBody].CursorStart()
	m.sampleStatus = status
}

// submitAddEndpoint checks the form and sends the new endpoint to fapi.
// fapi does the real validation; this only turns text into numbers and JSON.
func (m *model) submitAddEndpoint() tea.Cmd {
	port, err := strconv.Atoi(m.form.value(fieldPort))
	if err != nil {
		m.form.err = "Port must be a number"
		return nil
	}
	status, err := strconv.Atoi(m.form.value(fieldStatus))
	if err != nil {
		m.form.err = "Response status must be a number"
		return nil
	}
	body := m.form.value(fieldBody)
	if body != "" && !json.Valid([]byte(body)) {
		m.form.err = "The response body is not valid JSON"
		return nil
	}

	mock := NewMock{
		Port:   port,
		Method: strings.ToUpper(m.form.value(fieldMethod)),
		Path:   m.form.value(fieldPath),
		Status: status,
		Body:   json.RawMessage(body),
	}
	message := fmt.Sprintf("Added %s %s on port %d", mock.Method, mock.Path, mock.Port)

	m.form.err = ""
	m.form.saving = true
	client := m.client
	return func() tea.Msg {
		return formDoneMsg{message: message, err: client.AddMock(mock)}
	}
}

// submitPassThrough sends the proxy to fapi.
func (m *model) submitPassThrough() tea.Cmd {
	port, err := strconv.Atoi(m.form.value(fieldFapiPort))
	if err != nil {
		m.form.err = "fapi port must be a number"
		return nil
	}
	host := m.form.value(fieldUpstreamHost)
	upstreamPort := m.form.value(fieldUpstreamPort)

	shownHost := host
	if shownHost == "" {
		shownHost = m.settings.PassThrough.Host
	}
	message := fmt.Sprintf("Port %d now proxies to %s port %s", port, shownHost, upstreamPort)
	if upstreamPort == "" {
		message = fmt.Sprintf("Removed the proxy from port %d", port)
	}

	m.form.err = ""
	m.form.saving = true
	client := m.client
	return func() tea.Msg {
		return formDoneMsg{message: message, err: client.SetUpstream(port, upstreamPort, host)}
	}
}

// handleFormDone goes back to the main screen after a successful submit, or
// shows fapi's message in the form.
func (m *model) handleFormDone(msg formDoneMsg) tea.Cmd {
	m.form.saving = false
	if msg.err != nil {
		m.form.err = msg.err.Error()
		return nil
	}

	m.screen = screenMain
	m.showMessage(msg.message, nil)
	return m.refresh(!m.liveUpdates())
}
