---
page_title: "anthropic: anthropic_deployment"
subcategory: ""
description: |-
  Reads an existing Anthropic deployment by ID.
---

# Data Source: anthropic_deployment

Reads an existing Anthropic Managed Agents deployment by ID.

## Example Usage

```terraform
data "anthropic_deployment" "existing" {
  id = "depl_abc123"
}

output "schedule" {
  value = data.anthropic_deployment.existing.schedule
}
```

## Argument Reference

- `id` (Required) Deployment ID (`depl_...`).
- `workspace_id` (Optional) Workspace ID. Required when using WIF authentication.

## Attribute Reference

All other attributes documented on the `anthropic_deployment` resource are exported: `name`, `description`, `agent`, `environment_id`, `initial_events`, `schedule`, `vault_ids`, `metadata`, `paused`, `status`, `paused_reason`, `created_at`, `updated_at`, `archived_at`.
