package cli

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// View draws the whole screen: a header, the current screen and the result
// of the last action.
func (m *model) View() tea.View {
	var content strings.Builder
	content.WriteString(m.header() + "\n\n")

	switch m.screen {
	case screenConnecting:
		content.WriteString("Connecting to fapi at " + m.adminURL + "…")
	case screenNotRunning:
		content.WriteString(m.viewNotRunning())
	case screenStarting:
		content.WriteString("Starting fapi in the background…")
	case screenMain:
		content.WriteString(m.viewMain())
	case screenAddEndpoint, screenPassThrough:
		content.WriteString(m.form.view())
	case screenRequests:
		content.WriteString(m.viewRequests())
	}

	if m.message != "" && m.screen != screenAddEndpoint && m.screen != screenPassThrough {
		style := successStyle
		if m.messageIsErr {
			style = errorStyle
		}
		content.WriteString("\n\n" + style.Render(m.message))
	}

	view := tea.NewView(content.String())
	// The alternate screen is a separate, full-window screen that the
	// terminal puts away when the program ends, restoring what was there.
	view.AltScreen = true
	view.WindowTitle = "fapi"
	return view
}

// header shows whether fapi is running, and where.
func (m *model) header() string {
	title := titleStyle.Render("fapi")
	if !m.connected {
		return title + "  " + faintStyle.Render(m.adminURL)
	}

	running := fmt.Sprintf("%s running · pid %d · web UI and admin API at %s", m.status.Version, m.status.PID, m.adminURL)
	if !m.settings.Features.WebUI {
		running = fmt.Sprintf("%s running · pid %d · admin API at %s", m.status.Version, m.status.PID, m.adminURL)
	}
	return title + "  " + successStyle.Render("●") + " " + faintStyle.Render(running)
}

// handleNotRunningKey handles the answer to "Start fapi?".
func (m *model) handleNotRunningKey(key tea.KeyPressMsg) tea.Cmd {
	switch key.String() {
	case "y", "enter":
		m.screen = screenStarting
		m.message = ""
		return m.start()
	case "n", "q", "esc":
		return tea.Quit
	}
	return nil
}

// viewNotRunning offers to start fapi.
func (m *model) viewNotRunning() string {
	return "fapi isn't running: nothing answers at " + m.adminURL + ".\n\n" +
		"Start it in the background? It keeps running after you leave this screen,\n" +
		"until you stop it (press s on the main screen).\n\n" +
		faintStyle.Render("y start · n quit")
}
