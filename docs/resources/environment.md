---
page_title: "anthropic: anthropic_environment"
subcategory: ""
description: |-
  Manages an Anthropic cloud environment for agent sessions.
---

# Resource: anthropic_environment

Manages an Anthropic cloud execution environment. Environments define the runtime configuration for agent sessions: networking policy, pre-installed packages, and MCP server access.

Supports two authentication modes, controlled by what is set in the **provider block**:

| Mode | Provider attributes required | `workspace_id` |
|---|---|---|
| WIF | `federation_rule_id`, `organization_id`, `service_account_id` | Required |
| Workspace API key | `workspace_api_key` | Not needed |

When both are configured, WIF takes precedence.

On destroy the environment is archived by default. Set `force_delete = true` to permanently delete it.

## Example Usage

### WIF authentication

```terraform
provider "anthropic" {
  federation_rule_id = var.anthropic_federation_rule_id
  organization_id    = var.anthropic_organization_id
  service_account_id = var.anthropic_service_account_id
}

resource "anthropic_environment" "example" {
  workspace_id = anthropic_workspace.example.id
  name         = "default-env"
}
```

### Workspace API key authentication

```terraform
provider "anthropic" {
  workspace_api_key = var.anthropic_workspace_api_key
}

resource "anthropic_environment" "example" {
  name = "default-env"
}
```

### Limited networking with packages (WIF)

```terraform
resource "anthropic_environment" "example" {
  workspace_id    = anthropic_workspace.example.id
  name            = "python-env"
  networking_type = "limited"

  allowed_hosts          = ["pypi.org", "files.pythonhosted.org"]
  allow_package_managers = true

  packages = jsonencode({
    pip = ["pandas", "numpy", "requests"]
  })

  metadata = {
    team = "data-science"
  }
}
```

## Argument Reference

* `workspace_id` - (Optional, Forces new resource) Workspace ID. Required when using WIF authentication.
* `name` - (Required) Environment name.
* `description` - (Optional) Human-readable description.
* `networking_type` - (Optional) `unrestricted` (default) or `limited`.
* `allowed_hosts` - (Optional) Allowed outbound hostnames. Only applies when `networking_type = "limited"`.
* `allow_mcp_servers` - (Optional) Allow MCP server network access. Default `false`. Only applies when `networking_type = "limited"`.
* `allow_package_managers` - (Optional) Allow package manager network access (PyPI, npm, etc). Default `false`. Only applies when `networking_type = "limited"`.
* `packages` - (Optional) JSON-encoded packages to pre-install. Supported managers: `apt`, `cargo`, `gem`, `go`, `npm`, `pip`.
* `metadata` - (Optional) Map of arbitrary string key-value pairs.
* `force_delete` - (Optional) When `true`, permanently deletes on destroy. Default `false` (archives).

## Attribute Reference

* `id` - Environment ID (`env_...`).
* `created_at` - ISO 8601 creation timestamp.
* `updated_at` - ISO 8601 last-updated timestamp.
* `archived_at` - ISO 8601 archival timestamp, or null if active.

## Import

WIF (workspace_id known):

```shell
terraform import anthropic_environment.example wrks_xxx/env_yyy
```

Workspace API key (workspace_id not needed):

```shell
terraform import anthropic_environment.example env_yyy
```
