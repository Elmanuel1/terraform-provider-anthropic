---
page_title: "anthropic-wif_agent Resource"
description: |-
  Manages an Anthropic agent.
---

# anthropic-wif_agent

Manages an Anthropic agent. Agents are workspace-scoped and authenticate via WIF bearer token.

## Example Usage

### Minimal agent

```terraform
resource "anthropic-wif_agent" "example" {
  workspace_id = anthropic-wif_workspace.example.id
  name         = "my-agent"
  model        = "claude-sonnet-4-6"
  system       = "You are a helpful assistant."
}
```

### Agent with tools and MCP servers

```terraform
resource "anthropic-wif_agent" "example" {
  workspace_id = anthropic-wif_workspace.example.id
  name         = "procurement-agent"
  model        = "claude-opus-4-7"
  model_speed  = "standard"
  system       = "You are a procurement assistant."
  description  = "Handles purchase order workflows."

  tools = jsonencode([
    { "type" = "agent_toolset_20260401" }
  ])

  mcp_servers = jsonencode([
    {
      name = "erp-server"
      type = "url"
      url  = "https://erp.example.com/mcp"
    }
  ])

  metadata = {
    team = "procurement"
    env  = "production"
  }
}
```

### Multi-agent coordinator

```terraform
resource "anthropic-wif_agent" "coordinator" {
  workspace_id = anthropic-wif_workspace.example.id
  name         = "coordinator"
  model        = "claude-opus-4-7"

  multiagent = jsonencode({
    type   = "coordinator"
    agents = [anthropic-wif_agent.worker.id]
  })
}
```

## Import

Import by `workspace_id/agent_id`:

```shell
terraform import anthropic-wif_agent.example wrks_xxx/agt_yyy
```

## Argument Reference

| Argument | Type | Required | Description |
|---|---|---|---|
| `workspace_id` | string | Yes | Workspace ID. Changing this forces a new resource. |
| `name` | string | Yes | Agent name. |
| `model` | string | Yes | Model ID, e.g. `claude-opus-4-7` or `claude-sonnet-4-6`. |
| `model_speed` | string | No | Inference speed: `standard` (default) or `fast`. |
| `system` | string | No | System prompt. |
| `description` | string | No | Human-readable description. |
| `tools` | string | No | JSON-encoded tools array. Maximum 20 tools. |
| `mcp_servers` | string | No | JSON-encoded MCP servers array. Maximum 20 servers, names must be unique. |
| `skills` | string | No | JSON-encoded skills array. Maximum 20 skills. |
| `multiagent` | string | No | JSON-encoded multi-agent coordinator config. |
| `metadata` | map(string) | No | Arbitrary string key-value pairs. |

## Attribute Reference

| Attribute | Type | Description |
|---|---|---|
| `id` | string | Agent ID (`agt_...`). |
| `version` | number | Optimistic-lock version, incremented on each update. |
| `created_at` | string | ISO 8601 creation timestamp. |
| `updated_at` | string | ISO 8601 last-updated timestamp. |
| `archived_at` | string | ISO 8601 archival timestamp, or null if active. |
