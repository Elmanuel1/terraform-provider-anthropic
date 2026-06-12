# [PRD] anthropic_deployment Terraform resource

> High-level product requirements. Derived from the design ADR. See
> [.claude/designs/deployments-resource/adr.md](./adr.md) for the resolved decisions,
> dependency ledger, and schema. This PRD captures *why / who / what / how-we-measure*,
> not implementation detail.

## TL;DR

Platform engineers who run Anthropic Managed Agents have no declarative way to manage
**deployments**: scheduled, recurring agent runs. Today they click in the console or
script raw API calls, which drift, aren't peer-reviewed, and can't be reproduced across
environments. This adds an `anthropic_deployment` resource and data source to the
existing Anthropic Terraform provider, so a deployment (an agent + environment + optional
cron schedule + seed messages) can be defined as code with full plan / apply / drift
detection. v1 covers the text-message deployment case; richer seed content and mounted
resources are fast-follows. Success is teams managing deployments in Terraform with zero
console-vs-code drift.

## Context & Problem

Managed Agents deployments are recurring scheduled runs of an agent in an environment;
each fire spawns a fresh session. The provider already manages the adjacent resources
(`anthropic_agent`, `anthropic_environment`, `anthropic_vault`, `anthropic_memory_store`),
so deployments are the missing piece needed to express a complete agent stack as code.

Without a resource, deployments are created by hand in the console or via one-off API
scripts. That means no version control, no review, no drift detection, and no
reproducibility across dev/staging/prod.

## Goals & Non-Goals

**Goals**

- Declarative CRUD for deployments via `anthropic_deployment`.
- Drift detection and safe **in-place** updates (no surprise replacement on edits).
- A data source to look up existing deployments.
- Parity with existing provider resources (auth, patterns, docs, tests).

**Non-Goals**

- Imperative "run now" triggering stays an ops action, not a Terraform concern.
- Modeling non-text seed events (image/document/outcome/system) is deferred.
- Modeling mounted `resources` (repositories / files / memory stores) is deferred.

## Target Users

- **Primary:** platform / infrastructure engineers already using this provider to manage
  Anthropic org resources as code.
- **Secondary:** application teams who own an agent and want its scheduled deployment
  versioned and reviewed alongside the agent and environment it depends on.

## Solution Overview

Add an `anthropic_deployment` resource + data source wrapping the Managed Agents
Deployments API, mirroring the existing `environment` resource. A deployment binds an
agent (optionally pinned to a version) to an environment, seeds each run with one or more
text messages, and optionally runs on a cron schedule. Lifecycle is create / read /
in-place update / archive, plus pause and resume. The schema is fully typed and faithful
to the API. Endpoint mapping and the full attribute set live in the ADR.

## Key User Flows

1. **Define a scheduled deployment**: reference an existing agent and environment, set a
   cron schedule and a seed message, run `terraform apply`.
2. **Pause / resume**: flip `paused` and apply.
3. **Update**: change the schedule or seed message; apply updates in place.
4. **Destroy**: `terraform destroy` archives the deployment (soft delete).
5. **Import / look up**: import an existing deployment by id, or read one via the data
   source.

## Functional Requirements

- Users can declare a deployment with an agent reference, an environment, required seed
  events, and an optional cron schedule.
- The system must read deployment state back from the API to detect drift.
- The system must apply schedule / agent / environment / content changes in place without
  forcing replacement.
- Users can pause and resume a deployment declaratively.
- The system must map destroy to archive; there is no hard delete.
- Users can import an existing deployment and read one via a data source.
- The system must not emit "inconsistent result after apply" errors; computed fields are
  handled per the guardrails in the ADR.

## Success Metrics

- **TBD (needs input from owner/PM):** target number of deployments managed in Terraform
  at 1 / 3 / 6 months.
- **Qualitative:** zero console-vs-Terraform drift for managed deployments; deployment
  changes land through reviewed PRs rather than console edits.

## Dependencies & Risks

- **Anthropic Managed Agents Deployments API**: public beta, header
  `managed-agents-2026-04-01`. The beta surface may change before GA.
- **Provider auth**: reuses existing WIF / workspace-API-key credential handling; no new
  auth.
- **Risk:** API round-trip fidelity for seed events (order preservation / enrichment),
  closed by an acceptance test (`apply` → immediate `plan` must be a no-op).
- **Risk:** beta API changes before GA could require schema updates.

## Out of Scope

- Manual run trigger, non-text seed events, and mounted `resources` are each deferred; the
  rationale is in the ADR's Alternatives and Consequences.

## Open Questions

- Adoption / success targets: **needs owner/PM input.**
- GA timeline for the Deployments API beta: **needs Anthropic confirmation.**
