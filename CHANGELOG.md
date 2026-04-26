# Changelog

**Languages:** **English** · [Русский](CHANGELOG.ru.md) · [简体中文](CHANGELOG.zh.md)

All notable changes to the Go binding are tracked here. Format based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow
[Semantic Versioning](https://semver.org/) with the pre-1.0 convention
that a MINOR bump is breaking.

This changelog tracks **binding releases**, not changes to the Ktav format
itself — for the latter see
[`ktav-lang/spec`](https://github.com/ktav-lang/spec/blob/main/CHANGELOG.md).

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
