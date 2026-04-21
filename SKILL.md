---
name: dixa
description: Use the distributed Dixa CLI in sandboxed AI sessions with bundled install helpers, session-local bootstrapping, and JSON-first command patterns.
---

# Dixa CLI Skill Guide

Use `dixa` when you want a deterministic command-line interface to the Dixa API.

This skill should use the distributed CLI, not a source build. Because sandbox sessions are ephemeral, do not rely on a preinstalled `dixa` on `PATH`. Install `dixa` for the current session and invoke it through `DIXA_BIN`.

## Session Bootstrap

### macOS sandboxes

Install `dixa` into a session-local directory with the bundled installer helper:

```bash
export DIXA_VERSION="${DIXA_VERSION:-}"
export DIXA_SESSION_LABEL="${DIXA_SESSION_LABEL:-${DIXA_VERSION:-latest}}"
export DIXA_TMP_ROOT="${DIXA_TMP_ROOT:-${TMPDIR:-/tmp}}"
export DIXA_SESSION_DIR="${DIXA_SESSION_DIR:-$DIXA_TMP_ROOT/dixa-cli-$DIXA_SESSION_LABEL}"
mkdir -p "$DIXA_SESSION_DIR/bin"

if [[ -n "$DIXA_VERSION" ]]; then
  INSTALL_DIR="$DIXA_SESSION_DIR/bin" DIXA_VERSION="$DIXA_VERSION" ./scripts/install.sh
else
  INSTALL_DIR="$DIXA_SESSION_DIR/bin" ./scripts/install.sh
fi

export DIXA_BIN="$DIXA_SESSION_DIR/bin/dixa"
"$DIXA_BIN" --version
```

### Linux sandboxes

Install `dixa` into a session-local directory with the bundled installer helper:

```bash
export DIXA_VERSION="${DIXA_VERSION:-}"
export DIXA_SESSION_LABEL="${DIXA_SESSION_LABEL:-${DIXA_VERSION:-latest}}"
export DIXA_TMP_ROOT="${DIXA_TMP_ROOT:-${TMPDIR:-/tmp}}"
export DIXA_SESSION_DIR="${DIXA_SESSION_DIR:-$DIXA_TMP_ROOT/dixa-cli-$DIXA_SESSION_LABEL}"
mkdir -p "$DIXA_SESSION_DIR/bin"

if [[ -n "$DIXA_VERSION" ]]; then
  INSTALL_DIR="$DIXA_SESSION_DIR/bin" DIXA_VERSION="$DIXA_VERSION" ./scripts/install.sh
else
  INSTALL_DIR="$DIXA_SESSION_DIR/bin" ./scripts/install.sh
fi

export DIXA_BIN="$DIXA_SESSION_DIR/bin/dixa"
"$DIXA_BIN" --version
```

### Windows sandboxes

Install `dixa` into a session-local directory with the bundled installer helper:

```powershell
$env:DIXA_VERSION = if ($env:DIXA_VERSION) { $env:DIXA_VERSION } else { "" }
$env:DIXA_SESSION_LABEL = if ($env:DIXA_SESSION_LABEL) {
  $env:DIXA_SESSION_LABEL
} elseif ($env:DIXA_VERSION) {
  $env:DIXA_VERSION
} else {
  "latest"
}
$env:DIXA_SESSION_DIR = if ($env:DIXA_SESSION_DIR) {
  $env:DIXA_SESSION_DIR
} else {
  Join-Path $env:TEMP "dixa-cli-$env:DIXA_SESSION_LABEL"
}

$env:INSTALL_DIR = Join-Path $env:DIXA_SESSION_DIR "bin"
& ./scripts/install.ps1

$env:DIXA_BIN = Join-Path $env:INSTALL_DIR "dixa.exe"
& $env:DIXA_BIN --version
```

If a published release asset is not available in the sandbox, report that the skill is blocked on package installation rather than building the CLI from source.

## Environment

Prefer environment variables inside the skill:

```bash
export DIXA_API_KEY="YOUR_DIXA_API_KEY"
export DIXA_OUTPUT="json"
```

Optional override:

```bash
export DIXA_BASE_URL="https://dev.dixa.io/v1"
```

## Core Rules

Always run the CLI with JSON output inside the skill.

Use `"$DIXA_BIN"` for commands inside the skill. Do not assume `dixa` is globally available even after install.

Run the bootstrap snippets from the directory that contains this `SKILL.md`, so the relative `./scripts/install.sh` and `./scripts/install.ps1` paths resolve correctly.

Do not ask the user for IDs if you can discover them with `list` or `get` commands first.

Read commands do not need confirmation.

Any mutating command requires explicit user confirmation first, then rerun the command with `--yes`.

Treat anonymize, remove, and delete style commands as potentially irreversible.

Do not invent payload shapes. If a command needs nested JSON, use `--input` or pass the exact documented JSON object.

Prefer aggregated analytics first. Only use unaggregated analytics when aggregated data is clearly insufficient.

## Dixa Terms

End users are customers or contacts.

Agents are support staff handling conversations.

Admins are represented under the same `agents` entity in the API.

Many write operations require IDs for tags, teams, queues, agents, conversations, or custom attributes. Discover those IDs first.

## Recommended Command Style

Use simple read commands first:

```bash
"$DIXA_BIN" org get --output json
"$DIXA_BIN" agents list --page-limit 20 --output json
"$DIXA_BIN" end-users list --page-limit 20 --output json
```

For nested payloads, prefer `--input`:

```bash
"$DIXA_BIN" --yes custom-attributes update-conversation-custom-attributes conv-123 \
  --input payload.json \
  --output json
```

You can also pipe JSON through stdin:

```bash
printf '{"attr-1":"gold"}' | \
  "$DIXA_BIN" --yes custom-attributes update-conversation-custom-attributes conv-123 \
  --input - \
  --output json
```

## Conversation Search Rule

`conversations search` does not accept a bare array for `--filters`.

Text search:

```bash
"$DIXA_BIN" conversations search \
  --query '{"value":"refund","exactMatch":false}' \
  --output json
```

Filter-only search:

```bash
"$DIXA_BIN" conversations search \
  --filters '{"strategy":"All","conditions":[{"field":{"operator":{"values":["email"],"_type":"IncludesOne"},"_type":"Channel"}}]}' \
  --output json
```

Combined text + filters:

```bash
"$DIXA_BIN" conversations search \
  --query '{"value":"refund","exactMatch":false}' \
  --filters '{"strategy":"All","conditions":[{"field":{"operator":{"values":["email"],"_type":"IncludesOne"},"_type":"Channel"}}]}' \
  --output json
```

## Write Safety Rule

For any mutating command:

1. Ask the user for explicit confirmation first.
2. After confirmation, rerun with `--yes`.

Example:

```bash
"$DIXA_BIN" --yes tags add --name vip --color red --output json
```

## Common Workflow Patterns

### Tags

To tag a conversation:

1. Discover the tag ID.
2. Discover or confirm the conversation ID.
3. Only then mutate.

```bash
"$DIXA_BIN" tags list-tags --output json
"$DIXA_BIN" conversations get conv-123 --output json
"$DIXA_BIN" --yes conversations tag conv-123 tag-456 --output json
```

To remove a tag from a conversation:

```bash
"$DIXA_BIN" tags list conv-123 --output json
"$DIXA_BIN" --yes conversations remove conv-123 tag-456 --output json
```

### Conversations

Most conversation mutations require a `conversation_id`.

Find conversations first with either:

```bash
"$DIXA_BIN" conversations get conv-123 --output json
```

or:

```bash
"$DIXA_BIN" conversations search \
  --query '{"value":"refund","exactMatch":false}' \
  --output json
```

To claim a conversation, discover both the conversation and the agent:

```bash
"$DIXA_BIN" conversations get conv-123 --output json
"$DIXA_BIN" agents list --output json
"$DIXA_BIN" --yes conversations assign conv-123 --agent-id agent-456 --output json
```

To start a conversation, first find or create the requester and, for outbound flows, an agent:

```bash
"$DIXA_BIN" end-users list --output json
"$DIXA_BIN" agents list --output json
```

### Teams and Queues

To assign agents to a team:

```bash
"$DIXA_BIN" agents list --output json
"$DIXA_BIN" teams list --output json
"$DIXA_BIN" --yes teams add team-123 --agent-ids agent-1 --agent-ids agent-2 --output json
```

To assign agents to a queue:

```bash
"$DIXA_BIN" agents list --output json
"$DIXA_BIN" queues list --output json
"$DIXA_BIN" --yes queues assign queue-123 --agent-ids agent-1 --agent-ids agent-2 --output json
```

### Knowledge Base

To create or update an article, discover the current categories or articles first:

```bash
"$DIXA_BIN" knowledge list-knowledge-categories --output json
"$DIXA_BIN" knowledge list --output json
```

Then mutate only after confirmation:

```bash
"$DIXA_BIN" --yes knowledge add --title "FAQ" --content "Answer" --output json
```

### Custom Attributes

Do not guess custom attribute IDs.

First discover the definitions:

```bash
"$DIXA_BIN" custom-attributes list --output json
```

Then update the target entity with a UUID-to-value map:

```bash
"$DIXA_BIN" --yes custom-attributes update-conversation-custom-attributes conv-123 \
  --custom-attributes '{"2f5515b6-7e98-4f4d-9010-bfd2a27d4f35":"012345"}' \
  --output json
```

## Analytics Workflow

This workflow should be treated as mandatory inside the skill.

1. Discover available metrics.
2. Inspect the chosen metric to get valid filters and aggregations.
3. Fetch aggregated data first.
4. Only then consider unaggregated records if summary data is insufficient.

Discover metrics:

```bash
"$DIXA_BIN" analytics prepare-analytics-metric-query --output json
```

Inspect one metric:

```bash
"$DIXA_BIN" analytics prepare-analytics-metric-query \
  --metric-id closed_conversations \
  --output json
```

Fetch aggregated data:

```bash
"$DIXA_BIN" analytics aggregated-data \
  --metric-id closed_conversations \
  --timezone UTC \
  --filters '[{"attribute":"channel","values":["email"]}]' \
  --aggregations '["Count"]' \
  --output json
```

Only use unaggregated data when aggregated output is not enough, and only after the aggregated workflow above:

```bash
"$DIXA_BIN" analytics prepare-analytics-record-query --output json
"$DIXA_BIN" analytics prepare-analytics-record-query --record-id ratings --output json
"$DIXA_BIN" analytics unaggregated-data --record-id ratings --output json
```

Important interpretation rule for nested or pre-aggregated metrics:

- `Count` can mean number of groups, not total underlying items.
- `Sum` can represent the total across those groups.

Do not assume a grouped metric gives per-agent or per-queue breakdowns in one call.

## Skill Behavior Summary

Start with read commands.

Discover required IDs before mutations.

Use exact JSON payloads for nested inputs.

Ask for explicit user confirmation before every mutation.

Use `--yes` only after confirmation.

Use analytics in the documented aggregated-first order.
