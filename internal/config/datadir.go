package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// DataDir returns the directory for the user's data (mocks.json, the server
// log and the saved request log), and for the optional config.json.
//
// In order: the --data-dir flag, the FAPI_DATA_DIR environment variable, or
// "fapi" in the operating system's per-user config directory:
// ~/.config/fapi on Linux, ~/Library/Application Support/fapi on macOS and
// %AppData%\fapi on Windows.
//
// It can't be set in the settings file, because that file may itself live in
// the data directory.
func DataDir(dataDirFlag string) (string, error) {
	if dataDirFlag != "" {
		return dataDirFlag, nil
	}

	fromEnvironment := os.Getenv("FAPI_DATA_DIR")
	if fromEnvironment != "" {
		return fromEnvironment, nil
	}

	userConfig, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("finding the user config directory (set --data-dir or FAPI_DATA_DIR instead): %w", err)
	}
	return filepath.Join(userConfig, "fapi"), nil
}
