# Releasing go-apperror

How versions are numbered and how a release is cut. This is a single-module
repository (module path `github.com/ikonglong/go-apperror`, declared in the
root `go.mod`); the `numcase` sub-package ships under the same module
version, so there is exactly one version line to manage and tags are plain
`vX.Y.Z` with no path prefix.

---

## 1. Versioning strategy

We follow [Semantic Versioning](https://semver.org) as Go modules
interpret it, and derive the bump from the
[Conventional Commits](https://www.conventionalcommits.org) that landed
since the previous tag.

### 1.1 Commit type → version bump

Review the range first:

```bash
git log $(git describe --tags --abbrev=0)..HEAD --oneline
```

Then map the **highest-impact** change in that range:

| Change in the range | Bump (v1+) | Bump while v0 (see §1.2) |
|---|---|---|
| Any `!` / `BREAKING CHANGE:` footer | **major** | **minor** |
| `feat:` (new, backward-compatible API) | **minor** | **patch** |
| `fix:` / `perf:` (backward-compatible) | **patch** | **patch** |
| `docs:` / `test:` / `chore:` / `ci:` / non-breaking `refactor:` | no release on its own — rolls into the next release | same |

Raising the minimum Go version (the `go` directive in `go.mod`, currently
`go 1.22`) is a **soft breaking change** for consumers on older toolchains:
treat it as at least a **minor** bump and call it out in the release notes.

### 1.2 The v0 phase (where we are now)

Under SemVer, major version `0` means the public API is still settling and
*may* change at any time. The practical convention this repo uses while on
`0.y.z`:

- **breaking change → bump the minor** (`0.1.0` → `0.2.0`)
- **feature or fix → bump the patch** (`0.1.0` → `0.1.1`)

Be aware of the tooling consequence: for a v0 module, `go get -u` will pull
the latest `0.x` it can find, **including a breaking `0.minor` bump**,
because everything under major 0 is treated as one compatibility line.
That is the accepted cost of v0 and the reason to graduate to v1 before
many external consumers depend on you.

### 1.3 Graduating to v1.0.0

Cut `v1.0.0` when you are ready to **promise backward compatibility** for
the exported API (`AppError`, `RemoteError`, the `Code` taxonomy, the
factory functions, `numcase`). From v1 on, `go get -u` stays within `1.x`,
so consumers get fixes and features without surprise breakage.

### 1.4 Major versions (v2 and beyond)

A breaking release past v1 is expensive in Go: major version ≥ 2 requires
[semantic import versioning](https://go.dev/ref/mod#major-version-suffixes)
— the module path gains a `/v2` suffix (`github.com/ikonglong/go-apperror/v2`)
and **every importing file must change its import path**. (The repo already
lives with this rule for tooling: see the golangci-lint `/v2` note in the
`Makefile`.)

Consequences:

- Prefer **deprecation over removal** within a major line. Keep the old
  symbol, add `// Deprecated: use X instead.`, introduce the replacement,
  and only delete at the next major.
- **Batch** unavoidable breaking changes so you cross the v2 boundary once,
  not repeatedly.

> The `AddErrCtx` → `AddNote` rename (`refactor!:`) was done while still on
> v0, where breaking changes are allowed without a path bump. The same
> change after v1 would have required keeping an `AddErrCtx` shim marked
> `// Deprecated:` forwarding to `AddNote`, removed only at v2.

### 1.5 Pre-release tags

For a release you want to validate before blessing it, use a pre-release
suffix: `v1.0.0-rc.1`, `v0.4.0-beta.1`. Go orders these *below* the final
(`v1.0.0-rc.1` < `v1.0.0`) and `go get` will not select a pre-release
unless asked for explicitly, so they are safe to publish for a consumer to
opt into.

### 1.6 Tags are immutable

**Never move or delete a published tag.** The Go module proxy
(`proxy.golang.org`) and checksum database (`sum.golang.org`) cache the
tag → commit → content hash permanently the first time anyone fetches it.
Re-pointing a tag does not propagate and will trip `SECURITY ERROR:
checksum mismatch` for consumers. To correct a bad release, **publish a new
patch** (e.g. `v0.4.1`).

---

## 2. Release process

### 2.1 Pre-flight

1. On `main`, fully synced, clean working tree:
   ```bash
   git switch main && git pull --ff-only && git status
   ```
2. Full local check green (same suite as CI and the pre-push hook):
   ```bash
   make ci        # fmt-check + lint + vuln + test -race
   ```
   Confirm the [CI run](.github/workflows/ci.yml) on `main` is also green.
3. Pick the version per §1 from the commit range.
4. (Recommended) update `CHANGELOG.md` — see §3.

### 2.2 Tag and push

Use an **annotated** tag (carries a message, author, and date; signed `-s`
if you publish signed releases):

```bash
git tag -a v0.4.0 -m "v0.4.0"
git push origin v0.4.0
```

Pushing the tag triggers the CI `push`/tag workflow and makes the version
fetchable.

### 2.3 Verify the release is live

```bash
# Forces the proxy to fetch and register the version.
go list -m github.com/ikonglong/go-apperror@v0.4.0
```

Optionally prime `pkg.go.dev` by opening
`https://pkg.go.dev/github.com/ikonglong/go-apperror@v0.4.0`.

### 2.4 Consume the release

In a depending module (e.g. `go-app-template`):

```bash
go get github.com/ikonglong/go-apperror@v0.4.0
go mod tidy
```

If that module used a local `replace github.com/ikonglong/go-apperror =>
../go-apperror` during development, **remove it now** so the build resolves
the published, checksum-verified version.

---

## 3. Changelog

The project maintains a [CHANGELOG.md](./CHANGELOG.md) in
[Keep a Changelog](https://keepachangelog.com) format. Update it as
part of every release — add a new `## [version] — YYYY-MM-DD` section
at the top. Group entries under **Added**, **Changed**, **Removed**,
and **Fixed**. Call out **Breaking changes** first.

---

## 4. First-release recommendation (current state)

The module is still **untagged** — consumers resolve a pseudo-version
pinned to a commit. The [CHANGELOG](./CHANGELOG.md) already documents
the planned `v0.1.0` baseline.

**Recommended:** cut **`v0.1.0`** now as the first real tag — a clean,
honest "pre-1.0, API may still move" baseline. Graduate to **`v1.0.0`**
once the API has been exercised in production and you are ready to
promise compatibility.

**Alternative:** if you already consider the surface stable and want the
production app on a 1.x line (so routine `go get -u` can never pull a
breaking change), tag **`v1.0.0`** directly.

| | Start at `v0.1.0` | Start at `v1.0.0` |
|---|---|---|
| API still expected to move | ✅ honest signal | ❌ implies a stability promise |
| `go get -u` safety for consumers | ⚠️ can pull breaking `0.x` minors | ✅ stays within `1.x` |
| Cost of the next breaking change | minor bump | requires `/v2` path |

Given the API has been recently refined, `v0.1.0` now → `v1.0.0`
after it settles is the lower-risk path.

---

## 5. Cheat sheet

```bash
# 1. what changed since the last tag?
git log $(git describe --tags --abbrev=0 2>/dev/null)..HEAD --oneline

# 2. green light
make ci

# 3. tag (annotated) + push
git tag -a vX.Y.Z -m "vX.Y.Z" && git push origin vX.Y.Z

# 4. confirm it resolves
go list -m github.com/ikonglong/go-apperror@vX.Y.Z
```

Bump rule of thumb: **breaking → minor (v0) / major (v1+); feat → patch
(v0) / minor (v1+); fix → patch.** Never re-tag; fix forward.
