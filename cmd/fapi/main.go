// Command fapi is an API proxy for testing how a frontend handles API errors.
// It forwards requests to the real API, except for endpoints the user
// overrides with a response of their choice.
//
// This file reads the command line and starts the right thing. `fapi serve`
// (in serve.go) runs everything: the proxy and mock servers, the admin API and
// the web UI. `fapi` with no command opens the interactive terminal UI (internal/cli).
//
// Two build tags leave a UI out, to make the release editions:
//
//	go build ./cmd/fapi                  fapi: web UI and terminal UI
//	go build -tags notui ./cmd/fapi      fapi-web: web UI only
//	go build -tags nowebui ./cmd/fapi    fapi-cli: terminal UI only
//
// Every edition has the mock servers and the admin API. See tui_included.go,
// tui_excluded.go and web/embed.go.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"fapi/web"
)

// version is replaced at build time by the release tooling, with
// -ldflags "-X main.version=1.2.3". A plain `go build` leaves it as "dev".
// It's the only package-level variable, and nothing changes it at runtime.
var version = "dev"

const usage = `fapi: an API proxy for testing how a frontend handles API errors.

Usage:
` + tuiUsage + `  fapi serve [--config <file>] [--data-dir <folder>]
      Run fapi in the foreground: the proxy, the admin API and, in
      editions that have it, the web UI (http://127.0.0.1:3100 by default).
      Stop it with Ctrl+C.

  fapi help, fapi --help    Show this help
  fapi --version            Show the version

Options for serve:
  --config <file>        Settings file overriding the built-in defaults.
                         Default: $FAPI_CONFIG, or config.json in the data folder.
  --data-dir <folder>    Where endpoints and logs are saved.
                         Default: $FAPI_DATA_DIR, or "fapi" in your user config folder.

Settings are described in docs/configuration.md.
`

func main() {
	os.Exit(run(os.Args[1:]))
}

// run carries out the command in arguments and returns the exit code:
// 0 for success, 1 for failure and 2 for a mistake in the command line.
func run(arguments []string) int {
	if len(arguments) == 0 {
		// Without a terminal UI in this edition, or when the output is a pipe
		// or a file (which can't be interactive), show the usage.
		if !tuiIncluded || !isTerminal() {
			printUsage(os.Stdout)
			return 0
		}
		err := runTUI()
		if err != nil {
			fmt.Fprintf(os.Stderr, "fapi: %v\n", err)
			return 1
		}
		return 0
	}

	command := arguments[0]
	switch command {
	case "serve":
		err := serve(arguments[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "fapi: %v\n", err)
			return 1
		}
		return 0
	case "help", "--help", "-h", "-help":
		printUsage(os.Stdout)
		return 0
	case "--version", "-version":
		fmt.Printf("fapi %s (%s)\n", version, edition())
		return 0
	default:
		fmt.Fprintf(os.Stderr, "fapi: unknown command %q\n\n", command)
		printUsage(os.Stderr)
		return 2
	}
}

func printUsage(output io.Writer) {
	// Printing help can only fail if the terminal has gone away, and then
	// there's nowhere left to report it, so the error is ignored.
	_, _ = fmt.Fprint(output, usage)
}

// edition names what this build includes, such as "web UI and terminal UI".
func edition() string {
	var parts []string
	if web.Included {
		parts = append(parts, "web UI")
	}
	if tuiIncluded {
		parts = append(parts, "terminal UI")
	}
	if len(parts) == 0 {
		return "no UI"
	}
	return strings.Join(parts, " and ")
}

// isTerminal reports whether fapi is talking to a person in a terminal,
// rather than to a pipe or a file.
func isTerminal() bool {
	return isCharacterDevice(os.Stdin) && isCharacterDevice(os.Stdout)
}

// isCharacterDevice reports whether file is a terminal (a "character device"
// to the operating system).
func isCharacterDevice(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
