# Changelog

All notable changes to `workflow-models` will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0-beta.1] - 2026-07-24

### Added

- `pkg/dsl` — the compiled-plan DSL (`CompiledCollaboration`, `CompiledPlan`, `DepartmentDef`, `StageDef`, `ExecutionPlan`/`ExecutionStep` and its branch/step variants).
- `pkg/events` — `TemplatePublishedPayload`, the shared `workflow.template.published` event payload.
- `pkg/enums` — `StageType` constants and `EventTypeTemplatePublished`.
- Golden round-trip tests for `pkg/dsl` and `pkg/events`.
