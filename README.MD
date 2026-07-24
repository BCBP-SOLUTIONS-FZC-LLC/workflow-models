# workflow-models

Shared Go module between `workflow-definition-service` and the future Execution Service — every type crossing that service boundary lives here once, instead of being hand-mirrored independently on each side.

**Scope rule: only what both services actually need.** A type only one side reads or writes stays in that service's own domain package, not here.

- `pkg/dsl` — the compiled-plan DSL (`CompiledCollaboration`, `CompiledPlan`, `StageDef`, `ExecutionStep`, and friends) — the full contract, since Definition produces every field and Execution consumes every field.
- `pkg/events` — exactly one struct, `TemplatePublishedPayload` (`workflow.template.published`) — the only event payload that crosses this particular boundary. Everything else (Execution's own outbound/inbound event payloads) stays in Execution's own domain package.
- `pkg/enums` — `StageDef.Type` values plus the one shared event's wire-type constant.

Design spec / rationale: `design/LLD/workflow_models_lib.md` (BCBP-SOLUTIONS-FZC-LLC/design repo).

## Status

Design decided, module scaffolded (`pkg/dsl`/`pkg/events`/`pkg/enums` + golden round-trip tests). Not yet published/tagged, and `workflow-definition-service` has not yet migrated its own `internal/core/domain` types to alias this module — see the design doc's migration steps.
