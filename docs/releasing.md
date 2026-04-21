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
- Ensure the tap repository exists at `Dixa-public/homebrew-tap`.
- Ensure the tap repository has a `Formula/` directory on its default branch.
- Set the repository variable `ENABLE_HOMEBREW_TAP_UPLOAD=true` only when the public tap is ready to receive automated updates.
- Add a `HOMEBREW_TAP_TOKEN` repository secret in `Dixa-public/dixa-cli-public`.
  - The token must have content write access to `Dixa-public/homebrew-tap`.
  - Do not rely on the default workflow `GITHUB_TOKEN` for cross-repo tap updates.
  - If the variable or secret is not configured, GitHub Releases still publish normally and Homebrew tap updates are skipped.

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
   - the macOS tarballs
   - the Windows zip archives
   - `checksums.txt`
6. Verify the native installers work as the default install paths:

   macOS `.pkg`:

   - download `dixa-<version>-macos-universal.pkg`
   - install it

   ```bash
   dixa --version
   dixa --help
   ```

   Windows `.exe`:

   - download `dixa-installer_<version>_windows_<arch>.exe`
   - run it

   ```powershell
   dixa --version
   dixa --help
   ```

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

8. Verify the Homebrew tap received an updated formula in `Formula/dixa.rb`.
9. Verify the Homebrew fallback still works:

   ```bash
   brew install Dixa-public/tap/dixa
   dixa --version
   dixa --help
   ```

## Snapshot Validation

From the real git checkout, run:

```bash
rm -rf .release-extra
VERSION=0.1.0 OUTPUT_DIR=.release-extra ./scripts/build-macos-pkg.sh
goreleaser release --snapshot --clean
```

Validate that:

- the macOS `.pkg` is created in `.release-extra/`
- archives are produced for `darwin/amd64`, `darwin/arm64`, `windows/amd64`, and `windows/arm64`
- Windows installer executables are produced for `windows/amd64` and `windows/arm64`
- `checksums.txt` is generated
- Windows archives are emitted as `.zip`
- the built binary reports the injected version rather than `dev`

## Prereleases

- Tags such as `v0.1.0-rc1` are published as GitHub prereleases.
- Homebrew tap uploads are skipped automatically for prerelease tags, so the stable tap formula keeps pointing at the latest stable release.
