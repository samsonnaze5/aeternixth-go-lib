<!--
SYNC IMPACT REPORT
==================
Version change: (none) → 1.0.0
Bump rationale: Initial ratification of the project constitution. No prior version
exists; first establishment of governing principles is a MAJOR event.

Modified principles:
- (none — initial ratification)

Added sections:
- Core Principles (I. Code Quality & Simplicity, II. Testing Standards,
  III. API Consistency & Developer Experience, IV. Performance Requirements)
- Additional Constraints (Go toolchain, dependency policy, package boundaries)
- Development Workflow (gates, review, formatting)
- Governance (amendment, versioning, compliance)

Removed sections:
- (none — template placeholders replaced)

Templates requiring updates:
- ✅ .specify/memory/constitution.md — written
- ✅ .specify/templates/plan-template.md — Constitution Check section updated
  to reference the four principles as explicit gates
- ✅ .specify/templates/spec-template.md — no constitution-coupled fields; no change
- ✅ .specify/templates/tasks-template.md — task categories already cover
  test, polish, and performance phases; no structural change required
- ✅ README.md — no principle references; no change
- ✅ CLAUDE.md — already aligned with simplicity / goal-driven execution; no change

Follow-up TODOs:
- (none)
-->

# aeternixth-go-lib Constitution

## Core Principles

### I. Code Quality & Simplicity

Code MUST be the minimum that solves the stated problem; speculative abstractions,
configurability, or features are rejected. Every package MUST be independently
buildable (`go build ./...`) and free of cross-package dependencies beyond the
documented graph (`response → errors`; `fiber → middleware, errors, validator`;
`middleware → errors, jwt`). Exported identifiers MUST carry godoc comments with
at least one runnable example. Code MUST pass `gofmt` and `goimports` (`task format`)
before merge. Errors MUST be returned as values; `panic` is permitted only for
programmer errors (nil dereference of required deps) and never reachable from a
sentinel error path. Sentinel errors MUST follow the `Err{Description}` naming
pattern and be `errors.Is`-comparable.

**Rationale**: This is a utility library imported by many services; surface area
and cognitive load must stay low. Idiomatic Go and consistent error handling are
non-negotiable for long-term maintainability and predictable consumer behavior.

### II. Testing Standards (NON-NEGOTIABLE)

Every exported function MUST have a unit test covering the happy path and at least
one failure path. Tests MUST be table-driven where two or more inputs exercise the
same logic. `go test ./...` MUST pass before any merge to `main`. Cross-package
behavior (e.g., `middleware` + `jwt` + `fiber`) MUST be covered by integration tests
located in the consuming package or in `itestkit`. Tests MUST NOT mock packages
within this library — real implementations are used for internal collaborators;
mocking is reserved for external boundaries (HTTP clients, AWS, Gmail). For bug
fixes, a regression test reproducing the failure MUST be committed in the same
change as the fix and MUST fail without the fix applied.

**Rationale**: This library underpins financial calculations, auth, and HTTP
contracts. Mocked internal packages mask integration drift; table-driven tests with
real collaborators catch regressions that unit-only suites miss.

### III. API Consistency & Developer Experience

Public APIs across packages MUST follow consistent conventions: constructors named
`New{Type}` validate inputs and return `(*T, error)` rather than panic; converters
named `To{Type}` and `To{Type}Pointer` for symmetric pairs; sentinel errors named
`Err{Description}`. Optional or nullable values MUST be expressed via pointer types
or explicit `Null{Type}` wrappers — never via magic zero values. Generics MUST be
used only where they remove an unsafe `interface{}` assertion or eliminate a
runtime type check (current uses: `pagination.Response[T]`, `JWTService[T]`,
`GetRequestBody[T]`, `GetQueryParams[T]`). Directory-name and package-name
divergences (`aws`/`thirdpartyaws`, `jwt`/`jwtutil`, `defaults`/`defaultutil`,
`password`/`passwordutil`, `fiber`/`fiberutil`) MUST remain documented in
`CLAUDE.md` and MUST NOT be widened to new packages without amendment.

**Rationale**: The library's "users" are downstream Go developers. Predictable
naming, error shapes, and constructor signatures let a consumer guess the API
correctly on first try and reduce churn when adding new packages.

### IV. Performance Requirements

Hot-path code (parsers, formatters, rate-limit lookups, JWT signing/validation,
SQL null conversions) MUST have a `Benchmark*` test reporting allocations
(`go test -bench=. -benchmem`). The `decimal` package MUST remain zero-allocation
for string parsing and arithmetic where the upstream supports it. Concurrency-safe
types (e.g., `ratelimit.Limiter`) MUST document their thread-safety guarantee in
godoc and MUST be exercised by a `-race` test. Any code spawning goroutines MUST
provide a deterministic termination path via context cancellation or explicit
`Close`; goroutine leaks are merge-blockers. Memory-bound utilities MUST document
their growth characteristics (e.g., `ratelimit` map growth vs. cleanup policy).
Benchmark regressions greater than 10% on any covered hot path MUST be justified
in the PR description or rejected.

**Rationale**: Library code runs in every consumer's hot path. A 10% allocation
increase here multiplies across services. Explicit benchmarks and `-race`
coverage make performance and concurrency contracts visible at review time
instead of after a production incident.

## Additional Constraints

- **Go toolchain**: Minimum Go version is 1.25; the `go` directive in `go.mod`
  MUST match. Generics, `errors.Is/As`, and `any` are first-class.
- **Dependency policy**: New external module dependencies MUST be justified in
  the PR description (problem, alternatives considered, maintenance status of
  the dependency). Standard library is preferred; thin wrappers around vetted
  libraries (`shopspring/decimal`, `go-playground/validator`, `gofiber/fiber`)
  are accepted patterns.
- **Package boundaries**: The dependency graph stated in Principle I is the
  complete inter-package import set. New cross-package imports require an
  amendment.
- **License**: MIT. Any contribution MUST be compatible.

## Development Workflow

- All changes land via pull request against `main`. Direct pushes to `main` are
  prohibited.
- Required green checks before merge: `go build ./...`, `go test ./...`,
  `task format` produces no diff, and benchmarks for any modified hot-path
  package run without regression.
- At least one reviewer approval is required. The reviewer MUST verify the four
  principles against the diff and call out any unjustified deviation.
- Breaking changes (removed/renamed exports, signature changes, behavior shifts)
  MUST bump the library's MINOR or MAJOR version per semver and be listed in the
  PR description.
- Constitution compliance is part of code review. PRs that violate a principle
  without a documented justification (in `Complexity Tracking` of the plan or PR
  body) MUST be rejected or revised.

## Governance

This constitution supersedes ad-hoc practices and informal team conventions for
matters it covers. Amendments require:

1. A pull request modifying `.specify/memory/constitution.md` with the new text,
   an updated Sync Impact Report comment, and an incremented version line.
2. Documented rationale in the PR description, including migration notes for any
   principle that becomes stricter.
3. At least one reviewer approval; for MAJOR amendments, two reviewers.

Versioning policy (semantic):

- **MAJOR**: A principle is removed, a previously permitted practice is
  prohibited, or governance rules are restructured.
- **MINOR**: A new principle or section is added, or existing guidance is
  materially expanded.
- **PATCH**: Wording, typography, or clarification changes that do not alter
  meaning.

Compliance review occurs at every PR. Runtime/agent guidance lives in `CLAUDE.md`
and `README.md`; those documents MUST stay consistent with this constitution and
are updated in the same PR when an amendment changes their referenced rules.

**Version**: 1.0.0 | **Ratified**: 2026-05-10 | **Last Amended**: 2026-05-10
