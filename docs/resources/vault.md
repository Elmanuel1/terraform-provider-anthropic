---
page_title: "anthropic-wif_vault Resource"
description: |-
  Manages an Anthropic vault for storing credentials.
---

# anthropic-wif_vault

Manages an Anthropic vault. Vaults are workspace-scoped containers for storing MCP server credentials that agents can use during sessions.

Authenticates via WIF bearer token scoped to the `workspace_id`.

On destroy the vault is **archived** by default. Set `force_delete = true` to permanently delete it.

## Example Usage

```terraform
resource "anthropic-wif_vault" "example" {
  workspace_id = anthropic-wif_workspace.example.id
  display_name = "production-vault"

  metadata = {
    env  = "production"
    team = "platform"
  }
}
```

## Import

Import by `workspace_id/vault_id`:

```shell
terraform import anthropic-wif_vault.example wrks_xxx/vlt_yyy
```

## Argument Reference

| Argument | Type | Required | Description |
|---|---|---|---|
| `workspace_id` | string | Yes | Workspace ID. Changing this forces a new resource. |
| `display_name` | string | No | Human-readable vault name. |
| `metadata` | map(string) | No | Arbitrary string key-value pairs. |
| `force_delete` | bool | No | When `true`, permanently deletes on destroy. Default `false` (archives). |

## Attribute Reference

| Attribute | Type | Description |
|---|---|---|
| `id` | string | Vault ID (`vlt_...`). |
| `created_at` | string | ISO 8601 creation timestamp. |
| `updated_at` | string | ISO 8601 last-updated timestamp. |
| `archived_at` | string | ISO 8601 archival timestamp, or null if active. |
