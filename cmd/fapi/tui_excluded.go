// This file replaces tui_included.go in builds with the notui tag, such as
// the fapi-web edition. It doesn't import internal/cli, so Bubble Tea isn't
// compiled in.

//go:build notui

package main

// tuiIncluded reports whether this build contains the terminal UI.
const tuiIncluded = false

// tuiUsage is empty: there's no terminal UI to describe.
const tuiUsage = ""

// runTUI exists so main.go compiles the same way in every edition. main.go
// checks tuiIncluded first, so it's never called.
func runTUI() error {
	panic("runTUI called in a build without the terminal UI")
}
