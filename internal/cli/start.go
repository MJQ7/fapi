package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// startTimeout is how long to wait for a newly started fapi to answer.
const startTimeout = 10 * time.Second

// startInBackground runs `fapi serve` as a separate process that keeps
// running after this one exits, then waits until its admin API answers.
// If it doesn't, the error includes the end of the server log, which says why.
func startInBackground(client *Client, dataDir string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding the fapi executable: %w", err)
	}

	// serve writes its own log to fapi.log in the data folder, so its
	// terminal output isn't needed. Leaving Stdout and Stderr nil sends it
	// nowhere.
	command := exec.Command(executable, "serve", "--data-dir", dataDir)
	detach(command)

	err = command.Start()
	if err != nil {
		return fmt.Errorf("starting fapi: %w", err)
	}

	// This process won't wait for fapi to finish, so let the operating
	// system forget about the relationship between them.
	err = command.Process.Release()
	if err != nil {
		return fmt.Errorf("starting fapi: %w", err)
	}

	deadline := time.Now().Add(startTimeout)
	for time.Now().Before(deadline) {
		_, err = client.Status()
		if err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("fapi didn't start within %s. The end of its log:\n\n%s", startTimeout, logTail(dataDir, 15))
}

// logTail returns the last lines of the server log, or a note if it can't
// be read.
func logTail(dataDir string, lineCount int) string {
	path := filepath.Join(dataDir, "fapi.log")
	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("(could not read %s: %v)", path, err)
	}

	lines := strings.Split(strings.TrimRight(string(contents), "\n"), "\n")
	if len(lines) > lineCount {
		lines = lines[len(lines)-lineCount:]
	}
	return strings.Join(lines, "\n")
}
