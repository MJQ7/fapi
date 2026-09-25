// This file replaces tui_included.go in builds with the notui tag, such as
// the fapi-web edition. It doesn't import internal/cli, so Bubble Tea isn't
// compiled in.

//go:build notui

package main

import "errors"

// tuiIncluded reports whether this build contains the terminal UI.
const tuiIncluded = false

// tuiUsage is empty: there's no terminal UI to describe.
const tuiUsage = ""

// runTUI is never called when tuiIncluded is false; it exists so main.go
// compiles the same way in every edition.
func runTUI() error {
	return errors.New("this build of fapi doesn't include the terminal UI; use the fapi or fapi-cli edition for it")
}
