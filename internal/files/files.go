// Package files saves files safely. Both the core package (mocks.json,
// requests.json) and the config package (config.json) use it.
package files

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteAtomic writes contents to a temporary file next to path, then
// renames it over path. A rename replaces the file in one step, so a crash
// can't leave a half-written file behind.
func WriteAtomic(path string, contents []byte) error {
	directory := filepath.Dir(path)
	err := os.MkdirAll(directory, 0o755)
	if err != nil {
		return fmt.Errorf("creating %s: %w", directory, err)
	}

	temporary, err := os.CreateTemp(directory, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("creating a temporary file: %w", err)
	}

	_, err = temporary.Write(contents)
	if err != nil {
		closeAndRemove(temporary)
		return fmt.Errorf("writing %s: %w", temporary.Name(), err)
	}

	err = temporary.Close()
	if err != nil {
		removeQuietly(temporary.Name())
		return fmt.Errorf("closing %s: %w", temporary.Name(), err)
	}

	err = os.Rename(temporary.Name(), path)
	if err != nil {
		removeQuietly(temporary.Name())
		return fmt.Errorf("replacing %s: %w", path, err)
	}
	return nil
}

// closeAndRemove cleans up after a failed write. The write's own error is
// the one worth reporting, so errors here are ignored.
func closeAndRemove(file *os.File) {
	_ = file.Close() // ignored: see above
	removeQuietly(file.Name())
}

// removeQuietly deletes a leftover temporary file. Failing to delete it is
// harmless, so the error is ignored.
func removeQuietly(path string) {
	_ = os.Remove(path) // ignored: see above
}
