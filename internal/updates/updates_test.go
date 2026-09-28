package updates

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestAssetName(t *testing.T) {
	cases := map[string]string{
		InstallDeb:     "fapi-web_1.2.0_linux_amd64.deb",
		InstallRPM:     "fapi-web_1.2.0_linux_amd64.rpm",
		InstallWindows: "fapi-web_1.2.0_windows_amd64.zip",
		InstallDocker:  "",
		InstallManual:  "",
	}
	for installType, want := range cases {
		got := AssetName("fapi-web", "1.2.0", installType)
		if got != want {
			t.Errorf("AssetName(%s) = %q, want %q", installType, got, want)
		}
	}
}

func TestFindChecksum(t *testing.T) {
	sum := sha256.Sum256([]byte("contents"))
	listing := fmt.Sprintf("%x  fapi_1.2.0_linux_amd64.deb\n%x  fapi_1.2.0_linux_amd64.rpm\n", sha256.Sum256(nil), sum)

	got, err := findChecksum([]byte(listing), "fapi_1.2.0_linux_amd64.rpm", "1.2.0")
	if err != nil || !bytes.Equal(got, sum[:]) {
		t.Fatalf("findChecksum = %x, %v; want %x", got, err, sum)
	}
	_, err = findChecksum([]byte(listing), "fapi_1.2.0_windows_amd64.zip", "1.2.0")
	if err == nil {
		t.Fatal("findChecksum found a file that isn't listed")
	}
	_, err = findChecksum([]byte("nothex  a.deb\n"), "a.deb", "1.2.0")
	if err == nil {
		t.Fatal("findChecksum accepted an invalid checksum")
	}
}

// fakeGitHub serves a release list with version, and its files.
func fakeGitHub(t *testing.T, version string, files map[string][]byte, checksums map[string][]byte) *httptest.Server {
	t.Helper()
	var listing strings.Builder
	for name, contents := range files {
		sum := checksums[name]
		if sum == nil {
			computed := sha256.Sum256(contents)
			sum = computed[:]
		}
		fmt.Fprintf(&listing, "%s  %s\n", hex.EncodeToString(sum), name)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /releases", func(writer http.ResponseWriter, request *http.Request) {
		_ = json.NewEncoder(writer).Encode([]release{{TagName: "v" + version, HTMLURL: "https://example.com"}})
	})
	mux.HandleFunc("GET /download/v"+version+"/{name}", func(writer http.ResponseWriter, request *http.Request) {
		name := request.PathValue("name")
		if name == checksumsFile {
			_, _ = writer.Write([]byte(listing.String()))
			return
		}
		contents, ok := files[name]
		if !ok {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write(contents)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// newTestInstaller returns an installer using server, for installation.
func newTestInstaller(t *testing.T, server *httptest.Server, installation Installation, restart func()) (*Installer, string) {
	t.Helper()
	dataDir := t.TempDir()
	checker := NewChecker(server.URL+"/releases", "1.1.0", func() Installation { return installation })
	installer := NewInstaller(checker, dataDir, restart)
	installer.baseURL = server.URL + "/download"
	return installer, dataDir
}

// waitForState waits until the installer's job leaves the running states.
func waitForState(t *testing.T, installer *Installer) Job {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		job := installer.Status()
		if job.State != StateDownloading && job.State != StateInstalling {
			return job
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("the update didn't finish")
	return Job{}
}

func zipWithExecutable(t *testing.T, contents []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for name, data := range map[string][]byte{"LICENSE": []byte("MIT"), "fapi.exe": contents} {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = entry.Write(data)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestInstallWindows(t *testing.T) {
	folder := t.TempDir()
	executable := filepath.Join(folder, "fapi.exe")
	if err := os.WriteFile(executable, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	name := "fapi_1.2.0_windows_amd64.zip"
	server := fakeGitHub(t, "1.2.0", map[string][]byte{name: zipWithExecutable(t, []byte("new"))}, nil)
	var restarted atomic.Bool
	installation := Installation{Type: InstallWindows, Package: "fapi", CanInstall: true, CanDownload: true, executable: executable}
	installer, _ := newTestInstaller(t, server, installation, func() { restarted.Store(true) })

	_, err := installer.Start(context.Background(), "1.2.0")
	if err != nil {
		t.Fatal(err)
	}
	job := waitForState(t, installer)
	if job.State != StateRestarting {
		t.Fatalf("state %q (%s), want restarting", job.State, job.Error)
	}

	for path, want := range map[string]string{executable: "new", executable + ".old": "old"} {
		got, _ := os.ReadFile(path)
		if string(got) != want {
			t.Errorf("%s holds %q, want %q", path, got, want)
		}
	}
	outcome := installer.readOutcome()
	if outcome == nil || !outcome.OK || outcome.Version != "1.2.0" {
		t.Errorf("result.json = %+v, want 1.2.0 OK", outcome)
	}
	deadline := time.Now().Add(3 * time.Second)
	for !restarted.Load() && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if !restarted.Load() {
		t.Error("fapi wasn't restarted")
	}
}

func TestDownloadOnly(t *testing.T) {
	name := "fapi-web_1.2.0_linux_amd64.deb"
	server := fakeGitHub(t, "1.2.0", map[string][]byte{name: []byte("package")}, nil)
	installation := Installation{Type: InstallDeb, Package: "fapi-web", CanDownload: true}
	installer, dataDir := newTestInstaller(t, server, installation, nil)

	_, err := installer.Start(context.Background(), "1.2.0")
	if err != nil {
		t.Fatal(err)
	}
	job := waitForState(t, installer)
	path := filepath.Join(dataDir, updatesFolder, name)
	if job.State != StateDownloaded || job.File != path || job.Command != "sudo apt install "+path {
		t.Fatalf("job = %+v", job)
	}
	if job.Downloaded != int64(len("package")) {
		t.Errorf("downloaded %d bytes, want %d", job.Downloaded, len("package"))
	}
}

func TestChecksumMismatch(t *testing.T) {
	name := "fapi_1.2.0_linux_amd64.rpm"
	wrong := sha256.Sum256([]byte("something else"))
	server := fakeGitHub(t, "1.2.0", map[string][]byte{name: []byte("package")}, map[string][]byte{name: wrong[:]})
	installation := Installation{Type: InstallRPM, Package: "fapi", CanDownload: true}
	installer, dataDir := newTestInstaller(t, server, installation, nil)

	_, err := installer.Start(context.Background(), "1.2.0")
	if err != nil {
		t.Fatal(err)
	}
	job := waitForState(t, installer)
	if job.State != StateFailed || !strings.Contains(job.Error, "checksum") {
		t.Fatalf("job = %+v, want a checksum failure", job)
	}
	if exists(filepath.Join(dataDir, updatesFolder, name)) {
		t.Error("the file that failed its check was kept")
	}
}

func TestStartRefusals(t *testing.T) {
	server := fakeGitHub(t, "1.2.0", nil, nil)

	installer, _ := newTestInstaller(t, server, Installation{Type: InstallDocker, Reason: "a container"}, nil)
	_, err := installer.Start(context.Background(), "1.2.0")
	if err == nil || err.Error() != "a container" {
		t.Errorf("Docker: err = %v, want the reason", err)
	}

	installer, _ = newTestInstaller(t, server, Installation{Type: InstallDeb, Package: "fapi", CanDownload: true}, nil)
	_, err = installer.Start(context.Background(), "1.3.0")
	if err == nil {
		t.Error("started a version that isn't the latest")
	}
}

func TestTakeRequest(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, updatesFolder), 0o755); err != nil {
		t.Fatal(err)
	}
	request := filepath.Join(dataDir, updatesFolder, requestFile)
	if err := os.WriteFile(request, []byte(`{"version":"1.2.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	version, err := takeRequest(root)
	if err != nil || version != "1.2.0" {
		t.Fatalf("takeRequest = %q, %v", version, err)
	}
	if exists(request) {
		t.Error("the request wasn't deleted")
	}

	for _, bad := range []struct{ version, current string }{
		{"1.2.0; rm -rf /", "1.1.0"},
		{"../../1.2.0", "1.1.0"},
		{"1.1.0", "1.1.0"},
		{"1.0.9", "1.1.0"},
		{"1.2.0", "dev"},
	} {
		err := install(context.Background(), root, bad.version, bad.current, "fapi")
		if err == nil {
			t.Errorf("install accepted %q over %q", bad.version, bad.current)
		}
	}
}

// The helper runs as root, so links in the data folder mustn't lead it
// anywhere else.
func TestTakeRequestStaysInDataFolder(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, requestFile), []byte(`{"version":"1.2.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	dataDir := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dataDir, updatesFolder)); err != nil {
		t.Skipf("can't make links here: %v", err)
	}
	root, err := os.OpenRoot(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = takeRequest(root)
	if err == nil {
		t.Fatal("takeRequest followed a link out of the data folder")
	}
	if !exists(filepath.Join(outside, requestFile)) {
		t.Error("takeRequest deleted a file outside the data folder")
	}
}
