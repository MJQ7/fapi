package updates

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Files in the updates folder, inside the data folder. The folder is emptied
// when an update starts.
const (
	updatesFolder = "updates"
	// requestFile asks the .deb or .rpm's root helper to install an update
	// (see apply.go and packaging/fapi-update.path).
	requestFile = "request.json"
	// resultFile says how the last install went. The helper writes it, or
	// fapi itself on Windows.
	resultFile = "result.json"
)

// How long fapi waits for the helper: to take the request (it starts within
// a second when it's on), then to install the package.
const (
	helperStartTimeout   = 30 * time.Second
	helperInstallTimeout = 10 * time.Minute
)

// What an update is doing (Job.State).
const (
	StateIdle        = "idle"
	StateDownloading = "downloading"
	StateInstalling  = "installing"
	StateRestarting  = "restarting" // installed; fapi is restarting as the new version
	StateDownloaded  = "downloaded" // saved for the user to install, with Job.Command
	StateFailed      = "failed"
)

// ErrBusy is returned by Start while an update is already running.
var ErrBusy = errors.New("an update is already being installed")

// Job is an update's progress, for the web UI.
type Job struct {
	State      string `json:"state"` // one of the State constants
	Version    string `json:"version,omitempty"`
	Downloaded int64  `json:"downloaded"` // bytes so far
	Total      int64  `json:"total"`      // bytes in all, or 0 when unknown
	File       string `json:"file,omitempty"`
	// Command installs the downloaded File, for updates fapi can't install
	// itself, such as "sudo apt install /var/lib/fapi/updates/fapi_1.2.0_linux_amd64.deb".
	Command string `json:"command,omitempty"`
	Error   string `json:"error,omitempty"` // why it failed, worded for the user

	// Last is how the last install went, when no update is running and
	// there's been one since the updates folder was last emptied.
	Last *Outcome `json:"last,omitempty"`
}

// Outcome is how an install went, as saved in result.json.
type Outcome struct {
	Version string    `json:"version"`
	OK      bool      `json:"ok"`
	Error   string    `json:"error,omitempty"`
	Time    time.Time `json:"time"`
}

// helperRequest is the contents of request.json. The helper runs as root and
// fapi doesn't, so it only trusts the version: it works out the file name,
// and checks the file against GitHub's checksums itself.
type helperRequest struct {
	Version string `json:"version"`
}

// Installer downloads and installs updates, one at a time.
type Installer struct {
	checker *Checker
	dataDir string
	restart func() // stops fapi and starts the (new) executable again
	baseURL string // DownloadURL, or a test server
	client  *http.Client

	mutex sync.Mutex
	job   Job
}

// NewInstaller returns an installer for the releases checker finds. Updates
// are downloaded into the data folder dataDir. restart is called once fapi.exe
// has been replaced on Windows.
func NewInstaller(checker *Checker, dataDir string, restart func()) *Installer {
	return &Installer{
		checker: checker,
		dataDir: dataDir,
		restart: restart,
		baseURL: DownloadURL,
		client:  &http.Client{Timeout: 10 * time.Minute},
		job:     Job{State: StateIdle},
	}
}

// Status returns the running update's progress, or how the last one went.
func (i *Installer) Status() Job {
	i.mutex.Lock()
	job := i.job
	i.mutex.Unlock()

	if job.State == StateIdle {
		job.Last = i.readOutcome()
	}
	return job
}

// Start starts updating to version in the background: downloading it, then
// installing it if fapi can, and restarting. version must be the newest
// release, as the last check found it. Its errors are worded for the user.
func (i *Installer) Start(ctx context.Context, version string) (Job, error) {
	check := i.checker.Check(ctx, false)
	installation := check.Install
	switch {
	case !installation.CanDownload:
		return Job{}, errors.New(installation.Reason)
	case !check.UpdateAvailable || check.Latest != version:
		return Job{}, fmt.Errorf("fapi %s isn't a newer release than this one. Check for updates again", version)
	}

	i.mutex.Lock()
	defer i.mutex.Unlock()
	switch i.job.State {
	case StateDownloading, StateInstalling, StateRestarting:
		return Job{}, ErrBusy
	}
	i.job = Job{State: StateDownloading, Version: version}

	go i.run(installation, version)
	return i.job, nil
}

// run downloads and installs version, keeping i.job up to date.
func (i *Installer) run(installation Installation, version string) {
	log.Printf("Updates: downloading fapi %s", version)
	path, err := i.download(installation, version)
	if err != nil {
		i.fail(err)
		return
	}

	if !installation.CanInstall {
		log.Printf("Updates: downloaded fapi %s to %s, for installing by hand", version, path)
		i.update(func(job *Job) {
			job.State = StateDownloaded
			job.File = path
			job.Command = installCommand(installation.Type, path)
		})
		return
	}

	i.update(func(job *Job) { job.State = StateInstalling })
	if installation.Type == InstallWindows {
		err = i.installWindows(installation, version, path)
	} else {
		err = i.askHelper(version)
	}
	if err != nil {
		i.fail(err)
	}
}

// download downloads version's release file for installation into an empty
// updates folder, checks it against the release's checksums, and returns
// where it is.
func (i *Installer) download(installation Installation, version string) (string, error) {
	// Absolute, since the path is shown to the user.
	folder, err := filepath.Abs(filepath.Join(i.dataDir, updatesFolder))
	if err == nil {
		err = os.RemoveAll(folder)
	}
	if err == nil {
		err = os.MkdirAll(folder, 0o755)
	}
	if err != nil {
		return "", fmt.Errorf("couldn't empty the updates folder: %w", err)
	}

	ctx := context.Background()
	name := AssetName(installation.Package, version, installation.Type)
	checksum, err := fetchChecksum(ctx, i.client, i.baseURL, version, name)
	if err != nil {
		return "", err
	}

	path := filepath.Join(folder, name)
	partial := path + ".part"
	got, err := download(ctx, i.client, releaseFileURL(i.baseURL, version, name), partial, func(done int64, total int64) {
		i.update(func(job *Job) { job.Downloaded, job.Total = done, total })
	})
	if err != nil {
		return "", err
	}
	if !bytes.Equal(got, checksum) {
		_ = os.Remove(partial)
		return "", fmt.Errorf("the downloaded %s doesn't match the checksum GitHub lists for it, so fapi deleted it. Try again", name)
	}
	err = os.Rename(partial, path)
	if err != nil {
		return "", fmt.Errorf("couldn't save the update: %w", err)
	}
	return path, nil
}

// installWindows replaces the running fapi.exe with the one in the
// downloaded zip, then restarts fapi. Windows won't let a running executable
// be changed or deleted, but it can be renamed: it becomes fapi.exe.old,
// which the new fapi deletes when it starts (see CleanUp).
func (i *Installer) installWindows(installation Installation, version string, zipPath string) error {
	executable := installation.executable
	replacement := executable + ".new"
	err := extractExecutable(zipPath, replacement)
	if err != nil {
		return err
	}

	old := executable + ".old"
	_ = os.Remove(old) // left by an earlier update, if CleanUp couldn't delete it
	err = os.Rename(executable, old)
	if err != nil {
		_ = os.Remove(replacement)
		return fmt.Errorf("couldn't replace %s: %w", executable, err)
	}
	err = os.Rename(replacement, executable)
	if err != nil {
		_ = os.Rename(old, executable) // put the running version back
		return fmt.Errorf("couldn't replace %s: %w", executable, err)
	}

	log.Printf("Updates: replaced %s with fapi %s; restarting", executable, version)
	_ = os.Remove(zipPath) // no longer needed
	i.saveOutcome(Outcome{Version: version, OK: true, Time: time.Now()})
	i.update(func(job *Job) { job.State = StateRestarting })
	// A moment for the web UI to hear it's restarting.
	time.AfterFunc(time.Second, i.restart)
	return nil
}

// extractExecutable copies fapi.exe out of a release's zip to path.
func extractExecutable(zipPath string, path string) error {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("couldn't open the downloaded zip: %w", err)
	}
	defer func() {
		_ = archive.Close() // only read from
	}()

	for _, entry := range archive.File {
		if entry.Name != "fapi.exe" {
			continue
		}
		source, err := entry.Open()
		if err != nil {
			return fmt.Errorf("couldn't read fapi.exe from the downloaded zip: %w", err)
		}
		defer func() {
			_ = source.Close() // only read from
		}()

		target, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("couldn't save the new fapi.exe: %w", err)
		}
		_, copyErr := io.Copy(target, io.LimitReader(source, maxDownloadBytes))
		closeErr := target.Close()
		if copyErr != nil || closeErr != nil {
			_ = os.Remove(path)
			return fmt.Errorf("couldn't save the new fapi.exe: %w", errors.Join(copyErr, closeErr))
		}
		return nil
	}
	return errors.New("the downloaded zip has no fapi.exe")
}

// askHelper asks the package's root helper to install version, and waits
// for it. When it works, the package restarts the fapi service, which stops
// this process, so this only returns when something went wrong.
func (i *Installer) askHelper(version string) error {
	folder := filepath.Join(i.dataDir, updatesFolder)
	request := filepath.Join(folder, requestFile)
	contents, err := json.Marshal(helperRequest{Version: version})
	if err != nil {
		return err
	}
	// Written under another name and renamed, so the helper never sees half a file.
	err = os.WriteFile(request+".tmp", contents, 0o644)
	if err == nil {
		err = os.Rename(request+".tmp", request)
	}
	if err != nil {
		return fmt.Errorf("couldn't ask the fapi-update helper to install the update: %w", err)
	}
	log.Printf("Updates: asked the fapi-update helper to install fapi %s", version)

	// The helper deletes the request as soon as it starts.
	if !waitFor(helperStartTimeout, func() bool { return !exists(request) }) {
		_ = os.Remove(request)
		return errors.New("the fapi-update helper didn't start. Check it's on with: sudo systemctl enable --now fapi-update.path")
	}

	var outcome *Outcome
	waitFor(helperInstallTimeout, func() bool {
		outcome = i.readOutcome()
		return outcome != nil
	})
	switch {
	case outcome == nil:
		return errors.New("the update is taking longer than expected. See what the helper is doing with: journalctl -u fapi-update")
	case !outcome.OK:
		return errors.New(outcome.Error)
	}
	// Installed, but the package didn't restart fapi; ask to be restarted.
	i.update(func(job *Job) { job.State = StateRestarting })
	return nil
}

// readOutcome returns how the last install went, or nil if there's no
// result.json.
func (i *Installer) readOutcome() *Outcome {
	contents, err := os.ReadFile(filepath.Join(i.dataDir, updatesFolder, resultFile))
	if err != nil {
		return nil
	}
	var outcome Outcome
	if json.Unmarshal(contents, &outcome) != nil {
		return nil
	}
	return &outcome
}

// saveOutcome writes result.json.
func (i *Installer) saveOutcome(outcome Outcome) {
	contents, err := json.Marshal(outcome)
	if err == nil {
		err = os.WriteFile(filepath.Join(i.dataDir, updatesFolder, resultFile), contents, 0o644)
	}
	if err != nil {
		log.Printf("Updates: saving how the update went: %v", err)
	}
}

func (i *Installer) update(change func(job *Job)) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	change(&i.job)
}

func (i *Installer) fail(err error) {
	log.Printf("Updates: %v", err)
	i.update(func(job *Job) {
		job.State = StateFailed
		job.Error = sentence(err.Error())
	})
}

// installCommand returns the command that installs a downloaded package, or
// "" for a Windows zip, which is unzipped instead.
func installCommand(installType string, path string) string {
	if strings.ContainsAny(path, " '\"$") {
		path = "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
	}
	switch installType {
	case InstallDeb:
		return "sudo apt install " + path
	case InstallRPM:
		return "sudo dnf install " + path
	default:
		return ""
	}
}

// CleanUp deletes what an earlier update left beside executable: on Windows,
// the replaced fapi.exe.old. fapi calls it when it starts. Right after an
// update, the old fapi may still be exiting, and Windows won't delete an
// executable that's running, so it keeps trying for a few seconds, in the
// background.
func CleanUp(executable string) {
	old := executable + ".old"
	if !exists(old) {
		return
	}
	go func() {
		var err error
		gone := waitFor(10*time.Second, func() bool {
			err = os.Remove(old)
			return err == nil || errors.Is(err, os.ErrNotExist)
		})
		if !gone {
			log.Printf("Updates: couldn't delete the replaced %s: %v", old, err)
		}
	}()
}

// Relaunch starts executable with arguments as a new process of its own,
// which keeps running after this one exits. FAPI_RESTARTED tells it to wait
// for this process to free the admin port (see cmd/fapi/serve.go).
func Relaunch(executable string, arguments []string) error {
	command := exec.Command(executable, arguments...)
	command.Env = append(os.Environ(), "FAPI_RESTARTED=1")
	detach(command)
	err := command.Start()
	if err != nil {
		return fmt.Errorf("starting the new fapi: %w", err)
	}
	return command.Process.Release()
}

// waitFor checks done every half second until it's true or timeout passes,
// and returns whether it came true.
func waitFor(timeout time.Duration, done func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if done() {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return done()
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// sentence makes an error message a sentence, starting with a capital
// letter and ending with a full stop, since Go's errors don't.
func sentence(message string) string {
	if message == "" {
		return message
	}
	message = strings.ToUpper(message[:1]) + message[1:]
	if !strings.HasSuffix(message, ".") {
		message += "."
	}
	return message
}
