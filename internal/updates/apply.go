package updates

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// releaseVersion matches the versions the helper installs: release numbers
// only, such as "1.2.3".
var releaseVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

// ApplyRequest is the root helper for the .deb and .rpm: `fapi apply-update`,
// run as root by fapi-update.service when fapi writes updates/request.json in
// the data folder dataDir (see packaging/fapi-update.path). It installs the
// update fapi downloaded there, and writes updates/result.json. The package's
// own scripts then restart the fapi service. current is the installed
// version and packageName the installed edition (this executable's).
//
// fapi runs as an unprivileged user and can write to the data folder, so
// nothing there is trusted: the request only names a version, which must be
// newer than current; the file name comes from the version; the file is
// checked against the release's checksums.txt, which the helper downloads
// from GitHub itself; and it's installed from a copy in a folder only root
// can write to, so it can't be changed after it's checked. os.Root stops
// links in the data folder leading anywhere outside it.
func ApplyRequest(ctx context.Context, dataDir string, current string, packageName string) error {
	root, err := os.OpenRoot(dataDir)
	if err != nil {
		return fmt.Errorf("opening the data folder: %w", err)
	}
	defer func() {
		_ = root.Close() // nothing useful to do if closing fails
	}()

	version, err := takeRequest(root)
	if errors.Is(err, os.ErrNotExist) {
		log.Printf("No update was requested")
		return nil
	}
	if err == nil {
		log.Printf("Installing fapi %s over %s", version, current)
		err = install(ctx, root, version, current, packageName)
	}

	outcome := Outcome{Version: version, OK: err == nil, Time: time.Now()}
	if err != nil {
		outcome.Error = sentence(err.Error())
		// The package may have stopped fapi before failing; start it again,
		// as whichever version is installed now.
		_ = exec.Command("systemctl", "start", "fapi").Run()
	}
	writeOutcome(root, outcome)
	if err == nil {
		log.Printf("Installed fapi %s", version)
	}
	return err
}

// takeRequest reads and deletes request.json, returning the version it asks
// for. Deleting it first stops fapi-update.path from starting the helper
// again for the same request.
func takeRequest(root *os.Root) (string, error) {
	name := path.Join(updatesFolder, requestFile)
	file, err := root.Open(name)
	if err != nil {
		return "", err
	}
	contents, readErr := io.ReadAll(io.LimitReader(file, 4096))
	_ = file.Close()
	err = root.Remove(name)
	if err != nil {
		return "", fmt.Errorf("deleting the request: %w", err)
	}
	if readErr != nil {
		return "", fmt.Errorf("reading the request: %w", readErr)
	}

	var request helperRequest
	err = json.Unmarshal(contents, &request)
	if err != nil {
		return "", fmt.Errorf("reading the request: %w", err)
	}
	return request.Version, nil
}

// install checks and installs the downloaded package for version.
func install(ctx context.Context, root *os.Root, version string, current string, packageName string) error {
	wanted, versionOK := parseVersion(version)
	installed, installedOK := parseVersion(current)
	switch {
	case !releaseVersion.MatchString(version) || !versionOK:
		return fmt.Errorf("%q isn't a release version", version)
	case !installedOK:
		return fmt.Errorf("the installed fapi is a development build (%s), so it can't be updated this way", current)
	case compare(wanted, installed) <= 0:
		return fmt.Errorf("fapi %s isn't newer than the installed %s", version, current)
	case packageName == "":
		return errors.New("this build of fapi isn't a released edition")
	}

	contents, err := os.ReadFile(installTypeFile)
	kind := strings.TrimSpace(string(contents))
	if err != nil || (kind != InstallDeb && kind != InstallRPM) {
		return fmt.Errorf("%s doesn't say whether fapi came from a .deb or an .rpm", installTypeFile)
	}

	name := AssetName(packageName, version, kind)
	client := &http.Client{Timeout: time.Minute}
	checksum, err := fetchChecksum(ctx, client, DownloadURL, version, name)
	if err != nil {
		return err
	}

	private, err := os.MkdirTemp("", "fapi-update-")
	if err != nil {
		return fmt.Errorf("making a folder for the update: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(private)
	}()

	packagePath := filepath.Join(private, name)
	err = copyChecked(root, path.Join(updatesFolder, name), packagePath, checksum)
	if err != nil {
		return err
	}
	_ = root.Remove(path.Join(updatesFolder, name)) // the copy is installed instead

	var command *exec.Cmd
	if kind == InstallDeb {
		command = exec.CommandContext(ctx, "dpkg", "--install", packagePath)
		command.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	} else {
		command = exec.CommandContext(ctx, "rpm", "--upgrade", packagePath)
	}
	output, err := command.CombinedOutput()
	log.Printf("%s:\n%s", strings.Join(command.Args, " "), output)
	if err != nil {
		return fmt.Errorf("%s failed (%v): %s", command.Args[0], err, lastLines(string(output), 5))
	}
	return nil
}

// copyChecked copies the file name in root to target, and checks its
// SHA-256 is checksum.
func copyChecked(root *os.Root, name string, target string, checksum []byte) error {
	source, err := root.Open(name)
	if err != nil {
		return fmt.Errorf("opening the downloaded update: %w", err)
	}
	defer func() {
		_ = source.Close() // only read from
	}()

	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("copying the downloaded update: %w", err)
	}
	hash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(source, maxDownloadBytes))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		return fmt.Errorf("copying the downloaded update: %w", errors.Join(copyErr, closeErr))
	}
	if !bytes.Equal(hash.Sum(nil), checksum) {
		return errors.New("the downloaded update doesn't match the checksum GitHub lists for it, so it wasn't installed. Try again")
	}
	return nil
}

// writeOutcome saves outcome as result.json, for fapi to show. It's written
// under another name and renamed, so fapi never reads half a file. If fapi
// has deleted the updates folder meanwhile, there's nowhere to write it; the
// folder isn't made here, because one made by root couldn't be emptied by
// fapi later.
func writeOutcome(root *os.Root, outcome Outcome) {
	contents, err := json.Marshal(outcome)
	if err != nil {
		return
	}
	name := path.Join(updatesFolder, resultFile)
	err = root.WriteFile(name+".tmp", contents, 0o644)
	if err == nil {
		err = root.Rename(name+".tmp", name)
	}
	if err != nil {
		log.Printf("Saving the result: %v", err)
	}
}

// lastLines returns the last count lines of text, on one line.
func lastLines(text string, count int) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) > count {
		lines = lines[len(lines)-count:]
	}
	return strings.Join(lines, " ")
}
