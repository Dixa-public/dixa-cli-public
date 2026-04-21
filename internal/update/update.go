package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	DefaultRepo            = "Dixa-public/dixa-cli-public"
	DefaultAPIBaseURL      = "https://api.github.com"
	DefaultDownloadBaseURL = "https://github.com"
	DefaultCacheTTL        = 24 * time.Hour
	DefaultHTTPTimeout     = 2 * time.Second
)

var (
	ErrReleaseBuildRequired = errors.New("self-update only works on released dixa binaries")
	ErrUnsupportedPlatform  = errors.New("self-update is not supported on this platform")
)

type Service interface {
	Check(context.Context, string) (CheckResult, error)
	SelfUpdate(context.Context, string) (UpdateResult, error)
}

type CheckResult struct {
	CurrentVersion  string `json:"current_version,omitempty"`
	LatestVersion   string `json:"latest_version,omitempty"`
	LatestTag       string `json:"latest_tag,omitempty"`
	UpdateAvailable bool   `json:"update_available"`
}

type UpdateResult struct {
	CurrentVersion string `json:"current_version,omitempty"`
	LatestVersion  string `json:"latest_version,omitempty"`
	LatestTag      string `json:"latest_tag,omitempty"`
	ExecutablePath string `json:"executable_path,omitempty"`
	Status         string `json:"status,omitempty"`
	Scheduled      bool   `json:"scheduled,omitempty"`
	Message        string `json:"message,omitempty"`
}

type Manager struct {
	Repo            string
	StatePath       string
	APIBaseURL      string
	DownloadBaseURL string
	HTTPClient      *http.Client
	CacheTTL        time.Duration
	Now             func() time.Time
	GOOS            string
	GOARCH          string
	ExecutablePath  func() (string, error)
	RenameFile      func(string, string) error
	LaunchHelper    func(string) error
}

type state struct {
	LastCheckedAt time.Time `json:"last_checked_at"`
	LatestVersion string    `json:"latest_version"`
	LatestTag     string    `json:"latest_tag"`
}

type release struct {
	Tag     string
	Version string
	Semver  string
}

type githubReleaseResponse struct {
	TagName string `json:"tag_name"`
}

func DefaultStatePath(home string) string {
	return filepath.Join(home, ".config", "dixa", "update-state.json")
}

func NewManager(home string, client *http.Client) *Manager {
	if client == nil {
		client = &http.Client{Timeout: DefaultHTTPTimeout}
	}
	return &Manager{
		Repo:            DefaultRepo,
		StatePath:       DefaultStatePath(home),
		APIBaseURL:      DefaultAPIBaseURL,
		DownloadBaseURL: DefaultDownloadBaseURL,
		HTTPClient:      client,
		CacheTTL:        DefaultCacheTTL,
		Now:             time.Now,
		GOOS:            runtime.GOOS,
		GOARCH:          runtime.GOARCH,
		ExecutablePath:  os.Executable,
		RenameFile:      os.Rename,
		LaunchHelper:    defaultLaunchHelper,
	}
}

func IsReleaseVersion(version string) bool {
	_, err := normalizeVersion(version)
	return err == nil
}

func (m *Manager) Check(ctx context.Context, currentVersion string) (CheckResult, error) {
	currentSemver, err := normalizeVersion(currentVersion)
	if err != nil {
		return CheckResult{}, err
	}

	latest, err := m.latestRelease(ctx, true)
	if err != nil {
		return CheckResult{
			CurrentVersion: strings.TrimPrefix(currentSemver, "v"),
		}, err
	}

	return CheckResult{
		CurrentVersion:  strings.TrimPrefix(currentSemver, "v"),
		LatestVersion:   latest.Version,
		LatestTag:       latest.Tag,
		UpdateAvailable: semver.Compare(currentSemver, latest.Semver) < 0,
	}, nil
}

func (m *Manager) SelfUpdate(ctx context.Context, currentVersion string) (UpdateResult, error) {
	currentSemver, err := normalizeVersion(currentVersion)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("%w; current version is %q", ErrReleaseBuildRequired, strings.TrimSpace(currentVersion))
	}

	if err := m.validatePlatform(); err != nil {
		return UpdateResult{}, err
	}

	latest, err := m.latestRelease(ctx, false)
	if err != nil {
		return UpdateResult{}, err
	}

	result := UpdateResult{
		CurrentVersion: strings.TrimPrefix(currentSemver, "v"),
		LatestVersion:  latest.Version,
		LatestTag:      latest.Tag,
	}

	if semver.Compare(currentSemver, latest.Semver) >= 0 {
		result.Status = "up_to_date"
		result.Message = "dixa is already up to date."
		return result, nil
	}

	executablePath, err := m.currentExecutablePath()
	if err != nil {
		return result, fmt.Errorf("resolve current executable: %w", err)
	}
	result.ExecutablePath = executablePath

	archiveName, err := m.archiveName(latest.Version)
	if err != nil {
		return result, err
	}

	archive, err := m.downloadReleaseAsset(ctx, latest.Tag, archiveName)
	if err != nil {
		return result, err
	}

	switch m.goos() {
	case "darwin":
		if err := m.replaceUnixBinary(executablePath, archive); err != nil {
			return result, err
		}
		result.Status = "updated"
		result.Message = fmt.Sprintf("Updated dixa to %s.", latest.Version)
	case "linux":
		if err := m.replaceUnixBinary(executablePath, archive); err != nil {
			return result, err
		}
		result.Status = "updated"
		result.Message = fmt.Sprintf("Updated dixa to %s.", latest.Version)
	case "windows":
		if err := m.scheduleWindowsUpdate(executablePath, archive); err != nil {
			return result, err
		}
		result.Status = "scheduled"
		result.Scheduled = true
		result.Message = fmt.Sprintf("Scheduled update to dixa %s. The new version will be available after this process exits.", latest.Version)
	default:
		return result, ErrUnsupportedPlatform
	}

	return result, nil
}

func (m *Manager) latestRelease(ctx context.Context, allowCached bool) (release, error) {
	st, _ := m.loadState()
	if allowCached && m.cacheFresh(st) {
		if cached, ok := releaseFromState(st); ok {
			return cached, nil
		}
	}

	latest, err := m.fetchLatestRelease(ctx)
	if err != nil {
		if allowCached {
			st.LastCheckedAt = m.now()
			_ = m.saveState(st)
			if cached, ok := releaseFromState(st); ok {
				return cached, nil
			}
		}
		return release{}, err
	}

	st.LastCheckedAt = m.now()
	st.LatestVersion = latest.Version
	st.LatestTag = latest.Tag
	_ = m.saveState(st)

	return latest, nil
}

func (m *Manager) fetchLatestRelease(ctx context.Context) (release, error) {
	url := strings.TrimRight(m.apiBaseURL(), "/") + "/repos/" + m.repo() + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return release{}, fmt.Errorf("build latest release request: %w", err)
	}

	resp, err := m.httpClient().Do(req)
	if err != nil {
		return release{}, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return release{}, fmt.Errorf("fetch latest release: unexpected status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload githubReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return release{}, fmt.Errorf("decode latest release metadata: %w", err)
	}

	return parseReleaseTag(payload.TagName)
}

func (m *Manager) downloadReleaseAsset(ctx context.Context, tag, assetName string) ([]byte, error) {
	url := strings.TrimRight(m.downloadBaseURL(), "/") + "/" + m.repo() + "/releases/download/" + tag + "/" + assetName
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build release asset request: %w", err)
	}

	resp, err := m.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", assetName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("download %s: unexpected status %s: %s", assetName, resp.Status, strings.TrimSpace(string(body)))
	}

	archive, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", assetName, err)
	}
	return archive, nil
}

func (m *Manager) replaceUnixBinary(executablePath string, archive []byte) error {
	binary, err := extractTarGzFile(archive, "dixa")
	if err != nil {
		return err
	}

	info, err := os.Stat(executablePath)
	if err != nil {
		return fmt.Errorf("stat current executable %s: %w", executablePath, err)
	}

	dir := filepath.Dir(executablePath)
	tempFile, err := os.CreateTemp(dir, ".dixa-update-*")
	if err != nil {
		return m.wrapInPlaceUpdateError(executablePath, err)
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if _, err := tempFile.Write(binary); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("write temporary update file: %w", err)
	}

	mode := info.Mode().Perm()
	if mode == 0 {
		mode = 0o755
	}
	if err := tempFile.Chmod(mode); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("set executable permissions on temporary update file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temporary update file: %w", err)
	}

	if err := m.renameFile()(tempPath, executablePath); err != nil {
		return m.wrapInPlaceUpdateError(executablePath, err)
	}

	tempPath = ""
	return nil
}

func (m *Manager) scheduleWindowsUpdate(executablePath string, archive []byte) error {
	binary, err := extractZipFile(archive, "dixa.exe")
	if err != nil {
		return err
	}

	workDir, err := os.MkdirTemp("", "dixa-update-*")
	if err != nil {
		return fmt.Errorf("create temporary update directory: %w", err)
	}

	sourcePath := filepath.Join(workDir, "dixa.exe")
	if err := os.WriteFile(sourcePath, binary, 0o755); err != nil {
		_ = os.RemoveAll(workDir)
		return fmt.Errorf("write staged update binary: %w", err)
	}

	helperPath := filepath.Join(workDir, "apply-update.cmd")
	script := buildWindowsUpdateScript(executablePath, sourcePath, helperPath, workDir, os.Getpid())
	if err := os.WriteFile(helperPath, []byte(script), 0o700); err != nil {
		_ = os.RemoveAll(workDir)
		return fmt.Errorf("write update helper script: %w", err)
	}

	if err := m.launchHelper()(helperPath); err != nil {
		_ = os.RemoveAll(workDir)
		return fmt.Errorf("launch update helper: %w", err)
	}

	return nil
}

func (m *Manager) archiveName(version string) (string, error) {
	switch m.goos() {
	case "darwin":
		if !supportedArch(m.goarch()) {
			return "", fmt.Errorf("dixa update is not supported on %s/%s", m.goos(), m.goarch())
		}
		return fmt.Sprintf("dixa_%s_darwin_%s.tar.gz", version, m.goarch()), nil
	case "linux":
		if !supportedArch(m.goarch()) {
			return "", fmt.Errorf("dixa update is not supported on %s/%s", m.goos(), m.goarch())
		}
		return fmt.Sprintf("dixa_%s_linux_%s.tar.gz", version, m.goarch()), nil
	case "windows":
		if !supportedArch(m.goarch()) {
			return "", fmt.Errorf("dixa update is not supported on %s/%s", m.goos(), m.goarch())
		}
		return fmt.Sprintf("dixa_%s_windows_%s.zip", version, m.goarch()), nil
	default:
		return "", fmt.Errorf("%w on %s/%s; use the latest GitHub release installer instead", ErrUnsupportedPlatform, m.goos(), m.goarch())
	}
}

func (m *Manager) currentExecutablePath() (string, error) {
	path, err := m.executablePath()()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil && resolved != "" {
		return resolved, nil
	}
	return path, nil
}

func (m *Manager) validatePlatform() error {
	switch m.goos() {
	case "darwin", "linux", "windows":
		if supportedArch(m.goarch()) {
			return nil
		}
	}
	return fmt.Errorf("%w on %s/%s; use the latest GitHub release installer instead", ErrUnsupportedPlatform, m.goos(), m.goarch())
}

func (m *Manager) loadState() (state, error) {
	data, err := os.ReadFile(m.statePath())
	if errors.Is(err, os.ErrNotExist) {
		return state{}, nil
	}
	if err != nil {
		return state{}, fmt.Errorf("read update state %s: %w", m.statePath(), err)
	}

	var st state
	if err := json.Unmarshal(data, &st); err != nil {
		return state{}, fmt.Errorf("decode update state %s: %w", m.statePath(), err)
	}
	return st, nil
}

func (m *Manager) saveState(st state) error {
	if err := os.MkdirAll(filepath.Dir(m.statePath()), 0o755); err != nil {
		return fmt.Errorf("create update state directory: %w", err)
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("encode update state: %w", err)
	}
	if err := os.WriteFile(m.statePath(), data, 0o644); err != nil {
		return fmt.Errorf("write update state: %w", err)
	}
	return nil
}

func (m *Manager) cacheFresh(st state) bool {
	if st.LastCheckedAt.IsZero() {
		return false
	}
	return m.now().Sub(st.LastCheckedAt) < m.cacheTTL()
}

func (m *Manager) wrapInPlaceUpdateError(executablePath string, err error) error {
	if errors.Is(err, os.ErrPermission) {
		switch m.goos() {
		case "darwin":
			return fmt.Errorf("unable to update %s in place: %w. Reinstall with the latest macOS package or scripts/install.sh instead", executablePath, err)
		default:
			return fmt.Errorf("unable to update %s in place: %w. Reinstall with the latest release archive or scripts/install.sh instead", executablePath, err)
		}
	}
	return fmt.Errorf("replace current executable %s: %w", executablePath, err)
}

func parseReleaseTag(tag string) (release, error) {
	normalized, err := normalizeVersion(tag)
	if err != nil {
		return release{}, fmt.Errorf("latest release tag %q is not valid semver: %w", tag, err)
	}
	return release{
		Tag:     "v" + strings.TrimPrefix(normalized, "v"),
		Version: strings.TrimPrefix(normalized, "v"),
		Semver:  normalized,
	}, nil
}

func normalizeVersion(version string) (string, error) {
	trimmed := strings.TrimSpace(version)
	switch trimmed {
	case "", "dev":
		return "", ErrReleaseBuildRequired
	}
	if !strings.HasPrefix(trimmed, "v") {
		trimmed = "v" + trimmed
	}
	if !semver.IsValid(trimmed) {
		return "", fmt.Errorf("unsupported version %q", version)
	}
	return trimmed, nil
}

func releaseFromState(st state) (release, bool) {
	if strings.TrimSpace(st.LatestTag) == "" || strings.TrimSpace(st.LatestVersion) == "" {
		return release{}, false
	}
	rel, err := parseReleaseTag(st.LatestTag)
	if err == nil {
		return rel, true
	}
	rel, err = parseReleaseTag(st.LatestVersion)
	if err == nil {
		return rel, true
	}
	return release{}, false
}

func supportedArch(arch string) bool {
	switch arch {
	case "amd64", "arm64":
		return true
	default:
		return false
	}
}

func extractTarGzFile(archive []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("open tar.gz archive: %w", err)
	}
	defer gz.Close()

	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar.gz archive: %w", err)
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			continue
		}
		if filepath.Base(header.Name) != name {
			continue
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("extract %s from tar.gz archive: %w", name, err)
		}
		return data, nil
	}

	return nil, fmt.Errorf("%s not found in tar.gz archive", name)
}

func extractZipFile(archive []byte, name string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("open zip archive: %w", err)
	}
	for _, file := range reader.File {
		if filepath.Base(file.Name) != name {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open %s in zip archive: %w", name, err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("extract %s from zip archive: %w", name, err)
		}
		return data, nil
	}
	return nil, fmt.Errorf("%s not found in zip archive", name)
}

func buildWindowsUpdateScript(targetPath, sourcePath, helperPath, workDir string, pid int) string {
	return fmt.Sprintf(`@echo off
setlocal
set "TARGET=%s"
set "SOURCE=%s"
set "HELPER=%s"
set "WORKDIR=%s"
set "PARENT_PID=%d"
:wait
tasklist /FI "PID eq %%PARENT_PID%%" 2>NUL | find "%%PARENT_PID%%" >NUL
if not errorlevel 1 (
  timeout /t 1 /nobreak >NUL
  goto wait
)
:copy
copy /Y "%%SOURCE%%" "%%TARGET%%" >NUL
if errorlevel 1 (
  timeout /t 1 /nobreak >NUL
  goto copy
)
start "" /B cmd /c "ping 127.0.0.1 -n 2 >nul & del /f /q \"%%SOURCE%%\" \"%%HELPER%%\" >nul 2>&1 & rmdir /s /q \"%%WORKDIR%%\" >nul 2>&1"
exit /b 0
`, targetPath, sourcePath, helperPath, workDir, pid)
}

func defaultLaunchHelper(helperPath string) error {
	cmd := exec.Command("cmd.exe", "/C", "call", helperPath)
	return cmd.Start()
}

func (m *Manager) repo() string {
	if strings.TrimSpace(m.Repo) != "" {
		return strings.TrimSpace(m.Repo)
	}
	return DefaultRepo
}

func (m *Manager) apiBaseURL() string {
	if strings.TrimSpace(m.APIBaseURL) != "" {
		return strings.TrimSpace(m.APIBaseURL)
	}
	return DefaultAPIBaseURL
}

func (m *Manager) downloadBaseURL() string {
	if strings.TrimSpace(m.DownloadBaseURL) != "" {
		return strings.TrimSpace(m.DownloadBaseURL)
	}
	return DefaultDownloadBaseURL
}

func (m *Manager) httpClient() *http.Client {
	if m.HTTPClient != nil {
		return m.HTTPClient
	}
	return &http.Client{Timeout: DefaultHTTPTimeout}
}

func (m *Manager) cacheTTL() time.Duration {
	if m.CacheTTL > 0 {
		return m.CacheTTL
	}
	return DefaultCacheTTL
}

func (m *Manager) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

func (m *Manager) goos() string {
	if strings.TrimSpace(m.GOOS) != "" {
		return strings.TrimSpace(m.GOOS)
	}
	return runtime.GOOS
}

func (m *Manager) goarch() string {
	if strings.TrimSpace(m.GOARCH) != "" {
		return strings.TrimSpace(m.GOARCH)
	}
	return runtime.GOARCH
}

func (m *Manager) executablePath() func() (string, error) {
	if m.ExecutablePath != nil {
		return m.ExecutablePath
	}
	return os.Executable
}

func (m *Manager) renameFile() func(string, string) error {
	if m.RenameFile != nil {
		return m.RenameFile
	}
	return os.Rename
}

func (m *Manager) launchHelper() func(string) error {
	if m.LaunchHelper != nil {
		return m.LaunchHelper
	}
	return defaultLaunchHelper
}

func (m *Manager) statePath() string {
	if strings.TrimSpace(m.StatePath) != "" {
		return strings.TrimSpace(m.StatePath)
	}
	return DefaultStatePath("")
}
