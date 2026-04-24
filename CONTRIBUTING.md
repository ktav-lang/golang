# Contributing

**Languages:** **English** · [Русский](CONTRIBUTING.ru.md) · [简体中文](CONTRIBUTING.zh.md)

Thanks for helping. This repo contains **two things** stitched together:

- A small Rust crate (`crates/cabi/`) that wraps the reference `ktav`
  parser behind an `extern "C"` API.
- A Go package at the repo root that loads the compiled library at
  runtime through [`purego`](https://github.com/ebitengine/purego).

## Local development

Prerequisites: Rust (stable), Go 1.21+, and git.

```bash
# 1. Build the native library for your host platform.
cargo build --release -p ktav-cabi

# 2. Point Go at it and run tests.
export KTAV_LIB_PATH="$PWD/target/release/libktav_cabi.so"   # Linux
#      ="$PWD/target/release/libktav_cabi.dylib"             # macOS
#      ="$PWD/target/release/ktav_cabi.dll"                  # Windows

# 3. For conformance tests, point at the spec repo (git submodule).
export KTAV_SPEC_ROOT="$PWD/spec/versions/0.1/tests"

go test ./...
```

`git submodule update --init` pulls the shared spec fixtures.

## Architecture notes

- **Wire format.** Rust and Go exchange JSON, with `{"$i":"..."}` /
  `{"$f":"..."}` wrappers for typed integers / floats. This preserves
  arbitrary precision and the `:i` vs `:f` distinction through the FFI
  boundary.
- **Memory.** Rust allocates the output buffer; Go copies into a Go
  slice and immediately calls `ktav_free` on the Rust side. No buffer
  is long-lived across the FFI boundary.
- **Loader.** On the Go side, `internal/native` dlopens the shared
  library once per process (sync.Once). On Windows we go through
  `golang.org/x/sys/windows`; on Unix through `purego.Dlopen/Dlsym`.

## Releases

Tag `v<X.Y.Z>` on `main`. The release workflow cross-compiles the six
platform binaries, attaches them as GitHub Release assets, and Go proxy
picks up the tag automatically. The embedded `LibVersion` constant in
`internal/native/loader.go` must match the tag — change it in the same
commit as the tag message.

## License

By contributing you agree that your code is released under the MIT
license (see [LICENSE](LICENSE)).
