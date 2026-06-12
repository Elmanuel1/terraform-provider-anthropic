terraform {
  required_version = ">= 1.5.0"
  required_providers {
    anthropic = {
      source = "Elmanuel1/anthropic"
    }
  }
}

# Self-contained single-key smoke test for anthropic_deployment.
# Reads the workspace key from ANTHROPIC_API_KEY in the environment, then
# creates an environment, an agent, and two deployments (scheduled + manual).
provider "anthropic" {}

resource "anthropic_environment" "test" {
  name = "tf-smoke-deploy-env"
}

resource "anthropic_agent" "test" {
  name  = "tf-smoke-deploy-agent"
  model = "claude-haiku-4-5"
}

# Scheduled deployment.
resource "anthropic_deployment" "nightly" {
  name           = "tf-smoke-nightly"
  environment_id = anthropic_environment.test.id

  agent = {
    id = anthropic_agent.test.id
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
  environment_id = anthropic_environment.test.id

  agent = {
    id = anthropic_agent.test.id
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
