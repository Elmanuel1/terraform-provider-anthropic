---
page_title: "anthropic: anthropic_deployment"
subcategory: ""
description: |-
  Manages an Anthropic Managed Agents deployment.
---

# Resource: anthropic_deployment

Manages an Anthropic Managed Agents deployment: an agent bound to an environment with seed events and an optional cron schedule. Each run spawns a fresh session seeded with `initial_events`.

Omit the `schedule` block for a manual (on-demand) deployment. Include it to run automatically on a cron schedule.

Supports two authentication modes, controlled by what is set in the **provider block**:

| Mode | Provider attributes required | `workspace_id` |
|---|---|---|
| WIF | `federation_rule_id`, `organization_id`, `service_account_id` | Required |
| Workspace API key | `workspace_api_key` | Not needed |

When both are configured, WIF takes precedence.

On destroy the deployment is archived (soft delete). There is no hard delete.

## Example Usage

### Scheduled deployment (workspace API key)

```terraform
provider "anthropic" {
  workspace_api_key = var.anthropic_workspace_api_key
}

resource "anthropic_deployment" "nightly" {
  name           = "nightly-inbox-triage"
  environment_id = anthropic_environment.sandbox.id

  agent = {
    id = anthropic_agent.triage.id
  }

  initial_events = [
    {
      type = "user.message"
      content = [
        { type = "text", text = "Summarize today's support tickets and post to #digest." }
      ]
    }
  ]

  schedule = {
    expression = "0 9 * * 1-5"
    timezone   = "America/Los_Angeles"
  }
}
```

### Manual (on-demand) deployment

Omit `schedule`. The deployment runs only when triggered out of band (dashboard or API).

```terraform
resource "anthropic_deployment" "on_demand" {
  name           = "manual-triage"
  environment_id = anthropic_environment.sandbox.id

  agent = {
    id      = anthropic_agent.triage.id
    version = 3
  }

  initial_events = [
    {
      type    = "user.message"
      content = [{ type = "text", text = "Run triage now." }]
    }
  ]
}
```

## Argument Reference

- `name` (Required) Human-readable name.
- `agent` (Required) The agent to deploy.
  - `id` (Required) Agent ID (`agent_...`).
  - `version` (Optional) Agent version to pin. Omit to use the latest version at create time; the resolved version is then stored and not auto-updated.
- `environment_id` (Required) ID of the environment where sessions run.
- `initial_events` (Required) Between 1 and 50 events sent to each session at the start of every run. Text-only this iteration.
  - `type` (Required) Currently only `"user.message"`.
  - `content` (Required) Content blocks.
    - `type` (Required) Currently only `"text"`.
    - `text` (Required) The text content.
- `schedule` (Optional) Cron schedule. Omit for a manual deployment.
  - `expression` (Required) 5-field POSIX cron expression.
  - `timezone` (Required) IANA timezone identifier.
- `vault_ids` (Optional) Vault IDs supplying stored credentials for sessions.
- `metadata` (Optional) Arbitrary string key-value metadata (max 16 pairs).
- `paused` (Optional) Whether the deployment is paused. Defaults to `false`. Toggling calls the pause/unpause endpoints.
- `workspace_id` (Optional) Workspace ID. Required when using WIF. Changing it forces a new resource.

## Attribute Reference

- `id` Deployment ID (`depl_...`).
- `status` Lifecycle status: `"active"` or `"paused"`.
- `paused_reason` Why the deployment is paused (null when active): `type` and optional `error_type`.
- `schedule.last_run_at`, `schedule.upcoming_runs_at` Computed run timestamps.
- `created_at`, `updated_at`, `archived_at` Timestamps (RFC 3339).

## Import

Import by deployment ID, or `workspace_id/deployment_id` when using WIF:

```shell
terraform import anthropic_deployment.nightly depl_abc123
terraform import anthropic_deployment.nightly wrkspc_xxx/depl_abc123
```
