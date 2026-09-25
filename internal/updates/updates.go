// Package updates checks GitHub for a newer release of fapi. It only asks
// GitHub when a UI asks for the check (the web UI's Settings screen), never
// in the background, and remembers the answer for an hour so opening the
// screen again doesn't ask again.
package updates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ReleasesURL lists fapi's releases, newest first. Pre-releases and drafts
// in the list are skipped: only normal releases are offered as updates.
const ReleasesURL = "https://api.github.com/repos/MJQ7/fapi/releases?per_page=30"

// How long an answer is reused before GitHub is asked again. Failed checks
// are retried sooner. Without a login, GitHub allows 60 requests an hour.
const (
	cacheFor      = time.Hour
	cacheFailures = 5 * time.Minute
)

// Result is the answer to "is there a newer fapi?".
type Result struct {
	Current string `json:"current"` // the running version, such as "1.2.3" or "dev"

	// The newest release on GitHub, or "" when the check failed or there
	// are no releases yet.
	Latest      string `json:"latest"`
	LatestURL   string `json:"latestUrl"`   // the release's GitHub page
	PublishedAt string `json:"publishedAt"` // when it was released, in RFC 3339

	UpdateAvailable bool `json:"updateAvailable"`

	// DevelopmentBuild is true when the running version isn't a release
	// number (a plain `go build` reports "dev"), so it can't be compared.
	DevelopmentBuild bool `json:"developmentBuild"`

	CheckedAt time.Time `json:"checkedAt"`
	Error     string    `json:"error,omitempty"` // why the check failed, worded for the user
}

// Checker checks for updates, remembering the last answer.
type Checker struct {
	url     string
	current string
	client  *http.Client

	mutex sync.Mutex
	last  *Result // nil until the first check
}

// NewChecker returns a checker for the releases listed at url (normally
// ReleasesURL), comparing them with the running version current.
func NewChecker(url string, current string) *Checker {
	return &Checker{
		url:     url,
		current: current,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// Check returns whether there's a newer release, asking GitHub unless it
// answered recently. force asks GitHub regardless.
func (c *Checker) Check(ctx context.Context, force bool) Result {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.last != nil && !force {
		age := time.Since(c.last.CheckedAt)
		if age < cacheFor && (c.last.Error == "" || age < cacheFailures) {
			return *c.last
		}
	}

	result := c.ask(ctx)
	c.last = &result
	return result
}

// release is the part of GitHub's release JSON fapi reads.
type release struct {
	TagName     string `json:"tag_name"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
}

// ask asks GitHub for the releases and compares the newest with the running
// version.
func (c *Checker) ask(ctx context.Context) Result {
	result := Result{Current: c.current, CheckedAt: time.Now()}

	releases, err := c.fetchReleases(ctx)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	var newest *release
	var newestVersion [3]int
	for index := range releases {
		candidate := &releases[index]
		version, ok := parseVersion(candidate.TagName)
		if candidate.Draft || candidate.Prerelease || !ok {
			continue
		}
		// Releases come newest first, so on a tie the newer one is kept.
		if newest == nil || compare(version, newestVersion) > 0 {
			newest = candidate
			newestVersion = version
		}
	}
	if newest == nil {
		return result // no releases yet
	}

	result.Latest = strings.TrimPrefix(newest.TagName, "v")
	result.LatestURL = newest.HTMLURL
	result.PublishedAt = newest.PublishedAt

	current, ok := parseVersion(c.current)
	if !ok {
		result.DevelopmentBuild = true
		return result
	}
	result.UpdateAvailable = compare(newestVersion, current) > 0
	return result
}

// fetchReleases downloads the list of releases. Its errors are worded for
// the user.
func (c *Checker) fetchReleases(ctx context.Context) ([]release, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, fmt.Errorf("couldn't check for updates: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	// GitHub refuses requests without a User-Agent.
	request.Header.Set("User-Agent", "fapi/"+c.current)

	response, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("couldn't reach GitHub to check for updates: %w", err)
	}
	defer func() {
		_ = response.Body.Close() // nothing useful to do if closing fails
	}()

	switch {
	case response.StatusCode == http.StatusNotFound:
		return nil, errors.New("GitHub didn't find fapi's releases. They can only be checked while the repository is public")
	case response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusTooManyRequests:
		return nil, errors.New("GitHub is limiting how often this computer can check for updates. Try again later")
	case response.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("GitHub answered %s when checking for updates", response.Status)
	}

	var releases []release
	err = json.NewDecoder(response.Body).Decode(&releases)
	if err != nil {
		return nil, fmt.Errorf("couldn't read GitHub's list of releases: %w", err)
	}
	return releases, nil
}

// parseVersion reads the major, minor and patch numbers from a version such
// as "v1.2.3", "1.2.3" or "1.2.3-4-gcb1b4aa-dirty" (a local build 4 commits
// after 1.2.3, from scripts/build.sh). Anything after the numbers is
// ignored. It returns false for versions without them, such as "dev".
func parseVersion(text string) ([3]int, bool) {
	var version [3]int
	numbers, _, _ := strings.Cut(strings.TrimPrefix(text, "v"), "-")
	parts := strings.Split(numbers, ".")
	if len(parts) != 3 {
		return version, false
	}
	for index, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return version, false
		}
		version[index] = number
	}
	return version, true
}

// compare returns a positive number if a is newer than b, a negative number
// if it's older, and 0 if they're the same.
func compare(a [3]int, b [3]int) int {
	for index := range a {
		if a[index] != b[index] {
			return a[index] - b[index]
		}
	}
	return 0
}
