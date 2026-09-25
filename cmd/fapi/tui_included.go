// The terminal UI is optional at build time: this file adds it, and
// tui_excluded.go replaces it in builds with the notui tag (the fapi-web
// edition), which then don't contain Bubble Tea at all.

//go:build !notui

package main

import "fapi/internal/cli"

// tuiIncluded reports whether this build contains the terminal UI.
const tuiIncluded = true

// tuiUsage is the terminal UI's part of `fapi help`.
const tuiUsage = `  fapi
      Open the interactive terminal UI: see fapi's status, set up proxies,
      override endpoints and watch requests arrive. It offers to
      start fapi in the background if it isn't running.

`

// runTUI runs the terminal UI until the user quits.
func runTUI() error {
	return cli.RunInteractive()
}
