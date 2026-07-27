# Contributing

## Prerequisites

- Go 1.26.4+
- `GOPRIVATE=github.com/BCBP-SOLUTIONS-FZC-LLC/*` set in your shell
- No Docker required — this module has no integration/e2e tier (pure data
  types, no I/O).

## Dev setup

```bash
git clone git@github.com:BCBP-SOLUTIONS-FZC-LLC/workflow-models.git
cd workflow-models
make setup    # go mod download
make tidy
make lint
make test
```

## Project layout

```
workflow-models/
├── go.mod
├── README.MD
├── ARCHITECTURE.md
├── VERSIONING.md
├── CHANGELOG.md
├── SECURITY.md
├── Makefile
├── .golangci.yml
└── pkg/
    ├── dsl/      ← compiled-plan DSL types (CompiledCollaboration, CompiledPlan, StageDef, ExecutionStep, ...)
    ├── events/   ← TemplatePublishedPayload
    └── enums/    ← StageType constants + EventTypeTemplatePublished
```

There is no `internal/`, no `cmd/`, and no separate `test/` directory — tests
are co-located with the source they cover (`roundtrip_test.go` in each
package, using the black-box `dsl_test`/`events_test` package names).

## Adding a new type or field

1. Add or modify the struct in `pkg/dsl` or `pkg/events`.
2. Update that package's `roundtrip_test.go` golden fixture to exercise the
   new field — this is the drift tripwire; a fixture that doesn't cover the
   new field won't catch a shape regression later.
3. Run `make test` and confirm it passes.
4. If it's a new discriminator value, add the constant to `pkg/enums`.
5. Cross-check the field's semantics against the design doc
   (`design/LLD/workflow_models_lib.md` §2–§7) — this module's shape must
   stay field-for-field aligned with what Definition's compiler actually
   produces and what Execution's design actually consumes.
6. Update `CHANGELOG.md`'s `[Unreleased]` section.

## Testing requirements

| Layer | Location | Notes |
|---|---|---|
| Round-trip (golden) | `pkg/dsl/roundtrip_test.go`, `pkg/events/roundtrip_test.go` | marshal → unmarshal → DeepEqual; the drift tripwire (design doc §9). This is the only test tier — no unit/integration/e2e split, since there's no I/O to integration-test. |

## Before opening a PR

- [ ] `make ci` passes locally (`tidy vet lint test-ci build`)
- [ ] Golden round-trip fixtures updated for any struct/field change
- [ ] `CHANGELOG.md` `[Unreleased]` section updated
- [ ] New exported types/fields have godoc comments (enforced by `make lint`)
- [ ] Use the PR template's checklist

## Releasing

See [VERSIONING.md](./VERSIONING.md) for the full SemVer policy and the
maintainer release process — contributors don't need to tag releases
themselves, but should understand what counts as a MAJOR/MINOR/PATCH change
before proposing one.
