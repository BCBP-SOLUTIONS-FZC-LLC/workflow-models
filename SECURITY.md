# Security policy

## Supported versions

| Version | Supported |
|---------|-----------|
| `< v1.0.0` | No tagged releases yet |

*(Update this table at the first tag; when a new major line ships, the
previous major receives security fixes only for six months.)*

## Reporting a vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Report security issues by email to **vijay@bcbpsolutions.com** with the
subject line `[workflow-models] Security vulnerability`.

Include:

- A description of the vulnerability and its potential impact.
- Steps to reproduce (a minimal Go test is sufficient).
- The version(s) affected.
- Any suggested fix, if you have one.

### Response timeline

| Step | Target |
|------|--------|
| Initial acknowledgement | 48 hours |
| Severity assessment | 5 business days |
| Patch release (critical/high) | 14 days |
| Public disclosure | After patch ships |

We follow responsible disclosure. Reporters will be credited in release
notes unless anonymity is requested.

## Trust model and known limitations

This module is **passive data types only**: JSON-tagged Go structs and
string/const discriminators. It performs no I/O, opens no network
connections, executes no external code, and holds no secrets or
credentials — a materially smaller attack surface than the sibling
libraries (`platform-events`/`platform-pgcommon`/`platform-gincommon`),
which publish to SNS/SQS or connect to Postgres.

| Assumption | Implication |
|-----------|-------------|
| This module never decodes untrusted JSON itself | Decoding happens in the consuming service (Definition or Execution); this module only defines the shape. Bounding payload size, depth, and field counts is the consuming service's responsibility. |
| Struct field values are opaque strings/enums | Beyond Go's type system, the only checks are `ExpandCalls`'s structural ones (plan references, cycles, bindings, a stage budget); a malicious `Extras`/`IOMapping` value is a consuming-service concern, not this module's. |

## Scope

In scope for vulnerability reports:

- Supply-chain vulnerabilities in this module's own dev-tooling dependencies
  (`go.sum`, surfaced via `make vuln-check`).
- `dsl.ExpandCalls`, the one code path in `pkg/dsl` or `pkg/enums` that is
  not pure data: it runs on plans the consuming services decode, so a plan
  that makes it loop, panic or allocate without bound is in scope.

Out of scope: vulnerabilities in `workflow-definition-service`'s or the
future execution service's own decode/validation logic — that's each
consuming service's own `SECURITY.md`.
