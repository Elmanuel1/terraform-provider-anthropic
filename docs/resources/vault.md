---
page_title: "anthropic: anthropic_vault"
subcategory: ""
description: |-
  Manages an Anthropic vault for storing MCP server credentials.
---

# Resource: anthropic_vault

Manages an Anthropic vault. Vaults are workspace-scoped containers for storing MCP server credentials that agents can use during sessions.

Supports two authentication modes, controlled by what is set in the **provider block**:

| Mode | Provider attributes required | `workspace_id` |
|---|---|---|
| WIF | `federation_rule_id`, `organization_id`, `service_account_id` | Required |
| Workspace API key | `workspace_api_key` | Not needed |

When both are configured, WIF takes precedence.

On destroy the vault is archived by default. Set `force_delete = true` to permanently delete it.

## Example Usage

### WIF authentication

```terraform
provider "anthropic" {
  federation_rule_id = var.anthropic_federation_rule_id
  organization_id    = var.anthropic_organization_id
  service_account_id = var.anthropic_service_account_id
}

resource "anthropic_vault" "example" {
  workspace_id = anthropic_workspace.example.id
  display_name = "production-vault"

  metadata = {
    env  = "production"
    team = "platform"
  }
}
```

### Workspace API key authentication

```terraform
provider "anthropic" {
  workspace_api_key = var.anthropic_workspace_api_key
}

resource "anthropic_vault" "example" {
  display_name = "production-vault"
}
```

## Argument Reference

* `workspace_id` - (Optional, Forces new resource) Workspace ID. Required when using WIF authentication.
* `display_name` - (Required) Human-readable vault name.
* `metadata` - (Optional) Map of arbitrary string key-value pairs.
* `force_delete` - (Optional) When `true`, permanently deletes on destroy. Default `false` (archives).

## Attribute Reference

* `id` - Vault ID (`vlt_...`).
* `created_at` - ISO 8601 creation timestamp.
* `updated_at` - ISO 8601 last-updated timestamp.
* `archived_at` - ISO 8601 archival timestamp, or null if active.

## Import

WIF (workspace_id known):

```shell
terraform import anthropic_vault.example wrks_xxx/vlt_yyy
```

Workspace API key (workspace_id not needed):

```shell
terraform import anthropic_vault.example vlt_yyy
```
