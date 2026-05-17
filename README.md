# terraform-provider-anthropic-wif

Terraform provider for managing Anthropic platform resources using Workload Identity Federation (WIF) via Terraform Cloud OIDC.

Registry: [registry.terraform.io/providers/Elmanuel1/anthropic-wif](https://registry.terraform.io/providers/Elmanuel1/anthropic-wif/latest)

## Resources

| Resource | Auth | Description |
|---|---|---|
| `anthropic-wif_workspace` | Admin API key | Anthropic workspace |
| `anthropic-wif_agent` | WIF | Agent with model, tools, and skills |
| `anthropic-wif_environment` | WIF | Execution environment for agents |
| `anthropic-wif_vault` | WIF | Vault for storing credentials |
| `anthropic-wif_vault_credential` | WIF | MCP server credential in a vault |
| `anthropic-wif_memory_store` | WIF | Memory store for agent persistence |

## Quick Start

```terraform
terraform {
  required_providers {
    anthropic-wif = {
      source  = "Elmanuel1/anthropic-wif"
      version = "~> 0.4"
    }
  }
}

provider "anthropic-wif" {}

resource "anthropic-wif_workspace" "example" {
  name = "my-workspace"
}

resource "anthropic-wif_agent" "example" {
  workspace_id = anthropic-wif_workspace.example.id
  name         = "my-agent"
  model        = "claude-sonnet-4-6"
  system       = "You are a helpful assistant."
}
```

## Authentication

### Environment Variables

| Variable | Description | Required |
|---|---|---|
| `ANTHROPIC_ADMIN_API_KEY` | Admin API key (`sk-ant-admin-...`) | Always |
| `ANTHROPIC_FEDERATION_RULE_ID` | Federation rule ID (`fdrl_...`) | WIF resources |
| `ANTHROPIC_ORGANIZATION_ID` | Organization UUID | WIF resources |
| `ANTHROPIC_SERVICE_ACCOUNT_ID` | Service account ID (`svac_...`) | WIF resources |
| `TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC` | TFC-injected OIDC JWT | WIF resources |
| `TFC_WORKLOAD_IDENTITY_TOKEN` | Fallback OIDC JWT | WIF resources (fallback) |

Set `TFC_WORKLOAD_IDENTITY_AUDIENCE_ANTHROPIC=https://api.anthropic.com` on your TFC workspace to enable automatic JWT injection.

### Anthropic Console Setup

1. **Workload Identity Issuer**
   - Console → Settings → Workload Identity → Create issuer
   - Issuer URL: `https://app.terraform.io` | JWKS source: `discovery`

2. **Service Account**
   - Console → Settings → Service Accounts → Create

3. **Federation Rule**
   - Console → Settings → Federation Rules → Create
   - Audience: `https://api.anthropic.com`
   - Subject: `organization:<tfc-org>:project:<project>:workspace:<workspace>:run_phase:apply`
   - Target: service account from step 2
   - Scope: `workspace:developer`

## Local Development

```bash
go build -o terraform-provider-anthropic-wif .

# ~/.terraformrc
cat > ~/.terraformrc <<EOF
provider_installation {
  dev_overrides {
    "Elmanuel1/anthropic-wif" = "/path/to/provider/binary"
  }
  direct {}
}
EOF

export ANTHROPIC_ADMIN_API_KEY="sk-ant-admin-..."
export ANTHROPIC_FEDERATION_RULE_ID="fdrl_..."
export ANTHROPIC_ORGANIZATION_ID="00000000-..."
export ANTHROPIC_SERVICE_ACCOUNT_ID="svac_..."
export TFC_WORKLOAD_IDENTITY_TOKEN="<jwt>"

terraform plan
```
