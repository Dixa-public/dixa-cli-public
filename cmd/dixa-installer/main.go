//go:build windows

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sys/windows/registry"
)

var (
	version = "dev"
	repo    = "Dixa-public/dixa-cli-public"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "dixa installer failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	installDir := os.Getenv("INSTALL_DIR")
	if installDir == "" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("resolve home directory: %w", err)
			}
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		installDir = filepath.Join(localAppData, "Programs", "dixa", "bin")
	}

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}

	tag, releaseVersion := releaseTag(version)
	archiveName := fmt.Sprintf("dixa_%s_windows_%s.zip", releaseVersion, runtime.GOARCH)
	archiveURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, tag, archiveName)

	tmpDir, err := os.MkdirTemp("", "dixa-installer-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, archiveName)
	if err := downloadFile(archiveURL, archivePath); err != nil {
		return err
	}

	targetExe := filepath.Join(installDir, "dixa.exe")
	if err := extractFileFromZip(archivePath, "dixa.exe", targetExe); err != nil {
		return err
	}

	if err := ensureUserPathContains(installDir); err != nil {
		return err
	}

	fmt.Printf("Installed dixa %s to %s\n", releaseVersion, targetExe)
	fmt.Println("Open a new PowerShell or Command Prompt window before running `dixa`.")
	return nil
}

func releaseTag(v string) (tag string, versionWithoutV string) {
	if strings.HasPrefix(v, "v") {
		return v, strings.TrimPrefix(v, "v")
	}
	return "v" + v, v
}

func downloadFile(url, dst string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: unexpected status %s", url, resp.Status)
	}

	file, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

func extractFileFromZip(zipPath, wantedName, dst string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip %s: %w", zipPath, err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if filepath.Base(file.Name) != wantedName {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("open %s in zip: %w", file.Name, err)
		}

		out, err := os.Create(dst)
		if err != nil {
			rc.Close()
			return fmt.Errorf("create %s: %w", dst, err)
		}

		_, copyErr := io.Copy(out, rc)
		closeErr := rc.Close()
		syncErr := out.Close()
		if copyErr != nil {
			return fmt.Errorf("extract %s: %w", wantedName, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close %s in zip: %w", wantedName, closeErr)
		}
		if syncErr != nil {
			return fmt.Errorf("close %s: %w", dst, syncErr)
		}

		return nil
	}

	return fmt.Errorf("%s not found in %s", wantedName, zipPath)
}

func ensureUserPathContains(dir string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open user environment registry key: %w", err)
	}
	defer key.Close()

	currentPath, _, err := key.GetStringValue("Path")
	if err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("read user PATH: %w", err)
	}

	if pathContains(currentPath, dir) {
		return nil
	}

	newPath := dir
	if currentPath != "" {
		newPath = currentPath + ";" + dir
	}

	if err := key.SetExpandStringValue("Path", newPath); err != nil {
		return fmt.Errorf("update user PATH: %w", err)
	}

	return nil
}

func pathContains(pathValue, dir string) bool {
	for _, part := range strings.Split(pathValue, ";") {
		if strings.EqualFold(filepath.Clean(part), filepath.Clean(dir)) {
			return true
		}
	}
	return false
}
