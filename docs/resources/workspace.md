---
page_title: "anthropic-wif_workspace Resource"
description: |-
  Manages an Anthropic workspace.
---

# anthropic-wif_workspace

Manages an Anthropic workspace. Workspaces are the top-level organisational unit on the Anthropic platform. Agents, environments, vaults, and other resources are scoped to a workspace.

Authenticates with the Anthropic Admin API key (`ANTHROPIC_ADMIN_API_KEY`). WIF is not required for this resource.

On destroy the workspace is **archived** (soft-deleted). Anthropic does not expose a hard-delete endpoint for workspaces.

## Example Usage

```terraform
resource "anthropic-wif_workspace" "example" {
  name = "my-workspace"
}
```

## Import

Import by workspace name (resolved to ID at import time):

```shell
terraform import anthropic-wif_workspace.example my-workspace
```

## Argument Reference

| Argument | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Workspace name as it appears in the Anthropic Console. |

## Attribute Reference

| Attribute | Type | Description |
|---|---|---|
| `id` | string | Workspace ID (`wrks_...`). |
| `created_at` | string | ISO 8601 creation timestamp. |
| `archived_at` | string | ISO 8601 archival timestamp, or null if active. |
