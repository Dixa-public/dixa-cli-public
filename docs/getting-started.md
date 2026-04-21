# Getting Started

`dixa` is a terminal CLI for the Dixa API. This guide covers install, authentication, first commands, and the command patterns you will use most often.

If you want a broader cookbook with one example for every command group and subcommand, see [command-examples.md](./command-examples.md).

If you are setting up `dixa` inside a sandboxed AI tool, the matching GitHub Release also includes `skill-v<version>.zip`. That bundle includes the skill instructions, local install helpers, and bundled Linux binaries for sandbox environments that cannot download GitHub release assets directly.

## 1. Install `dixa`

### Preferred native installer

Use the installer asset attached to the release for your platform:

- macOS: `dixa-<version>-macos-universal.pkg`
- Windows: `dixa-installer_<version>_windows_<arch>.exe`

Linux currently uses the release archive or shell installer fallback instead of a native installer.

For installer behavior details and platform-specific notes, see [installers.md](./installers.md).

### Fallback options

PowerShell + GitHub (Windows fallback):

```powershell
irm https://raw.githubusercontent.com/Dixa-public/dixa-cli-public/main/scripts/install.ps1 | iex
```

Shell + GitHub (macOS/Linux fallback):

```bash
curl -fsSL https://raw.githubusercontent.com/Dixa-public/dixa-cli-public/main/scripts/install.sh | bash
```

Direct archive download:

Download the latest archive from [GitHub Releases](https://github.com/Dixa-public/dixa-cli-public/releases), unpack it, and place `dixa` on your `PATH`.

### Installer notes

- The macOS `.pkg` installs `dixa` to `/usr/local/bin`.
- The Windows `.exe` installs `dixa.exe` into `%LOCALAPPDATA%\Programs\dixa\bin` and updates the user `Path`.

Confirm the install:

```bash
dixa --version
```

Update to the latest stable release later with:

```bash
dixa update
```

## 2. Choose an auth method

### Option A: Keychain-backed profile

Best for interactive use on macOS.

```bash
dixa auth login --profile default --api-key YOUR_DIXA_API_KEY --set-default
```

Check what the CLI resolves:

```bash
dixa auth show
```

### Option B: Environment variable

Best for temporary sessions, CI, and agent workflows.

```bash
export DIXA_API_KEY="YOUR_DIXA_API_KEY"
export DIXA_OUTPUT="json"
```

Optional:

```bash
export DIXA_BASE_URL="https://dev.dixa.io/v1"
export DIXA_PROFILE="default"
```

## 3. Run your first commands

### Read commands

```bash
dixa org get
dixa agents list --page-limit 10
dixa conversations get conv-123
```

### JSON output

```bash
dixa org get --output json
dixa agents list --page-limit 10 --output json
```

`--output auto` is the default. In terminals it prefers readable tables. For automation, use `--output json`.

## 4. Search and discovery

Many write commands need IDs first. Discover them with read commands before mutating anything.

### Find tags

```bash
dixa tags list-tags --output json
```

### Search conversations

Text search:

```bash
dixa conversations search \
  --query '{"value":"refund","exactMatch":false}' \
  --output json
```

Filter-only search:

```bash
dixa conversations search \
  --filters '{"strategy":"All","conditions":[{"field":{"operator":{"values":["email"],"_type":"IncludesOne"},"_type":"Channel"}}]}' \
  --output json
```

## 5. Mutating commands

`dixa` is intentionally cautious:

- read commands never prompt
- write commands prompt in interactive shells
- non-interactive write commands require `--yes`
- destructive commands require `--yes` even in a TTY

Example:

```bash
dixa --yes tags add --name vip --color red
```

## 6. Nested JSON input

For larger payloads, use `--input`.

From a file:

```bash
dixa --yes custom-attributes update-conversation-custom-attributes conv-123 \
  --input payload.json \
  --output json
```

From stdin:

```bash
printf '{"attr-1":"gold"}' | \
  dixa --yes custom-attributes update-conversation-custom-attributes conv-123 \
  --input - \
  --output json
```

## 7. Analytics workflow

Use analytics in this order:

1. Discover metrics
2. Inspect the metric
3. Fetch aggregated data
4. Use unaggregated data only when summary data is not enough

```bash
dixa analytics prepare-analytics-metric-query --output json
dixa analytics prepare-analytics-metric-query --metric-id closed_conversations --output json
dixa analytics aggregated-data \
  --metric-id closed_conversations \
  --timezone UTC \
  --filters '[{"attribute":"channel","values":["email"]}]' \
  --aggregations '["Count"]' \
  --output json
```

## 8. Troubleshooting

### `dixa: command not found`

- open a fresh shell after install
- confirm the binary is on `PATH`

### `dixa version dev`

- this is expected for plain local builds
- distributed release binaries should report a real semver like `0.1.0`
- `dixa update` only works on release binaries, not local `dev` builds

### Update notice after commands

- `dixa` may print a quiet stderr notice when a newer stable release is available
- the notice is non-blocking and does not change stdout output
- run `dixa update` to install the latest stable release

### No Keychain access

Use `DIXA_API_KEY` instead of `dixa auth login`.

### Want agent-oriented usage guidance

See [SKILL.md](../SKILL.md) for the recommended behavior when `dixa` is used inside AI workflows.
