---
page_title: "anthropic-wif_memory_store Resource"
description: |-
  Manages an Anthropic memory store for agent persistence.
---

# anthropic-wif_memory_store

Manages an Anthropic memory store. Memory stores provide persistent storage for agents across sessions, enabling long-term context and knowledge retention.

Authenticates via WIF bearer token scoped to the `workspace_id`.

On destroy the memory store is **archived** by default. Set `force_delete = true` to permanently delete it.

~> **Note:** Memory store support is in beta (`managed-agents-2026-04-01`). Auth requirements may change.

## Example Usage

```terraform
resource "anthropic-wif_memory_store" "example" {
  workspace_id = anthropic-wif_workspace.example.id
  name         = "agent-memory"
  description  = "Persistent memory for the procurement agent."

  metadata = {
    env  = "production"
    team = "platform"
  }
}
```

## Import

Import by `workspace_id/memory_store_id`:

```shell
terraform import anthropic-wif_memory_store.example wrks_xxx/ms_yyy
```

## Argument Reference

| Argument | Type | Required | Description |
|---|---|---|---|
| `workspace_id` | string | Yes | Workspace ID. Changing this forces a new resource. |
| `name` | string | Yes | Memory store name. |
| `description` | string | No | Human-readable description. |
| `metadata` | map(string) | No | Arbitrary string key-value pairs. |
| `force_delete` | bool | No | When `true`, permanently deletes on destroy. Default `false` (archives). |

## Attribute Reference

| Attribute | Type | Description |
|---|---|---|
| `id` | string | Memory store ID. |
| `created_at` | string | ISO 8601 creation timestamp. |
| `updated_at` | string | ISO 8601 last-updated timestamp. |
| `archived_at` | string | ISO 8601 archival timestamp, or null if active. |
