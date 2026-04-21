# Command Examples

This guide shows one concrete example for every command surface in `dixa`.

It is intended as a structured cookbook rather than a strict API contract. The examples are designed to show how interaction with the CLI typically looks in a terminal, with realistic placeholder IDs and payloads you can replace with your own values.

## Conventions

- Examples assume you already authenticated with `dixa auth login` or exported `DIXA_API_KEY`.
- Replace placeholder IDs such as `agent-123`, `conv-123`, `user-123`, and `tag-123` with real values from your environment.
- Read commands are shown without `--yes`.
- Write commands are shown with `--yes` so the examples work in both interactive and non-interactive flows.
- Add `--output json` when you want machine-readable output for automation or AI tooling.
- For larger JSON payloads, you can often move the payload into a file and pass it with `--input payload.json` or `--input -`.

## Root And Discovery

### `dixa --help`
Show the top-level command groups.

```bash
dixa --help
```

### `dixa --version`
Print the installed CLI version.

```bash
dixa --version
```

### `dixa update`
Install the latest stable release over the current release binary.

```bash
dixa update
```

### `dixa <group> --help`
Inspect the commands available in a command group.

```bash
dixa conversations --help
```

### `dixa <group> <command> --help`
Inspect the flags and arguments for a specific command.

```bash
dixa conversations search --help
```

## Auth

### `dixa auth login`
Store an API key in Keychain for a named profile.

```bash
dixa auth login \
  --profile default \
  --api-key "YOUR_DIXA_API_KEY" \
  --set-default
```

### `dixa auth show`
Show the resolved profile, base URL, output mode, and masked API key.

```bash
dixa auth show
```

### `dixa auth logout`
Remove the saved API key for a profile.

```bash
dixa auth logout \
  --profile default \
  --remove-profile
```

## Completion

### `dixa completion --help`
Show the supported shell completion targets.

```bash
dixa completion --help
```

### `dixa completion bash`
Write Bash completion to a local file you can source or install yourself.

```bash
dixa completion bash > ./dixa.bash
```

### `dixa completion zsh`
Write Zsh completion to a local file you can place on your `fpath`.

```bash
dixa completion zsh > ./_dixa
```

### `dixa completion fish`
Install Fish completion for the current user.

```bash
dixa completion fish > ~/.config/fish/completions/dixa.fish
```

### `dixa completion powershell`
Load PowerShell completion in the current session.

```powershell
dixa completion powershell | Out-String | Invoke-Expression
```

## Organization

### `dixa org get`
Fetch organization details.

```bash
dixa org get
```

### `dixa org details`
Fetch organization details with the alternate command name.

```bash
dixa org details
```

## Settings

### `dixa settings list`
List contact endpoints in the organization.

```bash
dixa settings list --type email
```

### `dixa settings get`
Get a specific contact endpoint by ID.

```bash
dixa settings get contact-endpoint-123
```

### `dixa settings list-schedules`
List business-hours schedules.

```bash
dixa settings list-schedules
```

### `dixa settings check-business-hours-status`
Check whether a business-hours schedule is open at a given time.

```bash
dixa settings check-business-hours-status \
  schedule-123 \
  --timestamp "2026-04-21T09:00:00Z"
```

## Agents

### `dixa agents add`
Create a new agent or admin.

```bash
dixa --yes agents add \
  --display-name "Jane Doe" \
  --email "jane@example.com" \
  --phone-number "+4512345678"
```

### `dixa agents get`
Get a single agent by ID.

```bash
dixa agents get agent-123
```

### `dixa agents list`
List agents with optional filters and pagination.

```bash
dixa agents list \
  --email "jane@example.com" \
  --page-limit 20
```

### `dixa agents list-presence`
List presence status for all agents.

```bash
dixa agents list-presence
```

### `dixa agents list-teams`
List the teams for a specific agent.

```bash
dixa agents list-teams agent-123
```

### `dixa agents patch`
Patch selected fields on an agent.

```bash
dixa --yes agents patch \
  agent-123 \
  --display-name "Jane Q. Doe" \
  --avatar-url "https://example.com/avatar.jpg"
```

### `dixa agents set-agent-working-channel`
Update an agent's working-channel status.

```bash
dixa --yes agents set-agent-working-channel \
  agent-123 \
  --channel email \
  --working true
```

### `dixa agents update`
Replace an agent's fields with a full update.

```bash
dixa --yes agents update \
  agent-123 \
  --display-name "Jane Doe" \
  --email "jane@example.com" \
  --phone-number "+4512345678"
```

## End Users

### `dixa end-users add`
Create a single end user.

```bash
dixa --yes end-users add \
  --display-name "Alice Example" \
  --email "alice@example.com" \
  --external-id "crm-1001"
```

### `dixa end-users add-bulk`
Create multiple end users in a bulk action.

```bash
dixa --yes end-users add-bulk \
  --end-users '[{"displayName":"Alice Example","email":"alice@example.com"},{"displayName":"Bob Example","phoneNumber":"+4511111111"}]'
```

### `dixa end-users anonymize`
Request anonymization for an end user.

```bash
dixa --yes end-users anonymize user-123 --force true
```

### `dixa end-users get`
Get an end user by ID.

```bash
dixa end-users get user-123
```

### `dixa end-users list`
List end users with pagination.

```bash
dixa end-users list --page-limit 20
```

### `dixa end-users list-conversations`
List conversations for a specific end user.

```bash
dixa end-users list-conversations user-123 --page-limit 20
```

### `dixa end-users modify-end-users-bulk`
Patch multiple end users in one bulk request.

```bash
dixa --yes end-users modify-end-users-bulk \
  --end-users '[{"id":"user-123","displayName":"Alice Updated"},{"id":"user-456","externalId":"crm-2002"}]'
```

### `dixa end-users patch`
Patch selected fields on an end user.

```bash
dixa --yes end-users patch \
  user-123 \
  --email "alice.updated@example.com" \
  --display-name "Alice Updated"
```

### `dixa end-users update`
Fully update an end user.

```bash
dixa --yes end-users update \
  user-123 \
  --display-name "Alice Example" \
  --email "alice@example.com" \
  --external-id "crm-1001"
```

### `dixa end-users update-end-users-bulk`
Fully update multiple end users in one bulk request.

```bash
dixa --yes end-users update-end-users-bulk \
  --end-users '[{"id":"user-123","displayName":"Alice Example","email":"alice@example.com"},{"id":"user-456","displayName":"Bob Example","email":"bob@example.com"}]'
```

## Conversations

### `dixa conversations get`
Fetch a conversation by ID.

```bash
dixa conversations get conv-123
```

### `dixa conversations search`
Search conversations by text, filters, or both.

```bash
dixa conversations search \
  --query '{"value":"refund","exactMatch":false}' \
  --filters '{"strategy":"All","conditions":[{"field":{"operator":{"values":["email"],"_type":"IncludesOne"},"_type":"Channel"}}]}' \
  --page-limit 20
```

### `dixa conversations list`
List linked conversations for a conversation.

```bash
dixa conversations list conv-123
```

### `dixa conversations list-activity-log`
Show the activity log for one conversation.

```bash
dixa conversations list-activity-log conv-123
```

### `dixa conversations list-organization-activity-log`
Show organization-wide conversation activity.

```bash
dixa conversations list-organization-activity-log
```

### `dixa conversations list-flows`
List flows associated with a conversation.

```bash
dixa conversations list-flows conv-123
```

### `dixa conversations list-messages`
List messages in a conversation.

```bash
dixa conversations list-messages conv-123
```

### `dixa conversations list-notes`
List internal notes on a conversation.

```bash
dixa conversations list-notes conv-123
```

### `dixa conversations list-ratings`
List ratings on a conversation.

```bash
dixa conversations list-ratings conv-123
```

### `dixa conversations add-note`
Add a single internal note to a conversation.

```bash
dixa --yes conversations add-note \
  conv-123 \
  --message "Escalated to billing for manual review." \
  --agent-id agent-123
```

### `dixa conversations add-notes-bulk`
Add multiple internal notes in a single request.

```bash
dixa --yes conversations add-notes-bulk \
  conv-123 \
  --notes '[{"message":"Initial QA note"},{"message":"Second internal note"}]'
```

### `dixa conversations assign`
Assign or claim a conversation for an agent.

```bash
dixa --yes conversations assign \
  conv-123 \
  --agent-id agent-123 \
  --force true
```

### `dixa conversations close`
Close a conversation.

```bash
dixa --yes conversations close \
  conv-123 \
  --user-id agent-123
```

### `dixa conversations reopen`
Reopen a previously closed conversation.

```bash
dixa --yes conversations reopen conv-123
```

### `dixa conversations tag`
Tag a conversation with a known tag ID.

```bash
dixa --yes conversations tag conv-123 tag-123
```

### `dixa conversations remove`
Remove a tag from a conversation.

```bash
dixa --yes conversations remove conv-123 tag-123
```

### `dixa conversations tag-conversation-bulk`
Apply multiple tag names to a conversation asynchronously.

```bash
dixa --yes conversations tag-conversation-bulk \
  conv-123 \
  --tag-names billing \
  --tag-names vip
```

### `dixa conversations link`
Link a conversation to a parent conversation.

```bash
dixa --yes conversations link \
  conv-123 \
  --parent-conversation-id conv-parent-123
```

### `dixa conversations anonymize`
Request anonymization of a conversation.

```bash
dixa --yes conversations anonymize conv-123 --force true
```

### `dixa conversations anonymize-conversation-message`
Request anonymization of a single conversation message.

```bash
dixa --yes conversations anonymize-conversation-message \
  conv-123 \
  msg-123
```

### `dixa conversations set-conversation-followup-status`
Toggle follow-up status on a conversation.

```bash
dixa --yes conversations set-conversation-followup-status \
  conv-123 \
  --follow-up true
```

### `dixa conversations start`
Create a new conversation from the CLI.

```bash
dixa --yes conversations start \
  --requester-id user-123 \
  --conversation-type chat \
  --message-type text \
  --message-content "Hello from the CLI"
```

### `dixa conversations import`
Import conversations in bulk.

```bash
dixa --yes conversations import \
  --conversations '[{"requesterId":"user-123","conversationType":"email","messageType":"text","messageContent":"Imported message"}]'
```

## Tags

### `dixa tags add`
Create a new tag.

```bash
dixa --yes tags add --name vip --color red
```

### `dixa tags get`
Get a tag by ID.

```bash
dixa tags get tag-123
```

### `dixa tags list-tags`
List tags in the organization.

```bash
dixa tags list-tags --include-deactivated
```

### `dixa tags list`
List the tags currently attached to a conversation.

```bash
dixa tags list conv-123
```

### `dixa tags activate`
Activate a deactivated tag.

```bash
dixa --yes tags activate tag-123
```

### `dixa tags deactivate`
Deactivate a tag without deleting it.

```bash
dixa --yes tags deactivate tag-123
```

### `dixa tags remove`
Delete a tag and its associations.

```bash
dixa --yes tags remove tag-123
```

## Teams

### `dixa teams add-team`
Create a new team.

```bash
dixa --yes teams add-team --name "Support EMEA"
```

### `dixa teams get`
Get a team by ID.

```bash
dixa teams get team-123
```

### `dixa teams list`
List all teams.

```bash
dixa teams list
```

### `dixa teams list-agents`
List the agents in a team.

```bash
dixa teams list-agents team-123
```

### `dixa teams list-presence`
List presence for agents in a team.

```bash
dixa teams list-presence team-123 --page-limit 50
```

### `dixa teams add`
Add agents to an existing team.

```bash
dixa --yes teams add \
  team-123 \
  --agent-ids agent-123 \
  --agent-ids agent-456
```

### `dixa teams remove`
Remove agents from an existing team.

```bash
dixa --yes teams remove \
  team-123 \
  --agent-ids agent-123 \
  --agent-ids agent-456
```

### `dixa teams remove-team`
Delete a team.

```bash
dixa --yes teams remove-team team-123
```

## Queues

### `dixa queues add`
Create a queue with a few common flags.

```bash
dixa --yes queues add \
  --name "VIP Queue" \
  --priority 100 \
  --is-default false
```

### `dixa queues get`
Get a queue by ID.

```bash
dixa queues get queue-123
```

### `dixa queues list`
List queues in the organization.

```bash
dixa queues list
```

### `dixa queues list-agents`
List the agents assigned to a queue.

```bash
dixa queues list-agents queue-123
```

### `dixa queues assign`
Assign agents to a queue.

```bash
dixa --yes queues assign \
  queue-123 \
  --agent-ids agent-123 \
  --agent-ids agent-456
```

### `dixa queues remove`
Remove agents from a queue.

```bash
dixa --yes queues remove \
  queue-123 \
  --agent-ids agent-123 \
  --agent-ids agent-456
```

### `dixa queues check-queue-availability`
Check queue availability.

```bash
dixa queues check-queue-availability queue-123
```

### `dixa queues check-conversation-queue-position`
Check where a conversation currently sits in a queue.

```bash
dixa queues check-conversation-queue-position \
  queue-123 \
  conv-123
```

## Knowledge

### `dixa knowledge add`
Create a knowledge article.

```bash
dixa --yes knowledge add \
  --title "Refund Policy" \
  --content "Refunds are processed within 5 business days." \
  --published true
```

### `dixa knowledge get`
Get a knowledge article by ID.

```bash
dixa knowledge get article-123
```

### `dixa knowledge list`
List knowledge articles with pagination.

```bash
dixa knowledge list --page-limit 20
```

### `dixa knowledge add-knowledge-category`
Create a knowledge category.

```bash
dixa --yes knowledge add-knowledge-category \
  --name "Billing" \
  --parent-id category-root-123
```

### `dixa knowledge list-knowledge-categories`
List knowledge categories.

```bash
dixa knowledge list-knowledge-categories --page-limit 20
```

### `dixa knowledge modify-knowledge-article`
Patch an existing knowledge article.

```bash
dixa --yes knowledge modify-knowledge-article \
  article-123 \
  --title "Updated Refund Policy" \
  --published true
```

### `dixa knowledge remove`
Delete a knowledge article.

```bash
dixa --yes knowledge remove article-123
```

## Custom Attributes

### `dixa custom-attributes list`
List the custom attribute definitions in the organization.

```bash
dixa custom-attributes list
```

### `dixa custom-attributes get`
Get a custom attribute definition by ID.

```bash
dixa custom-attributes get custom-attribute-123
```

### `dixa custom-attributes update-conversation-custom-attributes`
Patch custom attribute values on a conversation.

```bash
dixa --yes custom-attributes update-conversation-custom-attributes \
  conv-123 \
  --custom-attributes '{"uuid-1":"gold","uuid-2":true}'
```

### `dixa custom-attributes update-end-user-custom-attributes`
Patch custom attribute values on an end user.

```bash
dixa --yes custom-attributes update-end-user-custom-attributes \
  user-123 \
  --custom-attributes '{"uuid-1":"enterprise","uuid-2":"north-europe"}'
```

## Analytics

Analytics is usually a discovery-first workflow:

1. Inspect available metrics or record types.
2. Pick the `metric-id` or `record-id` you need.
3. Fetch aggregated or unaggregated data with filters.

### `dixa analytics prepare-analytics-metric-query`
Inspect available metrics or inspect one metric in detail.

```bash
dixa analytics prepare-analytics-metric-query \
  --metric-id closed_conversations \
  --output json
```

### `dixa analytics aggregated-data`
Fetch aggregated analytics for a metric.

```bash
dixa analytics aggregated-data \
  --metric-id closed_conversations \
  --timezone UTC \
  --filters '[{"attribute":"channel","values":["email"]}]' \
  --aggregations '["Count"]' \
  --output json
```

### `dixa analytics prepare-analytics-record-query`
Inspect the available unaggregated record types before fetching records.

```bash
dixa analytics prepare-analytics-record-query --output json
```

### `dixa analytics unaggregated-data`
Fetch record-level analytics with a known record ID from `prepare-analytics-record-query`.

```bash
dixa analytics unaggregated-data \
  --record-id your-record-id \
  --timezone UTC \
  --page-limit 50 \
  --output json
```

## Example Workflow

If you are not sure where to begin, this is a common pattern:

1. Verify auth:

   ```bash
   dixa auth show
   ```

2. Discover IDs:

   ```bash
   dixa agents list --page-limit 20
   dixa tags list-tags
   dixa conversations search --query '{"value":"refund","exactMatch":false}'
   ```

3. Inspect the target record:

   ```bash
   dixa conversations get conv-123
   ```

4. Perform the mutation:

   ```bash
   dixa --yes conversations tag conv-123 tag-123
   ```

5. Verify the result:

   ```bash
   dixa tags list conv-123
   ```
