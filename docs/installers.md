# Installers

The default end-user install path for `dixa` is the native installer attached to each GitHub Release.

## Default Installer Assets

### macOS

Use the macOS package attached to the release:

`dixa-<version>-macos-universal.pkg`

Open the package and follow the installer prompts. It installs `dixa` to `/usr/local/bin`.

For internal builds from this repo:

```bash
VERSION=0.1.0 ./scripts/build-macos-pkg.sh
```

### Windows

Use the Windows installer executable attached to the release:

`dixa-installer_<version>_windows_<arch>.exe`

Run the installer executable. It:

- downloads the matching `dixa` release archive for the installer version
- installs `dixa.exe` into `%LOCALAPPDATA%\Programs\dixa\bin`
- updates the user `Path`

Open a new PowerShell or Command Prompt window after install before running `dixa`.

## Fallback Options

### Homebrew (macOS fallback, once the public tap is enabled)

```bash
brew install Dixa-public/tap/dixa
```

### PowerShell + GitHub (Windows fallback)

Install the latest release:

```powershell
irm https://raw.githubusercontent.com/Dixa-public/dixa-cli-public/main/scripts/install.ps1 | iex
```

To pin a version:

```powershell
$env:DIXA_VERSION = "0.1.0"
irm https://raw.githubusercontent.com/Dixa-public/dixa-cli-public/main/scripts/install.ps1 | iex
```

This fallback:

- resolves the latest release unless `DIXA_VERSION` is set
- downloads the matching Windows zip for `amd64` or `arm64`
- installs `dixa.exe` into `%LOCALAPPDATA%\Programs\dixa\bin`
- adds that directory to the user `Path`

### Shell + GitHub (macOS fallback)

Install the latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/Dixa-public/dixa-cli-public/main/scripts/install.sh | bash
```

To pin a version:

```bash
curl -fsSL https://raw.githubusercontent.com/Dixa-public/dixa-cli-public/main/scripts/install.sh | DIXA_VERSION=0.1.0 bash
```

This fallback:

- resolves the latest release unless `DIXA_VERSION` is set
- downloads the matching macOS archive for `arm64` or `amd64`
- installs `dixa` into `/usr/local/bin` when writable, otherwise `~/.local/bin`

### Direct archive download

Download the latest release archive from [GitHub Releases](https://github.com/Dixa-public/dixa-cli-public/releases), unpack it, and place `dixa` on your `PATH`.

## Verify

After install, open a fresh shell and run:

```bash
dixa --version
```

## Notes

- Native installers and fallback scripts depend on published GitHub Release assets.
- Local source builds are still useful for development, but they are not the recommended end-user install path.
