---
page_title: "Provider Matrix — All Auth Modes"
description: |-
  Full working example using all three provider aliases (admin, wif, workspace) in a single configuration, with guidance on TFC workspace separation.
---

# Provider Matrix — All Auth Modes

This guide shows a complete configuration that exercises all three authentication paths the provider supports: Admin API key, Workload Identity Federation (WIF), and workspace API key. It mirrors the layout used to test the provider itself.

## TFC Workspace Separation

~> **Recommendation:** Use **two separate Terraform Cloud workspaces** — one for workspace-level infrastructure and one for workload resources.

| TFC Workspace | Provider alias | Resources managed |
|---|---|---|
| `anthropic-admin` | `anthropic.admin` | `anthropic_workspace` |
| `anthropic-workloads` | `anthropic.wif`, `anthropic.workspace` | `anthropic_environment`, `anthropic_vault`, `anthropic_vault_credential`, `anthropic_agent` |

**Why separate them?** The `admin` provider uses an Admin API key scoped to your entire Anthropic organization and can create or delete workspaces. Keeping it in its own TFC workspace with tighter access controls (separate state, separate variable set, restricted team access) prevents a routine workload change from accidentally destroying a workspace and everything in it. Workspace IDs referenced by the workloads TFC workspace can be passed as remote state outputs or hardcoded after initial creation.

## providers.tf

```terraform
terraform {
  required_providers {
    anthropic = {
      source  = "Elmanuel1/anthropic"
      version = "~> 0.3.5"
    }
  }
}

# Admin API key — organization-level operations (workspace CRUD)
provider "anthropic" {
  alias         = "admin"
  admin_api_key = var.anthropic_admin_api_key
}

# WIF — workspace-scoped operations authenticated via Terraform Cloud OIDC
provider "anthropic" {
  alias              = "wif"
  federation_rule_id = var.anthropic_federation_rule_id
  organization_id    = var.anthropic_organization_id
  service_account_id = var.anthropic_service_account_id
}

# Workspace API key — workspace-scoped operations authenticated via static key
provider "anthropic" {
  alias             = "workspace"
  workspace_api_key = var.anthropic_workspace_api_key
}
```

## variables.tf

```terraform
variable "anthropic_admin_api_key" {
  type      = string
  sensitive = true
}

variable "anthropic_workspace_api_key" {
  type      = string
  sensitive = true
}

variable "anthropic_federation_rule_id" {
  type = string
}

variable "anthropic_organization_id" {
  type = string
}

variable "anthropic_service_account_id" {
  type = string
}

variable "mcp_token" {
  type      = string
  sensitive = true
}

variable "slack_mcp_url" {
  type = string
}
```

## workspaces.tf

Managed by the `admin` provider. Keep this in its own TFC workspace.

```terraform
# Primary workspace — all WIF workload resources live here
resource "anthropic_workspace" "primary" {
  provider = anthropic.admin
  name     = "my-team-workspace"
}

# Secondary workspace — tested via workspace API key
resource "anthropic_workspace" "secondary" {
  provider = anthropic.admin
  name     = "my-team-workspace-secondary"
}
```

## wif.tf

WIF-authenticated resources scoped to `anthropic_workspace.primary`.

```terraform
resource "anthropic_environment" "wif" {
  provider               = anthropic.wif
  workspace_id           = anthropic_workspace.primary.id
  name                   = "my-team-wif-env"
  description            = "Execution environment for WIF workloads"
  networking_type        = "limited"
  allowed_hosts          = ["api.anthropic.com", "pypi.org", "api.slack.com"]
  allow_mcp_servers      = true
  allow_package_managers = true
  packages               = jsonencode({ pip = ["requests"] })
  metadata = {
    team = "my-team"
    env  = "production"
  }
}

resource "anthropic_vault" "wif" {
  provider     = anthropic.wif
  workspace_id = anthropic_workspace.primary.id
  display_name = "my-team-vault"
  metadata = {
    team = "my-team"
    env  = "production"
  }
}

resource "anthropic_vault_credential" "wif" {
  provider       = anthropic.wif
  workspace_id   = anthropic_workspace.primary.id
  vault_id       = anthropic_vault.wif.id
  display_name   = "my-mcp-server-token"
  auth_type      = "static_bearer"
  mcp_server_url = "https://mcp.example.com"
  token          = var.mcp_token
  metadata = {
    team = "my-team"
    env  = "production"
  }
}

resource "anthropic_agent" "wif" {
  provider     = anthropic.wif
  workspace_id = anthropic_workspace.primary.id
  name         = "my-team-agent"
  model        = "claude-sonnet-4-6"
  model_speed  = "standard"
  system       = "You are a helpful assistant for my team."
  description  = "Primary agent"

  tools = jsonencode([
    {
      type            = "mcp_toolset"
      mcp_server_name = "my-server"
      default_config = {
        enabled           = true
        permission_policy = { type = "always_allow" }
      }
      configs = []
    },
    {
      type            = "mcp_toolset"
      mcp_server_name = "slack"
      default_config = {
        enabled           = true
        permission_policy = { type = "always_allow" }
      }
      configs = [
        {
          name              = "slack_send_message"
          enabled           = false
          permission_policy = { type = "always_allow" }
        }
      ]
    }
  ])

  mcp_servers = jsonencode([
    { type = "url", name = "my-server", url = "https://mcp.example.com" },
    { type = "url", name = "slack", url = var.slack_mcp_url }
  ])

  metadata = {
    team = "my-team"
    env  = "production"
  }
}
```

## workspace_apikey.tf

Same resource set authenticated via workspace API key. No `workspace_id` required — the key already scopes the request.

```terraform
resource "anthropic_environment" "workspace" {
  provider               = anthropic.workspace
  name                   = "my-team-workspace-env"
  description            = "Execution environment for workspace API key workloads"
  networking_type        = "limited"
  allowed_hosts          = ["api.anthropic.com", "pypi.org", "api.slack.com"]
  allow_mcp_servers      = true
  allow_package_managers = true
  packages               = jsonencode({ pip = ["requests"] })
  metadata = {
    team = "my-team"
    env  = "production"
  }
}

resource "anthropic_vault" "workspace" {
  provider     = anthropic.workspace
  display_name = "my-team-workspace-vault"
  metadata = {
    team = "my-team"
    env  = "production"
  }
}

resource "anthropic_vault_credential" "workspace" {
  provider       = anthropic.workspace
  vault_id       = anthropic_vault.workspace.id
  display_name   = "my-mcp-server-token-workspace"
  auth_type      = "static_bearer"
  mcp_server_url = "https://mcp.example.com"
  token          = var.mcp_token
  metadata = {
    team = "my-team"
    env  = "production"
  }
}

resource "anthropic_agent" "workspace" {
  provider    = anthropic.workspace
  name        = "my-team-agent-workspace"
  model       = "claude-sonnet-4-6"
  model_speed = "standard"
  system      = "You are a helpful assistant for my team."
  description = "Agent authenticated via workspace API key"

  tools = jsonencode([
    {
      type            = "mcp_toolset"
      mcp_server_name = "my-server"
      default_config = {
        enabled           = true
        permission_policy = { type = "always_allow" }
      }
      configs = []
    },
    {
      type            = "mcp_toolset"
      mcp_server_name = "slack"
      default_config = {
        enabled           = true
        permission_policy = { type = "always_allow" }
      }
      configs = [
        {
          name              = "slack_send_message"
          enabled           = false
          permission_policy = { type = "always_allow" }
        }
      ]
    }
  ])

  mcp_servers = jsonencode([
    { type = "url", name = "my-server", url = "https://mcp.example.com" },
    { type = "url", name = "slack", url = var.slack_mcp_url }
  ])

  metadata = {
    team = "my-team"
    env  = "production"
  }
}
```

## Tool permission policies

Each `mcp_toolset` entry in `tools` supports `default_config` (applies to all tools in the server) and `configs` (per-tool overrides). Both accept a `permission_policy` with one of two types:

| `permission_policy.type` | Behaviour |
|---|---|
| `always_allow` | Agent runs the tool automatically without asking the user |
| `always_ask` | Agent pauses and asks the user to approve before running the tool |

Use `configs` to disable specific tools (`enabled = false`) or override their policy:

```terraform
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
      # Always ask before scheduling
      {
        name              = "slack_schedule_message"
        enabled           = true
        permission_policy = { type = "always_ask" }
      }
    ]
  }
])
```

## Key differences between WIF and workspace API key resources

| Attribute | WIF | Workspace API key |
|---|---|---|
| `workspace_id` | Required on every resource | Omit — key is already workspace-scoped |
| Token source | Short-lived OIDC-minted token (auto-refreshed per run) | Static long-lived key (rotate manually) |
| Suitable for | CI/CD, Terraform Cloud, automated pipelines | Local development, simple scripts |

For WIF setup details see the [Authentication guide](./authentication.md).
