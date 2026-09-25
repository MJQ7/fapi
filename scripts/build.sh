#!/usr/bin/env bash
# Builds fapi from scratch, for this computer, into bin/.
#
#   scripts/build.sh              all three editions
#   scripts/build.sh fapi-cli     just the editions named (fapi, fapi-web, fapi-cli)
#
# Each run starts clean: it deletes the previous build and the web UI's
# installed packages, reinstalls those from package-lock.json (npm ci), builds
# the web UI and then the executables. Run it from anywhere; it works in the
# repository it's in. Needs Go, and Node.js 22 or later unless you only build
# fapi-cli.
#
# Works on Linux, macOS, WSL and Git Bash on Windows. It builds for the system
# it runs on: to build Windows .exe files, run it on Windows (Git Bash), and
# for Linux, in WSL or on Linux. For release files, use GoReleaser instead
# (see "Releasing" in README.md).

# Stop at the first failing command, including failures inside pipes, and
# treat unset variables as mistakes.
set -euo pipefail

# Work from the repository root, wherever the script was started.
cd "$(dirname "$0")/.."

editions=("$@")
if [ ${#editions[@]} -eq 0 ]; then
	editions=(fapi fapi-web fapi-cli)
fi

# The build tags that make each edition (see cmd/fapi/main.go).
tags_for() {
	case "$1" in
		fapi) echo "" ;;
		fapi-web) echo "notui" ;;
		fapi-cli) echo "nowebui" ;;
		*)
			echo "Unknown edition \"$1\". Choose from: fapi fapi-web fapi-cli" >&2
			exit 2
			;;
	esac
}

# Check the edition names before deleting anything.
needs_web_ui=false
for edition in "${editions[@]}"; do
	tags_for "$edition" >/dev/null
	if [ "$edition" != "fapi-cli" ]; then
		needs_web_ui=true
	fi
done

# The version fapi reports: the latest tag, plus the commit if there have been
# changes since, such as 0.1.0-3-g83dd064-dirty. "dev" outside a git checkout.
version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)
version=${version#v} # releases say 0.1.0, not v0.1.0

# ".exe" on Windows, nothing elsewhere.
exe_suffix=$(go env GOEXE)

echo "==> Cleaning"
rm -rf bin web/build/ui web/.svelte-kit

if [ "$needs_web_ui" = true ]; then
	echo "==> Installing the web UI's packages (npm ci)"
	npm --prefix web ci

	echo "==> Building the web UI"
	npm --prefix web run build
fi

mkdir -p bin
for edition in "${editions[@]}"; do
	output="bin/${edition}${exe_suffix}"
	echo "==> Building $output ($version)"
	# CGO_ENABLED=0 builds without C code, as the releases do.
	CGO_ENABLED=0 go build \
		-tags "$(tags_for "$edition")" \
		-ldflags "-s -w -X main.version=${version}" \
		-o "$output" \
		./cmd/fapi
done

echo
echo "Done:"
for edition in "${editions[@]}"; do
	output="bin/${edition}${exe_suffix}"
	echo "  $output: $("./$output" --version)"
done
