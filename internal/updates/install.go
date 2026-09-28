package updates

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// How fapi was installed, which decides how it's updated.
const (
	InstallDeb     = "deb"     // the .deb, running as the fapi service
	InstallRPM     = "rpm"     // the .rpm, running as the fapi service
	InstallWindows = "windows" // the .exe from a release's .zip
	InstallDocker  = "docker"  // the Docker image: updated by updating the container
	InstallManual  = "manual"  // anything else, such as a build from source
)

// Files the .deb and .rpm install (see .goreleaser.yaml). install-type holds
// "deb" or "rpm", so fapi knows which kind of package it came from.
const (
	packagedExecutable = "/opt/fapi/fapi"
	installTypeFile    = "/opt/fapi/install-type"
	helperUnitFile     = "/usr/lib/systemd/system/fapi-update.path"
)

// Installation describes the running fapi: its edition, how it was
// installed, and whether it can update itself.
type Installation struct {
	Type    string `json:"type"`    // one of the Install constants
	Package string `json:"package"` // the edition's release name: fapi, fapi-web or fapi-cli
	OS      string `json:"os"`      // as Go names it, such as "linux" or "windows"
	Arch    string `json:"arch"`    // as Go names it, such as "amd64"
	WSL     bool   `json:"wsl"`     // Linux running in Windows Subsystem for Linux

	// CanInstall is true when fapi can download an update, install it and
	// restart by itself.
	CanInstall bool `json:"canInstall"`
	// CanDownload is true when fapi can at least download an update, for
	// the user to install with the command it shows.
	CanDownload bool `json:"canDownload"`
	// Why fapi can't install (or download) updates itself, worded for the
	// user; "" when it can.
	Reason string `json:"reason,omitempty"`

	executable string // the running executable, for replacing it on Windows
}

// Detect works out how the running fapi was installed. packageName is the
// edition's release name ("" for a build with neither UI, which isn't
// released), executable is the running executable's path, and allowed is the
// installUpdates setting.
func Detect(packageName string, executable string, allowed bool) Installation {
	installation := Installation{
		Type:       InstallManual,
		Package:    packageName,
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		WSL:        runningInWSL(),
		executable: executable,
	}

	if os.Getenv("FAPI_INSTALL") == InstallDocker { // set by packaging/Dockerfile
		installation.Type = InstallDocker
		installation.Reason = "fapi runs in a container. To get a new version, update the container."
		return installation
	}

	if packageName == "" || runtime.GOARCH != "amd64" || (runtime.GOOS != "linux" && runtime.GOOS != "windows") {
		installation.Reason = "fapi is only released for Linux and Windows on amd64, so update this copy the way you built it."
		return installation
	}

	switch runtime.GOOS {
	case "windows":
		installation.Type = InstallWindows
		installation.CanDownload = true
		installation.CanInstall = executable != "" && canWriteTo(filepath.Dir(executable))
		if !installation.CanInstall {
			installation.Reason = fmt.Sprintf("fapi can't change the folder it runs from, %s. Download the update, then unzip fapi.exe over the old one yourself.", filepath.Dir(executable))
		}
	case "linux":
		detectPackage(&installation)
	}

	if !allowed && installation.CanDownload {
		installation.CanInstall = false
		installation.CanDownload = false
		installation.Reason = "Installing updates is turned off in fapi's settings (features.installUpdates)."
	}
	return installation
}

// detectPackage fills in installation for a Linux fapi installed from a .deb
// or .rpm. It can install updates itself when it runs as the fapi service
// and the package's update helper (fapi-update.path) is there to run the
// installer as root.
func detectPackage(installation *Installation) {
	contents, err := os.ReadFile(installTypeFile)
	packaged := err == nil && installation.executable == packagedExecutable
	kind := strings.TrimSpace(string(contents))
	if !packaged || (kind != InstallDeb && kind != InstallRPM) {
		installation.Reason = "This copy of fapi wasn't installed from a .deb or .rpm, so update it the way you installed it."
		return
	}

	installation.Type = kind
	installation.CanDownload = true

	// systemd sets INVOCATION_ID for the services it runs.
	_, err = os.Stat(helperUnitFile)
	switch {
	case os.Getenv("INVOCATION_ID") == "":
		installation.Reason = "fapi isn't running as the fapi service, so it can't install updates itself. Download the update, then install it with the command shown."
	case err != nil:
		installation.Reason = "The fapi-update helper isn't installed, so fapi can't install updates itself. Download the update, then install it with the command shown."
	default:
		installation.CanInstall = true
	}
}

// AssetName returns the name of the release file with version of the
// edition packageName for an install type, such as
// "fapi-web_1.2.0_linux_amd64.deb", or "" if there's none. The names are set
// in .goreleaser.yaml.
func AssetName(packageName string, version string, installType string) string {
	switch installType {
	case InstallDeb, InstallRPM:
		return fmt.Sprintf("%s_%s_linux_amd64.%s", packageName, version, installType)
	case InstallWindows:
		return fmt.Sprintf("%s_%s_windows_amd64.zip", packageName, version)
	default:
		return ""
	}
}

// runningInWSL reports whether this is Linux in Windows Subsystem for Linux,
// whose kernel names itself after Microsoft.
func runningInWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	release, err := os.ReadFile("/proc/sys/kernel/osrelease")
	return err == nil && strings.Contains(strings.ToLower(string(release)), "microsoft")
}

// canWriteTo reports whether fapi can create files in folder.
func canWriteTo(folder string) bool {
	file, err := os.CreateTemp(folder, ".fapi-write-test-*")
	if err != nil {
		return false
	}
	name := file.Name()
	_ = file.Close()
	_ = os.Remove(name)
	return true
}
