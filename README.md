# go-apperror

A Go error model for applications that follow a ports-and-adapters
architecture. Provides a standardized error taxonomy (`AppError`), a
dedicated type for remote-service failures (`RemoteError`), and the small
amount of glue needed to translate between them and the wire (HTTP/gRPC)
at the application boundary.

> **Design goal**: one in-app error type for application-domain errors,
> a *separate* type for "we called a remote service and it responded with
> failure" — distinguished by type, not by fields inside one shared type.
> Cross-cutting concerns (retry, logging, HTTP mapping) get a stable
> taxonomy to switch on; debugging keeps full access to the original
> protocol-level and remote-app-level signals.

## Install

```bash
go get github.com/ikonglong/go-apperror
```

Requires Go 1.22+.

## At a glance

```go
import "github.com/ikonglong/go-apperror"

// Construct via per-Code factories. No generic apperror.New(...) by design —
// every AppError must carry a Code from the standardized taxonomy.
// event is required (positional); message and the rest are options.
err := apperror.NewNotFound("user.lookup",
    apperror.WithMessage("user not found"),
    apperror.WithCase(apperror.NewStrCase("user_id_missing")),
)

err.Code()    // CodeNotFound  (canonical)
err.Event()   // user.lookup      (required)
err.Case()    // user_id_missing
err.Message() // user not found   (or Code.Description() if WithMessage omitted)
```

Calling a remote service and translating its failure into your taxonomy:

```go
// In a driven adapter, after receiving a 503 from user-service:
remoteErr := &apperror.RemoteError{
    Service:     "user-service",
    Operation:   "GetUser",
    Response:    &apperror.Response{StatusCode: 503, Body: rawBody},
    BodyCode:    "DEGRADED",
    BodyMessage: "service in maintenance",
    RetryAfter:  30 * time.Second,
}
// Classify into our taxonomy and wrap the RemoteError as the cause.
return apperror.NewUnavailable("user-service.GetUser",
    apperror.WithMessage("user-service degraded"),
    apperror.WithCause(remoteErr))
```

`RemoteError` is **not** a subtype of `AppError`. The canonical view is the
`AppError` that wraps it as a cause — that `AppError` is what propagates, and
`errors.As(err, &remoteErr)` recovers the remote-side root cause for a
boundary logger. See
[ERROR_HANDLING_GUIDE.md](./ERROR_HANDLING_GUIDE.md) for the full rationale.

## Core concepts

| Concept | What it answers |
|---|---|
| `Code` | (required) Category of failure, from a closed standardized taxonomy (NotFound, Unavailable, IllegalInput, ...). Picked via the factory you call. Use for cross-cutting decisions. |
| `Event` | (required) The operation/event during which the failure occurred (`"user.signup"`). Positional argument to every factory. For structured-log aggregation. Recommended format: `{namespace}.{operation}`. Empty event panics at construction time. |
| `Message` | (optional, via `WithMessage`) Human-readable description. Falls back to `Code.Description()` if omitted, so unstructured loggers still see a sensible string. |
| `Case` | (optional, via `WithCase`) The specific business condition (`"purchase_limit_exceeded"`). Orthogonal to Code. |
| `Cause` | (optional, via `WithCause`) Underlying error for `errors.Is` / `errors.As` chains. |

**When to define a Case.** Most errors don't need one — `Code` already
tells callers what category of failure happened. Reach for `Case` only
when product design or a caller has to branch on a specific business
condition inside that category, typically to render a tailored prompt
or trigger a different recovery path. For example, an `AlreadyExists`
during account creation might carry `email_taken` or `phone_taken` so
the UI can suggest "forgot your password? recover instead" rather than
a generic duplicate message. If no caller will branch on it, leave
`Case` unset.

For `RemoteError`, three layers of "code" coexist. The canonical layer
lives on the wrapping `AppError`; the other two on the `RemoteError`:

```go
appErr.Code()        // canonical: our taxonomy (CodeUnavailable)
remoteErr.StatusCode() // protocol: HTTP/RPC status (503)
remoteErr.BodyCode     // remote app: parsed from Response.Body ("DEGRADED")
```

Each layer answers a different question; log all three for full
observability fidelity.

## Code reference

The Code constants form a closed, standardized taxonomy. Construct an
AppError carrying a Code via the corresponding factory (e.g.
`NewNotFound("<event>", WithMessage("..."), ...)`). Descriptions and the
ambiguous-case rules below
are adapted from
[gRPC's status codes](https://github.com/grpc/grpc/blob/master/doc/statuscodes.md),
with names and notes adjusted for the HTTP-oriented use case.

Table entries omit the `Code` prefix for readability — the actual Go
identifiers are `CodeOK`, `CodeNotFound`, etc.

| Code | Num | HTTP | Description |
|---|---|---|---|
| `OK` | 0 | 200 | Not an error. Exists only for the Code↔HTTP mapping; no factory provided. |
| `Cancelled` | 1 | 499 | Operation was cancelled, typically by the caller (context cancelled, client disconnected). |
| `Unknown` | 2 | 500 | Unknown error; classification information is missing or the failure came from an unknown error space. |
| `IllegalInput` | 3 | 400 | Client supplied illegal input (malformed field, missing required value). |
| `Timeout` | 4 | 504 | Deadline expired before the operation could complete. For state-changing ops, may be returned even when the op later succeeds. |
| `NotFound` | 5 | 404 | A requested entity was not found. |
| `AlreadyExists` | 6 | 409 | The entity the client attempted to create already exists. |
| `PermissionDenied` | 7 | 403 | Caller is identified but lacks permission for this operation. |
| `TooManyRequests` | 8 | 429 | A resource has been exhausted: per-user quota, rate limit, per-resource budget. gRPC equivalent: `RESOURCE_EXHAUSTED`. |
| `FailedPrecondition` | 9 | 400 | System is not in the state required for the operation (e.g. non-empty `rmdir`). |
| `Conflict` | 10 | 409 | Concurrent operations conflicted (optimistic-locking version mismatch, transaction abort). gRPC equivalent: `ABORTED`. |
| `OutOfRange` | 11 | 400 | Operation attempted past a valid range (e.g. read past end of stream). |
| `Unimplemented` | 12 | 501 | Operation is defined but not implemented in this service/version. |
| `Internal` | 13 | 500 | An invariant expected by the underlying system has been broken. Reserved for serious internal errors. |
| `Unavailable` | 14 | 503 | Service is currently unavailable; typically transient — retry with backoff is reasonable (not always safe for non-idempotent ops). |
| `IllegalState` | 15 | 500 | Illegal/corrupt data in our datastore, unrecoverable data loss. Roughly gRPC's `DATA_LOSS`, slightly broader. |
| `Unauthenticated` | 16 | 401 | Request lacks valid authentication credentials for the operation. |
| `IllegalArg` | 29 | 500 | Illegal arguments passed *within our own code's layers* — a programmer-error contract violation. |
| `Unauthorized` | 30 | 401 | Credentials were valid but the session/token has expired; re-authentication is needed. |

### Choosing between similar codes

Several codes overlap in scope. The rules below resolve the ambiguity —
most of these are adapted from gRPC's guidance because the same questions
arise in any RPC-shaped system.

**`CodeFailedPrecondition` vs `CodeConflict` vs `CodeUnavailable`** —
all three reject an operation; what differs is *what the client should
do next*:

- `CodeUnavailable` — retry the same call later, with backoff.
- `CodeConflict` — retry at a higher level (e.g. restart the
  read-modify-write sequence when a test-and-set fails).
- `CodeFailedPrecondition` — do NOT retry until the system state has
  been externally fixed (e.g. an `rmdir` against a non-empty directory
  won't succeed until the contents are removed).

**`CodeIllegalInput` vs `CodeIllegalArg`** — both mean "wrong input",
distinguished by *who* supplied it:

- `CodeIllegalInput` (HTTP 400) — bad input from a *client* across an
  interface boundary; the client should fix and resubmit.
- `CodeIllegalArg` (HTTP 500) — bad args passed *within our own code* —
  a contract violation between internal layers. It's a bug.

**`CodeIllegalInput` vs `CodeOutOfRange`** — both HTTP 400:

- `CodeIllegalInput` — the input is problematic regardless of system
  state (malformed identifier, missing field).
- `CodeOutOfRange` — the problem may resolve as system state changes
  (e.g., reading past end-of-file). Prefer it when callers may iterate,
  because it makes "end of iteration" easy to detect programmatically.

**`CodeNotFound` vs `CodePermissionDenied`** — both restrict access:

- `CodeNotFound` — hide existence from an *entire class* of users
  (gradual rollout, undocumented allowlist).
- `CodePermissionDenied` — deny access for specific users within a
  class who would otherwise see the resource exists.

**`CodeUnauthenticated` vs `CodeUnauthorized`** — both HTTP 401:

- `CodeUnauthenticated` — no credentials, or credentials are
  fundamentally invalid (wrong signature, unknown subject).
- `CodeUnauthorized` — credentials *were* valid; the session
  has expired and re-authentication is needed.

## How it fits the architecture

This library is designed to pair with the ports-and-adapters style
described in [architecture.md](./architecture.md). Per-layer responsibility:

| Layer | Error responsibility |
|---|---|
| **Domain** | Constructs `AppError` for domain failures (NotFound, FailedPrecondition, OutOfRange, IllegalState). Knows nothing about HTTP/RPC. |
| **Application** | Propagates errors from below, may add context via `AddNote`, may construct use-case-level `AppError` (e.g. AlreadyExists for a duplicate signup). |
| **Driven adapter** | Owns translation of remote-service errors. When the server responded, constructs a `RemoteError` and wraps it as the cause of a canonical `AppError`; when no response was received, constructs a plain `AppError` (typically `NewUnavailable`/`NewTimeout`). Either way, an `AppError` is what propagates. |
| **Interfaces** | Catches errors at the wire boundary, maps `Code` → HTTP status via `apperror.HTTPStatusFor`, sanitizes outgoing payload. |

The full rules (when to use what, anti-patterns, code recipes per layer)
are in [ERROR_HANDLING_GUIDE.md](./ERROR_HANDLING_GUIDE.md). That document
is also designed to be referenced from a downstream app's `CLAUDE.md` so
Claude Code follows the same conventions.

## Package layout

```
github.com/ikonglong/go-apperror               # root package
├── apperror.go              AppError type, per-Code factories, options
├── code.go                  Code constants and metadata (Name, Description)
├── case.go                  Case interface, StrCase
├── httpstatus.go            HTTPStatus enum
├── http_op_mapping.go       Code ⇄ HTTP status mapping helpers
├── request_response.go      Captured wire artifacts for RemoteError
└── remoteerror.go           RemoteError type

github.com/ikonglong/go-apperror/numcase       # optional sub-package
└── ...                      Numeric Case identifiers (e.g. "1_3_1042")
                             for apps that need stable numeric error codes
```

## Development

Bootstrap a fresh clone:

```bash
make setup           # installs dev tools, enables git hooks, fetches deps
```

Common commands:

```bash
make ci              # full local CI flow: fmt-check + lint + vuln + test
make test            # tests with -race
make lint            # golangci-lint v2
make fmt             # auto-format with gofumpt
make help            # list all targets
```

### Tooling

- **Lint**: golangci-lint v2 (config: [`.golangci.yml`](./.golangci.yml));
  enables staticcheck, errcheck, govet, errorlint, gocritic, and more
- **Format**: gofumpt (stricter superset of gofmt)
- **Vulnerabilities**: govulncheck (call-graph-reachable CVEs only)
- **Pre-commit hook**: auto-formats staged `.go` files; enabled by
  `make install-hooks` (already done by `make setup`)
- **CI**: GitHub Actions ([`.github/workflows/ci.yml`](./.github/workflows/ci.yml))
  mirrors the local `make ci` flow with version-pinned tools

## Documentation

- [ERROR_HANDLING_GUIDE.md](./ERROR_HANDLING_GUIDE.md) — the full
  per-layer guide with code recipes and anti-patterns. The canonical
  reference for using this library in apps.
- [architecture.md](./architecture.md) — the architectural style this
  library is designed to support.

## License

[MIT](./LICENSE)
