//go:build !nowebui

// Package web compiles the built web UI into the fapi executable, so fapi can
// serve it without any other files. The rest of this folder is the SvelteKit
// project the UI is built from (see README.md here).
//
// This Go file has to live in this folder because //go:embed can only include
// files in the same folder as the Go file, or below it.
//
// Builds with the nowebui tag (`go build -tags nowebui`) leave the web UI out
// and use embed_excluded.go instead; the build line above says so.
package web

import (
	"embed"
	"io/fs"
)

// Included reports whether this build of fapi contains the web UI.
const Included = true

// The "all:" prefix includes every file under build/, even ones whose names
// start with "_" or ".", which SvelteKit's output uses (such as _app/).
//
//go:embed all:build
var buildFolder embed.FS

// Files returns the built web UI, and false if it hasn't been built (only
// build/README.md is there), so fapi can show how to build it instead.
func Files() (fs.FS, bool) {
	ui, err := fs.Sub(buildFolder, "build/ui")
	if err != nil {
		// fs.Sub only fails for an invalid path, and "build/ui" is valid.
		return nil, false
	}

	_, err = fs.Stat(ui, "index.html")
	if err != nil {
		return nil, false
	}
	return ui, true
}
