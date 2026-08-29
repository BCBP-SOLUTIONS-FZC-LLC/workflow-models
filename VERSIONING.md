# Versioning and releases

This repository is a **Go module** consumed by `workflow-definition-service`
and the Execution Service. Versions are published with **Git tags**
and described in [CHANGELOG.md](./CHANGELOG.md).

## Semantic versioning (SemVer)

We use [SemVer 2.0.0](https://semver.org/): `MAJOR.MINOR.PATCH` (e.g. `v1.0.0`).

| Bump | When you change | Examples |
|------|-----------------|----------|
| **MAJOR** | Breaking change in the public API (`pkg/dsl`, `pkg/enums`) | Removed/renamed exported field, incompatible struct-shape change, removed a `StageType` constant |
| **MINOR** | New backward-compatible capability | New optional field (`omitempty`), new `StageType` constant, new struct |
| **PATCH** | Backward-compatible fix | Doc-only correction, internal test fixture update |

### What counts as public API

| In scope (SemVer applies) | Out of scope |
|---------------------------|---------------|
| `pkg/dsl/*` — every exported type and field | Test-only helpers/fixtures (`roundtrip_test.go`, the `dsl_test` package) |
| `pkg/enums/*` — every exported constant | |

Unlike the sibling libraries (`platform-events`, `platform-pgcommon`,
`platform-gincommon`), this module has **no `internal/` package** — every
exported symbol under `pkg/` is public API (see [ARCHITECTURE.md](./ARCHITECTURE.md)).

### Guarantees

- **MAJOR `v1`:** no breaking changes in `pkg/*` within `v1.x`.
- **MINOR:** safe to `go get` without code changes.
- **PATCH:** drop-in replacement.

### Wire format guarantee

Every `pkg/dsl` struct is JSON-tagged and covered by a
golden round-trip test (`roundtrip_test.go` in each package). A field whose
`json:"..."` tag changes name, or that stops round-tripping through
`json.Marshal`/`json.Unmarshal`, is a **MAJOR** bump even if the Go field
name is unchanged — Definition and Execution communicate through the JSON
wire shape, not the Go struct name.

## Supported releases

| Version | Status | Go module | Supported until |
|---------|--------|-----------|-----------------|
| `< v1.0.0` | — | — | No tagged releases yet |

*(Update this table at the first tag.)*

## Consume a release

This is a **private module**.

```bash
go env -w GOPRIVATE=github.com/BCBP-SOLUTIONS-FZC-LLC/*
```

```bash
go get github.com/BCBP-SOLUTIONS-FZC-LLC/workflow-models@vX.Y.Z
go mod tidy
go mod vendor   # if the consuming service vendors dependencies
```

Never add a local `replace` directive pointing at a filesystem path — always
consume a tagged version.

## Maintainer release process

1. **Merge** all changes for the release to `main`.
2. **Update `CHANGELOG.md`:** move `[Unreleased]` entries into a new
   `## [X.Y.Z] - YYYY-MM-DD` section. The release workflow reads this section
   to populate the GitHub Release body.
3. **Run `make ci` locally** to confirm everything is green before tagging.
4. **Create and push an annotated tag:**
   ```bash
   git tag -a vX.Y.Z -m "vX.Y.Z"
   git push origin vX.Y.Z
   ```
5. **The release workflow** re-validates at the tagged commit and creates a
   GitHub Release with the CHANGELOG section as release notes.
6. **Notify consumers** (workflow-definition-service / execution-service
   teams) with upgrade notes if MINOR or MAJOR.

### Pre-release tags (optional)

| Tag pattern | Meaning |
|-------------|---------|
| `v1.1.0-rc.1` | Release candidate |
| `v1.1.0-beta.1` | Early integration testing |

## Compatibility matrix (library ↔ Go)

| workflow-models | Go (`go.mod`) |
|-----------------|---------------|
| `v1.0.x` (planned) | `1.26.4+` |

## Related files

| File | Purpose |
|------|---------|
| [CHANGELOG.md](./CHANGELOG.md) | User-facing history per version |
| [README.MD](./README.MD) | Scope summary and quick links |
| [CONTRIBUTING.md](./CONTRIBUTING.md) | Development guide and PR checklist |
| [.github/workflows/release.yml](./.github/workflows/release.yml) | Automated release pipeline |
| [go.mod](./go.mod) | Module path and minimum Go version |
