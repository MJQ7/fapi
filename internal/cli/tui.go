package cli

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"fapi/internal/config"
	"fapi/internal/core"
)

// The interactive UI is built with Bubble Tea, which follows "The Elm
// Architecture". There are three parts:
//
//   - the model: a struct holding everything on screen (model, below)
//   - Update: receives a message (a key press, an answer from fapi, a timer)
//     and changes the model. It never waits: anything slow, such as calling
//     fapi, is returned as a command (tea.Cmd), a function Bubble Tea runs in
//     the background. The command's result arrives later as another message.
//   - View: turns the model into the text shown in the terminal.
//
// Bubble Tea calls Update for each message and View after each Update.
// Docs: https://github.com/charmbracelet/bubbletea

// screen says which screen is showing.
type screen int

// The screens. iota numbers them 0, 1, 2, ... in order.
const (
	screenConnecting screen = iota
	screenNotRunning
	screenStarting
	screenMain
	screenAddEndpoint
	screenPassThrough
	screenRequests
)

// refreshInterval is how often the UI asks fapi for the latest state, so it
// shows changes made elsewhere (in the web UI, for example).
const refreshInterval = 2 * time.Second

// model is the UI's state. Update and View have pointer receivers (*model),
// so Update changes the model in place.
type model struct {
	client   *Client
	dataDir  string
	adminURL string

	screen    screen
	width     int
	height    int
	connected bool // fapi answered the last time the UI asked
	ticking   bool // a refresh timer is running

	// What fapi last said.
	status   Status
	settings config.Config
	data     MocksData
	requests []core.LoggedRequest

	// The main screen.
	selected   int           // index of the selected endpoint in data.Mocks
	confirming confirmAction // what y/n is being asked about, if anything

	// The last action's result, shown under the screen.
	message      string
	messageIsErr bool

	// The add-endpoint and proxy screens.
	form         form
	sampleStatus string // the status the body's sample was made for

	// The requests screen.
	requestsFor    string // show requests for this endpoint ID only; "" for all
	requestsOffset int    // how many requests are scrolled past

	// The live request stream; nil when not connected to it.
	events  <-chan streamEvent
	opening bool // the stream is being opened
}

func newModel(client *Client, dataDir string, adminURL string) *model {
	return &model{client: client, dataDir: dataDir, adminURL: adminURL, screen: screenConnecting}
}

// Messages: the results of commands, delivered to Update.

type connectedMsg struct {
	status   Status
	settings config.Config
	err      error
}

type snapshotMsg struct {
	status      Status
	data        MocksData
	requests    []core.LoggedRequest
	hasRequests bool // requests was fetched this time
	err         error
}

type tickMsg struct{}

type startedMsg struct{ err error }

type stoppedMsg struct{ err error }

type streamOpenedMsg struct {
	events <-chan streamEvent
	err    error
}

type streamEventMsg struct {
	events <-chan streamEvent // the stream it came from
	event  streamEvent
}

type streamClosedMsg struct{ events <-chan streamEvent }

// actionDoneMsg reports a change made from the main or requests screen.
type actionDoneMsg struct {
	message string
	err     error
}

// formDoneMsg reports the result of submitting a form.
type formDoneMsg struct {
	message string
	err     error
}

// Init is called once at the start: connect to fapi.
func (m *model) Init() tea.Cmd {
	return m.connect()
}

// Commands: functions Bubble Tea runs in the background. Each returns the
// message Update receives when it's done.

// connect asks fapi for its status and settings.
func (m *model) connect() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		status, err := client.Status()
		if err != nil {
			return connectedMsg{err: err}
		}
		settings, err := client.Config()
		return connectedMsg{status: status, settings: settings, err: err}
	}
}

// refresh asks fapi for its status and endpoints, and for the request log
// when includeRequests is true.
func (m *model) refresh(includeRequests bool) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		status, err := client.Status()
		if err != nil {
			return snapshotMsg{err: err}
		}
		data, err := client.Mocks()
		if err != nil {
			return snapshotMsg{err: err}
		}
		if !includeRequests {
			return snapshotMsg{status: status, data: data}
		}
		requests, err := client.Requests()
		return snapshotMsg{status: status, data: data, requests: requests, hasRequests: true, err: err}
	}
}

// tick waits refreshInterval, then sends tickMsg.
func tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

// start starts fapi in the background.
func (m *model) start() tea.Cmd {
	client, dataDir := m.client, m.dataDir
	return func() tea.Msg {
		return startedMsg{err: startInBackground(client, dataDir)}
	}
}

// openStream opens the live request stream, and starts a goroutine that
// passes its events into a channel.
func (m *model) openStream() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		stream, err := client.OpenRequestStream()
		if err != nil {
			return streamOpenedMsg{err: err}
		}
		events := make(chan streamEvent)
		go readRequestStream(stream, events)
		return streamOpenedMsg{events: events}
	}
}

// waitForEvent waits for the next event from the stream. Update calls it
// again after each event, so the UI keeps listening.
func waitForEvent(events <-chan streamEvent) tea.Cmd {
	return func() tea.Msg {
		// The second value is false once the channel is closed.
		event, open := <-events
		if !open {
			return streamClosedMsg{events: events}
		}
		return streamEventMsg{events: events, event: event}
	}
}

// run returns a command that calls action and reports it as an
// actionDoneMsg with message, or with action's error.
func run(message string, action func() error) tea.Cmd {
	return func() tea.Msg {
		err := action()
		if err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: message}
	}
}

// liveUpdates reports whether the request log arrives through the stream,
// rather than by asking for it every refreshInterval.
func (m *model) liveUpdates() bool {
	return m.settings.Features.RequestLog && m.settings.Features.LiveUpdates
}

func (m *model) showMessage(message string, err error) {
	if err != nil {
		m.message, m.messageIsErr = err.Error(), true
		return
	}
	m.message, m.messageIsErr = message, false
}
