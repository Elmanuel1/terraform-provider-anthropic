# ADR 0001: anthropic_deployment resource

- **Status:** Accepted
- **Date:** 2026-06-12
- **Initiative:** deployments-resource
- **Scope:** Lightweight (single-repo change to `Elmanuel1/terraform-provider-anthropic`)

This ADR is the combined design record. It folds in the resolved decision tree (Decision + Alternatives) and the dependency ledger (Consequences / evidence) produced during grill-me.

---

## Context

- The provider already wraps the Claude Managed Agents platform (`agent`, `environment`, `vault`, `vault_credential`, `memory_store`) plus the Admin API (`workspace`).
- A "deployment" is **not** an Admin API concept and **not** a release/version-promotion concept. In Managed Agents it is a **configured instance of an agent** bound to an environment + `initial_events`, with an **optional** cron schedule; each fire (scheduled or manual) spawns a fresh session.
- The Deployments API exposes a full CRUD surface (`POST` create, `GET` get, `GET` list, `POST /{id}` in-place partial update, `POST /{id}/archive` soft-delete) plus side actions (`/pause`, `/unpause`, `/run`). This maps cleanly onto a Terraform resource, mirroring the existing `environment` resource almost exactly.
- Team constraints: reuse existing components (no duplicate infrastructure); model the schema **typed, faithful to the API request/response** — a string only where the API field is a string.

---

## Decision

- Add a first-class **`anthropic_deployment`** resource + **`anthropic_deployment` data source**, mirroring the `environment` pattern (Create / Read / Update / Archive / Import + drift detection).
- **Update is in-place** (`POST /{id}`) for all mutable fields; **`paused`** is toggled via the dedicated `/pause` + `/unpause` endpoints; **Delete maps to archive** (no hard delete, hence no `force_delete`).
- **Fully-typed schema, faithful to the API request/response.** Typed nested attributes wherever the API nests (`agent {id, version}`, `schedule {expression, timezone}`, `initial_events [{type, content:[{type, text}]}]`); a string is used only where the API field is a string. No JSON-string, no list-of-strings.
- **`schedule` is optional** (manual/trigger-only deployments allowed); **`initial_events` is required** (1–50). Scope reduction is by *union variant* (text `user.message` only; `resources` deferred), not by typing.
- **Exclude `POST /{id}/run`** from the provider — an imperative "run now" trigger has no declarative home; it stays an ops action.
- **Reuse the entire shared spine** (`doWithCreds`, `auth.*`, `resolveWorkspaceCredentials`, `utils` helpers). New files: deployment client, resource, data source, docs, example, test, and one registration line.

### Decision Summary

| Decision | Resolved To |
|---|---|
| Semantics | `anthropic_deployment` = a configured instance of an agent bound to an environment + `initial_events`, with an **optional** cron schedule; each fire (scheduled or manual) spawns a fresh session. Wraps the real Managed Agents Deployments API — not a release/version-promotion concept. |
| API endpoints | Create `POST /v1/deployments`; Read `GET /v1/deployments/{id}`; List `GET /v1/deployments`; Update (in-place, partial) `POST /v1/deployments/{id}`; Archive `POST /v1/deployments/{id}/archive`. Pause/unpause via `POST /{id}/pause` + `/unpause`. Beta header `anthropic-beta: managed-agents-2026-04-01` (existing `auth.AgentsBeta`). |
| Manual run | `POST /{id}/run` deliberately **NOT exposed** — imperative action, no declarative home; ops via curl/CLI. |
| Auth | Reuse `resolveWorkspaceCredentials` + `auth.WithBeta(creds, auth.AgentsBeta)`. WIF or workspace API key, identical to `environment`/`agent`. No new auth code. |
| Update model | In-place partial update (`POST /{id}`, "omit to preserve") for `agent`, `environment_id`, `schedule`, `name`, `description`, `metadata`, `initial_events`, `resources`, `vault_ids` — **no force-new**. `paused` toggled via `/pause` + `/unpause`. |
| Delete | Archive only (`POST /{id}/archive`). No hard `DELETE`, so **no `force_delete`** (unlike `environment`). |
| Schema shape | **Fully typed, faithful to the API request/response.** Typed nested where the API nests; a string only where the API field is a string. No JSON-string, no list-of-strings. |
| Typed fields | `agent` nested `{ id (req string), version (opt+computed number) }`; `environment_id` (req string); `name` (req string); `description` (opt string); `metadata` (opt map); `vault_ids` (opt list(string)); `schedule` nested **(optional)** `{ expression (req string), timezone (req string) }`; `paused` (opt bool, default false); `initial_events` nested list **(required, 1–50)** `{ type (string), content: list of { type (string), text (string) } }`. |
| Scope cut (text version) | `initial_events` models only the `user.message` → `text` block path; image/document blocks, `user.define_outcome`, `system.message` not modeled this iteration. **`resources` deferred entirely.** Both additive follow-ups, both stay typed when added. |
| Schedule optional | `schedule` is **optional** (create schema `schedule: optional`). Omit for a manual/trigger-only deployment; within the block `expression` + `timezone` are required. |
| Computed | `id`, `status`, `paused_reason` nested `{ type, error_type }`, `created_at`, `updated_at`, `archived_at`, `agent.version` (resolved concrete), `schedule.last_run_at`, `schedule.upcoming_runs_at`. |
| Data source | Yes — `anthropic_deployment` data source reads by id. List endpoint (`agent_id`, `status`, `include_archived`, pagination) backs filtered lookups. |
| Import | By deployment id (`depl_...`); mirror `environment`'s `ImportState` (`workspace_id/id` for WIF, bare `id` for workspace key). |
| Delivery scope | **Full parity set**: `deployment_client.go` + `resource_deployment.go` + `datasource_deployment.go` + example + `docs/resources/deployment.md` + acceptance test + provider registration + version bump + CHANGELOG. |

### Resource Schema (sketch)

```hcl
resource "anthropic_deployment" "weekly_report" {
  agent = {
    id      = anthropic_agent.reporter.id
    version = anthropic_agent.reporter.version   # optional; omit to pin latest at create
  }
  environment_id = anthropic_environment.sandbox.id

  name        = "weekly-report"
  description = "Generates the weekly metrics report"
  paused      = false
  vault_ids   = [anthropic_vault.prod.id]

  # required, 1-50 events. Typed exactly as the API: user.message -> text block.
  initial_events = [
    {
      type = "user.message"
      content = [
        { type = "text", text = "Summarize last week's tickets." }
      ]
    }
  ]

  # optional — omit entirely for a manual/trigger-only deployment (no cron).
  schedule = {
    expression = "0 9 * * 1-5"          # 5-field POSIX cron
    timezone   = "America/Los_Angeles"  # IANA tz
  }

  # resources = [...] deferred to a later iteration (not yet in schema).
}
```

### Reuse Map (no new shared component)

| Need | Reused (already exists) |
|---|---|
| HTTP transport | `client.doWithCreds` + `defaultHTTPClient` (`internal/client/client.go`) |
| Beta header | `auth.AgentsBeta` + `auth.WithBeta()` |
| Auth (WIF / workspace key) | `resolveWorkspaceCredentials`, `validateWorkspaceCredentials` (`resource_credentials.go`) |
| Model helpers | `nullableString`, `nullableBool`, `fillMetadata` (`utils.go`) |
| Provider wiring | `providerData` + standard `Configure` block |
| Refs | existing `anthropic_agent` (`id`+`version`), `anthropic_environment` (`id`) |

**New files only:** `internal/client/deployment_client.go`, `internal/provider/resource_deployment.go`, `internal/provider/datasource_deployment.go`, `docs/resources/deployment.md`, `examples/deployment-test/`, `resource_deployment_test.go`, one line in `provider.go` `Resources()`/`DataSources()`.

---

## Consequences

- **Positive:** Consistent with every sibling resource; field-level plan diffs, plan-time validation, and references into `agent`/`schedule`/`initial_events`; real drift detection via `GET /{id}`; zero duplicate infrastructure; registry-publishable parity set.
- **Positive:** In-place update + archive-on-destroy → no surprise resource replacement on schedule/agent/env edits.
- **Trade-off (accepted):** text-only `initial_events` and deferred `resources` mean some valid API configs can't yet be expressed in HCL; both are additive, both stay typed.
- **No accepted-risk rows.** Every dependency below is evidence-backed.

### Guardrails from sibling-resource runtime fixes (#4 / #5 / #13 / #15)

These four bugs were fixed on `anthropic_agent` / `anthropic_skill`. Status of each against this design:

| Sibling bug | Root cause | This resource |
|---|---|---|
| #4 — state stripped `configs`/`default_config` | `marshalJSONList` dropped nested keys when storing the `tools` **JSON string** | **Avoided by construction** — no JSON-string field, no `marshalJSONList`. Typed nested attributes; Terraform tracks each field, nothing to strip. |
| #5 — plan≠state by JSON **key order** | raw plan JSON vs canonicalized state JSON | **Avoided by construction** — typed attributes have no serialized key order; comparison is field-by-field. (No `JSONSubsetValue`, no `StringSemanticEquals` needed.) |
| #13 — API returns **enriched** JSON | extra fields in the response broke string equality; needed subset matching | **Avoided in the string sense**; the typed analogue is handled below (every API-populated optional field must be `Computed`). |
| #15 — `updated_at` stale in plan after a real change | a `Computed` timestamp held its prior value, then changed on apply → "inconsistent result after apply" | **LIVE RISK — addressed by the computed-field rule below.** |

**Computed-field rule (the #15 lesson — do not get this wrong):**
- **Volatile computed fields → plain `Computed: true`, NO `UseStateForUnknown`.** Applies to `updated_at`, `status`, `paused_reason`, `schedule.last_run_at`, `schedule.upcoming_runs_at`. They go "(known after apply)" whenever the resource changes, so the plan never asserts a stale value. This mirrors the shipped `anthropic_skill.updated_at` fix.
- **`UseStateForUnknown` ONLY on immutable identity fields** that never change after create: `id`, `created_at`.
- If we ever pin a volatile field with `UseStateForUnknown` for less plan noise, we **must** also mark it unknown in `ModifyPlan` when any affecting input changes — exactly the #15 fix. Default to the simpler plain-`Computed` path; skip `ModifyPlan` mark-unknown unless proven necessary.

**Typed analogue of #13 (API enrichment) — the one place typed can still bite:**
- Every **optional input the API populates** must be `Optional + Computed`, else the user-null plan value ≠ the API-returned value → inconsistent result. Concretely: `agent.version` (API resolves a concrete version when omitted). Audit each field for this.
- Verify the API **preserves `initial_events` order** and does not reorder/enrich `content` blocks in a way element-wise typed comparison would flag. Cover with an acceptance test: `apply` → immediate `plan` must be empty (no-op).

### Watch-items (implementation gotchas)

- **`upcoming_runs_at` / `last_run_at`** change on every read (wall-clock) — plain `Computed` (per the rule above); never `UseStateForUnknown`.
- **`status` / `paused_reason`** are computed reflections of `paused`. On Update, diff `paused`: `true` → `POST /{id}/pause`, `false` → `POST /{id}/unpause`; all other field changes → `POST /{id}`.
- **`initial_events` (typed, text-only)** — nested attributes map one-to-one onto the API objects (no string encode/decode). Validate `type` values at plan time; reject unmodeled variants. An out-of-band non-text event surfaces a clear diagnostic on Read, not a silent drop.
- **`schedule` optional** — omit the block for a no-cron deployment (request omits `schedule`). With `/run` not exposed, a schedule-less deployment fires only via an out-of-band manual trigger — the user's choice.
- **`resources` deferred** — partial update ("omit to preserve") leaves any out-of-band value intact; Read does not map it into state. Modeled typed when added, never a string.
- **`agent` nested, `version` optional + computed** — omit `agent.version` → API resolves latest at create, returns concrete version; store it. A later agent-version bump does **not** auto-update the deployment; changing it is an explicit in-place update.
- **Delete = archive** — `Delete` always calls `POST /{id}/archive`; no hard delete, no `force_delete`.

---

## Dependency Ledger

Rows: **17 verified, 0 accepted-risk, 1 assumed (#10 — closed by an acceptance test, not docs).** Two rows corrected against the authoritative create schema (`schedule` optional; `initial_events` required and typed). The 3 `JSONSubset` sibling bugs (#4/#5/#13) are avoided by the typed-schema choice; the #15 computed-field bug is addressed by the computed-field rule in Consequences.

| # | Dependency | Assumed behavior we're relying on | Evidence | Status |
|---|---|---|---|---|
| 1 | `/v1/deployments` CRUD surface | create / list / get / update / archive all exist | platform.claude.com/docs/en/api/beta/deployments — quoted: Create `POST /v1/deployments`, List `GET /v1/deployments`, Get `GET /v1/deployments/{id}`, Update `POST /v1/deployments/{id}`, Archive `POST /v1/deployments/{id}/archive` | ✅ VERIFIED |
| 1a | Read endpoint | `GET /v1/deployments/{id}` returns the full deployment object for drift detection | platform.claude.com/docs/en/api/beta/deployments/retrieve — full schema + example curl | ✅ VERIFIED |
| 1b | Update endpoint | in-place partial `POST /{id}`, "omit to preserve"; no force-new needed | docs list mutable fields: agent, description, environment_id, initial_events, metadata, name, resources, schedule, vault_ids | ✅ VERIFIED |
| 1c | Delete | archive only; no hard `DELETE` | docs: "There is no DELETE endpoint... Only archive (soft delete)" | ✅ VERIFIED |
| 1d | `schedule` **optional** | a deployment may be created with no cron (manual/trigger-only); within the block `expression` + `timezone` are required | create schema quoted: `schedule: optional BetaManagedAgentsScheduleParams`. **Corrected** — an earlier summary-page reading inferred "required"; the authoritative create body says optional | ✅ VERIFIED — design was wrong (assumed required) |
| 1e | pause / unpause | `POST /{id}/pause` + `/unpause` toggle `status` between active/paused | retrieve schema `paused_reason.manual` = "caller invoked the pause endpoint"; managed-agents-api-reference.md lists both | ✅ VERIFIED |
| 1f | manual run | `POST /{id}/run` fires immediately, works while paused — excluded from provider by design | anthropics/skills managed-agents-api-reference.md (quoted) | ✅ VERIFIED (excluded) |
| 1g | `initial_events` **required** | array of 1–50 typed event objects; not optional, not strings | create schema quoted: `initial_events: array … At least 1, maximum 50`; each element a typed `user.message` with content blocks | ✅ VERIFIED — design was wrong (assumed optional / list-of-strings) |
| 2 | Clone target | `Elmanuel1/terraform-provider-anthropic`, public, default branch `main` | `gh repo view` JSON | ✅ VERIFIED |
| 3 | Plugin framework | terraform-plugin-framework v1.19.0, Go 1.25 | `go.mod` | ✅ VERIFIED |
| 4 | Auth | deployments accept workspace API key / WIF + beta `managed-agents-2026-04-01` | docs example curl uses `X-Api-Key` + beta header; same managed-agents family as `environment` | ✅ VERIFIED |
| 5 | Reuse spine exists | `doWithCreds`, `auth.{BaseURL,AgentsBeta,WithBeta,Credentials}`, `resolveWorkspaceCredentials`, `utils` helpers all present | read source in clone (`internal/client/`, `internal/auth/`, `internal/provider/`) | ✅ VERIFIED |
| 6 | `initial_events` typed shape | event = `{type:"user.message", content:[{type:"text", text}]}`; typed nested attributes, strings only at `type`/`text` leaves | create + retrieve schema — `BetaManagedAgentsUserMessageEventParams` + `BetaManagedAgentsTextBlock`. Text variant only; others deferred (still typed when added) | ✅ VERIFIED |
| 7 | `resources` deferred | not modeled this iteration; partial-update "omit to preserve" keeps any out-of-band value intact | retrieve doc + update "omit to preserve" semantics | ✅ VERIFIED (out of scope this iteration) |
| 8 | `agent` resolves concrete version | response always carries a concrete `agent.version` even if input omitted | create/retrieve doc: "A resolved agent reference with a concrete version" — informs `agent.version` optional+computed | ✅ VERIFIED |
| 9 | No existing deploy/schedule/run code | nothing to duplicate; resource is genuinely new | `grep -ri "deployment\|schedule\|cron\|pause" internal/` → empty | ✅ VERIFIED |
| 10 | API round-trips `initial_events` faithfully | response preserves event/content **order** and does not enrich modeled fields in a way element-wise typed comparison flags (the typed cousin of bug #13) | NOT verified from docs alone — **close via acceptance test**: `apply` → immediate `plan` must be a no-op | ❌ ASSUMED — verify in acceptance test before release |
| 11 | Computed-field plan stability | volatile computed fields go "known after apply" on change, never asserting a stale value (the #15 lesson) | Design rule set (plain `Computed`, `UseStateForUnknown` only on `id`/`created_at`); confirm with `apply`→no-op `plan` test | ✅ VERIFIED (by design) — test-confirmed at impl |

> The schema is **fully typed** — typed nested attributes mirroring the API request/response, a string only where the API field is a string. `json_subset_type.go` / `packages_type.go` are **not used** here.

### Failure-mode / NFR notes (provider context)

- **Partial failure (Create):** API returns the created object atomically; state set from the response. No half-created state.
- **Update split:** `paused` toggle and field updates are separate calls; on a mid-failure, Terraform reports the error and the next `plan` reconciles via Read. No data loss.
- **Drift on computed time fields:** mitigated by keeping `upcoming_runs_at`/`last_run_at`/`status` Computed-only.
- **Dependency outage:** CRUD calls surface HTTP errors through the standard `doWithCreds` error path (same as every other resource).
- **`initial_events` round-trip:** typed nested ↔ API event objects; a non-text event set out-of-band surfaces a diagnostic on Read.
- **Security:** the deferred `resources` field is what carries write-only credentials (github token); since it is out of scope this iteration, the provider stores none.

---

## Alternatives Considered

- **Model deployment as version-pinning / release-promotion** — rejected: the API has no such resource; `deployment` is a configured agent instance with an optional schedule.
- **JSON-string the polymorphic fields** (`initial_events`/`resources`) — initially proposed, **rejected** on the "use typed, follow the API docs" rule: it loses plan diffs, validation, and references. Variant scope is narrowed instead (text-only, resources deferred) while keeping modeled fields fully typed.
- **`initial_events` as a `list(string)`** — rejected: the API field is an array of typed event objects, not strings; a string would misrepresent the contract.
- **Mark `schedule` required** — rejected/corrected: the create schema says `schedule: optional`; a manual/trigger-only deployment is valid.
- **Expose `/run` as a one-shot resource or a framework Action** — rejected: imperative-in-declarative anti-pattern or net-new Action machinery requiring Terraform 1.14+; out of scope for the declarative resource.
- **Force-new on schedule/agent/environment changes** — rejected once the in-place `POST /{id}` update endpoint was verified.
