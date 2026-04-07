// Package updater provides rate-limited update detection for the agenthub app.
// It checks GitHub releases (via go-selfupdate) and persists the last-check
// timestamp to disk to avoid excessive network calls.
//
// Usage:
//
//	info, err := updater.Check(ctx, configDir, "scottkw/agenthub", version, updater.DefaultDetect, false)
//	if info != nil {
//	    // show update notification
//	}
package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	selfupdate "github.com/creativeprojects/go-selfupdate"
)

const (
	// rateLimitDuration is the minimum time between background update checks.
	rateLimitDuration = 1 * time.Hour

	// checkFile is the filename for the persisted last-check timestamp.
	checkFile = "update_check.json"
)

// DetectFunc is the injectable function type for detecting the latest release.
// It returns the latest version string, whether a release was found, and any error.
type DetectFunc func(ctx context.Context, slug string) (latestVersion string, found bool, err error)

// UpdateInfo holds information about an available update.
type UpdateInfo struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	ReleaseURL     string `json:"releaseURL"`
}

// lastCheckFile is the on-disk format for the rate-limit timestamp.
type lastCheckFile struct {
	LastCheck string `json:"last_check"`
}

// DefaultDetect uses go-selfupdate to check GitHub releases for the given slug.
// It returns the latest version string (without leading "v"), whether a release
// was found, and any network/API error.
func DefaultDetect(ctx context.Context, slug string) (string, bool, error) {
	latest, found, err := selfupdate.DetectLatest(ctx, selfupdate.ParseSlug(slug))
	if err != nil || !found {
		return "", found, err
	}
	return latest.Version(), true, nil
}

// Check performs a rate-limited update check.
//
// Returns nil, nil when:
//   - currentVersion is "dev" or "" (development build)
//   - rate limit is active (last check within 1 hour) and force is false
//   - no update is available (current is already latest)
//   - detectFunc returns an error or not-found (silent failure)
//
// When force is true, the rate limit is bypassed (for manual "Check for Updates").
//
// On a successful detection of a newer version, the last-check timestamp is
// persisted to configDir/update_check.json for future rate-limiting.
func Check(ctx context.Context, configDir, slug, currentVersion string, detect DetectFunc, force bool) (*UpdateInfo, error) {
	// Skip update check for development builds.
	if currentVersion == "dev" || currentVersion == "" {
		return nil, nil
	}

	// Check rate limit (skip if force=true).
	if !force {
		if withinRateLimit(configDir) {
			return nil, nil
		}
	}

	// Call the injected detect function.
	latestVersion, found, err := detect(ctx, slug)
	if err != nil || !found {
		// Silent failure — do not surface errors to the caller.
		return nil, nil
	}

	// Strip leading "v" from both versions before comparison.
	cleanCurrent := strings.TrimPrefix(currentVersion, "v")
	cleanLatest := strings.TrimPrefix(latestVersion, "v")

	// Use Masterminds/semver (transitive dep of go-selfupdate) for version comparison.
	vCurrent, err := semver.NewVersion(cleanCurrent)
	if err != nil {
		// Persist timestamp to avoid re-checking on every startup with a bad version.
		_ = persistTimestamp(configDir)
		return nil, nil
	}
	vLatest, err := semver.NewVersion(cleanLatest)
	if err != nil {
		_ = persistTimestamp(configDir)
		return nil, nil
	}

	// Persist the last-check timestamp regardless of whether there's an update.
	_ = persistTimestamp(configDir)

	// No update if latest is not greater than current.
	if !vLatest.GreaterThan(vCurrent) {
		return nil, nil
	}

	return &UpdateInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		ReleaseURL:     fmt.Sprintf("https://github.com/%s/releases/tag/v%s", slug, cleanLatest),
	}, nil
}

// withinRateLimit returns true if the last check was within the rate limit window.
func withinRateLimit(configDir string) bool {
	data, err := os.ReadFile(filepath.Join(configDir, checkFile))
	if err != nil {
		return false
	}
	var cf lastCheckFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return false
	}
	ts, err := time.Parse(time.RFC3339, cf.LastCheck)
	if err != nil {
		return false
	}
	return time.Since(ts) < rateLimitDuration
}

// persistTimestamp writes the current UTC time to configDir/update_check.json.
// The directory is created if it does not exist.
func persistTimestamp(configDir string) error {
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}
	cf := lastCheckFile{LastCheck: time.Now().UTC().Format(time.RFC3339)}
	data, err := json.Marshal(cf)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configDir, checkFile), data, 0600)
}
