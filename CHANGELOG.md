# Changelog

All notable changes to `workflow-models` will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.3.0-rc.3] - 2026-10-01

### Added

- `NodeKey(deptID, stage)`: a stage's key in both services, `<deptID>/<NodeID>` or `<deptID>/<Type>` without a `NodeID`.
- `StageDef.CreatesHumanTask`: false for a connector, a send task and a receive task.
- `StageDef.Name`: the BPMN task's name.

### Changed

- The goldens are re-made by Definition's publish and carry each stage's name.

## [1.3.0-rc.2] - 2026-09-26

### Added

- `dsltest.LibraryCallBoundaries`: a golden with an interrupting timer boundary and an interrupting message boundary on calls, each leading to an end event.
- `Terminates` on `BoundaryTimer`, `MessagePath`, `ErrorPath` and `TimerPath`: the boundary's path ends at an end event. Every boundary sets exactly one of `TargetDept` and `Terminates`.
- `CallPlanStep.Assignees` keys may be a path of calls to a task further down (`Review_Check::Check_Task`); the outermost call's entry wins.

### Changed

- `ExpandCalls` refuses a boundary that sets both or neither of `TargetDept` and `Terminates`, and an `Assignees` path through a call the plan does not make. An empty target no longer stands for the end of a path.
- The goldens are re-made by Definition's publish: a laneless module's node keys name it by process id without its version (`CA_Eng::Process_Review/Review_Task`), every task has a default user, the module keeps its own message, and boundaries to an end event terminate.

## [1.3.0-rc.1] - 2026-09-25

### Added

- `ExecutionStep.CallPlan` / `CallPlanStep`: a `callActivity` that calls another plan of the collaboration, compiled once, with its department bindings, call-site default assignees and boundary paths.
- `ExpandCalls` and `ErrPlanTooLarge`: the shared, pure expansion of `CallPlan` steps into `SubWorkflowStep`s with call-scoped department IDs (`<NodeID>::<department>`). Definition and Execution both run it, so node keys and IAM departments agree. A consumer must run it before interpreting a plan: an interpreter that meets a `call_plan` step treats it as an unknown variant.
- `pkg/dsl/dsltest` with `LibraryCalls`: a golden compiled collaboration written by Definition's publish and run by Execution's tests.

## [1.2.0-rc.4] - 2026-08-29

### Removed

- `pkg/events` (`TemplatePublishedPayload`) and `enums.EventTypeTemplatePublished` — the `workflow.template.published` event is retired; its only real behavior (Execution's compiled-plan cache prewarm) was already dead code, leaving nothing but payload validation and dedup-recording. Definition Service no longer publishes it, and Execution Service no longer consumes it.

### Added

- `enums.AllowedBPMNElements` — the shared Tier-1 BPMN element allowlist (`design/LLD/definition_service.md` §4.1.2), read by Definition Service's compiler for enforcement and by its new `GET /bpmn/allowed-elements` discovery endpoint (§3.3.20) for the modeler UI's palette. Previously defined only as inline logic in Definition Service's own `bpmn_compiler` package with no shared representation. Audited against Definition Service's actual parser (`bpmncore`) before this list was wired as real enforcement there: added `task`, `dataStoreReference`, `timerEventDefinition`, `errorEventDefinition`, `messageEventDefinition`, `timeDuration`, `incoming`, `outgoing` — all already-supported elements the first pass of this list omitted, which would otherwise have started rejecting them the moment the compiler began enforcing this list.

## [1.2.0-rc.2] - 2026-08-14

### Added

- `CompiledCollaboration.SchemaVersion` — the DSL schema major version, stamped by the compiler at publish time. `dsl.CurrentSchemaVersion` is the value this module encodes.
- `enums.StageTypeConnector` and `StageDef.ConnectorType`/`StageDef.IOMapping` — the connector-task DSL shape (`design/LLD/workflow_connectors.md` §3), reusing the existing `IOMapping`/`IOVar` shape rather than a new one.

## [1.0.0] - 2026-07-27

### Added

- `pkg/dsl` — the compiled-plan DSL (`CompiledCollaboration`, `CompiledPlan`, `DepartmentDef`, `StageDef`, `ExecutionPlan`/`ExecutionStep` and its branch/step variants).
- `pkg/events` — `TemplatePublishedPayload`, the shared `workflow.template.published` event payload.
- `pkg/enums` — `StageType` constants and `EventTypeTemplatePublished`.
- Golden round-trip tests for `pkg/dsl` and `pkg/events`.
