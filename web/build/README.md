`npm run build` writes the built web UI into `ui/` in this folder, and `embed.go` compiles it into the fapi executable.

This file is committed so the folder always exists: without it, `go build` would fail on a fresh checkout where the web UI hasn't been built.
