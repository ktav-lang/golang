# Changelog

**Languages:** **English** · [Русский](CHANGELOG.ru.md) · [简体中文](CHANGELOG.zh.md)

All notable changes to the Go binding are tracked here. Format based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow
[Semantic Versioning](https://semver.org/) with the pre-1.0 convention
that a MINOR bump is breaking.

This changelog tracks **binding releases**, not changes to the Ktav format
itself — for the latter see
[`ktav-lang/spec`](https://github.com/ktav-lang/spec/blob/main/CHANGELOG.md).

## [0.6.1] — 2026-06-05

- Docs: rewrite all README examples to spec 0.6 syntax (bare numbers instead of removed `:i`/`:f` markers; `##` comments instead of `#`).

## 0.6.0 — 2026-06-01

Sync to Ktav 0.6.0 — keys now support escaping.

### Added

- Keys process the full §3.7 escape set, with two new escapes:
  - `\.` → `.` (literal dot — does **not** split a dotted path)
  - `\:` → `:` (literal colon — does **not** act as the key/value separator)
- Examples: `a\.b: v` → `{"a.b": "v"}`, `a\:b: v` → `{"a:b": "v"}`,
  `x.y\.z: v` → `{"x": {"y.z": "v"}}`.

### Breaking

- A literal backslash inside a key now requires `\\` (previously `\` in
  a key was a plain byte). Rare in practice; per pre-1.0 SemVer this is
  a MINOR bump.

### Changed

- Tracks ktav-rust 0.6.0 / Ktav spec 0.6.0. Binding source unchanged —
  the escape change is internal to the Rust core and transparent across
  the purego FFI boundary.

---

## 0.5.0 — 2026-05-28

### Breaking

- **Spec 0.5.0**: typed markers `:i` / `:f` no longer exist. Numbers
  are inferred from the scalar body's lexical form (`42` → Integer,
  `3.14` → Float). Documents written for spec 0.1.x with explicit
  typed markers parse differently and must be updated.
- **`##` comments**: single `#` is now a literal character; comments
  require `##`. Update any Ktav source that used `# comment`.
- **Float normalisation**: Float values are stored in canonical
  shortest-decimal form (no underscores, no leading `+`). Byte-for-byte
  comparison against old serialised output may fail.
- **C ABI now exports six symbols** — `ktav_emit_canonical` is added;
  pinning `KTAV_LIB_PATH` to a pre-0.5.0 binary will fail with a
  missing symbol error.

### Added

- **`EmitCanonical(v any) (string, error)`** — renders a Go value as
  canonical Ktav (spec § 5.9): byte-deterministic output, no inline
  compounds, canonical integer / float normalisation. Two calls with
  identical inputs always produce identical bytes.
- **`TestConformanceCanonical`** — new conformance suite that verifies
  `EmitCanonical` output matches every `.canonical.ktav` oracle in the
  spec fixtures.

### Changed

- **Picked up `ktav 0.5.0`** — tracks the upstream Rust crate's
  spec 0.5 implementation: inferred numeric types, `##` comments,
  `emit_canonical` API. Spec submodule synced to tag `v0.5.0`. See the
  [`ktav` crate CHANGELOG](https://github.com/ktav-lang/rust/blob/main/CHANGELOG.md#050)
  for the full delta.
- **License changed to `MIT OR Apache-2.0`** — matching the rest of the
  `ktav-lang` ecosystem. `LICENSE` is renamed to `LICENSE-MIT`;
  `LICENSE-APACHE` is added. The SPDX expression in `Cargo.toml` is
  updated accordingly.
- **Conformance test suite updated to spec 0.5 fixtures** — path
  `spec/versions/0.5/tests`; `.canonical.ktav` files are excluded from
  the JSON-oracle test and handled by the new canonical suite.


## 0.3.1 — 2026-05-10

### Added

- **Top-level Array support** (spec 0.1.1, § 5.0.1) — `Loads` now
  returns a `[]any` when the document's first content line is an
  array-item line (bare scalar, typed marker, lone `{`/`[`, or
  multi-line opener). Previously top-level Arrays were rejected.
- **`Dumps` accepts top-level arrays** — pass any slice (`[]any`,
  `[]string`, etc.) and the rendered Ktav has bare item-per-line at
  the top, no surrounding `[...]`.
- **`DumpsForceStrings(v any) (string, error)`** — renders a Go value
  as Ktav with every scalar coerced to a String (typed integers,
  typed floats, booleans, null are flattened to their textual form
  via the raw-marker `::`). Compounds preserve their structure.
  The output round-trips back through `Loads` as the same set of
  String scalars — useful for environments or downstream consumers
  that don't understand the `:i` / `:f` typed markers.

### Changed

- **Picked up `ktav 0.3.1`** — tracks the upstream Rust crate's
  top-level Array support and `to_string_force_strings` API. Spec
  submodule synced to `7256816` (spec 0.1.1). See the
  [`ktav` crate CHANGELOG](https://github.com/ktav-lang/rust/blob/main/CHANGELOG.md#031--2026-05-10)
  for the full delta.
- **C ABI now exports five symbols** — added `ktav_dumps_force_strings`
  alongside the existing `ktav_loads` / `ktav_dumps` / `ktav_free` /
  `ktav_version`. The Go loader binds all five at first use; pinning
  `KTAV_LIB_PATH` to a pre-0.3.1 binary will fail with a missing
  symbol error.


## 0.3.0 — 2026-05-08

### Changed

- **Picked up `ktav 0.3.0`** — tracks ktav 0.3.0 (paren-string handling
  tightened: inline `(...)` paren strings are now invalid, must use
  multi-line form). Spec submodule synced to `46d94a7`. See the
  [`ktav` crate CHANGELOG](https://github.com/ktav-lang/rust/blob/main/CHANGELOG.md#030--2026-05-08)
  for the full delta.


## 0.2.0 — 2026-05-07

### Changed (breaking)

- **Picked up `ktav 0.2.0`** — multi-line strings now serialize in the
  indented stripped `( ... )` form by default (verbatim `(( ... ))`
  remains as fallback for content with leading whitespace or sole-`)`
  lines). `:f 42` accepts integer literals (parsed as `42.0`).
  See the
  [`ktav` crate CHANGELOG](https://github.com/ktav-lang/rust/blob/main/CHANGELOG.md#020--2026-05-07)
  for the full delta.

  Code comparing serialized output byte-for-byte to a baked-in
  `((...))` literal must be updated. Round-trip is unchanged.

### Spec

- spec submodule synced (typed_float_without_decimal moved invalid →
  valid/typed_float_integer_body).


## 0.1.2 — 2026-05-03

### Changed

- **Picked up `ktav 0.1.5`** — the upstream Rust crate now exposes
  `Error::Structured(ErrorKind)` with byte-offset spans, retroactive
  `#[non_exhaustive]` on the error enums, and a public `ktav::thin`
  event-based parser. The Go binding's user-visible behaviour is
  unchanged: returned `error` values carry the same human-readable
  message (Display strings for the seven canonical categories are
  byte-identical to ktav 0.1.4 — verified by ktav's own pinning
  tests). Mapping `ktav::ErrorKind` to typed Go error values
  (so callers can `errors.As(err, &ktav.MissingSeparatorSpaceError{})`
  etc.) is separate follow-up work tracked in the workspace's
  [`STRUCTURED_ERRORS.md`](https://github.com/ktav-lang/.github/blob/main/STRUCTURED_ERRORS.md).

`go get github.com/ktav-lang/golang@v0.1.2`.

## 0.1.1 — 2026-04-26

### Changed

- **Picked up `ktav 0.1.4`** — the upstream Rust crate's untyped
  `parse() → Value` path (which `cabi` uses) is now ~30% faster on
  small documents and ~13% faster on large ones, just from a one-
  line `Frame::Object` capacity tweak (4 → 8). Every `ktav.Loads`
  call benefits transparently.

## 0.1.0 — first public release

First release. Targets **Ktav format 0.1**.

### Module path

Published as `github.com/ktav-lang/golang`. `go get` picks up the Go
proxy automatically after the `v0.1.0` git tag lands.

### Public API

- `Loads(s string) (any, error)` — parse a Ktav document.
- `LoadsInto(s string, target any) error` — parse into `target` via
  `encoding/json`.
- `Dumps(v any) (string, error)` — render a Go value as Ktav text.
- `Error` — typed parse / render error surfaced through `errors.As`.

### Architecture

- **Native core** — the reference Rust `ktav` crate, wrapped with a tiny
  `extern "C"` C ABI (`crates/cabi`) and distributed as a prebuilt
  `.so` / `.dylib` / `.dll`.
- **Go loader** — `purego` (no cgo): the library is resolved at first
  call from `$KTAV_LIB_PATH` or downloaded once into `UserCacheDir`
  from the matching GitHub Release asset.
- **Wire format** — JSON between Rust and Go, with `{"$i":"..."}` /
  `{"$f":"..."}` tagged wrappers for lossless typed-integer / typed-float
  round-trips and arbitrary-precision integers (`*big.Int`).

### Type mapping

| Ktav             | Go                                              |
| ---------------- | ----------------------------------------------- |
| `null`           | `nil`                                           |
| `true` / `false` | `bool`                                          |
| `:i <digits>`    | `int64` if it fits, else `*big.Int`             |
| `:f <number>`    | `float64`                                       |
| bare scalar      | `string`                                        |
| `[ ... ]`        | `[]any`                                         |
| `{ ... }`        | `map[string]any` (insertion order preserved)    |

### Platforms

Prebuilt native binaries ship for:

- `linux/amd64`, `linux/arm64` (glibc)
- `darwin/amd64`, `darwin/arm64`
- `windows/amd64`, `windows/arm64`

Alpine (musl) is planned for a follow-up.

### Test coverage

Runs the full Ktav 0.1 conformance suite (all `valid/` and `invalid/`
fixtures) on Go 1.21 / 1.22 / 1.23 across Linux / macOS / Windows.

### Credits

Built on top of the reference `ktav` Rust crate. Dynamic loading via
[`ebitengine/purego`](https://github.com/ebitengine/purego).
