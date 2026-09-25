package cli

import (
	"errors"

	tea "charm.land/bubbletea/v2"

	"fapi/internal/core"
)

// Update receives every message and changes the model. Messages about fapi
// (answers, stream events, timers) are handled here; key presses go to the
// screen that's showing.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// A "type switch": run the case matching msg's type, with msg as that type.
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, m.handleKey(msg)

	case connectedMsg:
		return m, m.handleConnected(msg)

	case snapshotMsg:
		return m, m.handleSnapshot(msg)

	case tickMsg:
		return m, m.handleTick()

	case startedMsg:
		if msg.err != nil {
			m.screen = screenNotRunning
			m.showMessage("", msg.err)
			return m, nil
		}
		m.showMessage("Started fapi in the background", nil)
		return m, m.connect()

	case stoppedMsg:
		if msg.err != nil {
			m.showMessage("", msg.err)
			return m, nil
		}
		m.connected = false
		m.events = nil
		m.screen = screenNotRunning
		m.showMessage("Stopped fapi", nil)
		return m, nil

	case streamOpenedMsg:
		m.opening = false
		if msg.err != nil {
			m.showMessage("", msg.err)
			return m, nil
		}
		m.events = msg.events
		// Load the log once; the stream then brings each new request.
		return m, tea.Batch(waitForEvent(m.events), m.refresh(true))

	case streamEventMsg:
		if msg.events != m.events {
			return m, nil // from an old stream that has been replaced
		}
		m.applyEvent(msg.event)
		return m, waitForEvent(m.events)

	case streamClosedMsg:
		if msg.events == m.events {
			m.events = nil // reopened on the next tick, if fapi is still running
		}
		return m, nil

	case actionDoneMsg:
		m.confirming = confirmNothing
		m.showMessage(msg.message, msg.err)
		return m, m.refresh(!m.liveUpdates())

	case formDoneMsg:
		return m, m.handleFormDone(msg)
	}

	// Anything else (such as the text cursor blinking) belongs to the form.
	if m.screen == screenAddEndpoint || m.screen == screenPassThrough {
		return m, m.form.updateInput(msg)
	}
	return m, nil
}

// handleKey sends a key press to the screen that's showing.
func (m *model) handleKey(key tea.KeyPressMsg) tea.Cmd {
	switch m.screen {
	case screenNotRunning:
		return m.handleNotRunningKey(key)
	case screenMain:
		return m.handleMainKey(key)
	case screenAddEndpoint, screenPassThrough:
		return m.handleFormKey(key)
	case screenRequests:
		return m.handleRequestsKey(key)
	default:
		// While connecting or starting, only quitting is possible.
		if key.String() == "q" {
			return tea.Quit
		}
		return nil
	}
}

// handleConnected moves to the main screen once fapi answers, or offers to
// start it if it doesn't.
func (m *model) handleConnected(msg connectedMsg) tea.Cmd {
	if msg.err != nil {
		m.connected = false
		m.screen = screenNotRunning
		return nil
	}

	m.connected = true
	m.status = msg.status
	m.settings = msg.settings
	m.screen = screenMain

	commands := []tea.Cmd{m.refresh(m.settings.Features.RequestLog)}
	if !m.ticking {
		m.ticking = true
		commands = append(commands, tick())
	}
	if m.liveUpdates() {
		m.opening = true
		commands = append(commands, m.openStream())
	}
	return tea.Batch(commands...)
}

// handleSnapshot stores fapi's latest state, or notices it has stopped.
func (m *model) handleSnapshot(msg snapshotMsg) tea.Cmd {
	if msg.err != nil {
		if m.connected {
			m.connected = false
			m.events = nil
			m.screen = screenNotRunning
			m.showMessage("", errors.New("fapi has stopped"))
		}
		return nil
	}

	m.status = msg.status
	m.data = msg.data
	if msg.hasRequests {
		m.requests = msg.requests
	}
	// Keep the selection on an endpoint that still exists.
	m.selected = min(m.selected, max(len(m.data.Mocks)-1, 0))
	return nil
}

// handleTick refreshes the state and schedules the next tick, while fapi is
// running.
func (m *model) handleTick() tea.Cmd {
	if !m.connected {
		m.ticking = false
		return nil
	}

	commands := []tea.Cmd{tick(), m.refresh(m.settings.Features.RequestLog && !m.liveUpdates())}
	if m.liveUpdates() && m.events == nil && !m.opening {
		m.opening = true
		commands = append(commands, m.openStream())
	}
	return tea.Batch(commands...)
}

// applyEvent adds a new request to the log, or empties it.
func (m *model) applyEvent(event streamEvent) {
	if event.Request == nil {
		m.requests = nil
		m.requestsOffset = 0
		return
	}

	m.requests = append([]core.LoggedRequest{*event.Request}, m.requests...)
	maxEntries := m.settings.RequestLog.MaxEntries
	if maxEntries > 0 && len(m.requests) > maxEntries {
		m.requests = m.requests[:maxEntries]
	}
}
