# Contributing to ktav (Go)

**Languages:** **English** · [Русский](CONTRIBUTING.ru.md) · [简体中文](CONTRIBUTING.zh.md)

## Core rules

### 1. Every bug fix ships with a regression test

When you find a bug, **before fixing it**, write a test that reproduces
it — the test **must fail on `main`** and pass after the fix. Include
both in the same PR.

Tests live at the repo root:

| File                    | Scope                                                      |
| ----------------------- | ---------------------------------------------------------- |
| `ktav_smoke_test.go`    | Loads / Dumps happy paths, bigint, error surface.          |
| `conformance_test.go`   | Cross-language conformance against `ktav-lang/spec`.       |

### 2. Don't reinvent the format in the bindings

This Go package is deliberately a thin wrapper. Parser and format
behaviour belong in the Rust crate
([`ktav-lang/rust`](https://github.com/ktav-lang/rust)) — changing it
there updates every language binding at once. Only **Go-specific
ergonomics** (type mapping, purego loader, cache / download logic)
belong in this repo.

If your change requires a format change, start a discussion in
[`ktav-lang/spec`](https://github.com/ktav-lang/spec) first.

### 3. Public API changes note compatibility

If you touch anything exported from `ktav`, say in the PR description
whether it is:

- **semver-compatible** (additions, looser signatures, doc changes); or
- **semver-breaking** (renamed / removed items, changed signatures,
  tightened types) — in which case the version bump lands in the next
  MINOR while we are pre-1.0.

Update the CHANGELOG source units under `root-docs/CHANGELOG/` (all
three `>>>>> lang=` blocks) in the same PR and regenerate the output.

### 4. One concept per commit

Commits should be atomic: a bug fix and its test together, a feature
and its tests together, a rename on its own, a refactor on its own.
`git log --oneline` should read like a changelog. Don't prefix commit
messages with `feat:` / `fix:` — no conventional commits here.

### 5. Native library stays in lockstep with the Go module

The embedded `LibVersion` constant (`internal/native/loader.go`) **must**
match the git tag used to cut the release. If you bump the Go module
version, update `LibVersion` in the same commit. Mismatched values cause
consumers to download a library that doesn't match their code.

## Dev setup

You need:

- Go **1.21+**.
- A Rust toolchain via [`rustup`](https://rustup.rs/). MSRV: **1.71**.
- `git`.

Layout during development — the Go package loads the Rust-built
`ktav_cabi` cdylib via `purego`. Initialise the pinned spec submodule;
an adjacent spec checkout is not used:

```
ktav-lang/
└── golang/    ← this repo
    ├── spec/  ← pinned submodule with conformance fixtures
    └── crates/cabi/
```

```bash
git submodule update --init spec
cargo build --release -p ktav-cabi
```

The library is loaded from `target/release` (or
`$CARGO_TARGET_DIR/release`). Set `KTAV_LIB_PATH` only to override its path.

The Rust C ABI crate (`crates/cabi/`) depends on the published `ktav`
crate on crates.io by default. For local cross-repo edits, switch the
`workspace.dependencies.ktav` entry in `Cargo.toml` to
`{ path = "../rust" }`.

### Build

```bash
# 1. Build the native library for your host platform.
cargo build --release -p ktav-cabi

# 2. Point Go at it.
export KTAV_LIB_PATH="$PWD/target/release/libktav_cabi.so"   # Linux
#      ="$PWD/target/release/libktav_cabi.dylib"             # macOS
#      ="$PWD/target/release/ktav_cabi.dll"                  # Windows

# 3. Initialize the pinned spec corpus used by conformance tests.
git submodule update --init
```

`TestMain` uses the fixed path `spec/versions/0.8/tests`; there is no
`KTAV_SPEC_ROOT` override. The conformance guard checks the known category
manifest and fixture structure so corpus changes cannot be silently skipped.

### Test

```bash
go test -v ./...                # full suite
go test -run TestSmoke ./...    # filter by name
go test -run Conformance ./...  # spec fixtures only
```

Before any tests run, `TestMain` validates the corpus at
`spec/versions/0.8/tests`; missing files, companions, schema, or count
errors fail immediately. There is no `KTAV_SPEC_ROOT` override. The native library defaults to
`target/release` (or `$CARGO_TARGET_DIR/release`); `KTAV_LIB_PATH` is an
optional path override. Tests requiring the C ABI fail if the library is
missing; they do not skip.

### Lint

```bash
go vet -unsafeptr=false ./...
gofmt -l .                            # should print nothing
cargo fmt --all --check
cargo clippy --release -p ktav-cabi -- -D warnings
```

CI runs the same commands; run them locally before pushing.

## Architecture notes

- **Wire format.** Rust and Go exchange JSON over the FFI boundary,
  with `{"$i":"..."}` / `{"$f":"..."}` wrappers for integers / floats
  (spec 0.8: inferred from scalar form, no `:i`/`:f` markers in the
  text). The wire wrappers preserve Go integer precision; Ktav integer
  scalars outside `int64` parse as Strings.
- **Memory ownership.** Rust allocates the output buffer; Go copies it
  into a Go slice and immediately calls `ktav_free` on the Rust side.
  No buffer is long-lived across the FFI boundary.
- **Loader.** `internal/native` dlopens the shared library once per
  process (sync.Once). On Windows we go through
  `golang.org/x/sys/windows`; on Unix through `purego.Dlopen/Dlsym`.

## Release flow

Tag `v<X.Y.Z>` on `main`. The release workflow cross-compiles six
platform binaries (`linux` amd64/arm64, `darwin` amd64/arm64, `windows`
amd64/arm64), attaches them as GitHub Release assets, and the Go proxy
picks up the tag automatically. The embedded `LibVersion` constant in
`internal/native/loader.go` must match the tag — change it in the same
commit as the tag message.

## Philosophy

Ktav's motto: **"be the config's friend, not its examiner."** Before
proposing a new Go-specific feature, ask:

- Does this add a new rule the reader must hold in their head?
- Could this live in user code instead of the library?
- Does this erode the "no magic types" principle?

New rules are costly. Reject everything that doesn't clearly belong.

## Language policy

This repo participates in the org-wide three-language policy (EN / RU /
ZH). Every prose file lives in three parallel versions — see
[`ktav-lang/.github/AGENTS.md`](https://github.com/ktav-lang/.github/blob/main/AGENTS.md)
for the naming convention and the "update all three in one commit"
rule.

### License of contributions

Unless you explicitly state otherwise, any contribution intentionally
submitted for inclusion in this project by you, as defined in the
Apache-2.0 license, shall be dual-licensed as **MIT OR Apache-2.0**,
without any additional terms or conditions.
