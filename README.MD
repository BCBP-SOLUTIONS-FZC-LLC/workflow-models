# workflow-models

Shared Go module between Definition Service and Execution Service; every type crossing that service boundary lives here once, instead of being hand-mirrored independently on each side.

## Scope

- `pkg/dsl` — the compiled-plan DSL. Definition's BPMN compiler produces every field; Execution's workflow function consumes every field — the full contract, not a curated subset.
- `pkg/enums` — `StageDef.Type` discriminator values.

A type only one side reads or writes stays in that service's own domain package, not here.

## Contents

- [workflow-models](#workflow-models)
  - [Scope](#scope)
  - [Contents](#contents)
  - [Package dependency graph](#package-dependency-graph)
  - [Scope boundary](#scope-boundary)
  - [Adding to a consumer service](#adding-to-a-consumer-service)
  - [Quick start — pkg/dsl](#quick-start--pkgdsl)
  - [Package reference — pkg/dsl](#package-reference--pkgdsl)
  - [Package reference — pkg/enums](#package-reference--pkgenums)
  - [Key invariants](#key-invariants)
  - [Testing model](#testing-model)
  - [Versioning and releases](#versioning-and-releases)
  - [See also](#see-also)

## Package dependency graph

Neither package imports the other:

```mermaid
graph LR
    dsl(["pkg/dsl"])
    enums(["pkg/enums"])
```

## Scope boundary

- `internal/core/domain/compiled_plan.go` in `workflow-definition-service` defines the same types as `pkg/dsl` (`CompiledCollaboration`, `CompiledPlan`, `DepartmentDef`, `StageDef`, `ExecutionPlan`/`ExecutionStep` and its branch/step variants) — see [VERSIONING.md](./VERSIONING.md) and the design doc (§8) for the type-alias consolidation this module is the target of.
- Execution's 18 outbound events (`workflow.instance.*`, `workflow.task.*` — e.g. `workflow.instance.paused`, `workflow.task.reassigned`, `workflow.task.sla-breached`) belong to Execution's own domain package; consumers are Audit, Notification, Tender, LLM, and the Dashboard Stream Gateway, not Definition.
- The 5 IAM-owned inbound events Execution decodes — `DelegationStarted`, `DelegationEnded`, `TenantStateChanged`, `UserDeleted`, `UserAvailabilityChanged` — and their enum values (`initiator`, tenant `status`, delegation `scope`/`ended_reason`, force-route `direction`) live in Execution's own domain package.
- `platform-events`' `Envelope[T]` is not re-exported here — both services import `platform-events` directly for it.

## Adding to a consumer service

This is a **private module**:

```bash
go env -w GOPRIVATE=github.com/BCBP-SOLUTIONS-FZC-LLC/*
```

```bash
go get github.com/BCBP-SOLUTIONS-FZC-LLC/workflow-models@vX.Y.Z
go mod tidy
go mod vendor   # if the consuming service vendors dependencies
```

Never add a local `replace` directive pointing at a filesystem path — always consume a tagged version.

## Quick start — pkg/dsl

```go
import "github.com/BCBP-SOLUTIONS-FZC-LLC/workflow-models/pkg/dsl"

plan := &dsl.CompiledCollaboration{
    MainPlan: "Bid-No-Bid Review",
    Plans: []*dsl.CompiledPlan{
        {
            Name: "Bid-No-Bid Review",
            Departments: []dsl.DepartmentDef{
                {
                    ID:    "Bid-No-Bid Review/Tender Business",
                    Label: "Tender Business",
                    Stages: []dsl.StageDef{
                        {
                            Type:     "receive_task", // enums.StageTypeReceiveTask
                            Activity: "RFQ received",
                            Role:     "tender-business",
                            BoundaryTimer: &dsl.BoundaryTimer{
                                Duration:     "72h",
                                Interrupting: true,
                            },
                        },
                    },
                },
            },
            Execution: dsl.ExecutionPlan{
                Steps: []dsl.ExecutionStep{
                    {Sequential: []string{"Bid-No-Bid Review/Tender Business"}},
                },
            },
        },
    },
}

b, err := json.Marshal(plan)
// ...
var decoded dsl.CompiledCollaboration
err = json.Unmarshal(b, &decoded)
```

`StageDef.Type` is a plain `string` field, not `enums.StageType` — populate it with an `enums.StageType*` constant's value (`enums.StageTypeReceiveTask`) or the equivalent raw string.

## Package reference — pkg/dsl

```text
CompiledCollaboration                 (root artifact)
 ├─ Plans   []*CompiledPlan
 │            ├─ Departments []DepartmentDef
 │            │                ├─ Stages []StageDef
 │            │                │            ├─ BoundaryTimer  *BoundaryTimer
 │            │                │            └─ BoundaryMessage *MessagePath
 │            │                └─ (props map)
 │            ├─ Execution ExecutionPlan
 │            │              └─ Steps []ExecutionStep
 │            │                         ├─ Parallel  []ParallelBranch      → Steps []ExecutionStep (recursive)
 │            │                         ├─ Exclusive []ExclusiveBranch     → Steps []ExecutionStep (recursive)
 │            │                         ├─ SubWorkflow *SubWorkflowStep
 │            │                         │                ├─ Plan ExecutionPlan (recursive)
 │            │                         │                ├─ ErrorPaths []ErrorPath
 │            │                         │                ├─ TimerPaths []TimerPath
 │            │                         │                └─ MessagePaths []MessagePath
 │            │                         ├─ CallPlan  *CallPlanStep       → a plan in Plans, by name (see ExpandCalls)
 │            │                         ├─ IOMapping *IOMapping → Inputs/Outputs []IOVar
 │            │                         └─ MessagePaths []MessagePath
 │            └─ VisualElements []VisualElementDef
```

`ExecutionStep` is a struct of mutually exclusive optional fields, not a tagged union — which variant applies is just which field is non-nil.

| Type | Fields |
| --- | --- |
| `CompiledCollaboration` | `MainPlan string`, `Plans []*CompiledPlan`, `SchemaVersion int` |
| `CompiledPlan` | `Name`, `TaskQueue string,omitempty`, `Departments []DepartmentDef`, `Execution ExecutionPlan`, `VisualElements []VisualElementDef,omitempty` |
| `VisualElementDef` | `Kind`, `ID`, `Name,omitempty` |
| `DepartmentDef` | `ID`, `Label`, `IAMDepartmentID string,omitempty`, `Props map[string]string,omitempty`, `Stages []StageDef` |
| `StageDef` | `Type string` (holds `enums.StageType*` values), `Activity`, `NodeID,omitempty`, `Name,omitempty`, `Role`, `DefaultAssignees []string,omitempty`, `DueDate,omitempty`, `FollowUpDate,omitempty`, `BoundaryTimer *BoundaryTimer,omitempty`, `BoundaryMessage *MessagePath,omitempty`, `EngineNote,omitempty`, `Extras map[string]string,omitempty`, `IsZeebeUserTask bool,omitempty`, `ConnectorType string,omitempty`, `IOMapping *IOMapping,omitempty` |
| `BoundaryTimer` | `Duration`, `Interrupting bool`, `TargetDept,omitempty`, `Terminates bool,omitempty` |
| `ExecutionPlan` | `Steps []ExecutionStep` |
| `ExecutionStep` | `Sequential []string,omitempty`, `Parallel []ParallelBranch,omitempty`, `Exclusive []ExclusiveBranch,omitempty`, `SubWorkflow *SubWorkflowStep,omitempty`, `CallPlan *CallPlanStep,omitempty`, `IOMapping *IOMapping,omitempty`, `Extras map[string]string,omitempty`, `MessagePaths []MessagePath,omitempty` |
| `MessagePath` | `MessageName`, `Interrupting bool`, `TargetDept,omitempty`, `Terminates bool,omitempty` |
| `IOMapping` | `Inputs []IOVar,omitempty`, `Outputs []IOVar,omitempty` |
| `IOVar` | `Source`, `Target` |
| `ParallelBranch` | `DeptID`, `Steps []ExecutionStep` (recursive) |
| `ExclusiveBranch` | `Target,omitempty`, `TargetStage,omitempty`, `TargetNodeID,omitempty`, `TargetName,omitempty`, `ConditionExpression` (no omitempty), `Terminates bool,omitempty`, `Steps []ExecutionStep,omitempty`, `RevertToDept/RevertToStage/RevertToNodeID/RevertToName,omitempty` |
| `SubWorkflowStep` | `NodeID,omitempty`, `Name`, `Plan ExecutionPlan` (recursive), `ErrorPaths []ErrorPath,omitempty`, `TimerPaths []TimerPath,omitempty`, `MessagePaths []MessagePath,omitempty` |
| `ErrorPath` | `ErrorCode,omitempty`, `ShortCircuit bool` (no omitempty), `TargetDept,omitempty`, `Terminates bool,omitempty` |
| `TimerPath` | `Duration`, `Interrupting bool`, `TargetDept,omitempty`, `Terminates bool,omitempty` |
| `CallPlanStep` | `NodeID`, `Name,omitempty`, `Plan`, `Departments map[string]string,omitempty`, `Assignees map[string]string,omitempty`, `ErrorPaths`/`TimerPaths`/`MessagePaths,omitempty` |

`ExpandCalls(c, maxStages)` returns a copy of a collaboration with every `CallPlan` step replaced by the `SubWorkflowStep` Execution runs, and the called plan's departments cloned into the caller as `<NodeID>::<department>`, bound to the caller's IAM departments. Both services run it, so they agree on node keys. A call's `Assignees` key names a task of the called plan, or a path of calls to one further down (`Review_Check::Check_Task`), the outermost call's entry winning. Every boundary sets exactly one of `TargetDept` and `Terminates`, and `ExpandCalls` refuses one that does not. `ErrPlanTooLarge` reports a plan over `maxStages` stages after expansion. See the design doc §2.6.

## Package reference — pkg/enums

| Constant | Value |
| --- | --- |
| `StageTypePrep` | `"prep"` |
| `StageTypeReview` | `"review"` |
| `StageTypeApprove` | `"approve"` |
| `StageTypeSendTask` | `"send_task"` |
| `StageTypeReceiveTask` | `"receive_task"` |
| `StageTypeConnector` | `"connector"` |

An unrecognized `StageDef.Type` value is a valid forward-compat passthrough, not membership in a closed set — the compiler emits `UNKNOWN_STAGE_TYPE` as a warning, not an error, and compiles a passthrough `StageDef`.

## Key invariants

| Invariant | Why it matters |
| --- | --- |
| `Extras`/`IOMapping` unknown-key decode policy is asymmetric by prefix | An unrecognized `exec.`-prefixed key is routing/gating semantics a consumer predates and must hard-fail; any other unrecognized key is a Zeebe custom property that must soft-ignore. |
| Unknown `ExecutionStep` variant is a hard error; unknown `StageDef.Type` tolerates | A missing control-flow discriminator is workflow corruption with no safe default. A new stage type beyond `prep`/`review`/`approve` is an intentional, forward-compat product surface. |
| A `json:"..."` tag change is a MAJOR version bump | Definition and Execution communicate over the JSON wire shape, not the Go struct/field name. |

## Testing model

`pkg/dsl` has a `roundtrip_test.go`: marshal a fixture → JSON → unmarshal → `reflect.DeepEqual` against the original — the drift tripwire. `expand_test.go` covers `ExpandCalls` and `stage_test.go` covers `NodeKey` and `StageDef.CreatesHumanTask`, the package's executable code; everything else in `pkg/dsl` and `pkg/enums` is a declaration. No numeric coverage gate is wired into CI.

`pkg/dsl/dsltest` holds golden compiled plans, written only by Definition's golden test from its real publish and run by Execution's tests, so both services test one contract (design doc §2.7). `LibraryCalls()` is a workflow calling one library module from two lanes, with call-site assignees, a nested module whose task the workflow's calls name by path, and a module that keeps its own message. `LibraryCallBoundaries()` adds an interrupting timer and message boundary on calls, each leading to an end event and so terminating. `FlowOrder()` runs its tasks in flow order across two lanes, with a return to a lane, a call on an exclusive branch and a parallel fork whose branches share a lane.

## Versioning and releases

See [VERSIONING.md](./VERSIONING.md) for the full release process. Summary:

| Version bump | When |
| --- | --- |
| **MAJOR** | Breaking/renamed exported field in `pkg/dsl` or `pkg/enums`; a wire-format (JSON tag) change |
| **MINOR** | New backward-compatible field, `StageType`, or event-type constant |
| **PATCH** | Doc-only correction, internal test fixture update |

```bash
# Tag and push triggers the release workflow automatically.
git tag -a v1.0.0 -m "v1.0.0"
git push origin v1.0.0
```

## See also

| Document | Description |
| --- | --- |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Package dependency graph, testing model, key invariants |
| [VERSIONING.md](./VERSIONING.md) | SemVer rules, supported versions, release process |
| [CHANGELOG.md](./CHANGELOG.md) | Per-version changes |
| [CONTRIBUTING.md](./CONTRIBUTING.md) | Development setup, PR checklist |
| [SECURITY.md](./SECURITY.md) | Vulnerability reporting, trust model |
