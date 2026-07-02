# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] — 2026-07-02

First tagged release. Two first-class error types — `AppError` for
locally-originated failures, `RemoteError` for remote-service failures —
distinguished by type rather than by fields inside one shared struct.

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
- `Code` taxonomy (18 codes) with descriptions and HTTP status mappings,
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
