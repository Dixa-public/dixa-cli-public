package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIsReleaseVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version string
		want    bool
	}{
		{version: "0.1.2", want: true},
		{version: "v0.1.2", want: true},
		{version: "0.1.2-rc1", want: true},
		{version: "dev", want: false},
		{version: "main", want: false},
		{version: "", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()
			if got := IsReleaseVersion(tt.version); got != tt.want {
				t.Fatalf("IsReleaseVersion(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestCheckUsesFreshCache(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	statePath := filepath.Join(t.TempDir(), "update-state.json")
	if err := writeStateFile(statePath, state{
		LastCheckedAt: now.Add(-time.Hour),
		LatestVersion: "0.2.0",
		LatestTag:     "v0.2.0",
	}); err != nil {
		t.Fatalf("write state: %v", err)
	}

	serverCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalls++
		t.Fatalf("unexpected network request: %s", r.URL.Path)
	}))
	defer server.Close()

	manager := newTestManager(server, statePath, now)
	result, err := manager.Check(context.Background(), "0.1.2")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !result.UpdateAvailable {
		t.Fatalf("expected update to be available")
	}
	if result.LatestVersion != "0.2.0" {
		t.Fatalf("expected cached latest version, got %q", result.LatestVersion)
	}
	if serverCalls != 0 {
		t.Fatalf("expected no network calls, got %d", serverCalls)
	}
}

func TestCheckRefreshesExpiredCache(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	statePath := filepath.Join(t.TempDir(), "update-state.json")
	if err := writeStateFile(statePath, state{
		LastCheckedAt: now.Add(-48 * time.Hour),
		LatestVersion: "0.1.1",
		LatestTag:     "v0.1.1",
	}); err != nil {
		t.Fatalf("write state: %v", err)
	}

	serverCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalls++
		if r.URL.Path != "/repos/"+DefaultRepo+"/releases/latest" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": "v0.2.0"})
	}))
	defer server.Close()

	manager := newTestManager(server, statePath, now)
	result, err := manager.Check(context.Background(), "0.1.2")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !result.UpdateAvailable {
		t.Fatalf("expected refreshed update notice")
	}
	if result.LatestVersion != "0.2.0" {
		t.Fatalf("expected refreshed latest version, got %q", result.LatestVersion)
	}
	if serverCalls != 1 {
		t.Fatalf("expected one network call, got %d", serverCalls)
	}

	st, err := readStateFile(statePath)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if st.LatestVersion != "0.2.0" || st.LatestTag != "v0.2.0" {
		t.Fatalf("unexpected refreshed state: %#v", st)
	}
}

func TestCheckRefreshFailureUsesCachedRelease(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	statePath := filepath.Join(t.TempDir(), "update-state.json")
	if err := writeStateFile(statePath, state{
		LastCheckedAt: now.Add(-48 * time.Hour),
		LatestVersion: "0.2.0",
		LatestTag:     "v0.2.0",
	}); err != nil {
		t.Fatalf("write state: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	manager := newTestManager(server, statePath, now)
	result, err := manager.Check(context.Background(), "0.1.2")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !result.UpdateAvailable || result.LatestVersion != "0.2.0" {
		t.Fatalf("expected stale cached release to be used, got %#v", result)
	}

	st, err := readStateFile(statePath)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if !st.LastCheckedAt.Equal(now) {
		t.Fatalf("expected last_checked_at to update on refresh failure, got %v", st.LastCheckedAt)
	}
}

func TestCheckVersionComparisons(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	statePath := filepath.Join(t.TempDir(), "update-state.json")
	if err := writeStateFile(statePath, state{
		LastCheckedAt: now,
		LatestVersion: "0.1.2",
		LatestTag:     "v0.1.2",
	}); err != nil {
		t.Fatalf("write state: %v", err)
	}

	manager := newTestManager(nil, statePath, now)
	tests := []struct {
		name          string
		current       string
		wantAvailable bool
	}{
		{name: "older", current: "0.1.1", wantAvailable: true},
		{name: "equal", current: "0.1.2", wantAvailable: false},
		{name: "newer", current: "0.1.3", wantAvailable: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := manager.Check(context.Background(), tt.current)
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			if result.UpdateAvailable != tt.wantAvailable {
				t.Fatalf("Check(%q) update_available = %v, want %v", tt.current, result.UpdateAvailable, tt.wantAvailable)
			}
		})
	}
}

func TestSelfUpdateUpToDate(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/" + DefaultRepo + "/releases/latest":
			_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": "v0.1.2"})
		default:
			t.Fatalf("unexpected download for up-to-date binary: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	manager := newTestManager(server, filepath.Join(t.TempDir(), "update-state.json"), time.Now().UTC())
	result, err := manager.SelfUpdate(context.Background(), "0.1.2")
	if err != nil {
		t.Fatalf("SelfUpdate: %v", err)
	}
	if result.Status != "up_to_date" {
		t.Fatalf("expected up_to_date status, got %#v", result)
	}
}

func TestSelfUpdateReplacesBinaryOnDarwin(t *testing.T) {
	t.Parallel()

	exeDir := t.TempDir()
	exePath := filepath.Join(exeDir, "dixa")
	if err := os.WriteFile(exePath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	archive := mustTarGz(t, "dixa", []byte("new-binary"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/" + DefaultRepo + "/releases/latest":
			_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": "v0.1.3"})
		case "/" + DefaultRepo + "/releases/download/v0.1.3/dixa_0.1.3_darwin_arm64.tar.gz":
			_, _ = w.Write(archive)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	manager := newTestManager(server, filepath.Join(t.TempDir(), "update-state.json"), time.Now().UTC())
	manager.GOOS = "darwin"
	manager.GOARCH = "arm64"
	manager.ExecutablePath = func() (string, error) { return exePath, nil }

	result, err := manager.SelfUpdate(context.Background(), "0.1.2")
	if err != nil {
		t.Fatalf("SelfUpdate: %v", err)
	}
	if result.Status != "updated" {
		t.Fatalf("expected updated status, got %#v", result)
	}

	got, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatalf("read updated executable: %v", err)
	}
	if string(got) != "new-binary" {
		t.Fatalf("unexpected updated binary contents: %q", string(got))
	}
	info, err := os.Stat(exePath)
	if err != nil {
		t.Fatalf("stat updated executable: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("expected executable mode to be preserved, got %v", info.Mode().Perm())
	}
}

func TestSelfUpdateBlocksDevBuild(t *testing.T) {
	t.Parallel()

	manager := newTestManager(nil, filepath.Join(t.TempDir(), "update-state.json"), time.Now().UTC())
	_, err := manager.SelfUpdate(context.Background(), "dev")
	if !errors.Is(err, ErrReleaseBuildRequired) {
		t.Fatalf("expected ErrReleaseBuildRequired, got %v", err)
	}
}

func TestSelfUpdateRejectsUnsupportedPlatform(t *testing.T) {
	t.Parallel()

	manager := newTestManager(nil, filepath.Join(t.TempDir(), "update-state.json"), time.Now().UTC())
	manager.GOOS = "linux"
	manager.GOARCH = "amd64"
	_, err := manager.SelfUpdate(context.Background(), "0.1.2")
	if !errors.Is(err, ErrUnsupportedPlatform) {
		t.Fatalf("expected ErrUnsupportedPlatform, got %v", err)
	}
}

func TestSelfUpdateReturnsPermissionGuidance(t *testing.T) {
	t.Parallel()

	exeDir := t.TempDir()
	exePath := filepath.Join(exeDir, "dixa")
	if err := os.WriteFile(exePath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	archive := mustTarGz(t, "dixa", []byte("new-binary"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/" + DefaultRepo + "/releases/latest":
			_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": "v0.1.3"})
		case "/" + DefaultRepo + "/releases/download/v0.1.3/dixa_0.1.3_darwin_arm64.tar.gz":
			_, _ = w.Write(archive)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	manager := newTestManager(server, filepath.Join(t.TempDir(), "update-state.json"), time.Now().UTC())
	manager.GOOS = "darwin"
	manager.GOARCH = "arm64"
	manager.ExecutablePath = func() (string, error) { return exePath, nil }
	manager.RenameFile = func(oldPath, newPath string) error { return os.ErrPermission }

	_, err := manager.SelfUpdate(context.Background(), "0.1.2")
	if err == nil {
		t.Fatalf("expected permission error")
	}
	if !strings.Contains(err.Error(), "scripts/install.sh") {
		t.Fatalf("expected reinstall guidance, got %v", err)
	}
}

func TestSelfUpdateSchedulesWindowsReplace(t *testing.T) {
	t.Parallel()

	exeDir := t.TempDir()
	exePath := filepath.Join(exeDir, "dixa.exe")
	if err := os.WriteFile(exePath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	archive := mustZip(t, "dixa.exe", []byte("new-binary"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/" + DefaultRepo + "/releases/latest":
			_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": "v0.1.3"})
		case "/" + DefaultRepo + "/releases/download/v0.1.3/dixa_0.1.3_windows_amd64.zip":
			_, _ = w.Write(archive)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	manager := newTestManager(server, filepath.Join(t.TempDir(), "update-state.json"), time.Now().UTC())
	manager.GOOS = "windows"
	manager.GOARCH = "amd64"
	manager.ExecutablePath = func() (string, error) { return exePath, nil }

	var helperPath string
	manager.LaunchHelper = func(path string) error {
		helperPath = path
		return nil
	}

	result, err := manager.SelfUpdate(context.Background(), "0.1.2")
	if err != nil {
		t.Fatalf("SelfUpdate: %v", err)
	}
	if result.Status != "scheduled" || !result.Scheduled {
		t.Fatalf("expected scheduled result, got %#v", result)
	}
	if helperPath == "" {
		t.Fatalf("expected helper script to be launched")
	}

	script, err := os.ReadFile(helperPath)
	if err != nil {
		t.Fatalf("read helper script: %v", err)
	}
	if !strings.Contains(string(script), exePath) {
		t.Fatalf("expected helper script to reference target executable, got %s", string(script))
	}
	if !strings.Contains(string(script), "copy /Y") {
		t.Fatalf("expected helper script copy loop, got %s", string(script))
	}

	stagedBinary, err := os.ReadFile(filepath.Join(filepath.Dir(helperPath), "dixa.exe"))
	if err != nil {
		t.Fatalf("read staged binary: %v", err)
	}
	if string(stagedBinary) != "new-binary" {
		t.Fatalf("unexpected staged binary contents: %q", string(stagedBinary))
	}
}

func newTestManager(server *httptest.Server, statePath string, now time.Time) *Manager {
	manager := &Manager{
		Repo:       DefaultRepo,
		StatePath:  statePath,
		CacheTTL:   DefaultCacheTTL,
		Now:        func() time.Time { return now },
		GOOS:       "darwin",
		GOARCH:     "arm64",
		RenameFile: os.Rename,
	}
	if server != nil {
		manager.APIBaseURL = server.URL
		manager.DownloadBaseURL = server.URL
		manager.HTTPClient = server.Client()
	} else {
		manager.HTTPClient = &http.Client{Timeout: DefaultHTTPTimeout}
	}
	return manager
}

func writeStateFile(path string, st state) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readStateFile(path string) (state, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return state{}, err
	}
	var st state
	if err := json.Unmarshal(data, &st); err != nil {
		return state{}, err
	}
	return st, nil
}

func mustTarGz(t *testing.T, name string, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(data))}); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatalf("write tar body: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}

func mustZip(t *testing.T, name string, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}
