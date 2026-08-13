# Changelog

All notable changes to `workflow-models` will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `CompiledCollaboration.SchemaVersion` — the DSL schema major version, stamped by the compiler at publish time. `dsl.CurrentSchemaVersion` is the value this module encodes.
- `enums.StageTypeConnector` and `StageDef.ConnectorType`/`StageDef.IOMapping` — the connector-task DSL shape (`design/LLD/workflow_connectors.md` §3), reusing the existing `IOMapping`/`IOVar` shape rather than a new one.

## [1.0.0] - 2026-07-27

### Added

- `pkg/dsl` — the compiled-plan DSL (`CompiledCollaboration`, `CompiledPlan`, `DepartmentDef`, `StageDef`, `ExecutionPlan`/`ExecutionStep` and its branch/step variants).
- `pkg/events` — `TemplatePublishedPayload`, the shared `workflow.template.published` event payload.
- `pkg/enums` — `StageType` constants and `EventTypeTemplatePublished`.
- Golden round-trip tests for `pkg/dsl` and `pkg/events`.
