# Releasing

This project uses git tags as the single source of truth for published versions.

## Version Mapping

- Git tag: `v0.1.0`
- GitHub Release: `v0.1.0`
- GoReleaser `{{ .Version }}`: `0.1.0`
- `dixa --version`: `0.1.0`
- Plain local `go build ./cmd/dixa`: `dev`

Local builds default to `dev` because the version is injected only during release-style builds.

## Prerequisites

- Run releases from the real git checkout of `Dixa-public/dixa-cli-public` with full history and tags.

## Release Flow

1. Merge release-ready changes to `main`.
2. Create an annotated tag:

   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   ```

3. Push the tag:

   ```bash
   git push origin v0.1.0
   ```

4. Let GitHub Actions run `.github/workflows/release.yml`.
5. Verify the GitHub Release contains:
   - `dixa-<version>-macos-universal.pkg`
   - `dixa-installer_<version>_windows_<arch>.exe`
   - `skill-v<version>.zip`
   - the macOS tarballs
   - the Linux tarballs
   - the Windows zip archives
   - `checksums.txt`
6. Verify the native installers work as the default install paths:

   macOS `.pkg`:

   - download `dixa-<version>-macos-universal.pkg`
   - install it

   ```bash
   dixa --version
   dixa --help
   dixa update
   ```

   Windows `.exe`:

   - download `dixa-installer_<version>_windows_<arch>.exe`
   - run it

   ```powershell
   dixa --version
   dixa --help
   ```

   `dixa update` should either report that the binary is already current or, when a newer stable release exists, update the installed release binary in place.

7. Verify the fallback installers still resolve and download the matching release assets.

   Shell + GitHub fallback:

   ```bash
   curl -fsSL https://raw.githubusercontent.com/Dixa-public/dixa-cli-public/main/scripts/install.sh | DIXA_VERSION=0.1.0 bash
   dixa --version
   dixa --help
   ```

   PowerShell + GitHub fallback:

   ```powershell
   $env:DIXA_VERSION = "0.1.0"
   irm https://raw.githubusercontent.com/Dixa-public/dixa-cli-public/main/scripts/install.ps1 | iex
   dixa --version
   dixa --help
   ```

8. Verify the Claude skill bundle is self-contained:

   ```bash
   unzip -l skill-v<version>.zip
   ```

   Confirm the archive contains:

   - `dixa/SKILL.md`
   - `dixa/scripts/install.sh`
   - `dixa/scripts/install.ps1`

   Then extract it and confirm the bundled skill defaults to the matching CLI version unless `DIXA_VERSION` is overridden.

## Snapshot Validation

From the real git checkout, run:

```bash
rm -rf .release-extra
VERSION=0.1.0 OUTPUT_DIR=.release-extra ./scripts/build-macos-pkg.sh
VERSION=0.1.0 OUTPUT_DIR=.release-extra ./scripts/build-claude-skill-zip.sh
goreleaser release --snapshot --clean
```

Validate that:

- the macOS `.pkg` is created in `.release-extra/`
- the Claude skill zip is created in `.release-extra/`
- archives are produced for `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`, and `windows/arm64`
- Windows installer executables are produced for `windows/amd64` and `windows/arm64`
- `checksums.txt` is generated
- Windows archives are emitted as `.zip`
- the built binary reports the injected version rather than `dev`
- `unzip -l .release-extra/skill-v0.1.0.zip` shows:
  - `dixa/SKILL.md`
  - `dixa/scripts/install.sh`
  - `dixa/scripts/install.ps1`

## Prereleases

- Tags such as `v0.1.0-rc1` are published as GitHub prereleases.
