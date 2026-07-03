# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.2] — 2026-07-04

### Added

- `RemoteWithMessage`, `RemoteWithCase`, `RemoteWithDetails`, and
  `RemoteWithCause` option constructors for `RemoteError`, matching the
  option surface of `AppError`. Inline `RemoteOption` closures still work
  but are no longer necessary for common cases.

### Changed

- RemoteError construction doc: inline closure examples replaced with
  `RemoteWithCause`.
- README: added `Details` row to Core concepts table; option names now
  show both `WithXxx` and `RemoteWithXxx` forms.
- README RemoteError quick-start: added transport-failure example using
  `RemoteWithCause`.

## [0.1.1] — 2026-07-04

Documentation improvements aligned with the
[enterprise application error handling guide](https://my.feishu.cn/docx/Md8hdR4aGoPqECx68YscJnJGnec).

### Added

- "Fault" column to the README code reference table, showing each code's
  implicit responsibility (Client or Server).
- `// Fault: Client` / `// Fault: Server` annotations on every `Code`
  constant for IDE hover discoverability.
- `# Fault responsibility` sections to the `AppError` and `RemoteError`
  type docs, explaining the two responsibility contexts (app-as-server
  vs. app-as-client) and how they relate.
### Changed

- README guide links now point to the authoritative Feishu design document.
- RemoteError doc: "Canonical" view label → "Taxonomy" for consistency
  with the project-wide removal of the "canonical" term.
- RemoteError conventions comment: removed the ambiguous "neither is an
  error" phrasing.
- RELEASING.md: updated §3 to reflect the existing `CHANGELOG.md`; updated
  §4 status and version examples.
### Fixed

- `message` field reference in `apperror.go` constructor doc now uses
  backtick formatting, matching the `event` field.
- CHANGELOG code count clarified from "18 codes" to "18 error codes
  + `CodeOK`".

## [0.1.0] — 2026-07-02

First tagged release. Two first-class error types — `AppError` for
locally-originated failures, `RemoteError` for remote-service failures —
distinguished by type rather than by fields inside one shared struct.

Designed in concert with the
[enterprise application error handling guide](https://my.feishu.cn/docx/Md8hdR4aGoPqECx68YscJnJGnec).
The `Code` taxonomy and processing steps (translate → wrap → expose) are
defined there; this library is the reference implementation.

### Added

- `AppError` type with 18 per-`Code` factory functions (`NewNotFound`,
  `NewInternal`, …). No generic constructor — picking a factory IS the
  classification.
- `RemoteError` type with 18 parallel factory functions
  (`NewRemoteUnavailable`, …). Two evidence paths: `WithErrResp` for
  received responses, inline `RemoteOption` closure for transport errors.
- `RemoteErrorResp` DTO for normalising parsed remote error responses.
- Required `event` field on both error types (panics on empty) for
  structured-log aggregation.
- Functional options: `WithMessage`, `WithCase`, `WithDetails`, `WithCause`.
- `AddNote` for prepending context without changing error identity.
- `StackTrace` captured at construction time on both error types.
- `FlatMessage` for walking the error chain into a single-line
  human-readable summary.
- `Code` taxonomy (18 error codes + `CodeOK`) with descriptions and HTTP status mappings,
  adapted from gRPC's status codes.
- `Case` interface and `StrCase` for fine-grained business-condition
  discrimination.
- `HTTPStatus` enum and `Code ↔ HTTPStatus` mapping helpers
  (`HTTPStatusFor`, `CodeFor`).
- `numcase` sub-package for numeric `Case` identifiers.
- Dev tooling: `golangci-lint` v2, `gofumpt`, `govulncheck`, pre-commit
  and pre-push hooks, GitHub Actions CI.

### Changed

- `OpCancelled` → `Cancelled`, `UnknownError` → `Unknown`,
  `OpConflict` → `Conflict`, `InternalError` → `Internal`,
  `AuthorizationExpired` → `Unauthorized`.
- `AddErrCtx` → `AddNote`.
- `OpCodeFor` → `CodeFor`.

### Removed

- `StatusMethodNotAllowed` (no `Code` mapped to it).
- Generic `apperror.New(…)` constructor — callers must use per-Code factories.
