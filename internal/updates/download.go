package updates

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// DownloadURL is where release files are downloaded from, as
// DownloadURL/v1.2.3/<file name>. Only fapi's own releases are downloaded.
const DownloadURL = "https://github.com/MJQ7/fapi/releases/download"

// Limits on what's downloaded. A release file is a few megabytes.
const (
	maxDownloadBytes  = 200 << 20 // 200 MB
	maxChecksumsBytes = 1 << 20   // 1 MB
)

// checksumsFile lists each release file's SHA-256, one "<checksum>  <name>"
// line per file (see checksum in .goreleaser.yaml).
const checksumsFile = "checksums.txt"

// releaseFileURL returns where file name of release version is downloaded
// from, such as DownloadURL/v1.2.3/checksums.txt.
func releaseFileURL(baseURL string, version string, name string) string {
	return fmt.Sprintf("%s/v%s/%s", baseURL, version, name)
}

// fetchChecksum returns the SHA-256 checksum that release version's
// checksums.txt lists for file name. Its errors are worded for the user.
func fetchChecksum(ctx context.Context, client *http.Client, baseURL string, version string, name string) ([]byte, error) {
	response, err := get(ctx, client, releaseFileURL(baseURL, version, checksumsFile))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = response.Body.Close() // nothing useful to do if closing fails
	}()

	contents, err := io.ReadAll(io.LimitReader(response.Body, maxChecksumsBytes))
	if err != nil {
		return nil, fmt.Errorf("couldn't download fapi %s's checksums: %w", version, err)
	}
	return findChecksum(contents, name, version)
}

// findChecksum returns the checksum listed for file name in the contents of
// a checksums.txt.
func findChecksum(contents []byte, name string, version string) ([]byte, error) {
	scanner := bufio.NewScanner(bytes.NewReader(contents))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 || fields[1] != name {
			continue
		}
		checksum, err := hex.DecodeString(fields[0])
		if err != nil || len(checksum) != sha256.Size {
			return nil, fmt.Errorf("fapi %s's checksums.txt has an invalid checksum for %s", version, name)
		}
		return checksum, nil
	}
	return nil, fmt.Errorf("fapi %s has no %s to download", version, name)
}

// download saves the file at url to path, and returns its SHA-256 checksum.
// progress is called as it arrives, with the bytes so far and the total (0
// when the server doesn't say). Its errors are worded for the user.
func download(ctx context.Context, client *http.Client, url string, path string, progress func(done int64, total int64)) ([]byte, error) {
	response, err := get(ctx, client, url)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = response.Body.Close() // nothing useful to do if closing fails
	}()
	if response.ContentLength > maxDownloadBytes {
		return nil, fmt.Errorf("the update is %d MB, which is larger than fapi expects", response.ContentLength>>20)
	}

	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("couldn't save the update: %w", err)
	}

	hash := sha256.New()
	counter := &progressWriter{total: max(response.ContentLength, 0), report: progress}
	// One byte more than the limit, to tell a file of exactly the limit from
	// a larger one.
	written, copyErr := io.Copy(io.MultiWriter(file, hash, counter), io.LimitReader(response.Body, maxDownloadBytes+1))
	closeErr := file.Close()
	switch {
	case copyErr != nil:
		return nil, fmt.Errorf("the download stopped: %w", copyErr)
	case closeErr != nil:
		return nil, fmt.Errorf("couldn't save the update: %w", closeErr)
	case written > maxDownloadBytes:
		return nil, errors.New("the update is larger than fapi expects")
	}
	return hash.Sum(nil), nil
}

// get sends a GET request and returns the response if it's 200 OK. Its
// errors are worded for the user.
func get(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("couldn't download %s: %w", url, err)
	}
	// GitHub refuses requests without a User-Agent.
	request.Header.Set("User-Agent", "fapi")

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("couldn't reach GitHub to download the update: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return nil, fmt.Errorf("GitHub answered %s for %s", response.Status, url)
	}
	return response, nil
}

// progressWriter counts the bytes written through it, and reports them.
type progressWriter struct {
	done   int64
	total  int64
	report func(done int64, total int64)
}

func (w *progressWriter) Write(data []byte) (int, error) {
	w.done += int64(len(data))
	if w.report != nil {
		w.report(w.done, w.total)
	}
	return len(data), nil
}
