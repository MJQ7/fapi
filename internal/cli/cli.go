// Package cli is fapi's terminal interface. Running fapi with no command in
// a terminal opens an interactive UI, built with Bubble Tea, to see fapi's
// status, list, add and remove endpoints, turn them on and off, set
// proxies and watch requests arrive live. If fapi isn't running, it
// offers to start it in the background.
//
// Like the web UI, it's a client of the admin API: it calls it over HTTP
// (client.go) rather than using the core package, so it works with a fapi
// started any way (by this UI, `fapi serve`, systemd or Docker).
package cli

import (
	"fmt"
	"net"
	"strconv"

	tea "charm.land/bubbletea/v2"

	"fapi/internal/config"
)

// RunInteractive runs the interactive UI until the user quits. fapi itself
// keeps running afterwards.
func RunInteractive() error {
	dataDir, err := config.DataDir("")
	if err != nil {
		return err
	}

	// The same settings `fapi serve` would use, to find the admin port.
	settings, _, err := config.Load("", dataDir)
	if err != nil {
		return err
	}
	url := adminURL(settings)

	program := tea.NewProgram(newModel(NewClient(url), dataDir, url))
	_, err = program.Run()
	if err != nil {
		return fmt.Errorf("running the interactive UI: %w", err)
	}
	return nil
}

// adminURL returns the address of fapi's admin API, such as
// http://127.0.0.1:3100.
func adminURL(settings config.Config) string {
	host := settings.ListenAddress
	// 0.0.0.0 means "listen on every address"; it can't be connected to.
	if host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(settings.AdminPort))
}
