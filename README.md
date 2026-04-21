# `dixa` CLI

`dixa` is a standalone Go CLI for the Dixa API. It is designed for two modes of use:

- Human operators working directly in a terminal
- AI agent skills that need a deterministic command surface with JSON output

It ships as a single `dixa` binary with a stable command surface, profile-aware authentication, TTY-friendly output for humans, and JSON output for automation.

## Alpha / Tech Preview

`dixa` is currently an alpha tech preview.

Expect rough edges, incomplete polish, and potential breaking changes between early releases. It is ready for internal evaluation and early adopter workflows, but you should expect the install flow, command ergonomics, and some endpoint coverage details to keep evolving.

## Highlights

- Single binary named `dixa`
- Full Dixa API command surface across organization, users, conversations, tags, teams, queues, settings, knowledge, custom attributes, and analytics
- TTY-aware output: readable tables for humans, JSON for non-interactive use
- Profile-based auth and config in `~/.config/dixa/config.toml`
- macOS Keychain support through `dixa auth login`
- Guard rails for write commands:
  - read commands never prompt
  - write commands prompt in interactive shells
  - non-interactive write commands require `--yes`
  - destructive commands require `--yes` even in a TTY

## Install

### Native Installers (Preferred)

The preferred end-user install path is the native installer attached to each GitHub Release:

- macOS: `dixa-<version>-macos-universal.pkg`
- Windows: `dixa-installer_<version>_windows_<arch>.exe`

See [docs/installers.md](./docs/installers.md) for platform-specific install behavior and fallback options.

### Fallback Options

PowerShell + GitHub (Windows fallback):

```powershell
irm https://raw.githubusercontent.com/Dixa-public/dixa-cli-public/main/scripts/install.ps1 | iex
```

Shell + GitHub (macOS fallback):

```bash
curl -fsSL https://raw.githubusercontent.com/Dixa-public/dixa-cli-public/main/scripts/install.sh | bash
```

Direct archive download:

Download the latest release archive from [GitHub Releases](https://github.com/Dixa-public/dixa-cli-public/releases), unpack it, and place `dixa` somewhere on your `PATH`.

Installed release binaries can also self-update with:

```bash
dixa update
```

`dixa` checks for newer stable releases at most once every 24 hours and prints a quiet stderr notice when an update is available. It never interrupts command success or changes stdout output.

## Getting Started

Start here for install, auth, first commands, and troubleshooting:

- [docs/getting-started.md](./docs/getting-started.md)
- [docs/command-examples.md](./docs/command-examples.md)

If you want a self-contained agent bundle for sandboxed AI workflows, each GitHub Release also includes `skill-v<version>.zip`.

## Quick Start

### 1. Store a profile in Keychain

```bash
dixa auth login --profile default --api-key YOUR_DIXA_API_KEY --set-default
```

If Keychain is not available, use an environment variable instead:

```bash
export DIXA_API_KEY="YOUR_DIXA_API_KEY"
export DIXA_OUTPUT="json"
```

### 2. Run a read command

```bash
dixa org get
dixa agents list --page-limit 10
dixa conversations search \
  --query '{"value":"refund","exactMatch":false}' \
  --output json
```

For filter-only search, the API expects a structured filter object:

```bash
dixa conversations search \
  --filters '{"strategy":"All","conditions":[{"field":{"operator":{"values":["email"],"_type":"IncludesOne"},"_type":"Channel"}}]}' \
  --output json
```

### 3. Run a write command

```bash
dixa --yes tags add --name vip --color red
```

### 4. Inspect the current auth/config resolution

```bash
dixa auth show
```

### 5. Update to the latest stable release

```bash
dixa update
```

## Global Flags

Every command supports the same top-level flags:

- `--profile`: choose which named profile to load from `~/.config/dixa/config.toml`
- `--base-url`: override the API base URL, which is useful for non-default environments
- `--api-key`: provide an API key directly for the current command instead of using Keychain, env vars, or a saved profile
- `--output auto|json|table|yaml`: control the response format; `auto` prefers tables in a TTY and machine-readable output in automation
- `--debug`: print request and resolution details to help troubleshoot config, auth, or API issues
- `--yes`: skip confirmation prompts for write commands; this is required for non-interactive mutations

## Config Resolution

Config precedence is:

1. Explicit command flags
2. Environment variables
3. Named profile in `~/.config/dixa/config.toml`
4. Built-in defaults

Supported environment variables:

- `DIXA_API_KEY`
- `DIXA_BASE_URL`
- `DIXA_PROFILE`
- `DIXA_OUTPUT`

Default API base URL:

```text
https://dev.dixa.io/v1
```

## JSON Input

Commands that take larger payloads also support `--input <file>` or `--input -`:

```bash
dixa --yes custom-attributes update-conversation-custom-attributes conv-123 \
  --input payload.json
```

## Analytics Workflow

Analytics commands intentionally preserve the explicit discovery flow:

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

Only move to `unaggregated-data` when aggregated output is insufficient.

## Repo Guide

- [SKILL.md](./SKILL.md): canonical agent-oriented skill guide for sandboxed `dixa` usage
- `skill-v<version>.zip`: release-bundled copy of the skill with local install scripts and a matching default CLI version
- [docs/getting-started.md](./docs/getting-started.md): setup, auth, examples, and troubleshooting
- [docs/command-examples.md](./docs/command-examples.md): structured example commands for the full CLI surface
- [docs/installers.md](./docs/installers.md): default macOS `.pkg` and Windows `.exe` installer flows, plus fallbacks
- [docs/parity-matrix.md](./docs/parity-matrix.md): generated surface parity table
- [docs/releasing.md](./docs/releasing.md): GitHub Releases, native installers, and the Claude skill bundle release flow

## Development

```bash
go test ./...
go build ./cmd/dixa
```
