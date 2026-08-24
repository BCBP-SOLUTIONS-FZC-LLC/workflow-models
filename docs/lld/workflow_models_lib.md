> **Note:** This is the in-repo copy of this document. It is kept content-identical to the design repo's published copy, allowed to differ from it only on embed-vs-link mechanics — never on content.

# Workflow Models Shared Library — Low-Level Design

`workflow-models` (`github.com/BCBP-SOLUTIONS-FZC-LLC/workflow-models`) is the shared Go module carrying every type that crosses the boundary between Definition Service and Execution Service: the compiled-plan DSL, the one event payload both sides touch, and the enum/constant discriminators both sides need. This document specifies the module's contents field-by-field, how Definition's compiler populates each field, and how Execution's design consumes/dispatches on it — the full usage detail on both sides of the boundary, not just what the module contains.

---

## Table of Contents

1. Overview & Status
2. Package Reference: `pkg/dsl`
   - 2.1 `CompiledCollaboration` / `MessageDef`
   - 2.2 `CompiledPlan` / `DepartmentDef` / `VisualElementDef`
   - 2.3 `StageDef` / `BoundaryTimer`
   - 2.4 `ExecutionPlan` / `ExecutionStep`
   - 2.5 Branch & Step Variants
3. Package Reference: `pkg/events`
4. Package Reference: `pkg/enums`
5. Scope Boundary: What's Not In This Module
6. Versioning
   - 6.1 DSL Schema Versioning
   - 6.2 Event Versioning
7. `Extras` / `IOMapping` Key Registry
8. Definition Service Migration
9. Test Plan
10. Package / File Layout
11. Build & Publish Conventions
12. Repo Scaffolding Gaps vs. Sibling Platform Libraries
13. Relationship to `compiled_plan_contract.md`

- Appendix A: Design Decisions
- Appendix B: Open Items
  - Appendix B.1: Repo Readiness Checklist
- Revision history

---

## 1. Overview & Status

Go 1.26, zero external dependencies. Three packages: `pkg/dsl`, `pkg/events`, `pkg/enums`. It exists to close two structural drift sources that are otherwise kept in sync by discipline, not by the compiler:

1. **The compiled-plan DSL travels as an opaque JSON string.** `DefinitionService.GetCompiledWorkflow` returns `compiled_plan_json` as a plain `string`, not a typed message. Nothing stops Definition's compiler from adding a BPMN element handler, an `ExecutionStep` variant, or a struct field without Execution ever finding out until an instance fails at runtime. Two drift bugs already exist in the codebase from this class of problem: `eventBasedGateway` silently dropped by XML parsing, and `inclusiveGateway` inconsistently handled across code and documentation.
2. **Event payload structs are hand-mirrored against each service's own AsyncAPI spec.** Definition's `internal/core/domain/eventpayloads.go` `TemplatePublishedPayload` must match `WorkflowTemplatePublishedPayload` in its own `api/asyncapi.yaml` by discipline alone; Execution does the same for its 18 outbound payloads. `workflow.template.published` is the sharp case — Definition produces it, Execution consumes it for cache pre-warm (`execution_service.md` §6.2) — so two independently hand-maintained copies of the same contract sit on either side of the event bus with no compiler check between them.

Both failure classes are the same shape: a second source of truth kept in sync by discipline. Publishing the real Go structs as an importable module turns a silent runtime surprise into a compile-time error the moment a consumer bumps the dependency.

**Status.** Pre-release, tagged for integration testing. `pkg/dsl`/`pkg/enums`/`pkg/events` exist with golden round-trip tests; `v0.1.0-beta.1` is tagged and consumed by `workflow-definition-service`, which has migrated its own compiled-plan and event-payload types to reference the module directly (§8) — the real `v1.0.0` tag is still pending. Execution Service imports this module directly from day one, with no interim hand-rolled mirror ever written to retire.

**Relationship to the gRPC/proto contract.** `DefinitionService.GetCompiledWorkflow`/`ExecutionService.CheckActiveInstances`/`PauseUserTasks` (`api/proto/definition/v1/definition.proto`, `execution/v1/execution_service.proto`) are a separate, already-solved sharing mechanism — `buf`-generated stubs already give both services one structurally-shared contract for that RPC layer. This module doesn't wrap or duplicate it; see `definition_service.md` §3.4 / `execution_service.md` §5.3 for that contract.

**Wire vs. in-process authority.** For the one shared event, two artifacts must agree: the JSON Schema registered in AWS Glue (the wire contract, governed by `platform-schemagov`, driven off each service's own `api/asyncapi.yaml`) and the Go struct in this module (the in-process contract). The schema stays the cross-language wire authority; the struct is the Go-side compile authority; §9's golden round-trip test is the tripwire that fails CI if they diverge. For the compiled-plan DSL there is no separate wire authority — the wire is an opaque JSON string — so the module's struct is the only authority that exists.

---

## 2. Package Reference: `pkg/dsl`

The compiled-plan DSL, moved field-for-field from Definition's own `internal/core/domain/compiled_plan.go` (byte-identical to the module's copy). Genuinely shared: Definition produces every field below, Execution consumes every field below — this is why the DSL, unlike events, gets the whole type family rather than a curated subset (§5).

### 2.1 `CompiledCollaboration` / `MessageDef`

The top-level entry point — one `CompiledCollaboration` per BPMN collaboration diagram, wrapping every participant pool as a `CompiledPlan`.

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `MainPlan` | `string` | `main_plan` | Name of the primary pool — the Temporal workflow entry point. |
| `Plans` | `[]*CompiledPlan` | `plans` | Every compiled pool, including pools marked `Ignored` (§2.2) — never filtered out at this level. |
| `Messages` | `[]MessageDef` | `messages` | Every named BPMN message crossing pool boundaries in this collaboration. |

`MessageDef`:

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `Name` | `string` | `name` | The BPMN message name, the correlation key both a `sendTask` and its matching `receiveTask` reference. |
| `SourcePlan` | `string` | `source_plan` | Process name of the sending participant. |
| `TargetPlan` | `string` | `target_plan` | Process name of the receiving participant. |

**Definition populates it via** the collaboration-level compile step (`internal/bpmn_compiler/bpmncore/compile.go`), which assembles one `CompiledPlan` per participant pool and collects every `bpmn:messageFlow` into `Messages`.

**Execution consumes it via** `GetCompiledPlanActivity`, the workflow function's first activity (`execution_service.md` §2.1, Compiled-Plan DSL Shape) — fetched exactly once per instance and held for the workflow's entire lifetime, which is what makes an in-flight instance immune to a later republish of the same template. `MainPlan` selects which `CompiledPlan` the root workflow function drives; `Messages` backs the instance-wide, node-keyed message-correlation buffer that matches a `sendTask` to its `receiveTask`, including across sibling parallel branches.

`SchemaVersion int`: major DSL schema version, stamped on `CompiledCollaboration` by the compiler at publish time (§6.1). Major only — minor/patch revisions are additive-by-convention and never need a discriminator. Consumed by `execution_service.md` §2.5's Factory/Strategy compatibility layer.

### 2.2 `CompiledPlan` / `DepartmentDef` / `VisualElementDef`

One `CompiledPlan` per BPMN pool/participant.

`CompiledPlan`:

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `Name` | `string` | `name` | Pool name — matches a `SourcePlan`/`TargetPlan`/`MainPlan` reference. |
| `TaskQueue` | `string` | `task_queue,omitempty` | Intended to resolve the Temporal task queue this pool's instances run on, tier-routed from the tenant's plan (`wf-queue-default` vs. an isolated `wf-queue-<tenant_uuid>`). |
| `Ignored` | `bool` | `ignored,omitempty` | True for a pool the modeler marked out of scope — produces an admin-stub task, not real work. |
| `Departments` | `[]DepartmentDef` | `departments` | Every BPMN lane in this pool. |
| `Execution` | `ExecutionPlan` | `execution` | The step sequence driving the workflow function (§2.4). |
| `VisualElements` | `[]VisualElementDef` | `visual_elements,omitempty` | Diagram-only elements with no execution semantics. |

`DepartmentDef`:

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `ID` | `string` | `id` | BPMN lane identifier — a local, plan-scoped key. |
| `Label` | `string` | `label` | Human-readable lane name. |
| `IAMDepartmentID` | `string` | `iam_department_id,omitempty` | Meant to carry a real IAM department UUID. |
| `Ignore` | `bool` | `ignore,omitempty` | Lane marked out of scope. |
| `Props` | `map[string]string` | `props,omitempty` | Free-form lane-level properties. |
| `Stages` | `[]StageDef` | `stages` | Every task/stage assigned to this lane (§2.3). |

`VisualElementDef`:

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `Kind` | `string` | `kind` | BPMN element type (e.g. `dataStoreReference`). |
| `ID` | `string` | `id` | BPMN element ID. |
| `Name` | `string` | `name,omitempty` | Human-readable label. |

**Definition populates it via** `bpmncore/compile.go`'s per-pool assembly and `bpmncore/qualify.go`'s lane-to-department resolution. `IAMDepartmentID` is **never populated today** — every code path (`bpmncore/traverse.go`'s `DeptOf`, `bpmncore/graph.go`'s `LaneNameFor`) emits only the BPMN lane's `name` attribute, never a real IAM department UUID (Appendix B). `CompiledPlan.TaskQueue` is documented intent only — no compiler logic resolves a tenant's plan tier into a queue name yet (Appendix B).

**Execution consumes it via** `execution_service.md` §3.2 (Worker Topology & Task-Queue Registration) for `TaskQueue` (snapshotted once at instantiation onto `workflow_instance.task_queue`); `Ignored` drives the ignored-pool admin-stub dispatch (§8.2's worked example); `IAMDepartmentID` is the field Execution's own `workflow_task.department_id uuid NOT NULL` column depends on and currently cannot populate correctly until Definition ships a real capture — a confirmed cross-repo blocker (Appendix B), not a mere type mismatch.

### 2.3 `StageDef` / `BoundaryTimer`

One `StageDef` per task/stage in a department's lane.

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `Type` | `string` | `type` | Stage-type discriminator — `pkg/enums.StageType` values (§4.1). |
| `Activity` | `string` | `activity` | The Temporal Activity name this stage type dispatches to. |
| `NodeID` | `string` | `node_id,omitempty` | BPMN element ID — stable, machine-addressable routing target. |
| `Role` | `string` | `role` | Required assignee role/level for this stage. |
| `DefaultAssignees` | `[]string` | `default_assignees,omitempty` | Compile-time default assignee list. |
| `DueDate` | `string` | `due_date,omitempty` | SLA due-date expression. |
| `FollowUpDate` | `string` | `follow_up_date,omitempty` | SLA follow-up-date expression. |
| `BoundaryTimer` | `*BoundaryTimer` | `boundary_timer,omitempty` | An attached BPMN timer boundary event, if any. |
| `BoundaryMessage` | `*MessagePath` | `boundary_message,omitempty` | An attached BPMN message boundary event, if any. |
| `EngineNote` | `string` | `engine_note,omitempty` | Free-text compiler annotation — e.g. the `UNKNOWN_STAGE_TYPE` explanation (§7). |
| `Extras` | `map[string]string` | `extras,omitempty` | Free-form bag (§7). |
| `IsZeebeUserTask` | `bool` | `is_zeebe_user_task,omitempty` | Zeebe-modeler provenance marker. |
| `ConnectorType` | `string` | `connector_type,omitempty` | Set when `Type` is `enums.StageTypeConnector` — the connector's registered name (e.g. `storage`, `send-email`, `rest-call`), parsed once at compile time from the BPMN `<zeebe:taskDefinition type="connector:<name>"/>` attribute's `connector:` prefix. A dedicated field, not folded into the compound `Type` string, so neither side ever re-parses a prefix — same rationale as `department_id` being a real column (`execution_service.md` §4.3). |
| `IOMapping` | `*IOMapping` | `io_mapping,omitempty` | Input/output variable mapping for a connector-typed stage, from the element's `<zeebe:ioMapping>` — reuses the exact same `IOMapping`/`IOVar` shape `CallPoolStep` already uses (§2.5), not reinvented. `nil` for every non-connector stage type. |

`BoundaryTimer`:

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `Duration` | `string` | `duration` | ISO-8601 timer duration. |
| `Interrupting` | `bool` | `interrupting` | Whether firing cancels the task it's attached to. |
| `TargetDept` | `string` | `target_dept,omitempty` | Department the timer routes to on fire. |

**Definition populates it via** a small stage-type registry, `internal/bpmn_compiler/stage_types.go`'s `defaultStageTypes()` map (`"prep"`→`PrepActivity`, `"review"`→`ReviewActivity`, `"approve"`→`ApproveActivity`), plus `element/send_task.go`/`element/receive_task.go` for the `send_task`/`receive_task` types. `IsZeebeUserTask` is set by the parser but has no compiler-side consumer — confirmed inert, write-only (Appendix B). `ConnectorType`/`IOMapping` are populated by the `serviceTask` element handler (`definition_service.md` §4.1.2/§4.1.3.3) — the same handler that produces `Type = enums.StageTypeConnector`.

**Execution consumes it via** `execution_service.md` §2.4 (Stage-Type Dispatch) for `Type`/`Activity`; §2.8 (SLA Semantics) for `DueDate`/`FollowUpDate`, raced against the task's own resolution in a Temporal `Selector`; §2.2 (Boundary Events) for `BoundaryTimer` → `workflow.NewTimer`; §4.3 for `ConnectorType`, snapshotted onto `workflow_task.connector_type` at task creation. `IOMapping.Inputs` applies the same entry-only way `CallPoolStep.IOMapping` already does (§2.5) — copied into `context_json` before the task is created; `IOMapping.Outputs` is interpreted downstream, by whichever process runs the connector worker (`workflow_connectors.md` §6.5), to decide which of its result fields become which `context_json` variables when it calls the existing task-completion path — Execution itself does not interpret `Outputs`.

### 2.4 `ExecutionPlan` / `ExecutionStep`

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `Steps` | `[]ExecutionStep` | `steps` | The ordered step sequence the workflow function walks. |

`ExecutionStep` — a struct of **mutually exclusive optional fields**, not a tagged union; "which variant" is just "which field is non-nil," already expressed in Go's own type system with nothing further to add:

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `Sequential` | `[]string` | `sequential,omitempty` | An ordered list of node IDs to execute in sequence. |
| `Parallel` | `[]ParallelBranch` | `parallel,omitempty` | Concurrent branches (§2.5). |
| `Exclusive` | `[]ExclusiveBranch` | `exclusive,omitempty` | Conditional branches (§2.5). |
| `SubWorkflow` | `*SubWorkflowStep` | `sub_workflow,omitempty` | A nested BPMN subprocess (§2.5). |
| `CallPool` | `*CallPoolStep` | `call_pool,omitempty` | A hand-off to another compiled pool (§2.5). |
| `IOMapping` | `*IOMapping` | `io_mapping,omitempty` | Variable input/output declarations (§2.5). |
| `Extras` | `map[string]string` | `extras,omitempty` | Free-form bag (§7). |
| `MessagePaths` | `[]MessagePath` | `message_paths,omitempty` | Message boundary events attached to this step (§2.5). |

**Definition populates it via** the graph-walk assembly across `bpmncore/{traverse,compile,state,qualify}.go` — each BPMN control-flow construct (sequence flow, gateway, subprocess, call activity) emits exactly one `ExecutionStep` with exactly one of the above fields set.

**Execution consumes it via** `runSteps`, the workflow function's execution algorithm (`execution_service.md` §2.5, Workflow-Function Execution Algorithm) — one non-nil field per step drives dispatch to the matching handler (sequential dispatch, parallel-branch fan-out, exclusive-gateway evaluation, subworkflow/call-pool recursion).

### 2.5 Branch & Step Variants

`ParallelBranch`:

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `DeptID` | `string` | `dept` | The department this branch executes under. |
| `Steps` | `[]ExecutionStep` | `steps` | This branch's own step sequence. |

Execution: `execution_service.md` §2.7 (Force-Back and the Parallel-Gateway History Model) — a failed branch's siblings keep running; `DEGRADED` parks the instance at the aggregation point, not mid-branch.

`ExclusiveBranch`:

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `Target` | `string` | `target,omitempty` | Target department (legacy/human addressing). |
| `TargetStage` | `string` | `target_stage,omitempty` | Target stage within that department. |
| `TargetNodeID` | `string` | `target_node_id,omitempty` | BPMN element ID of the branch's first task — stable, machine-addressable routing. |
| `TargetName` | `string` | `target_name,omitempty` | Human-readable target name. |
| `ConditionExpression` | `string` | `condition_expression` | Evaluated at runtime; empty string is the implicit-else branch. |
| `Terminates` | `bool` | `terminates,omitempty` | True when this branch leads directly to an end event. |
| `RevertToDept` / `RevertToStage` / `RevertToNodeID` / `RevertToName` | `string` | `revert_to_*,omitempty` | Set when this branch is a back-edge (a guarded revert/loop) rather than a forward branch. |

Execution: `execution_service.md` §2.6 (Exclusive-Gateway Evaluation) — a small built-in comparator evaluates `ConditionExpression` for today's dominant binary case; the one empty-`ConditionExpression` branch is the implicit else. The `RevertTo*` fields drive force-back/cyclic-revert flow (§2.7), bounded by Definition's own max-loop-iteration guard against an unbounded revert cycle — the same bound that lets Execution's design skip Continue-As-New (`execution_service.md` Appendix A.2 #28).

`SubWorkflowStep`:

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `NodeID` | `string` | `node_id,omitempty` | BPMN element ID. |
| `Name` | `string` | `name` | Subprocess name. |
| `Plan` | `ExecutionPlan` | `plan` | The subprocess's own nested step sequence. |
| `ErrorPaths` | `[]ErrorPath` | `error_paths,omitempty` | Attached error boundary events. |
| `TimerPaths` | `[]TimerPath` | `timer_paths,omitempty` | Attached timer boundary events. |
| `MessagePaths` | `[]MessagePath` | `message_paths,omitempty` | Attached message boundary events. |

`ErrorPath`:

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `ErrorCode` | `string` | `error_code,omitempty` | BPMN error code to match. |
| `ShortCircuit` | `bool` | `short_circuit` | Whether this path bypasses remaining subprocess steps. |
| `TargetDept` | `string` | `target_dept,omitempty` | Department the path routes to on fire. |

`TimerPath`:

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `Duration` | `string` | `duration` | ISO-8601 timer duration. |
| `Interrupting` | `bool` | `interrupting` | Whether firing cancels the subprocess. |
| `TargetDept` | `string` | `target_dept,omitempty` | Department the path routes to on fire. |

Execution: `execution_service.md` §2.3 (subProcess, CallPool, and callActivity Constructs) for `SubWorkflowStep`'s nested dispatch, and the same §2.2 boundary-event machinery as `StageDef.BoundaryTimer`/`BoundaryMessage` for `ErrorPath`/`TimerPath`/`MessagePath` attached at the subprocess level.

`CallPoolStep`:

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `Pool` | `string` | `pool` | Name of the compiled pool control hands off to. |

Execution: §2.3 there — a Temporal child workflow dispatch; only the main pool may emit this.

`IOMapping` / `IOVar`:

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `Inputs` / `Outputs` | `[]IOVar` | `inputs,omitempty` / `outputs,omitempty` | Variable declarations for a `CallPoolStep`. |
| `Source` / `Target` (`IOVar`) | `string` | `source` / `target` | A single variable mapping. |

Execution: applied **entry-only** — inputs are copied into `context_json` before the segment runs; there is no separate exit re-application, since the segment's own steps write `context_json` directly as they execute (`execution_service.md` §2.1).

`MessagePath`:

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `MessageName` | `string` | `message_name` | Correlates to a `MessageDef.Name` at the collaboration level (§2.1). |
| `Interrupting` | `bool` | `interrupting` | Whether receipt cancels the attached task/subprocess. |
| `TargetDept` | `string` | `target_dept,omitempty` | Department the message routes to on receipt. |

Execution: the instance-wide, node-keyed message-correlation buffer (§2.1 above), including cross-sibling-parallel-branch correlation, with one accepted residual — a consumed message survives a force-back past its sending branch; the re-fire becomes a fresh pending entry (`execution_service.md` §8.2's worked example).

**Definition populates every struct in this subsection via** the same `bpmncore/{traverse,compile,state,qualify}.go` graph walk as §2.4, plus `element/{subprocess,call_activity}.go` for `SubWorkflowStep`/`CallPoolStep` specifically and `element/gateway_xor.go` for `ExclusiveBranch`.

---

## 3. Package Reference: `pkg/events`

Exactly one struct: `TemplatePublishedPayload` — the only event payload that actually crosses the Definition↔Execution boundary.

| Field | Type | JSON tag | Purpose |
| --- | --- | --- | --- |
| `WorkflowID` | `string` | `workflow_id` | The published workflow's ID. |
| `WorkflowKey` | `string` | `workflow_key` | Tenant-scoped human key. |
| `VersionID` | `string` | `version_id` | The specific version published. |
| `VersionNumber` | `int32` | `version_number` | That version's ordinal. |
| `ArtifactHash` | `string` | `artifact_hash` | Content hash of the compiled artifact. |
| `PublishedBy` | `string` | `published_by` | Acting user's IAM UUID. |
| `PromotedFromVersionID` | `*string` | `promoted_from_version_id,omitempty` | Set only when this publish is a promotion of an existing version, never a fresh compile. |

**Definition populates and produces it** at the publish transaction (`definition_service.md` §7.2, Event Payload Schemas; §5, Publish Transaction Flow) — Definition's real, currently-live struct is `internal/core/domain/eventpayloads.go`'s `TemplatePublishedPayload`, byte-identical to this module's copy, mirroring its own governed AsyncAPI `WorkflowTemplatePublishedPayload` schema 1:1.

**Execution consumes it** for compiled-plan cache pre-warm (`execution_service.md` §6.2, Inbound Event Catalogue) — a fetch of the compiled plan via `GetCompiledWorkflow`, refreshing the `workflow_key → active version_id` map in the shared Valkey cache, recency-guarded against out-of-order redelivery, fail-open on fetch failure since the plan loads lazily on the next instantiation regardless.

**Why exactly one struct.** Execution's 18 outbound event payloads (`workflow.instance.*`/`workflow.task.*`) are consumed by Audit/Notification/Tender/LLM/Dashboard — never by Definition — and stay in Execution's own domain package, the same way Definition's own event structs already live in `eventpayloads.go` rather than here. `platform-events`' generic `Envelope[T]` is deliberately not re-exported — both services already import `platform-events` directly for it, and a re-export buys neither service anything neither can already reach (§5).

---

## 4. Package Reference: `pkg/enums`

Shrinks to exactly what `pkg/dsl` and the one shared event need — five `StageType` string constants and one event-type constant, nothing else.

### 4.1 `StageType`

| Constant | Value | Producing registry (Definition) | Dispatch branch (Execution) |
| --- | --- | --- | --- |
| `StageTypePrep` | `"prep"` | `stage_types.go`'s `defaultStageTypes()` | `execution_service.md` §2.4 — `PrepActivity` |
| `StageTypeReview` | `"review"` | same | §2.4 — `ReviewActivity` |
| `StageTypeApprove` | `"approve"` | same | §2.4 — `ApproveActivity` |
| `StageTypeSendTask` | `"send_task"` | `element/send_task.go` | §2.4 + §2.2 (message dispatch) |
| `StageTypeReceiveTask` | `"receive_task"` | `element/receive_task.go` | §2.4 + §2.2 (message correlation) |
| `StageTypeConnector` | `"connector"` | `definition_service.md` §4.1.2/§4.1.3.3's `serviceTask` handler | `execution_service.md` §2.4 — dispatched like `prep`/`review`/`approve`; completed by `workflow-connectors` (`workflow_connectors.md`) |

An unrecognized `StageDef.Type` value is a valid forward-compat passthrough, not a member of an exhaustive set — the validator emits an `UNKNOWN_STAGE_TYPE` warning (not an error) and compiles a passthrough `StageDef` whose `EngineNote` records why (`definition_service.md` §3.1.4, §4.1) — these six are only what the compiler's own registry currently emits, not a closed set IAM is barred from extending (§7).

`ExecutionStep`'s variants have no wire discriminator string to constantize here — as §2.4 states, "which variant" is just "which field is non-nil," already expressed in Go's type system.

### 4.2 `EventTypeTemplatePublished`

`EventTypeTemplatePublished = "workflow.template.published"` — the one shared wire-type constant. Definition remains the sole owner/registrar of this schema in AWS Glue; Execution's own `asyncapi.yaml` documents it only as a `receive` operation for completeness, payload schema owned upstream, never re-registered.

Removed from this package during scope correction (Execution-only, moved to Execution's own enum constants, §5): the 18 outbound wire-type strings, and the payload enums `initiator`/tenant `status`/delegation `scope`/`ended_reason`/force-route `direction` — Definition never references any of these.

---

## 5. Scope Boundary: What's Not In This Module

The module holds only what both Definition Service and Execution Service actually need — not everything either service happens to touch. Applied strictly:

**Execution's 18 outbound event payloads** (`workflow.instance.started`/`.paused`/`.resumed`/`.cancelled`/`.terminated`/`.degraded`/`.failed`/`.finished`/`.force-routed`, `workflow.task.created`/`.claimed`/`.completed`/`.deferred`/`.reassigned`/`.superseded`/`.failed`/`.sla-warning`/`.sla-breached`, per `execution_service.md` §6.4) are consumed by Audit, Notification, Tender, LLM, and the Dashboard Stream Gateway — never by Definition. They stay in Execution's own domain package.

**The 5 inbound payloads Execution decodes that IAM owns** (`delegation.started`, `delegation.ended`, `tenant.state.changed`, `user.deleted`, `user.availability.changed`, per `execution_service.md` §6.2) are Execution-only too — Definition never touches them.

**Their enum values** — `initiator` (`admin`/`tenant_state`/`safety_net`/`ooo`/`degraded_recovery`/`override`/`delegation`), tenant `status`, delegation `scope`/`ended_reason`, force-route `direction` — live in Execution's own domain package for the same reason.

**`platform-events`' `Envelope[T]`** is not re-exported (§3) — a convenience neither service requires from this module.

Why this matters as its own boundary, not an implementation detail: a module that quietly grows to hold "everything Execution happens to touch" stops being a compile-time-shared contract and becomes an unversioned dumping ground neither service can safely evolve independently. The one-struct/five-constant footprint above is deliberate, re-derived once already (rev 1.0 → rev 1.1's scope correction, see Revision history) — not an oversight to be quietly grown back.

---

## 6. Versioning

### 6.1 DSL Schema Versioning

Not yet built. Planned:

- `CompiledCollaboration` gains a `SchemaVersion string`, stamped by the compiler at publish time (`version_publish.go`) — a plan is compiled once and stored immutably, so the version is captured at that moment, never re-derived later.
- `GetCompiledWorkflowResponse` gains an additive `dsl_schema_version` field (non-breaking under `buf`'s `breaking: use: FILE` policy), so Execution can check compatibility without parsing the JSON blob and fail closed on a major-version mismatch.
- Orthogonal to `version_number` (the workflow template's own draft/published revision, not the DSL shape). The producer stays decoupled: Definition need not know what DSL version Execution supports; Execution self-checks the artifact it receives.

### 6.2 Event Versioning

Follows the AsyncAPI `.v2` rule already governing both services' specs: an additive payload change is a new Glue schema version, same wire `type`, and a same-shaped additive Go struct field (module minor version bump); a breaking change bumps `type` to `<name>.v2`, adds a new payload struct alongside the old (kept until retirement), and dual-publishes for one release cycle — Execution's own outbound catalogue settled this window at 30 days (`execution_service.md` §6.8); this module's own event hasn't needed one yet (Appendix B). Consumers pin a module version; forward-compatibility (open schemas, `additionalProperties: true`) means a not-yet-upgraded consumer tolerates an additive producer change at runtime even before it bumps the module.

---

## 7. `Extras` / `IOMapping` Key Registry

The module gives the shape of these bags (`StageDef.Extras`, `ExecutionStep.Extras`, `IOMapping.Inputs`/`Outputs`), not their meaning.

| Key | Meaning | Emitted by | Interpreted by | Passthrough? |
| --- | --- | --- | --- | --- |
| _(populate as keys are given meaning — empty at creation)_ | | | | |

Reserved: an **`exec.`** prefix for keys carrying execution-semantic meaning — routing/gating decisions the workflow function must act on, not display-only or audit-only properties.

Execution's decoder policy:

- Unknown `ExecutionStep` variant → **hard error, fail instance start** — a structural control-flow discriminator with no matching case is workflow corruption, not a forward-compat gap.
- Unknown `StageDef.Type` → **tolerate** — Definition's own compiler already treats this as forward-compat by design (`definition_service.md` §3.1.4/§4.1: `UNKNOWN_STAGE_TYPE` is a warning, not an error; IAM may introduce stage types beyond `prep`/`review`/`approve`, and the task compiles as a passthrough stage). Execution's decoder honors the same policy on its own side.
- Unknown top-level struct field → **tolerate** (Go's zero-value default is backward-safe for an additive producer change).
- Unknown `Extras`/`IOMapping` key **with** the `exec.` prefix → **hard error** (routing/gating semantics this build predates).
- Any other unknown `Extras`/`IOMapping` key → **soft-ignore, pass through, log at debug** (Zeebe custom properties are open by design; hard-failing would block Definition from adding a UI-only or audit-only property).

---

## 8. Definition Service Migration

A mechanical procedure — no behavior change, only where types are defined. Definition migrates via **direct reference**, not a type alias: every caller imports and references `pkg/dsl`/`pkg/events`/`pkg/enums` directly, and the local `domain` copies are deleted outright rather than kept alive under their old names.

1. Tag and publish `workflow-models@vX.Y.Z` (a pre-release tag is sufficient for integration testing; see `VERSIONING.md`) with `pkg/dsl` populated from `compiled_plan.go`'s current field-for-field shape (§2).
2. `go get github.com/BCBP-SOLUTIONS-FZC-LLC/workflow-models@vX.Y.Z` in `workflow-definition-service`; `go mod tidy && go mod vendor`.
3. Delete `internal/core/domain/compiled_plan.go` entirely — every type it declared now lives in `pkg/dsl`. Delete `TemplatePublishedPayload`/`EventTypeTemplatePublished` from `internal/core/domain/eventpayloads.go`, keeping only the Definition-specific `EventSource` constant (not part of the shared module).
4. Repoint every caller at the module directly: `domain.CompiledPlan` → `dsl.CompiledPlan`, `domain.StageDef` → `dsl.StageDef`, `domain.TemplatePublishedPayload` → `events.TemplatePublishedPayload`, `domain.EventTypeTemplatePublished` → `enums.EventTypeTemplatePublished`, and so on for every other `pkg/dsl` type. In practice this touches every file that previously imported `domain` for a compiled-plan or event-payload type — confirmed scope on the first migration: **247 occurrences across 27 files**, spanning `internal/bpmn_compiler/**`, `internal/core/service/**`, `internal/core/port/**` (including the generated mock), `internal/adapter/inbound/grpc/server.go`, and their test counterparts. Where a file already imports `platform-events/pkg/events` under the unaliased name `events`, alias the module's import (e.g. `wfevents`) to avoid a package-name collision — this came up once, in an integration test that used both `events.NewSNSPublisher` (platform-events) and the module's `events.TemplatePublishedPayload` in the same file.
5. Regenerate the mock (`make mock`) rather than hand-editing it, once `internal/core/port/services.go`'s signatures reference `dsl.CompiledPlan`/`dsl.CompiledCollaboration`.
6. Run the full test suite unchanged (`make test`, `make arch-lint`, `make build`) — this is a pure mechanical rename with no behavior change, so no test assertion should need updating; a failure here indicates the module's struct isn't actually field-for-field identical to what it replaced, not a bug in the test. `.go-arch-lint.yml` needs no edit — `depOnAnyVendor: true` already leaves vendored-module imports unrestricted regardless of which internal component does the importing.
7. Add the golden round-trip tests (§9) as new files; wire into CI alongside the existing `make test` target.

Direct reference was chosen over a type alias for one source of truth on the type with no indirection layer, at the cost of a wider one-time diff than an alias would have required (a 2-file change touching only `compiled_plan.go`/`eventpayloads.go`). See Appendix A #8 for the full trade-off.

Execution Service simply imports `pkg/dsl`/`pkg/events` directly from day one — no migration, no interim hand-rolled type ever exists to retire.

---

## 9. Test Plan

One test per artifact class, run in each consuming repo's own CI — not centrally in the module, since each repo's own registered schema is what its own copy must stay honest against.

| Test | Asserts | Lives in | Fails when |
| --- | --- | --- | --- |
| DSL round-trip | Marshal a `pkg/dsl.CompiledCollaboration` fixture → JSON → unmarshal → deep-equal the original; separately, unmarshal a real stored `compiled_plan_json` blob (from a real `workflow_version` row or a `design/Workflows/*.compiled.json` fixture) → the struct has no unexpected zero-valued required field | `workflow-definition-service`'s own test suite, and `workflow-models`'s own `pkg/dsl/roundtrip_test.go` | The compiler starts emitting a shape `pkg/dsl` doesn't have a field for, or a struct field's JSON tag changes without a corresponding compiler update |
| Event round-trip | Marshal a `pkg/events.TemplatePublishedPayload` fixture → JSON → structurally diff against Definition's own registered `internal/eventschema/WorkflowTemplatePublishedPayload.json` (the file `make extract-schemas` produces from `api/asyncapi.yaml`) | `workflow-definition-service`'s own CI, alongside the existing `schema-validate`/`schema-diff` targets | The struct and the registered AsyncAPI schema diverge |
| Execution decode-side sanity | Execution's own test suite decodes a real, currently-registered `workflow.template.published` envelope (a fixture captured from Definition's actual output, not hand-constructed) using `pkg/events.TemplatePublishedPayload` and asserts no field is silently dropped | `workflow-execution-service`'s own test suite (once that repo has code) | Definition's real wire shape and the shared struct have drifted from Execution's own decode assumptions |

This does not replace `platform-schemagov`'s own CI validate/diff/register jobs (§1) — it's an additional, narrower check that the Go representation hasn't drifted from the wire one, running alongside the existing schema-governance pipeline, not instead of it.

---

## 10. Package / File Layout

```text
workflow-models/
├── go.mod                          module github.com/BCBP-SOLUTIONS-FZC-LLC/workflow-models, go 1.26
├── README.MD
├── .gitignore
└── pkg/
    ├── dsl/
    │   ├── collaboration.go        CompiledCollaboration, MessageDef
    │   ├── plan.go                 CompiledPlan, DepartmentDef, VisualElementDef
    │   ├── stage.go                StageDef, BoundaryTimer
    │   ├── execution_step.go       ExecutionPlan, ExecutionStep, ParallelBranch, ExclusiveBranch,
    │   │                           SubWorkflowStep, CallPoolStep, IOMapping, IOVar,
    │   │                           MessagePath, ErrorPath, TimerPath
    │   └── roundtrip_test.go       golden struct↔JSON drift test (package dsl_test)
    ├── enums/
    │   └── stage_type.go           StageType constants + the one shared event's wire-type constant
    └── events/
        ├── template_published.go   TemplatePublishedPayload
        └── roundtrip_test.go       golden struct↔JSON + required-field drift test (package events_test)
```

File split follows the same struct-family grouping `compiled_plan.go` already uses internally (collaboration/plan/stage/step) — a mechanical reorganization into multiple files, not a redesign; each consuming repo's own `go.mod` picks up the module as a single dependency regardless of this internal file count.

---

## 11. Build & Publish Conventions

Publishing/versioning matches the other org private Go libs (`platform-events`/`platform-pgcommon`/`platform-gincommon`): module path `github.com/BCBP-SOLUTIONS-FZC-LLC/workflow-models`, SemVer git tags, private via `GOPRIVATE=github.com/BCBP-SOLUTIONS-FZC-LLC/*`, fetched with `go get …@vX.Y.Z` then `go mod tidy` + `go mod vendor`. Never a local `replace` directive.

Residual build-list items not already covered by §6–§9 above:
- Populate the `Extras`/`IOMapping` key registry (§7) as real keys are identified; add the `exec.`-prefix validator.
- Build Execution's parser and event codec against the module from day one — no interim hand-rolled types.
- Extend `buf`'s breaking-change gate to cover the new `dsl_schema_version` proto field (§6.1) once added.

---

## 12. Repo Scaffolding Gaps vs. Sibling Platform Libraries

`workflow-models`' entire repo root today is `.gitignore`, `README.MD`, `go.mod`, `pkg/` — nothing else. All three sibling private Go libs (`platform-events`, `platform-pgcommon`, `platform-gincommon`) additionally carry the following, none of which exist here yet:

| Missing | Present in | Purpose |
| --- | --- | --- |
| `ARCHITECTURE.md` | all three | Layer model, package dependency graph, per-package public-API tables, sequence-diagram flows, a "key invariants" table, performance characteristics |
| `CHANGELOG.md` | all three | Keep a Changelog format, one entry per release, per-symbol behavior + migration notes |
| `VERSIONING.md` | all three | Explicit SemVer policy — what counts as public API (`pkg/dsl`/`pkg/events`/`pkg/enums` here) vs. internal, MAJOR/MINOR/PATCH criteria |
| `CONTRIBUTING.md` | all three | Contribution workflow |
| `SECURITY.md` | all three | Vulnerability reporting process |
| `Makefile` | all three | `setup`/`lint`/`test`/`test-ci`/`build`/`cover`/`race`/`godoc`/`ci` targets |
| `.golangci.yml` | all three | Lint configuration |
| `.github/` | all three | `CODEOWNERS`, PR template, `dependabot.yml`, issue templates, and CI workflows (`ci.yml`, `validate.yml`, `release.yml`) |

**Not applicable, not a gap:** `docker-compose.yml`, `.env-example`, `Dockerfile` — all three sibling libs carry these to stand up integration-test infrastructure for real I/O (e.g. `platform-events`' LocalStack SNS/SQS containers). `workflow-models` is pure data types with no I/O surface — there is nothing for these to stand up, and adding them would be cargo-culting the sibling libs' shape rather than meeting a real need.

---

## 13. Relationship to `compiled_plan_contract.md`

That doc made the original DSL-half decision (shared module, schema versioning, `Extras`/`IOMapping` registry, unknown-discriminator policy) under the name `platform-workflow-dsl`. This doc supersedes it: the same DSL decisions are carried here verbatim (now as `pkg/dsl`), broadened to also hold event and enum types under the renamed `workflow-models`. `compiled_plan_contract.md` keeps a superseded banner pointing here; no decision from it was reversed — only the name changed and the scope grew.

---

## Appendix A: Design Decisions

| # | Decision | Rationale |
| --- | --- | --- |
| 1 | Named `workflow-models`, not `platform-workflow-dsl` or `platform-models` | Broadens the earlier DSL-only name once the module also holds event and enum types; `platform-models` is IAM's own unbuilt, platform-wide name (`iam-hld.md` §15.4) carrying a cross-team-review requirement this actively-changing, 2-service module doesn't need — referred to IAM (`IAM/platform-models-status-sync.md`), not claimed here. |
| 2 | A shared Go module, not a hand-maintained JSON Schema or proto message | The Go structs are already the source of truth; publishing them as an imported module makes a new field/element a compile-time Go error the instant a consumer bumps the dependency. Proto was rejected for the plan shape specifically: `ExecutionStep` is a 7-way, recursive, variant-heavy union, both consumers are already Go, and a `.proto` would itself become a second hand-synced source of truth. |
| 3 | Wire contract and in-process contract are two separate, both-required artifacts for events | The Glue-registered JSON Schema stays the cross-language wire authority (governed by `platform-schemagov`); this module's Go struct is the Go-side compile authority. A golden round-trip test (§9) is the tripwire that fails CI if they diverge — this module complements schema governance, it doesn't replace it. |
| 4 | Moving `TemplatePublishedPayload` here changes nothing about `platform-schemagov`'s extraction pipeline | Extraction is entirely AsyncAPI-YAML-driven (`api/asyncapi.yaml` → `make extract-schemas` → `internal/eventschema/*.json`), never Go-source-driven. Each service still hand-authors its own `asyncapi.yaml` entry and runs its own independent validate/diff/register job against Glue regardless of where the Go struct lives. |
| 5 | The module's real value is working around Go's `internal/` package visibility, not reducing AsyncAPI/Glue-side duplication | `TemplatePublishedPayload` lives in Definition's `internal/core/domain` today — a compiler-enforced restriction stops Execution importing it. The only alternatives are an independent hand-mirrored struct (the exact failure class this module exists to close) or publishing it once, externally, so both services import the identical definition. |
| 6 | Scope rule: only what both services actually need, re-derived once already | A type only one side reads or writes stays in that service's own domain package. Applied to shrink `pkg/events`/`pkg/enums` from an earlier, broader draft down to one struct and one constant family — Execution's 18 outbound + 5 inbound payloads and their enums are Execution-only (§5). |
| 7 | `platform-events`' `Envelope[T]` is not re-exported | Both services already import `platform-events` directly for it; a re-export is a minor convenience, not something either service requires from this module. |
| 8 | Definition migrates via direct reference to the module, not a type alias | Superseded a rev-2.0 decision to alias (`type CompiledPlan = dsl.CompiledPlan`). Direct reference gives one source of truth for the type with no indirection layer through `domain`, at the cost of a wider one-time diff — 247 occurrences across 27 files, versus the alias approach's 2-file change (`compiled_plan.go`/`eventpayloads.go`). `.go-arch-lint.yml`'s import-direction rules still need no edit either way: `depOnAnyVendor: true` leaves vendored-module imports unrestricted regardless of which internal component does the importing (§8). |
| 9 | `Extras`/`IOMapping` unknown-key decode policy is asymmetric by prefix, not uniform | An unrecognized `exec.`-prefixed key is routing/gating semantics this build predates and must hard-fail; any other unrecognized key is a Zeebe custom property that must soft-ignore, since hard-failing on it would block Definition from adding a UI-only or audit-only property (§7). |
| 10 | Unknown `ExecutionStep` variant is a hard error; unknown `StageDef.Type` tolerates | A missing control-flow discriminator is workflow corruption with no safe default; a new stage type beyond `prep`/`review`/`approve` is an intentional, forward-compat product surface IAM may extend, matching the compiler's own `UNKNOWN_STAGE_TYPE` warning-not-error policy (§7). |
| 11 | `StageDef.ConnectorType` is a dedicated field, not parsed out of a compound `Type` string (`"connector:<name>"`) at runtime | Definition Service already parses the BPMN `connector:` prefix once, at compile time (`definition_service.md` §4.1.2); carrying the raw compound string forward would make Execution Service re-parse it, a repeated-parsing pattern the module has no other precedent for. Matches `department_id`'s own "needs efficient filtering, deserves a real field" precedent. `StageDef.IOMapping` reuses `CallPoolStep`'s existing `IOMapping`/`IOVar` shape rather than inventing a second one, for the same reason `ExecutionStep.IOMapping` itself was reused instead of a bespoke shape when it was first introduced. |

## Appendix B: Open Items

| Item | Owner | Status |
| --- | --- | --- |
| `CompiledCollaboration.SchemaVersion` + `GetCompiledWorkflowResponse.dsl_schema_version` | Definition Service | Factory/Strategy compatibility layer, `execution_service.md` §2.5 (§6.1) |
| `Extras`/`IOMapping` key registry | Definition Service / Execution team | Empty — populate as keys are given real meaning (§7) |
| `platform-models` naming collision | IAM | Referred, no answer yet (`IAM/platform-models-status-sync.md`) |
| This module's own event `.v2` dual-publish window | Execution team | Not yet needed; Execution's own outbound catalogue set 30 days as precedent (`execution_service.md` §6.8) — carry the same number here once a breaking change actually happens (§6.2) |
| `CompiledPlan.TaskQueue` tier-based routing logic | Definition Service | Documented intent only, zero compiler logic exists (§2.2) |
| `DepartmentDef.IAMDepartmentID` never populated | Definition Service | Hard cross-repo blocker — Execution's `workflow_task.department_id uuid NOT NULL` can't be populated until Definition reads a real IAM department UUID from a BPMN lane's `extensionElements` (§2.2) |
| Module tag/publish (`v1.0.0`) | Definition Service | `v0.1.0-beta.1` pre-release tagged and consumed; `v1.0.0` not yet cut (§8) |

### Appendix B.1: Repo Readiness Checklist

One row per §12 gap — Blocker (must exist before the `v1.0.0` tag) or Deferred (doesn't block the first tag).

| # | Item | Type |
| --- | --- | --- |
| 1 | `Makefile` (lint/test/build/ci targets) | Blocker |
| 2 | `.golangci.yml` | Blocker |
| 3 | `.github/workflows/{ci,validate,release}.yml` | Blocker |
| 4 | `VERSIONING.md` | Blocker — the public-vs-internal API scope table is exactly what Definition's migration (§8) needs to know before pinning a version |
| 5 | `CHANGELOG.md` | Blocker — a `v1.0.0` tag with no changelog entry breaks the org's own SemVer/Keep-a-Changelog convention on day one |
| 6 | `ARCHITECTURE.md` | Deferred — this LLD is the design-rationale document; a sibling-lib-format `ARCHITECTURE.md` restating the package reference is a nice-to-have, not a blocker for a 3-package, zero-dependency module |
| 7 | `CONTRIBUTING.md` | Deferred |
| 8 | `SECURITY.md` | Deferred |
| 9 | `.github/` non-CI items (`CODEOWNERS`, PR template, `dependabot.yml`, issue templates) | Deferred |
| 10 | `docker-compose.yml` / `.env-example` / `Dockerfile` | **N/A** — pure data-types module, no I/O surface to stand up integration-test infrastructure for (§12) |

---

## Revision history

| Rev | Date | Change |
| --- | --- | --- |
| 1.0 | 2026-07-17 | Initial doc, superseding `compiled_plan_contract.md`. Broadened the shared module from DSL-only (`platform-workflow-dsl`) to DSL + event payload/envelope types + shared enums, renamed `workflow-models`. Carried the DSL decisions (aliasing, `SchemaVersion` + `dsl_schema_version`, `Extras`/`IOMapping` registry + `exec.` prefix + decode policy) verbatim; added `pkg/events` (mirrors both services' AsyncAPI `<Name>Payload` schemas + re-exports `platform-events` `Envelope[T]`), `pkg/enums`, event `.v2` versioning aligned with the AsyncAPI rule, the wire-vs-in-process authority split, and a per-payload golden round-trip drift tripwire. Naming decision (`workflow-models`, not `platform-models`, not `platform-workflow-dsl`) surfaced for review. |
| 1.1 | 2026-07-21 | Expanded into an implementation-ready spec. Scope correction: `pkg/events`/`pkg/enums` shrink to only what both services genuinely need — one struct (`TemplatePublishedPayload`) and one wire-type constant plus the `StageDef.Type` values; Execution's 18 outbound payloads, its 5 IAM-owned inbound payloads, and their enums (`initiator`, `scope`, `ended_reason`, `direction`) are Execution-only and stay in Execution's own domain package, not this module. Dropped the `Envelope[T]` re-export. Added a concrete package/file layout, Definition Service's migration-to-alias procedure, a per-artifact-class test plan, and a precise statement of what moving to a shared module does and doesn't change about `platform-schemagov`. |
| 2.0 | 2026-07-24 | Rewritten into full LLD format: Table of Contents, numbered sections, a field-level table per struct in every package with an explicit "Definition populates it via" / "Execution consumes it via" pair citing real files (`internal/bpmn_compiler/**`, `internal/core/domain/*.go`) and real `execution_service.md`/`definition_service.md` section numbers (corrected two stale citations against the sibling doc's actual current section numbering in the process), a dedicated scope-boundary section (§5), Appendix A (Design Decisions) and Appendix B (Open Items) distilled from the prior revisions' prose, and a new Appendix B.1 (Repo Readiness Checklist) closing a gap analysis against the three sibling platform libraries (`platform-events`/`platform-pgcommon`/`platform-gincommon`) — `workflow-models` is missing `ARCHITECTURE.md`/`CHANGELOG.md`/`VERSIONING.md`/`CONTRIBUTING.md`/`SECURITY.md`/`Makefile`/`.golangci.yml`/`.github/` relative to all three; `docker-compose.yml`/`.env-example`/`Dockerfile` are confirmed not applicable (no I/O surface). No design decision reversed — restructured and cross-referenced against Execution's actual spec sections and Definition's actual current source. |
| 2.1 | 2026-07-26 | `workflow-models@v0.1.0-beta.1` tagged and consumed by `workflow-definition-service` for integration testing. §8 and Appendix A #8 revised: Definition migrates via **direct reference** to `pkg/dsl`/`pkg/events`/`pkg/enums`, not the type alias originally specified — `compiled_plan.go`/`eventpayloads.go` are deleted outright and every caller (247 occurrences across 27 files) repoints at the module directly, confirmed working end-to-end (`make test`/`make arch-lint`/`make build` all green). This reverses the rev-1.0/2.0 alias decision; the trade-off (wider diff, no indirection layer) was made deliberately, not discovered as a defect in the prior approach. |
| 2.2 | 2026-08-11 | Connector-task support added to `pkg/dsl`/`pkg/enums`, closing a gap `automatic_connector_tasks.md` had described conceptually but never given concrete Go field names. §2.3: `StageDef` gains `ConnectorType string` (`connector_type,omitempty`) and `IOMapping *IOMapping` (`io_mapping,omitempty`, reusing §2.5's existing `IOMapping`/`IOVar` shape). §4.1: new `StageTypeConnector = "connector"` constant, sixth member of the `StageType` table. Appendix A gained decision #11 (dedicated field vs. compound-string parsing). Full worker-runtime and connector-catalogue design that consumes these fields now lives in `workflow_connectors.md`. |
| 2.3 | 2026-08-11 | `automatic_connector_tasks.md` consolidated into `workflow_connectors.md`; the former deleted, the latter's internal sections renumbered. §2.3's `IOMapping.Outputs` citation repointed at the new §6.5 (Runtime Loop) — no field or behavior change, citation-only. |
</content>
