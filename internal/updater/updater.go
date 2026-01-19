package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const (
	repoOwner = "KilimcininKorOglu"
	repoName  = "Hermes-Autonomous-Agent"
	apiURL    = "https://api.github.com/repos/%s/%s/releases/latest"
)

// Release represents a GitHub release
type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []Asset `json:"assets"`
	HTMLURL string  `json:"html_url"`
}

// Asset represents a release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Updater handles version checking and updates
type Updater struct {
	currentVersion string
	httpClient     *http.Client
}

// New creates a new Updater
func New(currentVersion string) *Updater {
	return &Updater{
		currentVersion: currentVersion,
		httpClient:     &http.Client{},
	}
}

// CheckUpdate checks for a new version
func (u *Updater) CheckUpdate() (*Release, bool, error) {
	url := fmt.Sprintf(apiURL, repoOwner, repoName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Hermes-Updater")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, false, fmt.Errorf("no releases found")
	}

	if resp.StatusCode != 200 {
		return nil, false, fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, false, fmt.Errorf("failed to parse release: %w", err)
	}

	hasUpdate := u.compareVersions(release.TagName)
	return &release, hasUpdate, nil
}

// compareVersions returns true if remote version is newer
func (u *Updater) compareVersions(remoteTag string) bool {
	remote := strings.TrimPrefix(remoteTag, "v")
	current := strings.TrimPrefix(u.currentVersion, "v")

	remoteParts := strings.Split(remote, ".")
	currentParts := strings.Split(current, ".")

	for i := 0; i < len(remoteParts) && i < len(currentParts); i++ {
		remoteNum, _ := strconv.Atoi(remoteParts[i])
		currentNum, _ := strconv.Atoi(currentParts[i])

		if remoteNum > currentNum {
			return true
		}
		if remoteNum < currentNum {
			return false
		}
	}

	return len(remoteParts) > len(currentParts)
}

// GetAssetName returns the expected asset name for current platform
func (u *Updater) GetAssetName() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	name := fmt.Sprintf("hermes-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

// FindAsset finds the matching asset for current platform
func (u *Updater) FindAsset(release *Release) *Asset {
	expectedName := u.GetAssetName()
	for _, asset := range release.Assets {
		if asset.Name == expectedName {
			return &asset
		}
	}
	return nil
}

// DownloadAndReplace downloads the new binary and replaces the current one
func (u *Updater) DownloadAndReplace(asset *Asset) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	req, err := http.NewRequest("GET", asset.BrowserDownloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Hermes-Updater")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(execPath), "hermes-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write update: %w", err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpPath, 0755); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	}

	oldPath := execPath + ".old"
	os.Remove(oldPath)

	if err := os.Rename(execPath, oldPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	if err := os.Rename(tmpPath, execPath); err != nil {
		os.Rename(oldPath, execPath)
		return fmt.Errorf("failed to install update: %w", err)
	}

	os.Remove(oldPath)

	return nil
}
