// This file replaces embed.go in builds made with the nowebui tag
// (`go build -tags nowebui`), such as the fapi-cli release. Those builds
// leave the web UI out, so they don't need Node.js to build and are smaller.

//go:build nowebui

package web

import "io/fs"

// Included reports whether this build of fapi contains the web UI.
const Included = false

// Files returns false: this build has no web UI.
func Files() (fs.FS, bool) {
	return nil, false
}
