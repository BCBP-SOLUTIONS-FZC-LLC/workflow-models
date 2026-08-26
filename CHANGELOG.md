# Changelog

All notable changes to `workflow-models` will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
