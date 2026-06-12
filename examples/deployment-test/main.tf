terraform {
  required_version = ">= 1.5.0"
  required_providers {
    anthropic = {
      source = "Elmanuel1/anthropic"
    }
  }
}

# Single-key smoke test for anthropic_deployment.
# Set ANTHROPIC_API_KEY (workspace API key) in the environment, then provide an
# existing agent_id and environment_id from the same workspace.

variable "workspace_api_key" {
  type    = string
  default = ""
}

variable "agent_id" {
  type        = string
  description = "An existing agent_... ID in the workspace."
}

variable "environment_id" {
  type        = string
  description = "An existing env_... ID in the workspace."
}

provider "anthropic" {
  workspace_api_key = var.workspace_api_key
}

# Scheduled deployment.
resource "anthropic_deployment" "nightly" {
  name           = "tf-smoke-nightly"
  environment_id = var.environment_id

  agent = {
    id = var.agent_id
  }

  initial_events = [
    {
      type    = "user.message"
      content = [{ type = "text", text = "Summarize today's support tickets." }]
    }
  ]

  schedule = {
    expression = "0 9 * * 1-5"
    timezone   = "UTC"
  }
}

# Manual deployment (no schedule).
resource "anthropic_deployment" "manual" {
  name           = "tf-smoke-manual"
  environment_id = var.environment_id

  agent = {
    id = var.agent_id
  }

  initial_events = [
    {
      type    = "user.message"
      content = [{ type = "text", text = "Run triage now." }]
    }
  ]
}

output "nightly_id" {
  value = anthropic_deployment.nightly.id
}

output "nightly_upcoming_runs" {
  value = anthropic_deployment.nightly.schedule.upcoming_runs_at
}

output "manual_status" {
  value = anthropic_deployment.manual.status
}
