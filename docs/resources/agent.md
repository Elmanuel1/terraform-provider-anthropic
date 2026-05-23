---
page_title: "anthropic: anthropic_agent"
subcategory: ""
description: |-
  Manages an Anthropic agent.
---

# Resource: anthropic_agent

Manages an Anthropic agent.

Supports two authentication modes, controlled by what is set in the **provider block**:

| Mode | Provider attributes required | `workspace_id` |
|---|---|---|
| WIF | `federation_rule_id`, `organization_id`, `service_account_id` | Required |
| Workspace API key | `workspace_api_key` | Not needed |

When both are configured, WIF takes precedence.

For Terraform Cloud WIF setup and debugging token exchange failures, see the [Authentication guide](../guides/authentication.md).

## Example Usage

### WIF authentication

```terraform
provider "anthropic" {
  federation_rule_id = var.anthropic_federation_rule_id
  organization_id    = var.anthropic_organization_id
  service_account_id = var.anthropic_service_account_id
}

resource "anthropic_workspace" "example" {
  name = "my-workspace"
}

resource "anthropic_agent" "example" {
  workspace_id = anthropic_workspace.example.id
  name         = "my-agent"
  model        = "claude-sonnet-4-6"
  system       = "You are a helpful assistant."
}
```

### Workspace API key authentication

```terraform
provider "anthropic" {
  workspace_api_key = var.anthropic_workspace_api_key
}

resource "anthropic_agent" "example" {
  name   = "my-agent"
  model  = "claude-sonnet-4-6"
  system = "You are a helpful assistant."
}
```

### Agent with MCP server

When `mcp_servers` is declared, every server must be referenced by a `mcp_toolset` entry in `tools`.

```terraform
resource "anthropic_vault" "example" {
  workspace_id = anthropic_workspace.example.id
  display_name = "my-vault"
}

resource "anthropic_vault_credential" "example" {
  workspace_id   = anthropic_workspace.example.id
  vault_id       = anthropic_vault.example.id
  display_name   = "erp-token"
  auth_type      = "static_bearer"
  mcp_server_url = "https://erp.example.com/mcp"
  token          = var.mcp_token
}

resource "anthropic_agent" "example" {
  workspace_id = anthropic_workspace.example.id
  name         = "procurement-agent"
  model        = "claude-opus-4-7"
  model_speed  = "standard"
  system       = "You are a procurement assistant."
  description  = "Handles purchase order workflows."

  tools = jsonencode([
    {
      type            = "mcp_toolset"
      mcp_server_name = "erp-server"
      default_config = {
        enabled           = true
        permission_policy = { type = "always_allow" }
      }
      configs = []
    }
  ])

  mcp_servers = jsonencode([
    {
      type = "url"
      name = "erp-server"
      url  = "https://erp.example.com/mcp"
    }
  ])

  metadata = {
    team = "procurement"
    env  = "production"
  }
}
```

~> **Note:** The `mcp_server_name` in each `mcp_toolset` tool entry must match the `name` of a declared `mcp_servers` entry exactly. The API rejects agents where a declared MCP server has no corresponding toolset.

### Agent with tool permission policies

Use `default_config` to set the default permission for all tools on an MCP server, and `configs` to override individual tools. Set `enabled = false` to hard-block a specific tool.

```terraform
resource "anthropic_agent" "example" {
  workspace_id = anthropic_workspace.example.id
  name         = "assistant"
  model        = "claude-sonnet-4-6"
  system       = "You are a helpful assistant."

  tools = jsonencode([
    {
      type            = "mcp_toolset"
      mcp_server_name = "slack"
      default_config = {
        enabled           = true
        permission_policy = { type = "always_allow" }
      }
      configs = [
        # Hard-block direct messages
        {
          name              = "slack_send_message"
          enabled           = false
          permission_policy = { type = "always_allow" }
        },
        # Require approval before scheduling
        {
          name              = "slack_schedule_message"
          enabled           = true
          permission_policy = { type = "always_ask" }
        }
      ]
    },
    {
      type            = "mcp_toolset"
      mcp_server_name = "confluence"
      default_config = {
        enabled           = true
        permission_policy = { type = "always_ask" }
      }
      configs = []
    }
  ])

  mcp_servers = jsonencode([
    { type = "url", name = "slack",      url = var.slack_mcp_url },
    { type = "url", name = "confluence", url = var.confluence_mcp_url }
  ])
}
```

`permission_policy.type` accepts:
- `always_allow` — the agent runs the tool automatically.
- `always_ask` — the agent pauses and asks the user to approve before running the tool.

### Agent with Anthropic skills

```terraform
resource "anthropic_agent" "example" {
  workspace_id = anthropic_workspace.example.id
  name         = "data-agent"
  model        = "claude-sonnet-4-6"
  system       = "You are a data analysis assistant."

  skills = jsonencode([
    { type = "anthropic", skill_id = "xlsx" },
    { type = "anthropic", skill_id = "web_search" }
  ])
}
```

### Multi-agent coordinator

```terraform
resource "anthropic_agent" "worker" {
  workspace_id = anthropic_workspace.example.id
  name         = "worker"
  model        = "claude-sonnet-4-6"
  system       = "You are a worker agent."
}

resource "anthropic_agent" "coordinator" {
  workspace_id = anthropic_workspace.example.id
  name         = "coordinator"
  model        = "claude-opus-4-7"

  multiagent = jsonencode({
    type   = "coordinator"
    agents = [anthropic_agent.worker.id]
  })
}
```

## Argument Reference

* `workspace_id` - (Optional, Forces new resource) Workspace ID. Required when using WIF authentication.
* `name` - (Required) Agent name.
* `model` - (Required) Model ID, e.g. `claude-opus-4-7` or `claude-sonnet-4-6`.
* `model_speed` - (Optional) Inference speed: `standard` (default) or `fast`.
* `system` - (Optional) System prompt.
* `description` - (Optional) Human-readable description.
* `tools` - (Optional) JSON-encoded tools array. Maximum 128 tools. Each declared `mcp_servers` entry must have a corresponding `{ type = "mcp_toolset", mcp_server_name = "..." }` entry here.
* `mcp_servers` - (Optional) JSON-encoded MCP servers array. Maximum 20 servers, names must be unique.
* `skills` - (Optional) JSON-encoded skills array. Maximum 20 skills.
* `multiagent` - (Optional) JSON-encoded multi-agent coordinator config.
* `metadata` - (Optional) Map of arbitrary string key-value pairs.

## Attribute Reference

* `id` - Agent ID (`agt_...`).
* `version` - Optimistic-lock version, incremented on each update.
* `created_at` - ISO 8601 creation timestamp.
* `updated_at` - ISO 8601 last-updated timestamp.
* `archived_at` - ISO 8601 archival timestamp, or null if active.

## Import

WIF (workspace_id known):

```shell
terraform import anthropic_agent.example wrks_xxx/agt_yyy
```

Workspace API key (workspace_id not needed):

```shell
terraform import anthropic_agent.example agt_yyy
```
